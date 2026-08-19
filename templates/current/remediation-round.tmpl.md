---
round: 1
dose: findings-only
hitlist: docs/remediation/<effort>/hitlist-1.md
run_id:
verdict:
# extension-ratified-by: <owner>   # required for round 4 and beyond
---

# Round 1

Copy this file to `docs/remediation/<effort>/round-N.md`, one file per round,
numbered from 1. `dose:` is one of `findings-only`, `prescriptive`,
`raw-review`. The budget is 3 rounds per effort: a round-4-or-later record
needs `extension-ratified-by:` naming the owner who ratified the extension —
uncomment the frontmatter key above. `spine audit stages` advises (never
blocks) when it is missing.

## Findings

The table keys on the results-contract `code`, so "did this finding fail last
round?" is a lookup against the previous round's record, not a judgment about
prose.

| code | status | note |
| --- | --- | --- |
| `go@1/tskip` | fixed | assertion replaces the skip; fixture is now required. |
| `go@1/errsink` | open | still dropped at the same call seam. |
| `go@1/mutate` | regressed | M004 survives now — the do-not-regress row. |

`status` is one of `open`, `fixed`, `regressed`.

## Verdict

State the round's outcome and what the next round's dose should be. Escalate
one step at a time, and only after a round fails on the same `code`.
