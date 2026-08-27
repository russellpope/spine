---
id: I044
title: unattributed-transcript verdict + source-file naming in details
severity: med
status: fixed
affects: [audit, I009]
blocked-by: [I042]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Design D24. New warn-level verdict `unattributed-transcript`, same
non-blocking severity band as `no-transcript`: repo-scoped, ticket-relevant
codex material exists but none met attribution (guardian-only matches,
token absent from the opening message, orchestrator-only mentions). The
detail names what was found, why it was excluded, and the source transcript
file. `no-transcript` narrows to mean literally nothing found, and its
wording stops claiming "no dispatch or transcript evidence" when near-miss
material exists.

Every judged codex verdict's detail names its source transcript file — the
I008 requirement (silent-descent names its source) satisfied as a special
case. Found-but-unusable is not nothing-found.

Note from I041 review (referred-Q3): thread_spawn actuals link by ROOT
session id only — that granularity is all I009's facts support. When more
than one dispatch under a single root names distinct tickets and the merged
actuals differ, the detail line should disclose the coarse linkage so a
surprising verdict is diagnosable at a glance.

Ratified at I044 review: "merged actuals differ" is implemented as "the
shared root's linked actual superseded the individually declared alias"
(linked[root] true with ≥2 distinct tickets under the root). A literal
cross-ticket differ check is near-dead code — root-keyed linking gives
tickets sharing a root identical merged actual sets by construction. The
implemented gate fires exactly on the operator-surprise case the note
exists for (declared luna/sol, actual terra visible to both).

## Acceptance criteria

- [ ] Guardian-only match yields `unattributed-transcript` with a why-excluded detail naming the file
- [ ] Mid-transcript-only token match yields `unattributed-transcript`, not `no-transcript`
- [ ] Ticket with zero scoped material still yields `no-transcript`
- [ ] Judged codex verdicts (match, descent, escalation, unmapped) name their source file in the detail
- [ ] New verdict never blocks; exit codes unchanged on all prior scenarios
- [ ] `go test ./...` green

## Blocked by

- I042

## Resolution — closed 2026-08-26 (ledger reconciliation)

Shipped; never closed. The `unattributed` verdict appears in
`internal/audit/audit.go`, `internal/audit/codex.go`,
`internal/audit/teamspawn.go`, and `internal/audit/i090_test.go`.

Closed transitively by **I048** (`fixed` 2026-07-27), which lists this ticket in
`blocked-by`. I048 also resolves this ticket's carry-forward (review Minor 3,
near-miss visibility) as **moot in practice**: M4a rows attribute via the worker
scan as `unmapped-dispatch`, which is better than `unattributed`, and no live
case surfaced where model-less spawn text was the only signal.
