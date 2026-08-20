---
id: I100
title: "spine's own suite writes fixture findings into the stage's MAIPIPE_RESULTS: `go test ./...` is green but `fast/test` fails the run"
severity: high
status: fixed
affects: [I082, I089, I092]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: [cross-task-integration]
review-tier: n/a
---

## Problem

`maipipe run full --wait` on spine at `716b644` → `run verdict: failed`,
stage `fast/test` `exit_code=1`, summary `go@1/n-plus-one: 2 finding(s)`,
two findings against `perrow.go:8` and `:11` — a file that does not exist in
the repo. The five `gates/*` stages skipped behind it. `make test` was green
at the same commit.

`perrow.go` is `nPlusOneSeeded`, a gate test fixture
(`cmd/spine/gate_test.go:369`). The `fast/test` stage runs `go test ./...`
with `MAIPIPE_RESULTS` exported; the gate tests drive results-emitting
commands, and any that does not redirect the variable itself writes its
**fixture** findings into the stage's real results file. maipipe reads that
file, sees a failing check, and fails the stage. The suite's own exit code is
irrelevant — the leak decides the verdict.

Same family as I092, other direction. I092 fixed the leak *out* of spine
(the mutate battery handing the stage env down to a fixture suite) and pinned
it with `TestGateMutateVerifyEnvScrubbed`. This is the leak *in*: spine's suite
inheriting the stage env from maipipe. Nothing guarded that direction.

Reproduction (before the fix):

```
$ env MAIPIPE_RESULTS=/tmp/leak.json go test ./cmd/spine/ -count=1
--- FAIL: TestGatePositiveControls/tskip/good … /n-plus-one/good   (8 subtests)
--- FAIL: TestGateResultsFileOnlyWhenEnvSet
--- FAIL: TestGateMutateHumanTable
$ cat /tmp/leak.json
{"status": "fail", "summary": "go@1/n-plus-one: 2 finding(s)", "findings": [ … perrow.go:8, perrow.go:11 … ]}
```

The last writer wins: `TestGateSyntacticClassesTolerateTestdata`'s
`n-plus-one/seeded` arm (`gate_test.go:1364`) never sets `MAIPIPE_RESULTS`, so
its two seeded findings are what maipipe read. Three other tests fail outright
under the stage env because they assert on stdout or on the *absence* of a
results file — output the inherited variable diverts.

## Root cause

Every affected test assumes `MAIPIPE_RESULTS` and `SPINE_GATE_*` are unset in
the ambient environment. That held on a developer's terminal and stopped
holding the moment spine started running its own suite under its own gate
pack (I089). The assumption was never asserted anywhere.

## Fix

`TestMain` in `cmd/spine` scrubs the stage's variables before any test runs
(`cmd/spine/main_test.go`): `MAIPIPE_RESULTS` plus every `SPINE_GATE_*` key.
Tests that need either still set it with `t.Setenv`, so no existing test
changes. Per-test redirects were rejected — that is the fix that already
half-happened, and it re-breaks with the next test that forgets.

Two controls:

- `TestScrubStageEnvRemovesStageVars` — unit: both families go, an unrelated
  `SPINE_*` key survives.
- `TestSuiteWritesNoResultsFileUnderAStage` — end-to-end: re-runs this
  package's seeded gate test in a child `go test` with `MAIPIPE_RESULTS`
  exported, the way the stage runs it, and fails if the path is written.

Negative control (scrub disabled in `TestMain`, everything else unchanged):

```
--- FAIL: TestSuiteWritesNoResultsFileUnderAStage
    stage results path written by the tree's own suite: … "file": "perrow.go", "line": 8 …
```

Verification: `make test` green; `env MAIPIPE_RESULTS=… SPINE_GATE_N_PLUS_ONE_CLIENTS=Query
go test ./... -count=1` green with the results path absent afterwards;
`maipipe run full --wait` → `run verdict: passed`.

## Notes

Only `cmd/spine` is affected — `internal/gate` tests call the checks directly
and never go through the results emitter's env lookup
(`grep -rln 'MAIPIPE_RESULTS\|SPINE_GATE_' internal/` hits no `_test.go` that
emits). The scrub lives in `cmd/spine` for that reason; a second package that
starts emitting needs its own `TestMain`, and the end-to-end control above is
the pattern to copy.

Generalizes the handoff's open item 3 (producer-side contract test): the
lesson is the same one as I092 item 5 — spine's positive controls exercise
spine, and anything that only shows up when maipipe is the caller needs a
control that puts spine *under* a stage, not beside one.
