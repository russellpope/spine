---
id: I089
title: "Self-enable gate_pack go@1 on spine (WORKFLOW.md keys, maipipe.toml region, doctor D10 round-trip)"
severity: med
status: open
affects: [I085]
blocked-by: [I088]
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

spine ships the go@1 gate pack (I082–I086) but does not run it on itself.
Dogfood plan (deepthought handoff 2026-08-19 §1b–d): set `gate_pack: go@1`
+ `gate_pack_config.tskip_allow` in spine's WORKFLOW.md; `spine update
--dir . ` dry-run then `--write`; inspect the `# spine:begin gate-pack go@1`
region in `maipipe.toml`; `spine doctor` → D10 silent; hand-edit one line
inside the region → D10 fires; revert via `spine update --write`. Then the
maipipe composition check (a `full` lane stage `pipeline = "gate-go"`
outside the region; `maipipe validate`; findings carry `code = go@1/<check>`)
and `mutate` with a `docs/mutation-spec.json` (2–3 probes incl. one
`report_only`; deliberate build break → control failure exit 1).

## Fix

Record each step's command + output in the ticket resolution; any defect
found gets its own ticket.
