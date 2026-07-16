---
id: I018
title: Cursor format + parser + spine cursor subcommand
severity: med
status: fixed
affects: [cli, cursor]
blocked-by: []
execution-mode: subagent-driven
tier: routine
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Plan Task 1 (docs/specs/2026-07-15-stage-cursor-controls-plan.md). The canonical cursor grammar (`<!-- spine:cursor -->` block: effort, prd, tickets, stages with `[x]`/`[<]`/`[ ]`), a strict parser in a new `internal/cursor` package reading the head of `.superpowers/sdd/progress.md`, and a read-only `spine cursor` subcommand printing the parsed cursor (+ "derivation: n/a" placeholder until I019) with `--quiet` for hook use. Grammar text written once, reusable verbatim by the gen 8 template (I020).

## Acceptance criteria

- [ ] `spine cursor` prints a well-formed cursor from a fixture repo; exit 0
- [ ] `--quiet` exits 0 silently when not a spine repo / no cursor
- [ ] Parse errors (malformed block, two `[<]` markers, unknown stage name vs WORKFLOW.md `stages:`) are findings, not panics
- [ ] Fixture tests in the doctor/audit testdata style cover valid/malformed/missing/two-HERE/unknown-stage
- [ ] `go test ./...` green
