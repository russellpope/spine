---
id: I061
title: "Cursor writes: gen 10 fleet sweep + handoff skills to verbs-only procedure"
severity: med
status: open
affects: [fleet, skills]
blocked-by: [I060]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Parent

PRD: docs/specs/2026-08-06-cursor-writes-design.md (spec: skills change in the
same window as the sweep — the estate never instructs a procedure its tooling
can't perform, and never keeps a hand-writing instruction alive after the
sole-writer rule lands).

## What to build

Roll the rule out everywhere at once (precedent: gen 8 sweep I015, gen 9 sweep
I029):

1. **Fleet sweep**: `spine update` dry-run review → `--write` → commit, per
   estate repo, until every repo's WORKFLOW.md carries the gen 10 sole-writer
   text. Unexpected drift → owner-reviewed diff before `--force`.
2. **Skills**: `handoff` and `handoff-to-codex` (the only two skills
   referencing the cursor, verified by sweep at grill time) rewritten: the
   snapshot embed happens automatically via `spine handoff new`; cursor state
   changes only via the verbs; hand-editing instructions removed.
3. **Live verification** per estate convention: in one real swept repo, run
   the real commands (a verb, `handoff new`, `audit stages`) and show output —
   not just diffs.

## Acceptance criteria

- [ ] Every estate repo swept to gen 10; per-repo commits; drift (if any)
      resolved with owner-reviewed diffs
- [ ] Both skills contain no hand-edit or manual-embed instruction; procedure
      references the verbs and automatic embed
- [ ] Live-verify evidence pasted: verb write + handoff embed + clean
      `spine audit stages` in a swept repo
- [ ] No repo left mixed-rule (old embed-verbatim text alongside new tooling)

## Blocked by

- [I060] — the gen 10 template the sweep distributes.
