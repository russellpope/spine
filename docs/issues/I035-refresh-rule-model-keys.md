---
id: I035
title: Refresh rule — model keys leave the choice-vs-default path
severity: high
status: open
affects: [update, models]
blocked-by: [I033]
execution-mode: subagent-driven
tier: primary
effort:
risk-triggers: [cross-task-integration, plan-flagged-ambiguity]
review-tier: primary
---

## What to build

Design D6–D7. Teach `spine update` to resolve model-routing values through the table instead of the generic choice-vs-default rule, and remove those keys from that rule's extraction path entirely.

Today any on-disk value differing from the current template default is classified as a deliberate per-repo choice and carried forward — verified empirically, this is why changing a model id in the template propagates nothing. After this ticket: a value matching any historical default is **inherited** and refreshed to the current default; a value matching no known default is an **override** and preserved untouched. Every refresh is itemized in the update plan, distinct from unrelated template prose churn, so a seventeen-repo sweep does not hide model changes among wording diffs.

The mirror's on-disk format is unchanged here — still bare tier keys, claude only. This is deliberately the expand step: the refresh mechanism is proven against the existing format before the format moves.

**This ticket is where the Opus 5 refresh actually reaches repos.** A repo carrying the previous fallback default gets it refreshed; a repo that pinned something else keeps it.

The failure mode to guard: if the routing keys stay in the generic extraction path, the old re-render rule and the new resolver both claim them and the original propagation trap returns in new clothing.

## Acceptance criteria

- [ ] A repo whose fallback carries the previous shipped default is refreshed to the current default
- [ ] A repo whose fallback carries an unrelated value keeps it, reported as an override
- [ ] Each refreshed value is itemized in the plan naming old value, new value, and that it was inherited
- [ ] Nothing is written without the existing write flag; the plan is reviewable first
- [ ] Model-routing keys no longer appear in generic choice extraction — verified by test, not inspection
- [ ] Non-model keys (profile, reviewers, stages, harness) still follow the unchanged choice-vs-default rule
- [ ] Existing generation-migration tests still pass unchanged
- [ ] `go test ./...` green
