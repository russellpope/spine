---
id: I039
title: Fleet sweep — all repos to gen 10
severity: low
status: open
affects: [fleet]
blocked-by: [I036, I037]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Bring every spine-scaffolded repo in the estate to generation 10, following the sweep pattern used for the gen-8 and gen-9 sweeps.

Each repo's update plan is reviewed before writing, with attention to one thing the previous sweeps did not have to check: whether any model value reported as a refresh was in fact a deliberate override. The refresh rule cannot distinguish a deliberate choice that happens to equal a prior default, so the sweep is the human checkpoint for that ambiguity.

After the sweep, a routing audit runs against a repo with real transcripts to confirm resolution end to end against real-format data rather than synthetic fixtures.

## Acceptance criteria

- [ ] Every scaffolded repo reports generation 10; none left behind
- [ ] Each repo's diff reviewed before write; any refresh of a suspected deliberate override raised rather than applied
- [ ] Repos with genuine model overrides retain them
- [ ] A routing audit against real transcripts resolves tiers correctly and reports no new unmapped dispatches
- [ ] `spine doctor` healthy in each swept repo
