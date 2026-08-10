---
id: I064
title: "Handoff cursor parser latches onto a quoted cursor-marker literal in prose"
severity: low
status: open
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
