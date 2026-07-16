---
id: I015
title: Fleet sweep scope
severity: med
status: fixed
affects: [fleet]
blocked-by: []
labels: [wayfinder:grilling]
parent: I010
assignee: russell
---

## Question

Once gen 8 exists, which repos get swept — all 17 spine repos (9 on gen 5, 2 on gen 6, 6 on gen 7 as of 2026-07-15), only the ~9 active since July, or active-now with an on-touch nudge for dormant ones?

## Resolution

(2026-07-15, owner) **All 17 repos, ultima-dci-edition last** — after its reconciliation path ([I017](I017-ultima-gen8-reconciliation.md)) lands. Controls only hold if no repo is a gap, and updating a dormant notes repo is cheap (dry-run review → `--write` → commit). Runs as one overnight AFK build ticket. Partial sweeps rejected: the fleet stays split and the next adherence audit reproduces the same skew finding.
