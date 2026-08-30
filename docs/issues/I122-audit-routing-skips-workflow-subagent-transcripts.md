---
id: I122
title: "audit routing: workflow (spawnDepth 1) subagent transcripts are not scanned"
severity: med
status: open
affects: []
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

`spine audit routing --transcripts <project dir>` does not scan transcripts of agents spawned
by the Claude Code Workflow tool, which live under
`<project>/<session>/subagents/workflows/wf_*/agent-<id>.jsonl` with a minimal
`agent-<id>.meta.json` of `{"agentType":"workflow-subagent","spawnDepth":1}`.

Observed live in maikanban on 2026-08-29: workflow `wf_6f0d3028-076` ran 14 agents whose
opening messages begin "Implement ticket I052 (docs/issues/…)" etc. — exactly the D21
first-line attribution shape — yet the audit reported `no-transcript` for I052–I055. The only
matches for that effort came from top-level Agent-tool dispatches. Result: an entire
workflow-executed batch is invisible to the routing audit, so both matches and descents inside
workflows go unjudged (a real sonnet reviewer descent in that run was never flagged).

## Fix

Include `subagents/workflows/*/agent-*.jsonl` in the transcript sweep, applying the existing
attribution rules (D21 first-line token, D23-style structural exclusions as applicable), and
read the model from the workflow run metadata when the transcript doesn't carry it. Tests: a
fixture workflow transcript tree with a ticket token in an agent's first line attributes and
tier-judges that agent; a guardian-excluded shape inside a workflow stays excluded (negative
control).

## Related

- I121 — range-token attribution (same audit, same observed session).
- maikanban `.superpowers/sdd/progress.md` ROUTING AUDIT FACTS entry, 2026-08-29.
