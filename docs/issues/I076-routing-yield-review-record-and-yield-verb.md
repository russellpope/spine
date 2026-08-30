---
id: I076
title: Routing-yield forward build — REVIEW record line + spine yield verb
severity: low
status: open
affects: [audit, cli, workflow]
blocked-by: [I073]
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

Implementation is blocked by [I073](I073-flavor-to-harness-rename-migration.md):
I073 must be fixed and independently verified at its exact final SHA, including
the required exact-SHA maipipe result, before I076 code starts. This ticket
remains open pending that prerequisite and implementation.
