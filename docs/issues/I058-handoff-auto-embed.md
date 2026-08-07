---
id: I058
title: "Cursor writes: handoff new auto-embeds the committed snapshot"
severity: med
status: fixed
affects: [cli, handoff]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Parent

PRD: docs/specs/2026-08-06-cursor-writes-design.md. Glossary: CONTEXT.md
"Stage cursor" (working home / committed snapshot).

## What to build

`spine handoff new` captures the committed snapshot automatically: the file it
scaffolds carries the current cursor block verbatim, killing the hand-performed
copy step (the ADR 0013 staleness class). Read-side only — needs the existing
parser, not the I057 verbs.

- Working home has a cursor → the scaffolded handoff embeds the canonical
  block.
- No cursor present (wayfinder efforts, pre-cursor repos) → scaffold without a
  block, print a note, exit zero.
- Committed snapshots are historical: nothing ever retro-mutates a previously
  created handoff. The existing audit check (newest snapshot matches working
  home) stays the cross-home consistency gate.

## Acceptance criteria

- [x] Fixture with a cursor: created handoff carries the block verbatim;
      `spine audit stages` accepts the pair as fresh
- [x] Fixture without a cursor: note printed, exit zero, no block in the file
- [x] Existing `handoff new` behavior (naming, listing, --fleet) unchanged
- [x] Tests at the existing handoff command seam, external behavior only
- [x] `go test ./...` green

## Blocked by

- None — can start immediately.

## Resolution

Fixed in `c4266f2`, `0ef3c55`, and `bacbdf7`. `spine handoff new` embeds the
current canonical cursor automatically, refuses malformed input before file
creation, and the shared stage/doctor derivation now rejects malformed or
stale full-state snapshots while preserving historical handoffs.
