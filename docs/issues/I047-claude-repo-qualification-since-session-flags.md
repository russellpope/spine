---
id: I047
title: Claude-side repo qualification + --since/--session filters
severity: high
status: fixed
affects: [audit, I008]
blocked-by: [I040]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [plan-flagged-ambiguity]
review-tier: primary
---

## What to build

Design D28 — the recorded I008 incident (a shared controller dir's spine
I003 dispatch false-blocking praxis's unrelated I003) becomes
unreproducible. A claude dispatch claims a ticket only if its
description/prompt also references the audited repo (absolute path or
basename token) OR its session shows cwd evidence inside the repo. New
`--since <time>` and `--session <id>` filters scope the transcript set as
operator escape hatches.

Flagged ambiguity: the repo-reference heuristic's false-negative surface
(legit dispatches that never name the repo). The review must weigh what the
heuristic excludes on real transcripts, not just what it admits — losing
legitimate evidence downgrades verdicts to no-transcript, which is honest
but must be a conscious trade, and unmatched dispatches remain visible in
the report's informational list.

No started-date anchoring (rejected at grill: it blinds multi-milestone
repos to earlier builds' transcripts).

## Acceptance criteria

- [ ] I008-shaped fixture: same ticket id in two repos, shared controller transcript dir — neither audit sees the other's dispatch; no false silent-descent
- [ ] Dispatch naming the repo path/basename claims normally; cwd-evidence path also claims
- [ ] `--since` excludes older sessions; `--session` restricts to one; both compose with defaults
- [ ] Excluded dispatches surface in the unmatched informational list, not silently dropped
- [ ] `go test ./...` green

## Blocked by

- I040

## Resolution — closed 2026-08-26 (ledger reconciliation)

Shipped; never closed. Both halves verified in the working tree:

- **Repo qualification** — `repoQualifies` is live at
  `internal/audit/audit.go:401` and `:453`, both gated on
  `flavor == "claude"`. This is the D28 predicate.
- **`--since` / `--session` filters** — parsed at the CLI layer with acceptance
  tests in `cmd/spine/main_test.go:1053-1109`, covering both the filtering
  behaviour and the exit-2 usage error on an unparseable value.
- Dedicated regression file `internal/audit/i047_test.go`.

Closed transitively by **I048** (`fixed` 2026-07-27), which lists this ticket in
`blocked-by`.

**Still-live hazard — see [I111].** The two predicates above are gated on
*flavor*, not on transcript source. The two were the same thing when this
shipped and stop being the same the moment open-weights records are tagged
`openweights`: those records would fall out of the D28 gate and begin claiming
tickets they should not. Closing this ticket does not close that hazard; I111
owns it and no existing test covers it.
