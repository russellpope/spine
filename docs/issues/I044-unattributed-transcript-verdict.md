---
id: I044
title: unattributed-transcript verdict + source-file naming in details
severity: med
status: open
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

## Acceptance criteria

- [ ] Guardian-only match yields `unattributed-transcript` with a why-excluded detail naming the file
- [ ] Mid-transcript-only token match yields `unattributed-transcript`, not `no-transcript`
- [ ] Ticket with zero scoped material still yields `no-transcript`
- [ ] Judged codex verdicts (match, descent, escalation, unmapped) name their source file in the detail
- [ ] New verdict never blocks; exit codes unchanged on all prior scenarios
- [ ] `go test ./...` green

## Blocked by

- I042
