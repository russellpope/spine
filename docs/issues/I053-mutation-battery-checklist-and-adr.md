---
id: I053
title: "Mutation battery: checklist doc in docs/ + ADR 0013 (packaging)"
severity: med
status: fixed
affects: [docs]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

The grilled battery convention (research doc *Grill outcomes*, 2026-08-06) has no
normative home. The checklist must live in spine `docs/` — **not** `templates/`
(ADR 0004 makes templates a code change + generation bump + fleet refresh).

## Scope

1. `docs/mutation-battery-checklist.md`: the 10 runnable classes **with provenance
   marks intact** — 8/9 `[report-only]` (run, excluded from scored denominator),
   2/10 `[CANDIDATE]` (no probe data; graduate when a wired tree runs them);
   record format (per-class verdict matrix + required one-line distinct-cause
   summary; scalar = killed/valid-scorable-probes); reporting rules (build-breakers
   excluded and disclosed; presence required by the `/model-eval` skill's process,
   no threshold); the fixture-strength **reviewer instruction** (former class 11 —
   explicitly not a battery entry).
2. `docs/adr/0013-*.md`: packaging decision — checklist in `docs/` never
   `templates/`; runner bundled with the `/model-eval` skill, not spine and not a
   standalone repo; record rides `spine eval` opaque `stage`/`score` (ADR 0007);
   no enforcement code without a threshold.

## Acceptance

Design-doc criterion 1: docs-only diff (`git diff --stat` shows no `templates/`, no
Go changes); provenance marks present verbatim; ADR indexed per `docs/adr/README.md`
convention.

## Resolution

Shipped 2026-08-06 (mutation-battery effort, spine main `86cf362`):
`docs/mutation-battery-checklist.md` and `docs/adr/0013-mutation-battery-packaging.md`
(amended in place per owner ratification R2) landed as a docs-only diff. Evidence:
`docs/handoffs/2026-08-06-mutation-battery-shipped.md`; both files verified present
2026-08-09. Status flipped in the 2026-08-09 ledger hygiene sweep — the flip had
lagged the ship.
