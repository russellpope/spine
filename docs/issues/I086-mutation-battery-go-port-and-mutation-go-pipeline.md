---
id: I086
title: "Mutation battery Go port: spine gate go mutate + mutation-go pipeline"
severity: med
status: fixed
affects: [gate, update, templates]
blocked-by: [I085]
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
pack — `mutate`). ADR 0015 (supersedes ADR 0013 items 2 and 4). Source
behaviour: `docs/mutation-battery-checklist.md` and the `/model-eval`
skill's Python runner (mutate.py, sites.sh) — port, don't reinvent.

## What to build

`spine gate go mutate` runs the behavioural mutation battery in Go: reads
the per-tree mutation spec, applies each mutation, runs the configured
verify command, and emits killed/survived rows through the results contract
(one finding per row, `code = go@1/mutate`, detail naming the mutation site
and outcome), with the unmutated negative control mandatory — exit 1 only
when the control fails, otherwise exit 0 (advisory lane). `spine update`
now also renders `[pipelines.mutation-go]` (profile audit, one stage) into
the region. The `/model-eval` skill's runner is documented as superseded by
the binary (skill change is estate-side, out of this repo).

## Acceptance criteria

- [ ] Fixture with a known killed and a known survived mutation produces
      the expected rows; control failure → exit 1 with a finding.
- [ ] Rows validate against the results contract; killed rows carry enough
      detail for a do-not-regress block (site, mutation, outcome).
- [ ] Region renders `mutation-go`; a repo pinned before this ticket
      refreshes to include it (inherited) without touching user lanes.
- [ ] Positive-control pair added like every other class.

## Blocked by

- I085 (region rendering + gen 11).

## Resolution

Fixed 2026-08-19 on branch `local-harness-conventions` (commits 3a12b38,
53c278f). `spine gate go mutate` (`internal/gate/mutate.go`) ports
`mutate.py`: spec JSON `{id, file, find, replace, report_only?, desc?}` from
`SPINE_GATE_MUTATE_SPEC` (env-only knob; default `docs/mutation-spec.json`),
verify `go build ./... && go test ./...` (override `SPINE_GATE_MUTATE_VERIFY`,
timeout `SPINE_GATE_MUTATE_TIMEOUT`, default 15m); runs in a temp copy of the
tracked tree, never in `--dir`; unmutated control first — failure → one
`go@1/mutate` finding carrying the verify-output tail and the kept copy's
path, exit 1; otherwise one finding per probe (KILLED/SURVIVED/NO-SITE/
BUILD-ERR, `[report-only]`), file:line of the site, both kill-rate lines as
summary, exit 0 (advisory lane). `spine update` renders
`[pipelines.mutation-go]` (`profile = "audit"`, one stage) after `gate-go`
in the region; `mutate` is excluded from `gate-go`; a pre-existing gate-go
region refreshes to include it as inherited. Ruling: `gate_pack_disabled:
[mutate]` removes the pipeline — a user lane composing `mutation-go` would
then dangle. Rulings: the positive-control pair for an advisory class is
rows + rates plus the control-failure exit-1 fixture; rows carry site, probe
id, outcome (mutation recoverable from the spec by id).
`docs/mutation-battery-checklist.md` notes the binary supersedes the Python
runner (ADR 0015; the `/model-eval` skill change is estate-side). Review +
scoped re-review clean.
