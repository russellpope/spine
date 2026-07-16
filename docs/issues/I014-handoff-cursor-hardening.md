---
id: I014
title: Handoff cursor hardening
severity: med
status: fixed
affects: [skills, doctor, audit]
blocked-by: []
labels: [wayfinder:grilling]
parent: I010
assignee: russell
---

## Question

How is /handoff hardened so handoffs can't drop a stage? A hook can't reliably intercept "a handoff is being written," so a hard mechanical block at write time isn't available — the incident's channel was a prose resume prompt naming an abbreviated stage path.

## Resolution

(2026-07-15, owner) **/handoff (and resume-prompt authoring) must run `spine cursor` and embed its output verbatim** — never a prose paraphrase of "what's next." Backstop: a doctor advisory check plus an `audit stages` blocking check that the newest file in `docs/handoffs/` contains a cursor block whenever a cursor exists. Prose paraphrases die; omissions get caught by the gate that already blocks. Skill-text-only rejected (the control class that failed); consuming-end-only blocking rejected (bad handoff would already be the only record).
