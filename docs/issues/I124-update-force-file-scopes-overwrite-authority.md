---
id: I124
title: "spine update --force-file scopes overwrite authority to named managed files"
severity: low
status: fixed
commits: [61d4c40, 5372110, 9ca0dd8, 99caf16, 2519fe8, b822954, 47db6ba, df9ea49]
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

- [x] A named maipipe file regenerates while an unrelated WORKFLOW edit stays byte-identical.
- [x] Repeated flags authorize exactly their named planned files.
- [x] Unknown, duplicate, absolute, traversal, and unmanaged paths fail before writes.
- [x] Existing global `--force` remains compatible.
- [x] A selected candidate that fails preflight writes no file.

## Related

- I093 item 4. Owner selected additive scoped forcing on 2026-08-30.
- Accepted design and implementation plan: `docs/specs/2026-08-30-i124-update-force-file-authority-design.md` and `docs/specs/2026-08-30-i124-update-force-file-authority-plan.md`.
- Round-2 ruling: implementation and closure are already owner-authorized; close after the plan's primary review, independent verification, ticket evidence, and exact-SHA lane gates, unless a genuine contradiction or out-of-scope expansion requires a stop.

## Resolution

Fixed 2026-08-30 at final I124 product SHA `df9ea49`. Repeatable
`--force-file` authority is limited to canonical managed paths in the complete
current update plan, rejects malformed, duplicate, unknown, unmanaged, and
mixed global/scoped requests deterministically before writes, preserves
selected-marker and candidate-preflight protections, and leaves standalone
`--force` compatible. Dirty-repository rejection diagnostics are exact and
invalid-set output is permutation-independent. A fresh primary review and a
different independent primary verifier passed compiled clean/dirty,
portable-path, real/fake preflight, whole-plan no-write, full/race/vet/build,
and Windows compile gates. The single batch-final exact-SHA maipipe lane
remains the ship gate.
