---
id: I017
title: Ultima WORKFLOW.md gen7→8 reconciliation
severity: med
status: fixed
affects: [update, template]
blocked-by: []
labels: [wayfinder:research]
parent: I010
assignee: claude
---

## Question

Ultima-dci-edition's WORKFLOW.md carries the hand-written "Stage cursor (consistency rule)" section (lines ~20–32) — a local edit to a machine-owned file. `spine update` classifies such lines as unrecognized and skips the file (`SkippedUnrecognized`, `internal/update/update.go` ~121–140). Once gen 8's template contains an equivalent section, does ultima upgrade cleanly, or does it need `--force` (which would also drop any other local edits)?

## Resolution

(2026-07-15, Claude — verified against `internal/update/update.go`) Unrecognized detection renders `expectedOld` from the repo's *recorded* generation (7) plus extracted keys, so ultima's hand-written lines stay unrecognized even after gen 8 ships the section — a plain update still skips. But update.go already has the sanctioned channel: the **`supersededLines`** map (~line 393+) recognizes literal lines "a prior generation emitted." **Gen 8 adds ultima's exact hand-written section lines to `supersededLines`**, making them recognized-and-replaceable; plain `spine update --write` then upgrades ultima cleanly. Fallback if drift is found at sweep time: `--force` with an owner-reviewed dry-run diff. The build ticket for the template change must capture ultima's section lines verbatim.
