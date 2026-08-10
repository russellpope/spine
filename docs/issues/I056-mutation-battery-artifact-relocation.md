---
id: I056
title: "Mutation battery: relocate per-tree specs + preserve evidence"
severity: low
status: fixed
affects: [docs, scripts]
blocked-by: [I054]
execution-mode: subagent-driven
tier: mechanical
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

Per-tree mutation specs (`scripts/{opus,gpt55,laguna}.json`, `scripts/specs/*.json`)
are eval artifacts living in spine `scripts/`; they belong beside the eval records
they describe. After I054 the runner has left spine too.

## Scope

1. Move per-tree specs to the consuming repo's `docs/evals/` area, beside its eval
   records. **Never touch eval trees** in `local-model-evaluation` — records/specs
   area only. While relocating, mark the B7 and B8 entries in every spec with
   `"report_only": true` (amendment R1, 2026-08-06 — the runner distinguishes raw
   vs scorable rates via this flag).
2. Preserve reproduction evidence in spine: copy
   `scripts/overnight/results-20260806-090020/` (combined.txt + per-tree logs,
   ~100 lines) under `docs/research/` beside the research doc.
3. Update references: research doc + overnight README pointers to the runner's and
   specs' new homes; remove relocated files from spine `scripts/`.

## Acceptance

Design-doc criterion 6: spine `scripts/` carries no per-tree specs; evidence
retained under `docs/research/`; no dangling references
(`find docs -name '*.md' -print0 | xargs -0 grep -l 'scripts/specs'` is empty).

## Resolution

Shipped 2026-08-06 (mutation-battery effort): specs relocated to
local-model-evaluation `be49c9e` beside the eval records (B7/B8
`report_only: true` per amendment R1); reproduction evidence preserved at
`docs/research/2026-08-06-mutation-battery-repro/`. Re-verified 2026-08-09:
`scripts/specs/` and `scripts/*.json` absent from spine.

Noted deviation on the acceptance grep: `grep -l 'scripts/specs'` over `docs/`
is not literally empty — the two hits are the preserved
`overnight-README.md` (verbatim historical evidence this ticket's own Scope 2
requires keeping) and the design doc's descriptive text of the relocation.
Neither is a dangling reference to a live path, so the criterion's intent
(no stale pointers to relocated files) is met; the literal grep and the
preserve-evidence requirement contradict each other and are resolved in favor
of preservation. Status flipped in the 2026-08-09 ledger hygiene sweep.
