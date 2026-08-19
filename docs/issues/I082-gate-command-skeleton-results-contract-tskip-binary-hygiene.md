---
id: I082
title: "spine gate go skeleton + results-contract emitter + tskip, binary-hygiene"
severity: high
status: open
affects: [gate, cli]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md` (Gate
pack). ADR 0015. Glossary: "gate pack", "check class", "positive control".

## What to build

The tracer bullet through the whole gate path. `spine gate go <check>
[--dir D]` exists with the exit-code contract (0 pass, 1 findings, 2
misconfiguration). When `MAIPIPE_RESULTS` is set the check writes the maipipe
results contract there (`maipipe_results: 0`, `status`, `summary`,
`findings[]` each with `severity`, `message`, `file`, `line`, and
`code = "go@1/<check>"`); otherwise a human table on stdout. Two check
classes ship: `tskip` (any `t.Skip*` call in `_test.go`; allowlist via
config env) and `binary-hygiene` (tracked files that are executables/
archives by content, plus stray second module trees). Each ships with a
positive-control fixture pair in spine's test suite — a known-good repo that
passes and a seeded violation that fails. Pack version constant `go@1`
lives in one place.

## Acceptance criteria

- [ ] `spine gate go tskip` and `spine gate go binary-hygiene` pass on the
      good fixture and fail (exit 1) on the seeded fixture, at the CLI seam.
- [ ] With `MAIPIPE_RESULTS` set, the JSON written validates the required
      keys and each finding's `code` is `go@1/<check>`; without it, stdout
      carries a human table and no file is written.
- [ ] Unknown check or missing config → exit 2 with a message.
- [ ] Emitter is reusable by later classes without duplication.

## Blocked by

- None — can start immediately.
