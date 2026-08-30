---
id: I124
title: "spine update --force-file scopes overwrite authority to named managed files"
severity: low
status: open
affects: [I093]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [plan-flagged-ambiguity]
review-tier: primary
---

## What to build

Add repeatable `--force-file <repo-relative-managed-path>` to `spine update`.
It authorizes regeneration only for exact named files in the current plan.
Existing boolean `--force` retains global behavior; candidate preflight and
atomic writes stay unchanged.

## Acceptance criteria

- [ ] A named maipipe file regenerates while an unrelated WORKFLOW edit stays byte-identical.
- [ ] Repeated flags authorize exactly their named planned files.
- [ ] Unknown, duplicate, absolute, traversal, and unmanaged paths fail before writes.
- [ ] Existing global `--force` remains compatible.
- [ ] A selected candidate that fails preflight writes no file.

## Related

- I093 item 4. Owner selected additive scoped forcing on 2026-08-30.
