---
id: I082
title: "spine gate go skeleton + results-contract emitter + tskip, binary-hygiene"
severity: high
status: fixed
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

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commits 3d39947, c25f1a6).
New package `internal/gate`: check registry, one results-contract emitter
(JSON to `MAIPIPE_RESULTS` when set, else human table), one exit-code owner
(`gate.Run`), pack constants `go@1` in one place (`PackName`/`PackVersion`,
`Code(check)`), CLI `spine gate go <check> [--dir D]`. Classes `tskip`
(go/ast; receivers `t`/`b`/`tb` and `.T()` accessors; allowlist
`SPINE_GATE_TSKIP_ALLOW` = comma-separated `path` or `path:line`, unset = no
allowlist) and `binary-hygiene` (tracked files by magic bytes incl. tar at
offset 257; `go.mod` in a tracked non-root dir = stray module tree). Positive
control pairs at the CLI seam in `cmd/spine/gate_test.go`. Config env
convention: `SPINE_GATE_` + upper-snake(gate_pack_config key) via
`gate.EnvVar`. Recorded constraint: legitimate `tools/go.mod` tool modules
false-positive under go@1's stray-module rule (escape: `gate_pack_disabled`;
an allow key is go@2 territory). Task review + scoped re-review clean.
