---
id: I055
title: "Do-not-regress block: template + dispatch-prep instruction in /model-eval"
severity: med
status: fixed
affects: [model-eval skill]
blocked-by: [I054]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Remediation rounds self-inflict regressions: ornith-35b introduced a latent
ordering bug (r1) and a firing `t.Skip` (r2); Laguna's r2 gains were "paid for by a
regression." Nothing tells a remediation dispatch what the previous round already
proved working.

## Scope

Second deliverable of the mutation-battery PRD (same skill as I054, hence
blocked-by). Add to the `/model-eval` skill:

1. The DNR template (from the research doc): generated block listing each verified
   behaviour as `file:line — behaviour — proven by <test/mutation id>`, closing
   with BOTH fixed lines: "Breaking one of these costs more than any fix below
   gains." and "Report any that you must break, and why, before you break it."
   (Amended 2026-08-06, reviewer finding RA1: the research doc's two-line close
   is normative; the earlier single-line quote here was an abbreviation.)
2. Dispatch-prep instruction: every remediation dispatch is prepended with the
   block generated from the previous round's verified criteria.

## Acceptance

Design-doc criterion 5: template present; a sample block generated from the Laguna
round history renders correctly.

## Resolution

Shipped 2026-08-06 (mutation-battery effort, deepthought main `4c06342`): DNR
template (`references/do-not-regress-template.md`, both fixed closing lines per
amendment RA1) plus the Laguna sample block
(`references/do-not-regress-example-laguna.md`) and the dispatch-prep instruction
in the `/model-eval` skill sections. Live skill files re-verified present
2026-08-09. Evidence: `docs/handoffs/2026-08-06-mutation-battery-shipped.md`.
Status flipped in the 2026-08-09 ledger hygiene sweep.
