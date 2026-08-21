---
id: I097
title: "Clearing `gate_pack` leaves the region in place and still executing — there is no uninstall, and the naive one breaks spine's own repo"
severity: med
status: fixed
affects: [I085, I095]
blocked-by: [I096]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-19 from `docs/research/2026-08-19-gate-pack-region-ownership-analysis.md`.

Reproduced 2026-08-19 on a clean checkout: with `gate_pack:` cleared in
WORKFLOW.md, `planMaipipe` returns `ok=false` and explicitly leaves an existing
region alone (`internal/update/gatepack.go:137-143`), maipipe.toml disappears
from the `spine update` report entirely, and `spine doctor` is **silent**
because `gatePackCheck` returns nil on an empty pack
(`internal/doctor/doctor.go:170-172`). The rendered `gate-go` stages remain in
the file and keep executing, because the repo's own lane composes them. The same
holds for a pack name this binary does not ship (`:146-153`): skipped, reported,
stale region still running.

So opting out is not an operation spine can perform. But the naive
implementation is worse than none: splicing the region out of spine's own
`maipipe.toml` yields `pipeline "full" stage "gates": composes unknown pipeline
"gate-go"`, exit 1 — the repo's lanes stop loading. That composition
(`maipipe.toml:66-68`) is repo-owned by ADR 0016's design, and maipipe's Phase 1
will add a second such reference when additive `init` scaffolds an `audit` lane
composing `mutation-go` (maipipe I204).

## Fix

Make opt-out a planned two-step, not a splice:

1. **Detect out-of-region references first.** Before planning removal, scan the
   file for stages that compose `gate-go` or `mutation-go` from outside the
   markers. If any exist, **refuse**: report `gate_pack cleared but N stage(s)
   still compose the pack — remove them, then re-run`, naming pipeline and stage
   for each, and leave the file untouched.
2. **With no such references, plan the region's deletion** (markers included)
   as an ordinary diff in the `spine update` report, applied by `--write`.
3. **An unknown/unshipped pack name** must report the stale region as a doctor
   finding rather than silence — today `gatePackCheck` returns nil and nothing
   says the repo is running stages this binary cannot render.

The removal plan is subject to I096's parse-before-write gate, which is why this
is blocked on it.

## Acceptance criteria

- [x] Fixture with a region and no out-of-region reference: `gate_pack: ""` →
      plan shows the region removed; `--write` removes it; the resulting file
      parses and `maipipe validate` exits 0
- [x] Fixture matching spine's own layout (repo-owned `full` lane composing
      `gate-go`): `gate_pack: ""` → refusal naming `full`/`gates`, file
      unchanged. Removing the refusal reproduces the unloadable file
- [x] `gate_pack: go@99` (unshipped) with an existing region → doctor reports
      the stale region, not silence
- [x] Re-running opt-out after removal is a clean no-op

## Notes

Depends on the I095 owner call only for message wording (whether a divergent
in-region value is worth reporting as it goes), not for the removal semantics.

## Resolution (2026-08-20)

`spine update` now reads only stage declarations outside the managed region
before an opt-out deletion. Any `gate-go` or `mutation-go` composition refuses
the whole plan and names every owning pipeline and stage. Without those
consumers, the normal plan carries a marker-inclusive deletion; ADR 0018's
existing maipipe preflight still validates it before `--write`; composition and
preflight refusals happen before any writes, so every pending file remains
untouched. Doctor now adds a D10 stale-region warning when an
unshipped pack pin leaves an existing region in place. Focused tests include a
real `maipipe validate` load-bearing control and opt-out no-op rerun.
