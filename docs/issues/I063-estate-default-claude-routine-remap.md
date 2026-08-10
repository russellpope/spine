---
id: I063
title: "Estate default claude.routine still claude-sonnet-5 despite owner ban"
severity: low
status: fixed
affects: [models, update, audit]
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

## Resolution

- The embedded `claude.routine` default is now `claude-opus-5 @ low`.
  Current aliases are `claude-opus-5` and `opus`; the displaced
  `claude-sonnet-5` pair is history with omitted effort, preserving the
  medium effort at which it actually shipped. Sonnet shorthand is not a
  historical alias; exact historical ids remain auditable.
- Existing inherited Sonnet-medium rows refresh through ADR 0011's normal
  model-table resolver and receive an itemized old/new update-plan entry.
  Unrelated ids and mismatched effort pairs remain reported overrides.
- Legacy generation 5-9 customized top-level effort now replaces an inherited
  table suffix rather than being skipped. Both `xhigh` and the routine tier's
  nominal `medium` survive as deliberate per-entry overrides against the new
  low-effort current pair; explicit per-entry effort still wins.
- **Generation decision:** no template bump. `{{MODEL_ROUTING_ROWS}}` already
  renders from the embedded table and `applyModelRouting` refreshes historical
  pairs independently of template generation, so this is a plain sweep
  refresh under ADR 0011, not a template-format change under ADR 0004.
  `templates/VERSION` remains 10 and no new ADR is required.
- Spine's own pre-existing Opus-low row was normalized with the generated
  mirror alignment and is locked by a write-enabled regression using the
  checked-in `WORKFLOW.md`; first and second updates are byte-stable.
- Implementation landed in `b71a20b`; primary blind review found the spine-row
  byte-stability gap, corrected in `c84c5bd`, and primary re-review passed with
  no findings. Fresh primary verification passed uncached full tests, focused
  race tests, vet, a clean offline build, real compiled-CLI acceptance cases,
  formatting/diff integrity, and routing audit. Historical generation fixture
  contents were not rewritten.
- No push, binary installation, or estate sweep was performed; those remain
  owner deployment actions.
