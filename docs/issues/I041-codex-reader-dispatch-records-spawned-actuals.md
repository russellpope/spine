---
id: I041
title: Codex reader core — dispatch records, spawned-thread actuals, guardian exclusion
severity: high
status: open
affects: [audit, I009]
blocked-by: [I040]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Design D20, D23, and the discovery/degrade clauses of the 2026-07-26
codex-audit design. The audit discovers codex sessions (default
`$CODEX_HOME/sessions`, else `~/.codex/sessions`; new `--codex-sessions DIR`
override mirroring `--transcripts`) and reads them as evidence:

- Dispatch records: `spawn_agent` calls (explicit model field; ticket token
  matched case-insensitively in the task name) and team spawn commands whose
  arguments carry an explicit model (`herdr agent start … -- -m X` and cmux
  equivalent). Dispatch records claim tickets exactly as claude Task
  dispatches do.
- Spawned-thread actuals: `thread_spawn` subagent files' per-turn
  turn_context models, linked to their tree by root session id (no
  parent-walking); actuals supersede the dispatch's declared model where
  linkable.
- Guardian auto-review threads are structurally excluded from every evidence
  path (their reported model is synthetic).
- Model evidence is always per-turn; session_meta's model field is null and
  never read.
- Degrade-never-fail: missing dir, unreadable file, unrecognized shape →
  report warning, never an error. D14's generation gate applies unchanged.

Verified format facts are recorded dated in I009 — build fixtures from
those, not from re-inspection.

## Acceptance criteria

- [ ] A herdr-shaped fixture (lead with `-m` spawn records + terra worker thread) judges its routine ticket `match`
  (ratified at review: the worker thread is present-but-inert until I042's
  worker-session scan lands — the match comes from the dispatch record; the
  fixture asserts the worker session does NOT leak into evidence)
- [ ] A `spawn_agent` fixture with lowercase task-name token claims the right ticket with the declared model
- [ ] Spawned-thread actuals supersede the dispatch's declared model when both exist
- [ ] A guardian-only fixture contributes no evidence to any ticket
- [ ] A model-switching session contributes each turn's model, not one per file
- [ ] Missing/unreadable codex dir degrades to a warning; claude-only repos audit exactly as before
- [ ] Flavor of codex-sourced tokens is codex; mixed claude+codex evidence judges per token per flavor
- [ ] Existing blocking/advisory behavior unchanged on all prior scenarios
- [ ] `go test ./...` green

## Blocked by

- I040
