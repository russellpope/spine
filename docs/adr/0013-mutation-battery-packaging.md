---
id: "0013"
title: "Mutation battery packaging: docs-only, runner outside spine, no enforcement code"
status: Superseded by 0015
date: 2026-08-06
---

Amended 2026-08-06 (pre-merge, R2): record home corrected from stage/score fields
to the Audit/Rescore body.

# 0013: Mutation battery packaging: docs-only, runner outside spine, no enforcement code

The behavioural mutation battery (`docs/research/2026-08-05-behavioural-mutation-battery.md`,
grilled 2026-08-06) needed a packaging decision spanning four separable pieces:
where the checklist convention lives, where the runner lives, how the result rides
the eval record, and whether spine gains enforcement code. An earlier draft proposed
shipping the checklist in spine `templates/` as a "zero-code" change and giving the
runner its own standalone repo.

Decided (grill, 2026-08-06):

1. **The checklist lives in spine `docs/`, never `templates/`.** ADR 0004 makes
   `templates/` a code change: templates compile into the binary behind a single
   integer generation, so adding one is a code change plus a generation bump plus a
   fleet refresh across 17 repos. `docs/` avoids that machinery entirely. The
   checklist itself is `docs/mutation-battery-checklist.md`.
2. **The runner (`mutate.py`, `sites.sh`, the batch flow) is bundled with the
   `/model-eval` skill, not spine and not a standalone repo.** It is colocated with
   its only consumer. A standalone repo was rejected: the redistribution story had
   no named external customer, and the estate does not need a 19th repo for ~60
   lines of Python.
3. **The battery record rides the eval record's Audit / Rescore *body* — zero spine
   code and no schema change.** `stage` keeps its fixed vocabulary and `score:`
   stays the rubric total; neither is repurposed to carry battery data. Presence of
   the record is required by the `/model-eval` skill's process, not by tooling; a
   `spine doctor` presence check is explicitly out of scope (ADR 0007-consistent)
   unless a pass threshold is ever adopted.
4. **No enforcement code without a threshold.** There is no pass threshold (see the
   checklist's reporting rules), so there is nothing for enforcement code to gate on.
   Adding a `spine doctor` check ahead of a threshold would enforce presence of a
   field with no consumer for its value — deferred until a threshold decision, if
   one is ever made.

Per-tree mutation specs (JSON) live beside the eval records they describe, in the
consuming repo's `docs/evals/` area — they are eval artifacts, not spine tooling,
and do not live in spine `scripts/`.

Rejected alternatives: checklist in `templates/` (false "zero-code" framing, ruled
out by ADR 0004); a standalone runner repo (no named customer, redistribution
story didn't hold up under the grill); enforcement code ahead of a threshold
(nothing to enforce against, and conflates requiredness with tooling-checked
requiredness).
