---
id: I064
title: "Handoff cursor parser latches onto a quoted cursor-marker literal in prose"
severity: low
status: wontfix
affects: []
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Found live 2026-08-09 writing the i062-tiebreak codex handoff: a Gotchas
bullet that QUOTED the cursor block's opening marker in backticks ("never
hand-edit any `...spine:cursor...` comment block") made the newest-handoff
parser treat that prose position as the block opener. Everything after it
parsed as cursor-block lines ("unrecognized line in cursor block" for each
prose line, "unknown key" when it hit the real block's opener), and both
`spine doctor` (D9) and `spine audit stages` (handoff blocking) went red on a
handoff whose real spine-owned block, at the end of the file, was pristine.

Docs that teach the sole-writer rule naturally want to name the marker; the
parser punishes exactly that.

## Fix

Anchor block detection to the LAST (or a line-anchored, column-0) occurrence
of the opening marker, or require the marker to be the entire line —
backtick-quoted inline mentions in prose should never open a block. Add a
regression fixture: a handoff quoting the literal marker mid-prose ahead of a
well-formed block must parse clean.

## Resolution — closed 2026-08-26 as a duplicate (ledger reconciliation)

**The bug is real and still unfixed.** This ticket closes as `wontfix` only
because it is the same defect as **[I109]**, filed twice — not because the
defect is accepted.

I109 supersedes this one and is the ticket to work:

- it is `severity: med` where this is `low`, and carries
  `execution-mode: subagent-driven`, `tier: routine`, `review-tier: primary`,
  where this one's routing fields are empty;
- it names the mechanism and the exact site — a bare `strings.Index` substring
  scan in `parse()`, `internal/cursor/cursor.go:267` — where this ticket
  describes only the symptom observed live on 2026-08-09.

The live evidence recorded here (the i062-tiebreak codex handoff, where a
backticked marker in a Gotchas bullet turned both `spine doctor` D9 and
`spine audit stages` red against a pristine spine-owned block) remains the
best field reproduction of the defect, and the regression fixture proposed
above still applies. Both carry forward to I109.

Closing the duplicate so the frontier shows one ticket for this defect rather
than two.
