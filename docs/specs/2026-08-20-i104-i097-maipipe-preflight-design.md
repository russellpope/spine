# I104/I097 maipipe preflight and gate-pack opt-out

Status: approved sequential implementation: the first fixed range/branch implements I104 only; I097 is implemented only on its descendant stacked branch, and this candidate combines both ranges
Date: 2026-08-20
Tickets: I104, followed by I097
Decision record: ADR 0018 (to be created by I104); respects ADRs 0001, 0015, and 0017

## Problem

`spine update` currently contains a hand-written TOML duplicate/balance scanner as
one half of its `maipipe.toml` preflight. It duplicates an authority already
provided by `maipipe validate`, but is neither a TOML parser nor needed when
maipipe is available. Separately, clearing a gate pack leaves its managed region
running; I097 addresses that behavior only in the descendant stacked branch.

## I104 decision and scope

Adopt I104 option B. When `gate_pack` is non-empty, `maipipe` on `PATH` is a
precondition for planning or writing `maipipe.toml`. The update plan must report
that `maipipe.toml` is skipped and why when the binary is unavailable, then apply
all other planned files and exit successfully. The skipped file remains
byte-identical. When maipipe is present, its `validate` command is the sole
pre-write preflight and an invalid candidate is refused before any writes.

Delete the spine-side structural scanner and all scanner-specific tests. Retain
the shared `maipipeOnPath` lookup and validation path. Each plan must name the
preflight that ran: `maipipe validate` or the no-binary skip.

The I104 ADR records the decision without changing ADR 0001. I104 becomes fixed
with its execution/routing metadata completed, and I096 gains a dated note that
its structural half has been removed.

## I097 scope, stacked after I104 review

I097 is approved and implemented on the descendant stacked branch after I104
passes review. Clearing a gate pack must begin with an out-of-region composition scan
for `gate-go` and `mutation-go`. If it finds one or more references, update
refuses to remove the managed region, names every affected pipeline and stage,
and leaves the file byte-identical. The spine-layout fixture is load-bearing:
removing that refusal must reproduce `composes unknown pipeline "gate-go"`.

With no out-of-region references, update plans and writes a marker-inclusive
deletion of the region. Re-running after deletion is a clean no-op. An unknown
configured pack with an existing stale region is a doctor finding, not silence.
No I097 production code, tests, or ticket resolution belong in this I104 branch.

## Acceptance criteria

- With non-empty `gate_pack` and no `maipipe`, the plan names `maipipe.toml`,
  states it was skipped because `maipipe` is unavailable, identifies that
  preflight, writes other planned files, returns 0, and never changes the file.
- With a resolvable `maipipe`, a bad rendered candidate is refused; the existing
  invalid-candidate negative controls remain meaningful.
- No structural scanner implementation or scanner-only tests remain.
- The work stays stdlib-only (ADR 0001) and preserves the ADR 0017 rule that the
  plan is the review surface for managed-region changes.
- The I104 fixed range contains no I097 production code, tests, or ticket
  resolution; the descendant I097 range implements the approved stacked scope.

## Verification

Record each red/green scoped test command. Before completion run `gofmt -l .`,
`go vet ./...`, and `SPINE_REQUIRE_MAIPIPE=1 make test` with a writable,
task-specific Go cache if necessary. Run a fresh-context spec review against
this document, `spine audit routing`, and commit the required report.
