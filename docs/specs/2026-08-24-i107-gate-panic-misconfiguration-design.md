# I107 gate panic becomes a misconfiguration exit

Status: approved (grilled 2026-08-24)
Date: 2026-08-24
Tickets: I107
Decision record: ADR 0021; respects ADRs 0001, 0015; defers the doctor advisory to I108

## Problem Statement

`spine gate go@1 dead-code-callgraph` and `spine gate go@1
deferred-cleanup-errcheck` both call `loadModule`, which type-checks the audited
module against an export-data importer built from `go/importer` and the stdlib
`gcimporter` compiled into the spine binary (ADR 0001: stdlib only). That
importer decodes only up to the export-data format version its own build
toolchain knew. Upgrade the machine's Go after installing spine and the build
cache holds newer export data than the binary can read, at which point
`gcimporter` panics instead of returning an error:

```
panic: cannot decode "bytes", export data version 4 is greater than maximum
supported version 2 [recovered, repanicked]
...
github.com/russellpope/spine/internal/gate.typeCheck(...)
	internal/gate/load.go:192
```

Observed live on 2026-08-24: `~/bin/spine` built at `b417d73` with Go 1.26.7,
machine toolchain since upgraded to 1.27.0. `maipipe run full` #20 at `ebc73f4`
failed both classes on a **docs-only** commit while the other five stages
passed. `make install` cleared it; #21 passed unchanged otherwise.

Two defects, independent of the upgrade itself:

1. The failure surfaces as a Go runtime panic and stack trace rather than a
   diagnosable condition. Nothing in the output says "rebuild spine" — the
   operator has to read a `gcimporter` trace to work it out.
2. Only the two `go/types`-loading classes are affected, so the lane fails
   *partially* and reads as a code defect in whatever change is in flight. That
   is a false accusation against an innocent commit.

The exit code is not itself the defect, but not for the reason the ticket gave.
`gate.Run` owns a three-value contract — 0 pass, 1 findings, 2 misconfiguration
— and honours it by printing a returned error to stderr and returning 2 without
reaching `emit` (`internal/gate/gate.go:315`). On the panic path `Run` never
finishes: the 2 an operator sees is the Go runtime's exit status for an
unrecovered panic. The absence of a `$MAIPIPE_RESULTS` file is the same kind
of accident — the process dies before `emit` can write it
(`internal/gate/results.go:53-65`). Both guarantees are coincidences of where
the process happens to die, and this work makes them real.

## Solution

A panic inside a gate check is converted into the misconfiguration path rather
than being allowed to terminate the process, at two seams.

`loadModule` recovers and **classifies**. A panic identifying an export-data
version mismatch becomes an actionable error naming the verbatim panic text, the
toolchain the binary was built with, the toolchain on PATH, and the remedy. Any
other panic becomes an internal error naming the package under check and
carrying the panic value and stack verbatim — nothing is silenced.

`gate.Run` wraps every check invocation in a blanket recover of the same shape,
so the exit contract is true for all present and future check classes rather
than the two this ticket happened to touch.

Both paths return an `error` and therefore inherit the existing contract:
stderr, exit 2, `emit` unreached, no results file written.

Detection keys on the **real condition** — the importer actually failed to
decode. Comparing `runtime.Version()` against the `go` on PATH before loading,
which the ticket proposed as "cheaper and stronger", is rejected: the panic is
keyed to the export-data format version, which does not change on every Go
release, so a 1.26.7-built binary reads a 1.26.8-populated cache without
complaint. A version-string comparison would refuse that working configuration
and turn a healthy lane red — the same false accusation this work removes, moved
one layer earlier and made unconditional. Rejected on evidence in ADR 0021, not
deferred.

## User Stories

1. As a repo owner whose gate stage fails after a Go upgrade, I want the output
   to name the stale binary and the remedy, so that I fix my environment instead
   of debugging the commit under test.
2. As a repo owner, I want the two affected classes to fail the same way as any
   other misconfiguration, so that a partial lane failure is legible as an
   environment problem rather than a code defect.
3. As a repo owner, I want a module that genuinely fails to type-check to behave
   exactly as it does today, so that this fix cannot hide a real defect.
4. As a repo owner, I want a panic that is *not* an export-data mismatch to keep
   its panic value and stack, so that a genuine `go/types` bug is never
   laundered into "rebuild your binary".
5. As a repo owner, I want a panic in any check class — not only these two — to
   exit 2 with a message rather than crash, so that `gate.Run`'s published
   contract is something maipipe can rely on.
6. As a repo owner, I want a failed check to write no results document, so that
   a stage that could not judge the code never reports a verdict.

## Implementation Decisions

**D38 — recover at `loadModule`.** `loadModule` recovers panics raised anywhere
beneath it, including inside `typeCheck`'s `conf.Check` where the importer runs
lazily. It returns an error; it never re-panics. The recover covers the whole
function body so the `goList` call is inside it too.

**D39 — classification.** A recovered value is an *export-data mismatch* when
its string form identifies the `gcimporter` version condition. Match on the
stable substring `export data version`, not on the surrounding wording, package
name or numbers. Everything else is an *internal error*.

**D40 — mismatch message.** Names four things: the verbatim recovered value (it
already carries both version numbers), the toolchain the binary was built with
from `runtime.Version()`, the toolchain on PATH from `go env GOVERSION`, and the
remedy `make install`. The PATH probe is a subprocess on an already-fatal path.
`go` is present for every panic that can actually reach this branch, because an
`export data version` panic comes from the importer, which runs only after
`goList` has succeeded. That ordering is a convenience, not the guarantee —
D38 deliberately puts `goList` itself inside the recover, so the probe must
stand on its own: if it fails, that clause is omitted and the rest of the
message stands; one failure never becomes two.

**D41 — internal-error message.** Names the package under check — or the `--dir`
value, when the panic precedes the per-package loop and no package has been
entered yet — the recovered value, and the stack captured at recover time
(`debug.Stack()`), verbatim. This
class is deliberately more alarming than the mismatch class and must never be
phrased as a configuration problem.

**D42 — vocabulary separation.** Neither class may reuse the existing
`--dir %s does not type-check` phrasing, which belongs to the error path a
module that genuinely fails to compile already takes. A reader must be able to
tell "your code does not compile" from "spine cannot decode this build cache"
from "spine hit a bug" by the first line alone.

**D43 — blanket recover at `gate.Run`.** `Run` wraps the `fn(abs, cfg)` and
`rfn(abs, cfg)` invocations so a panic anywhere in any class becomes `runErr`,
taking the existing stderr-and-return-2 branch. It uses D41's internal-error
shape; it does not classify, because classification needs loader context D38
has and `Run` does not. The two recovers are not redundant: D38 produces the
actionable message, D43 guarantees the contract.

**D44 — no swallowing.** Every recovered panic produces output and exit 2.
There is no path on which a recover results in exit 0, exit 1, findings, or
silence. A module that does not type-check returns an error today and never
panics, so no recover sits on that path at all.

## Testing Decisions

- An unexported seam in `loadModule` lets a test inject a panicking importer,
  so both classifications are driven without a second Go toolchain installed.
- Mismatch case: inject a panic whose text carries `export data version 4 is
  greater than maximum supported version 2`. Assert exit 2, the message names
  the panic text, `runtime.Version()`, and `make install`, and that it does not
  contain the `does not type-check` phrasing (D42).
- Internal-error case: inject a panic with unrelated text. Assert exit 2, the
  panic value and a stack are present, and the message does **not** suggest
  rebuilding.
- If cheap, a synthetic export-data blob with a bumped version byte, fed through
  the importer, as an end-to-end confirmation the D39 substring actually matches
  what `gcimporter` raises. If the blob proves fiddly, drop it — the injected
  seam is the shipped test.
- Results-file guarantee: set `$MAIPIPE_RESULTS` to a path in a temp dir,
  trigger the injected panic, assert exit 2 **and** that no file was created.
- D43: a temporary check class registered in a test that panics, driven through
  `Run`, asserting exit 2 and a message rather than a crash.
- **Negative control:** remove the D38 recover and the mismatch test must panic
  and fail; remove D43's and the class-level test must too. The guards have to
  be load-bearing, proven by observation and not by argument.
- Constraint control: a fixture module that genuinely fails to type-check keeps
  today's exact message and exit code, unchanged by this work.
- Installing a second Go toolchain (`golang.org/dl/go1.26.7`) for a true
  end-to-end reproduction is **not** done. It proves the same thing at a
  hundred times the cost and pins the suite to two Go releases.

## Out of Scope

- Comparing toolchain versions to refuse up front, in the gate. Rejected on
  evidence (ADR 0021), not deferred.
- A `spine doctor` toolchain-skew advisory. Filed as **I108**; the version-string
  proxy is tolerable there only because an advisory cannot fail a lane, and that
  is its own design question.
- Clearing a stale `$MAIPIPE_RESULTS` file left by a previous run. The
  promise is "writes nothing", not "the path is clean".
- Making the importer itself version-tolerant, or taking a
  `golang.org/x/tools` dependency to get one. ADR 0001 stands.

## Further Notes

The remedy string is `make install` because that is what the operator ran on
2026-08-24 and what the repo's own lane uses. A consumer who installed spine
another way reads it as "rebuild spine from a checkout with the current
toolchain"; naming every install path in an error message is worse than naming
the repo's.
