---
id: I038
title: Team skills resolve models through spine (deepthought tree)
severity: high
status: fixed
affects: [skills, deepthought]
blocked-by: [I034]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Design D18. The team and handoff skills stop carrying model ids in prose and resolve each dispatch through the CLI. Lands in the deepthought tree; tracked in this ledger so one stage cursor covers the whole effort.

The codex-team playbook's pinned worker model becomes a per-dispatch resolution against the ticket's declared tier — this is the bug that already shipped: a single pinned worker model flattens per-tier routing so every worker runs the routine-tier model regardless of tier, under-modelling primary-tier tickets. The handoff playbook's literal lead argument resolves the same way.

The claude-team lead-model paragraph is **deleted, not updated**. It currently instructs the lead to size its own model to project difficulty and remaining credits with a floor — that is per-project primary selection performed by hand, and a per-repo override of the primary entry expresses it directly. Replacing the floor's id with a newer id would miss the point.

A spine-presence check joins the shared preflight each of these skills already runs, refusing early with an install hint exactly as they do for a missing codex binary or unsupported frontend.

A grep-style shell regression test beside the existing preflight test asserts no team skill contains a hardcoded model id outside a documented example block, and that dispatch paths invoke the resolver. Deliberately a weak test of behaviour and a strong guard against one specific regression that has already occurred once.

## Acceptance criteria

- [ ] No hardcoded model id remains in any team or handoff skill outside documented example blocks
- [ ] Worker dispatch resolves per ticket tier at the finest granularity the frontend's worker lifecycle permits — per dispatch on herdr, per task cluster at spawn on cmux, where a cluster's tier is the **highest** tier among its tasks — so a primary-tier ticket never spawns below the primary entry. Scope: the flavors whose dispatch lines name a model; claude-team's workers select via SDD's upstream capability heuristics and are out of scope (amended 2026-07-25 per I038 review RA2/RA3 — the original wording was unsatisfiable at per-task granularity on cmux given the ticket's own "do not change pane topology" constraint)
- [ ] Lead spawn resolves the primary tier for its flavor
- [ ] claude-team's lead-model guidance paragraph is removed, not re-pinned
- [ ] Spine presence is checked in the shared preflight and refuses early with an install hint
- [ ] The regression test fails when a literal id is reintroduced into a dispatch path
- [ ] Existing preflight shell test still passes
