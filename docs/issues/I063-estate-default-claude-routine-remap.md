---
id: I063
title: "Estate default claude.routine still claude-sonnet-5 despite owner ban"
severity: low
status: open
affects: []
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

The owner banned claude-sonnet-5 (2026-08-06, mid mutation-battery run;
substitute claude-opus-5 @ low effort — recorded in project memory and the
shipped handoff's open items). Spine still emits `claude.routine:
claude-sonnet-5` as the estate default: the rows behind
`{{MODEL_ROUTING_ROWS}}` in `templates/current/WORKFLOW.md.tmpl` come from
code (`internal/update`), so every new scaffold and every sweep-refreshed
unedited WORKFLOW.md keeps proposing a banned model.

Spine's own WORKFLOW.md got the owner override edit
(`claude.routine: claude-opus-5 @ low`, 2026-08-09 maintenance sweep), which
sweeps preserve — but that covers one repo, not the estate default.

## Fix

Change the estate default `claude.routine` row to `claude-opus-5 @ low` in
the spine code that generates `{{MODEL_ROUTING_ROWS}}`, with whatever
refresh-rule/migration handling the change needs (I035 refresh-rule model
keys is prior art). Decide whether unedited estate WORKFLOW.md files pick the
new default up via the normal sweep refresh or need a generation bump per
ADR 0004. Owner ratifies before build — only worth doing if the sonnet-5 ban
is permanent.

**Ratified 2026-08-10:** owner confirmed the sonnet-5 ban is permanent and
approved the build. Ticket is dispatchable; frontier once claimed.
