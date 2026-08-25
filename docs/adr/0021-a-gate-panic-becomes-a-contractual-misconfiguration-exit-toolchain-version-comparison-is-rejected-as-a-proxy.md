---
id: "0021"
title: "A gate panic becomes a contractual misconfiguration exit; toolchain version comparison is rejected as a proxy"
status: Accepted
date: 2026-08-24
---

# 0021: A gate panic becomes a contractual misconfiguration exit; toolchain version comparison is rejected as a proxy

## Context

`gate.Run` documents a three-value exit contract — 0 pass, 1 findings, 2
misconfiguration — and owns it in one place: a check class returns an error,
`Run` prints it to stderr and returns 2 without reaching `emit`
(`internal/gate/gate.go:315`). The contract holds for every failure a class
reports as an *error*. It does not hold for a panic.

Two classes reach code that can panic. `dead-code-callgraph` and
`deferred-cleanup-errcheck` both call `loadModule`, which type-checks the
audited module against an export-data importer built from `go/importer` and the
stdlib `gcimporter` **compiled into the spine binary** (ADR 0001: stdlib only,
no `golang.org/x/tools`). That importer decodes only up to the export-data
format version its own build toolchain knew. When the machine's Go is upgraded
after spine was installed, the build cache holds newer export data and
`gcimporter` panics rather than returning an error:

```
panic: cannot decode "bytes", export data version 4 is greater than maximum
supported version 2 [recovered, repanicked]
```

Observed live on 2026-08-24 (I107): `~/bin/spine` built at `b417d73` with Go
1.26.7, machine toolchain since upgraded to 1.27.0. `maipipe run full` #20 at
`ebc73f4` failed both classes on a **docs-only** commit while the other five
stages passed; `make install` cleared it and #21 passed unchanged otherwise.

The failure is doubly misleading. The operator sees a raw Go stack trace instead
of a diagnosable condition, and nothing in it says "rebuild spine". And because
only the two `go/types`-loading classes are affected, a lane fails *partially*
and reads as a code defect in whatever change is in flight — a false accusation
against an innocent commit.

The exit code was never the problem, but not for the reason it appears: the 2 an
operator sees today is the Go runtime's exit status for an unrecovered panic,
not `Run`'s misconfiguration code. `Run` never finishes. The correct exit is
reached by accident, and so is the absence of a results document — the process
dies before `emit` can write `$MAIPIPE_RESULTS`. Both guarantees are
coincidences of where the process happens to die.

I107 proposed, as "cheaper and stronger", comparing `runtime.Version()` against
the `go` on PATH before loading and refusing up front. That comparison is a
proxy for the wrong thing. The panic is keyed to the **export-data format
version**, which does not change on every Go release; a 1.26.7-built binary
reads a 1.26.8-populated cache without complaint. A version-string comparison
would refuse that working configuration, converting a healthy lane into a red
one — the same class of false accusation the ticket exists to remove, moved one
layer earlier and made unconditional.

## Decision

**A panic inside a gate check is converted into the misconfiguration path, not
allowed to terminate the process.** Two seams:

1. `loadModule` recovers and classifies. A panic whose value identifies an
   export-data version mismatch becomes an actionable error naming the verbatim
   panic text, the toolchain the binary was built with (`runtime.Version()`),
   the toolchain on PATH (`go env GOVERSION`, degrading to omission if the probe
   fails), and the remedy (`make install`). Any other panic becomes an internal
   error naming the package under check and carrying the panic value and stack
   verbatim.
2. `gate.Run` wraps every check invocation in a blanket recover of the same
   shape, so the exit contract is true for all present and future check classes
   rather than for the two this ticket happened to touch.

Both paths return an `error`, so they inherit the existing contract: stderr, exit
2, `emit` unreached, no `$MAIPIPE_RESULTS` file written. The guarantees stop
being coincidences.

**Version comparison is rejected as a detection mechanism.** The gate keys on the
real condition — the importer actually failed to decode — and never on a proxy
for it. Recovering the panic cannot false-positive; comparing version strings
can, and would.

Genuine type-check failures are untouched. A module that does not type-check
returns an error today and never panics, so no recover is on that path. A
recovered panic must never be reported in the vocabulary of a type-check
failure, and a non-export-data panic must never be laundered into "rebuild your
binary": the two message classes stay distinguishable to the operator.

## Consequences

- Recovering panics reads as an anti-pattern in Go and will invite "fixing".
  It is deliberate: `Run` publishes an exit-code contract to maipipe, and a
  process that dies mid-check cannot honour it. The recover is what makes the
  documented contract true rather than accidental.
- A panic in any check class now costs a stack trace on stderr and exit 2
  instead of a crash. Nothing is silenced — the internal-error class prints the
  panic value and stack verbatim.
- Exit 2 writes nothing, which means a **stale** results file from a previous
  run at `$MAIPIPE_RESULTS` is left in place. The promise is "writes
  nothing", not "the path is clean".
- The operator gets no warning of toolchain skew until a gate stage fires.
  Earlier notice is possible as a non-blocking `spine doctor` advisory, where
  the version-string proxy is tolerable because it cannot fail a lane; that is
  filed as I108 and deliberately not built here.
- Every future check class inherits the Run-level net without doing anything.

## Related

- I107 — the ticket, and the live evidence above.
- I108 — the deferred `spine doctor` toolchain-skew advisory.
- I084 — added the two `go/types`-loading classes.
- ADR 0001 — stdlib-only loader, which is why the importer is frozen at the
  binary's build toolchain in the first place.
