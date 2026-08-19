---
id: I092
title: "First live mutation-go run fails results_invalid: findings emit line 0 / file \".\"; battery leaks MAIPIPE_RESULTS into the tree's suite"
severity: high
status: fixed
affects: [I082, I086]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: [cross-task-integration]
review-tier: n/a
---

## Problem

Dogfood §1c/d (2026-08-19): `maipipe run mutation-go --wait` on spine →
`run verdict: error`, stage `mutate` `error_kind=results_invalid`,
`finding line must be a positive 64-bit integer`. Two defects, one masking
the other:

1. **Results contract** (I082): `jsonFinding` always emits `file` and
   `line`; site-less findings (mutate's control failure with `file: "."`,
   `line: 0`; gitignore-control arm 1 and binary-hygiene with `line: 0`)
   are rejected by maipipe (`src/results.rs`: line optional but ≥1; file
   optional but must be a safe relative path). Any such finding turns the
   stage into `results_invalid` instead of `fail`.
2. **Battery env leak** (I086): the verify commands inherit the stage's
   `MAIPIPE_RESULTS` (and `SPINE_GATE_*`). spine's own gate tests exercise
   the results contract, so under maipipe the control went red
   (`TestGatePositiveControls/*`, `TestGateResultsFileOnlyWhenEnvSet`) —
   reproduced locally: `MAIPIPE_RESULTS=x go test ./cmd/spine` in the kept
   working copy fails; without it, passes.

## Fix

- `results.go`: `file,omitempty` / `line,omitempty`; mutate's control
  finding carries neither.
- `mutate.go`: `verifyEnv` strips `MAIPIPE_RESULTS=` and `SPINE_GATE_*`
  from the verify command env.
- Tests (`cmd/spine/gate_test.go`): `TestGateResultsOmitLineZero`,
  `TestGateMutateVerifyEnvScrubbed`, control-failure raw-key assertion.
  Negative controls: reverting each fix fails its test (3/3).
- Evidence: `maipipe run mutation-go --wait` on spine → stage `mutate`
  pass, 3 rows with `code = go@1/mutate` (recorded below after reinstall).
