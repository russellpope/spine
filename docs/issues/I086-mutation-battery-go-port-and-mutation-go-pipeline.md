---
id: I086
title: "Mutation battery Go port: spine gate go mutate + mutation-go pipeline"
severity: med
status: open
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
