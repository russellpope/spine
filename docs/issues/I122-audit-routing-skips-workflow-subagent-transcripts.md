---
id: I122
title: "audit routing: workflow (spawnDepth 1) subagent transcripts are not scanned"
severity: med
status: fixed
commits: [3294058, 59dc240, eaa8ff3, 9ceabf3, da4bf93, b676642, 8fe8dbf, eb080d3, 16b3945, b9e9f22, 3595c9b, 6bc5554, a348868, e077838, aba1f1e]
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

## Resolution

Fixed 2026-08-30 at final I122 product SHA `aba1f1e`. Routing audit now
discovers admitted depth-1 workflow workers, latches only a coherent first user
opening, preserves independently attributable nested dispatches, and obtains
missing model evidence from one exact agent entry in the real heterogeneous
workflow run format. Sidecar, JSONL, and run evidence are bounded where
specified, exact-key validated, descriptor-rooted, and retained as one snapshot
whose root, ancestors, artifact identities, and cross-read consistency are
revalidated against symlink and atomic replacement attacks. I121 range grammar,
I073 harness/source separation, I075 declarations, I074 verdicts, session/cwd
scope, exclusions, and exact nested identities remain intact. A fresh primary
review and a separate fresh primary verifier passed 34 hostile groups repeated
ten times plus full/race/vet/build and Windows gates. The batch-final exact-SHA
maipipe lane remains the ship gate.
