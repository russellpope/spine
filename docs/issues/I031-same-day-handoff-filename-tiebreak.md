---
id: I031
title: Same-day handoff filename-DESC tiebreak makes second same-day handoff naming load-bearing
severity: low
status: open
affects: [I014, I025]
blocked-by: []
execution-mode:
tier: mechanical
effort:
risk-triggers: []
review-tier:
---

## Problem

`handoff.List` orders same-day handoffs by filename DESC (documented behavior). Consequence observed live
2026-07-16: the derivation-polish handoff was first written as `2026-07-16-derivation-polish-shipped.md` and
lost "newest" to the same morning's `2026-07-16-stage-cursor-controls-built.md` — `spine audit stages` (I025
effort-matched backstop) then blamed the *older* doc for carrying the wrong effort's cursor block. The doc had
to be renamed (`...-stage-cursor-polish-shipped.md`) so it sorted after the earlier one. Writing a second
handoff on one day makes the filename itself load-bearing, and nothing warns the author: the failure surfaces
only as a stale-effort block pointing at a document the author didn't just write.

Papercut, not a correctness bug — the block is honest once you know the tiebreak rule. It has now cost one
live rename plus a gotcha line in two handoffs/PICKUPs.

## Fix

(open — candidates, pick at effort assignment)

- Prefer an effort-matched cursor block over pure filename-DESC when selecting "newest" for the I025 backstop
  (the doc whose block names the live effort is the one the check should read), falling back to filename DESC.
- Or: cheap discoverability — when audit stages raises the stale-effort finding and a *different* same-day
  handoff carries an effort-matched block, name the tiebreak in the detail ("an effort-matched handoff exists
  but sorts earlier: <file> — same-day handoffs order by filename DESC").
- Or: `/handoff` skill guard only (check the new filename sorts after existing same-day docs) — weakest, fixes
  the author path but not the reader's confusion.
