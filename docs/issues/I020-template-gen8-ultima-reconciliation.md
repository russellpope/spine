---
id: I020
title: Template gen 8 — stage-cursor section + ultima supersededLines
severity: med
status: fixed
affects: [template, update]
blocked-by: [I018]
execution-mode: subagent-driven
tier: routine
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Plan Task 3. Bump the template generation to 8; add the spine-owned "Stage cursor (consistency rule)" section to `WORKFLOW.md.tmpl` (adapted from ultima-dci-edition's hand-written section, embedding the I018 grammar and the verbatim-handoff rule); capture ultima's current WORKFLOW.md section lines **verbatim** into `supersededLines` in `internal/update`; gen7→8 migration tests including an ultima fixture proving plain update yields zero unrecognized lines.

## Acceptance criteria

- [ ] `spine version` reports gen 8; init/adopt scaffold the new section
- [ ] Gen7→8 test: a pristine gen-7 WORKFLOW.md updates cleanly
- [ ] Ultima fixture test (hbmview_test.go style, real file copied verbatim): plain `spine update` reports zero unrecognized lines and replaces the hand-written section
- [ ] No gen-mismatch message hardcodes 7
- [ ] `go test ./...` green
