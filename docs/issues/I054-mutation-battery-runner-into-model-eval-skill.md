---
id: I054
title: "Mutation battery: bundle runner into /model-eval skill + record convention"
severity: med
status: fixed
affects: [model-eval skill]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

The runner (`scripts/mutate.py`, `scripts/overnight/sites.sh`, batch flow) sits in
spine `scripts/` with no consumer relationship. Grill Q1/Q6: the battery is an
agent-assisted instrument for the `/model-eval` loop; the runner belongs inside the
skill (`~/.claude/skills/model-eval/`), colocated with its only consumer.

## Scope

1. Move `mutate.py` + `sites.sh` (+ a generic batch wrapper) into the skill's files.
2. Skill instruction additions:
   - Site-authoring loop: `sites.sh` candidates → agent writes the **exact literal**
     spec → standalone validation run; a `NO-SITE` row means the spec is wrong, not
     the tree.
   - Reporting requirement: every eval record carries the full verdict matrix + a
     one-line distinct-cause summary for survivors (riding `spine eval` opaque
     score — zero spine code, per ADR 0007).
3. Verify end-to-end from the new home on one wired tree; kill rate must match
   `scripts/overnight/results-20260806-090020/`.

## Acceptance

Design-doc criteria 2–4, including the **negative control**: corrupt one spec
string → the run reports a `NO-SITE` row as excluded-and-disclosed, proving the
literal-match guard fires from the skill-bundled path. Note: the skill dir is
outside spine's git — verification checks the live skill files.

## Resolution

Shipped 2026-08-06 (mutation-battery effort, deepthought main `4c06342`): runner
(`mutate.py` with `report_only` + dual rates, `sites.sh`, manifest-driven
`run-battery.sh`) bundled into the `/model-eval` skill, live via the
`~/.claude/skills/model-eval` symlink. Live skill files re-verified present
2026-08-09. Build/verify record: `docs/handoffs/2026-08-06-mutation-battery-shipped.md`
and the effort ledger `.superpowers/sdd/progress.md`. Status flipped in the
2026-08-09 ledger hygiene sweep.
