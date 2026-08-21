---
title: "I103 pack pin attribution complete on reviewed branch"
created: 2026-08-21
handoff_ordinal: 13
---

# Handoff — I103 pack pin attribution complete on reviewed branch (2026-08-21)

## Context

I103 closes pack-pin attribution on branch `i103-pack-pin`, based on
`acc0fe8` and with approved code SHA `dbf8300`
(`dbf83009046bf4efdd1616f6af30a1148e90d2a0`). Authoritative inputs were the
[I103 implementation handoff](2026-08-21-i103-pack-pin-attribution-codex.md),
the [design](../specs/2026-08-21-i103-pack-pin-attribution-design.md) and
[plan](../specs/2026-08-21-i103-pack-pin-attribution-plan.md),
[ADR 0019](../adr/0019-the-pack-pin-rides-the-stage-run-line-not-an-env-var.md),
and the **pack pin** glossary in `CONTEXT.md` §Gate pack.

The versioned gate command now resolves an explicit `<pack>@<v>` against its
frozen per-version table. It attributes findings and summaries to that pin;
an unknown pin or class outside the pin exits 2 before writing a results
document. Bare `spine gate go <check>` is cwd-independent and resolves to the
current binary pack. The I103 routine review found the future-binary test
seam incomplete; its separate `dbf8300` fix simulates go@2, proves explicit
go@1 stays go@1, and mutation-proves a pinned finding cannot regress to
binary attribution. Routine re-review and fresh primary whole-branch review
both ended **APPROVED**.

## State (verify before relying)

- The renderer carries the pin on every managed run line, including mutation.
  The reader regards legacy bare lines as a safe stale migration, recognises
  only this repo's pinned form, and reports a foreign pin as unrecognised
  region ownership. `spine update` reports byte-only stage-definition changes
  separately from adds/removals and names the required `maipipe gate
  approve-definition` re-approval.
- Story 23 now describes both frozen class membership and attribution; I098
  records I103's follow-through; I103 is `status: fixed` with its Resolution.
- Verified at approved SHA: `gofmt -l .`, `go vet ./...`, and
  `SPINE_REQUIRE_MAIPIPE=1 make test` all passed. Candidate-built update has
  no pending generated file; it reports only the pre-existing skipped
  `docs/issues/README.md` issue-index row. Routing audit exited 0:
  I103 is `escalated-with-reason` for `cmux cluster highest-tier rule`, with
  no silent descent. `git diff --check acc0fe8...HEAD` passed.
- Live maipipe run #3 (`3ae06580-9687-444c-ad03-9eb0b30b2916`) was pinned to
  `dbf8300` and passed all seven full-profile stages: `fast/vet`, `fast/test`,
  `binary-hygiene`, `dead-code-callgraph`, `deferred-cleanup-errcheck`,
  `gitignore-control`, and `tskip`. Mutation is audit-profile and therefore
  not scheduled by `full`.
- Runs #1 and #2 were diagnostic failures: first, a test leaked findings to
  its enclosing results path; second, the daemon used an older spine that did
  not understand `go@1`. Both were fixed without a global install. The daemon
  was temporarily restarted with the candidate binary for run #3, then idle-
  stopped and restored to its normal PATH.

## Next steps

Owner: integrate this single reviewed branch from current `main`. No push or
merge was performed by this worker. Because this docs-only handoff commit
moves HEAD, the lead must re-run `maipipe run full --wait` at the final
handoff SHA; the team report will carry that final evidence.

## Gotchas

- Keep `<pack>@<v>` as the pack-pin carrier on stage run lines; do not revive
  the superseded environment-variable carrier (ADR 0019).
- A future pack must add a frozen class-table entry. Bare `go` follows the
  binary's current pack, whereas an explicit pin never does.
- A definition-only region rewrite still requires maipipe definition approval,
  even when no stage was added or removed.
<!-- spine:cursor -->
effort: i103-pack-pin-attribution
prd: docs/specs/2026-08-21-i103-pack-pin-attribution-design.md
tickets: I103
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
