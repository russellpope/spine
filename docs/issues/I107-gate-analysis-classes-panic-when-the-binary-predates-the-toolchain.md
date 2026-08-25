---
id: I107
title: "gate analysis classes panic with a raw stack trace when the spine binary predates the Go toolchain"
severity: med
status: fixed
affects: [I084]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: primary
---

## Problem

`spine gate go@1 dead-code-callgraph` and `spine gate go@1
deferred-cleanup-errcheck` load the module with `go/types` + the stdlib
`gcimporter` compiled into the spine binary. That importer only decodes export
data up to the version its own build toolchain knew. When the machine's Go is
upgraded after spine was installed, the build cache holds newer export data and
the checks panic:

```
$ spine gate go@1 dead-code-callgraph --dir .
panic: cannot decode "bytes", export data version 4 is greater than maximum
supported version 2 [recovered, repanicked]
...
go/internal/gcimporter.Import(...)
	/opt/homebrew/Cellar/go/1.26.7/libexec/src/go/internal/gcimporter/gcimporter.go:80
github.com/russellpope/spine/internal/gate.typeCheck(...)
	internal/gate/load.go:192
$ echo $status
2
```

Observed live on 2026-08-24: `~/bin/spine` built at `b417d73` with Go 1.26.7,
machine toolchain since upgraded to 1.27.0. `maipipe run full` #20 at `ebc73f4`
failed with `gates/dead-code-callgraph` and `gates/deferred-cleanup-errcheck`
both exit 2 on a **docs-only** commit; the other five stages passed. `make
install` fixed both (exit 0), and #21 passed unchanged otherwise.

Two things are wrong, independent of the upgrade itself:

1. The failure surfaces as a Go runtime panic and stack trace, not as a
   diagnosable misconfiguration. Exit 2 is the right code; the output is not.
   Nothing in it says "rebuild spine" — the operator has to read a
   `gcimporter` trace to work out that the binary is stale.
2. Only the two `go/types`-loading classes are affected, so a lane fails
   partially and looks like a code defect in the commit under test rather than
   an environment problem. That is a false accusation against whatever change
   happens to be in flight.

## Fix

Recover the panic at the `loadModule` / `typeCheck` seam and report it as
misconfiguration with an actionable message — name the export-data version
mismatch, the toolchain the binary was built with, the toolchain on PATH, and
the remedy (`make install` / `go install`). Exit 2 as today, no findings
emitted, no results document written.

Optionally cheaper and stronger: compare `runtime.Version()` against `go
version` on PATH before loading and refuse up front with the same message, so
the panic path is never reached in the common case.

Constraint: this must not swallow genuine type-check failures in the audited
module. A module that legitimately fails to type-check keeps today's behavior.

## Grilled 2026-08-24 — two corrections to the above

**"Exit 2 is right" is half wrong.** Today's 2 is the Go runtime's exit status
for an unrecovered panic, not `gate.Run`'s misconfiguration code — `Run` never
finishes. The absence of a results document is the same accident: the process
dies before `emit` writes `$MAIPIPE_RESULTS`. Both guarantees are
coincidences of where the process happens to die. The fix does not preserve
them; it makes them real.

**The "optionally cheaper and stronger" preflight is rejected**, not deferred.
The panic is keyed to the export-data *format* version, which does not change on
every Go release, so a 1.26.7-built binary reads a 1.26.8-populated cache fine.
A `runtime.Version()` comparison would refuse that working configuration and
turn a healthy lane red — the same false accusation this ticket exists to
remove, moved one layer earlier and made unconditional. Detection keys on the
importer actually failing. Recorded on evidence in ADR 0021.

Design: `docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md`
(D38–D44). Scope grew by one item: a blanket recover at `gate.Run` so the exit
contract holds for every check class, not only the two named here. The `spine
doctor` toolchain-skew advisory is split out as **I108**.

## Resolution

Fixed 2026-08-25, merged to `main` at `612980b` (`--no-ff`). See ADR 0021 and
`docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md` (D38–D44).

`loadModule` recovers and classifies; `gate.Run` wraps every check invocation in
a blanket recover of the same error-returning shape. Both feed the existing
`runErr != nil` branch, so the 0/1/2 contract and the "no results document"
guarantee are now real rather than consequences of where the process happened to
die. The `runtime.Version()` preflight the ticket proposed as "optionally cheaper
and stronger" was **rejected on evidence**, not deferred — the panic keys on the
export-data format version, which does not move every Go release, so the
comparison would refuse working setups.

Verified against the live artifact after `make install` rebuilt `~/bin/spine` at
the merge commit, exit codes read unpiped:

- `spine gate go@1 dead-code-callgraph --dir .` → EXIT=0, `no findings`
- `spine gate go@1 deferred-cleanup-errcheck --dir .` → EXIT=0, `no findings`
- a genuinely broken module → EXIT=2, message unchanged
  (`--dir … does not type-check: example.com/i107live: ./broken.go:3:9: undefined:
  undefinedIdentifier`), and no file written at `$MAIPIPE_RESULTS`

Reviewed independently before merge rather than by the building team: a cold
spec-review against the PRD (0 missing, 0 scope creep) and a fresh-context verify
that re-ran the negative controls from scratch instead of accepting the team's
claims. That re-run found the team's Task 2 control inert as prescribed — both
target tests stub the very seam it mutates — and reddened it one level down at
`goVersionOnPATH`. The team's one declared deviation (substituting Task 5's
control) was adjudicated first and **upheld**: the plan's own control cannot
discriminate, because a genuine type-check failure returns via `p.Error` before
any importer runs, exactly as D44 already stated. Four spec defects found this
way were corrected at `39a6972`.

## Related

- I084 — the ticket that added the two analysis classes.
- The handoff gotcha "old spine binary exits 2 on `spine gate go@1 …`" is a
  *different* stale-binary failure (unknown pack). This one has the same
  symptom (exit 2 from gate stages after a toolchain or binary change) and a
  completely different cause, which is part of why it reads as confusing.
- Correction to a standing handoff assumption: no `maipipe daemon` restart was
  needed after `make install`. Stages exec `spine` from PATH per run — run #21
  picked up the rebuilt binary with the daemon still up (it had admitted work
  for another project and refused `stop-if-idle`).
