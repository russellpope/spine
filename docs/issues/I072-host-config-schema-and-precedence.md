---
id: I072
title: Host config schema and the preference/constraint precedence story
severity: med
status: open
affects: [model, workflow, fleet, cli]
blocked-by: [I070, I071]
labels: [wayfinder:grilling]
parent: I066
assignee:
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration, plan-flagged-ambiguity]
review-tier: primary
---

## Question

[I068](I068-host-scoped-availability-and-tier-pins.md) ratified the shape:
per-host spine config as constraint, estate/repo tables as preference,
owner-ratified tier equivalence pins. Design the concrete schema: where
exactly the host config lives; how it declares available harnesses, reachable
models/gateways, and per-tier pins (model @ effort); the precedence order
estate default → repo override → host constraint/pin; what `spine model` and
`MirrorRows()`/WORKFLOW.md mirror rows display when the true routing is
host-dependent; and what doctor/audit check when a pinned model is not
reachable from the current host. Grill with the owner, then feed the standard
gate chain (PRD → tickets).

## Correction evidence (2026-08-30)

Status remains **open** pending the required fresh primary requirements review
and independent verification. Commit `2d5843d` (`fix(I072): keep alternate
host-blind`) removes the remaining review regression: a present malformed host
file still fails structural loading, while a valid file cannot gate, replace,
or filter `spine model --alternate`. The correction has command/model red-green
coverage for a config missing the selected flavor, an unavailable selected
harness, an unreachable selected primary route, and byte-identical no-config
alternate output. Its exact-SHA clean-worktree evidence includes focused/full/
race Go suites, vet, build, compiled alternate plus ordinary/effort/validate
matrix, both go@1 lanes, and `maipipe run full --wait` (`run HEAD #11`, passed).

## Correction 2 scope (2026-08-30)

Status remains **open**. Plan correction `4e2ce60`
(`docs(I072): ratify host config validation boundary`) makes the evidence-backed
`hostconfig.Load(path, flavors, lookup)` interface binding and records that
`Load` is the complete exported validation boundary. It does not add an
exported `Validate` seam or remove any schema, routing, audit, or validation
requirement.

The correction adds regressions for legacy no-config and valid-config
`model --alternate --json` bytes with no requested/host/pin trail, while a
malformed present config remains an exit-2/no-stdout failure. It also adds the
missing parser attacks for `base_url`, `modelOverrides`, and `auth_header`,
retains every prior prohibited-field case, and covers two available doctor
harnesses in lexical order with parallel explicit-path seams. Required fresh
primary requirements review, independent verification, both go@1 lanes, and
exact-final-SHA maipipe evidence remain pending before this ticket can close.
