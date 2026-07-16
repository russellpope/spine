---
id: I021
title: Global SessionStart cursor hook
severity: med
status: fixed
affects: [harness]
blocked-by: [I018]
execution-mode: subagent-driven
tier: mechanical
review-tier: routine
---

## What to build

Plan Task 4. One SessionStart hook entry in `~/.claude/settings.json`: when the session's project dir is a spine repo (WORKFLOW.md containing `template_version`), run `spine cursor --quiet` and let its output land as session context. Keep it a one-line shell wrapper. Document the hook in deepthought alongside the existing hook notes.

## Acceptance criteria

- [ ] Hook entry added without disturbing existing SessionStart hooks (open-brain recall etc.)
- [ ] In a spine repo with a cursor: output injected; in a non-spine cwd or cursor-less repo: silent, exit 0
- [ ] Documented in deepthought
- [ ] settings.json remains valid JSON (verified by loading a session)
