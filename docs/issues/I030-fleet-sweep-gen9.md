---
id: I030
title: Fleet sweep — all 17 repos to gen 9, hand-fold objectstudio/maipipe
severity: low
status: open
affects: [fleet]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

The gen-9 binary shipped 2026-07-16 (I024-I027 batch: I026 grammar line + I027 doctor-advises clause) but the
fleet is entirely gen 8 (17/17 as of the I028 closure), one generation behind the binary. Spine's own
WORKFLOW.md is also gen 8.

## Fix

Per repo (enumerate live; 17 expected per the I023 sweep table): `spine update` dry-run → review diff →
`spine update --write` → `spine doctor` → commit sweep content only (no-mixing rule — pre-existing dirt stays
out). Each clean repo is a two-line WORKFLOW.md content delta (tickets grammar line + doctor-advises clause)
plus the `template_version: 9` stamp, verified non-destructive by the gen8to9 recognition tests + the parent
batch's FT6 dry-run on deepthought.

Known deviations, owner-ruled in advance (gen-8/I028 precedent): **objectstudio** and **maipipe** WORKFLOW.md
will skip with unrecognized-edits listings (their standing hand-folded state) — hand-fold the two gen-9 lines
+ stamp there, preserving their local edits verbatim (objectstudio: vfb gate + framebuffer harness; maipipe:
Codex model remap). maipipe may still be mid-build on `feat/i021-events-wait-sse` — single-file pathspec
commit like 931a52e if so. spine's own WORKFLOW.md updates on the effort branch.

No `--force` anywhere; a repo that fails to update cleanly is reported and skipped. No pushes during the
sweep (push backlog is a separate owner-worded step this session).

## Acceptance criteria

- [ ] Every repo shows `template_version: 9` on disk and a passing `spine doctor` (standing D4/D5/D6 states noted, not new findings)
- [ ] objectstudio/maipipe hand-folds preserve their local edits verbatim (diff shows only the two gen-9 lines + stamp)
- [ ] Per-repo results table (gen before/after, files, doctor, commit) appended to the build ledger
- [ ] No repo force-updated; failures listed, never silently skipped
