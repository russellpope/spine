---
id: I013
title: Session-start cursor surfacing
severity: med
status: fixed
affects: [cli, harness]
blocked-by: []
labels: [wayfinder:grilling]
parent: I010
assignee: russell
---

## Question

How do session starts surface the stage cursor so "check it at session start" stops depending on the model remembering — a global hook, a template-shipped per-repo hook, or documentation only?

## Resolution

(2026-07-15, owner) **One global SessionStart hook** in `~/.claude/settings.json`: when the cwd is a spine repo with a `progress.md` cursor, inject the output of a new read-only **`spine cursor`** subcommand (parsed cursor + advisory derivation verdict) into context. Fleet-wide with zero per-repo wiring; new repos covered automatically. Per-repo template-shipped hooks rejected (spine owning harness config in 17 repos, stale-gen repos uncovered); documentation-only rejected (the memory-dependent control that already failed).
