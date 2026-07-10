---
id: I006
title: Fleet sweep to gen 6 + supersede conflated memory entries
severity: med
status: open
affects: []
blocked-by: [I005]
parent: [I001]
labels: [ready-for-agent]
execution-mode: inline
tier: routine
effort: medium
risk-triggers: []
review-tier: n/a
---

## Parent

I001 — rollout stage 2 of `docs/specs/2026-07-09-model-routing-design.md` (D10).

## What to build

After one real build has exercised the machinery end-to-end (praxis I001 is
the natural candidate — external gate, not part of this ticket set), the
remaining gen-5 fleet repos update to gen 6 via the standard
dry-run-diff-then-write flow. Persistent-memory entries that conflate
ultracode with subagent-driven are superseded with the pinned CONTEXT.md
vocabulary, dated, in the same writeback.

## Acceptance criteria

- [ ] Gate confirmed: one real build ran annotated tickets → routed dispatches → `spine audit routing` at verify
- [ ] All fleet repos report generation 6 (`spine doctor`)
- [ ] Memory writeback supersedes the ultracode/subagent-driven conflations (dated, stale entries named)

## Blocked by

- I005 — and the external live-exercise gate above.
