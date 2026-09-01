---
id: I076
title: Routing-yield forward build — REVIEW record line + spine yield verb
severity: low
status: open
affects: [audit, cli, workflow]
blocked-by: []
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

Build the forward-looking half recommended by
`docs/research/2026-08-05-routing-yield-feasibility.md` (which resolved
[I052](I052-routing-yield-feedback-charter.md)): one review-time record line in
the progress-ledger grammar, spine parsing records only (never filenames), and
`spine yield` (per-repo + `--fleet`) that always prints counts and refuses
rates below a floor (n≥20 low-confidence, n≥40 stated rate). Heterogeneous
amendment: the proposed line keyed `<flavor>/<tier>` must instead carry the
**actual model id** (and harness, post-[I073](I073-flavor-to-harness-rename-migration.md)
naming) — per-family yield is precisely the evidence the equivalence pins of
[I068](I068-host-scoped-availability-and-tier-pins.md) need. Escalation
frequency needs no new record (derivable from existing ESCALATION/FALLBACK
lines).

## Accepted design

The accepted I076 design is
[the 2026-08-30 routing-yield review-record PRD](../specs/2026-08-30-routing-yield-review-record-design.md)
with its [paired implementation plan](../specs/2026-08-30-routing-yield-review-record-plan.md).
It records explicit task-gate REVIEW lines keyed by canonical harness, actual
model ID, and effective tier, plus a separate final-review series that never
changes a task-rate denominator. It does not infer outcomes from filenames or
transcripts and introduces no rating rule.

## Owner-directed batch-lane amendment (2026-08-31)

The original I073 prerequisite and its standalone exact-SHA maipipe rationale
are preserved in the accepted design and plan as historical context. The owner
has superseded that standalone-lane requirement only: I073 is now satisfied by
fixed product SHA `46b2324`, the PASS all-20 primary fleet result, closure
`dcb1c3e`, and a fresh primary post-fleet PASS. Canonical `harness`
terminology and its bounded compatibility surface are therefore available to
I076; the product contract still has no `flavor` alias.

I076 Task 1 may pass on that durable evidence. This ticket remains open for its
gated tail. After focused and full review, an independent verification, and the
final requirements attack pass, I076 closes as `fixed, pending batch ship`.
The owner-directed batch-final exact-SHA `maipipe run full --wait` runs only
after every included ticket has that fixed, blocked, or surfaced disposition,
following the final whole-branch review, routing audit, and fresh handoff. No
standalone I073 or I076 lane is required or claimed. The batch-final lane is
the ship verdict; any post-lane commit invalidates it and requires a rerun at
the new exact SHA.
