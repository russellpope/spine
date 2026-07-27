---
id: I040
title: Prefactor — audit entry point takes options; flavor threads per token
severity: med
status: open
affects: [audit]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Design 2026-07-26 codex-audit, groundwork for D20–D28. The audit entry point
takes an options struct (repo dir, claude transcripts dir, codex sessions
dir, time/session filters) instead of positional args, so later tickets add
inputs without churning the signature. Flavor stops being derived once per
run and instead travels beside each evidence token into judgment (D15's
seam made real) — with only the claude reader live, every token still
carries claude, and behavior is byte-identical.

Pure prefactor: no new inputs are read yet, no verdict changes.

## Acceptance criteria

- [ ] Entry point accepts an options value; CLI callers updated; `--transcripts` behavior unchanged
- [ ] Each judged token carries its own flavor through judgment; resolution per token happens within that flavor's table
- [ ] All existing audit scenarios and CLI tests pass unchanged — identical verdicts, warnings, exit codes
- [ ] `go test ./...` green

## Blocked by

- None — can start immediately.
