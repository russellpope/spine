---
id: I073
title: Flavor → harness rename migration
severity: med
status: open
affects: [model, cli, workflow, fleet]
blocked-by: [I072]
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

[I067](I067-harness-vs-flavor-axis.md) ratified the rename: flavor becomes
harness. Inventory every touchpoint — `models/defaults.json` keys,
`spine model <flavor> <tier>`, `MirrorRows()` → WORKFLOW.md mirror rows,
CONTEXT.md "artifacts never name a flavor", WORKFLOW.md §Model routing
language, audit output, estate repos carrying mirror rows — and execute the
migration in an order that keeps `spine doctor`/`audit` green throughout,
including the fleet sweep. Sequenced after
[I072](I072-host-config-schema-and-precedence.md) so the rename lands the new
schema's names once, not twice.
