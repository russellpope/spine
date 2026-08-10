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
