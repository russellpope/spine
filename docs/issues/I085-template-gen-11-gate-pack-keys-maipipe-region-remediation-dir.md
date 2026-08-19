---
id: I085
title: "Template gen 11: gate_pack keys, maipipe.toml region (gate-go), docs/remediation scaffold, doctor region integrity"
severity: high
status: fixed
affects: [templates, update, scaffold, doctor, workflow]
blocked-by: [I082]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md` (Gate
pack — delivery; Remediation — scaffold). ADR 0002, 0004, 0015, 0016.

## What to build

A repo owner opts into the Go gate pack by setting `gate_pack: go@1` in
WORKFLOW.md and running `spine update`: spine renders a `# spine:begin
gate-pack go@1` … `# spine:end` region into `maipipe.toml` (creating the file
with only the region when absent, leaving the owner's own lanes untouched
when present) containing `[pipelines.gate-go]` — profile full, one stage per
enabled check class running `spine gate go <check>` with `env` from
`gate_pack_config`. `gate_pack`, `gate_pack_disabled`, `gate_pack_config`
are preserved choice keys; disabled classes are omitted from the render.
Template generation bumps to 11 (new keys, and `docs/remediation/README.md`
scaffolded with the remediation convention text: 3-round budget, dose
escalation rule, rescore-as-fresh cross-ref to the eval seam). `spine
doctor` checks region integrity (markers present, content canonical for the
pinned pack version) and nothing else per repo. Repos without `gate_pack`
get no `maipipe.toml` (negative control). `mutation-go` is NOT rendered
here (I086 adds it with the command).

## Acceptance criteria

- [ ] gen-10→11 migration test in the existing gen-test pattern; VERSION
      bumped; unchanged repos refresh cleanly.
- [ ] `spine update` on a fixture: absent `maipipe.toml` → created with the
      region only; present with user lanes → lanes byte-identical, region
      added/refreshed; `gate_pack_disabled: [tskip]` → no tskip stage.
- [ ] Edits inside the region are reported as unrecognized (ADR 0002), not
      kept silently.
- [ ] Doctor fires on a broken marker; silent on a canonical region.
- [ ] `docs/remediation/README.md` scaffolded by init and update.
- [ ] Repo without `gate_pack` writes no `maipipe.toml`.

## Blocked by

- I082 (the rendered stages must invoke commands that exist).

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commit a9427bb).
Template generation 11: WORKFLOW.md gains preserved choice keys `gate_pack`
(empty = opt-out), `gate_pack_disabled: []`, and a `gate_pack_config:` block
with sub-keys `test_enum_spec`, `fixture_manifest`, `build_outputs`,
`n_plus_one_clients`, `tskip_allow` (extracted as dotted keys). `spine update`
renders `# spine:begin gate-pack go@1` … `# spine:end` into `maipipe.toml`
(`internal/update/gatepack.go`): `[pipelines.gate-go]`, `profile = "full"`,
one `[[pipelines.gate-go.stages]]` per enabled class in `gate.CheckNames()`
order with `run = "spine gate go <check>"` and `env` from non-empty config;
absent file → region-only create; user lanes byte-preserved; region refreshed
in place; `mutation-go` deferred to I086. Ruling (ADR 0002 applied to a
render that is a pure function of WORKFLOW.md keys): drift recognition is
shape-based — lines any configuration could have rendered refresh silently
(inherited), everything else is unrecognized and reported (skip unless
`--force`); marker damage is never `--force`-repaired. Doctor D10: markers
well-formed and content canonical for the pinned pack, only when `gate_pack`
is set. `docs/remediation/README.md` embedded (`remediation-README.md`),
scaffolded by init and managed by update. gen 10→11 migration lock
(`testdata/spine-gen10`, `gen10to11_test.go`). Spine's own repo self-adopts
gen 11 in a separate chore commit. Primary-tier review clean, no fix round.
