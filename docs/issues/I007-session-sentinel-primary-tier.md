---
id: I007
title: "`primary: session` sentinel — let the primary tier track the harness session model"
severity: low
status: open
affects: [I006]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

The tier→id mapping is per-repo remappable (ADR 0010), which covers projects
that are permanently not fable-class: edit `model_routing` once. But the remap
is a file edit while the session model is a one-keystroke `/model` choice, and
nothing connects the two. Today the two can silently disagree: a session run
on opus against a repo whose mapping says `primary: claude-fable-5` produces a
routing audit judged against a mapping the owner had already overridden in
their head. Worse, a floating primary is currently contract-illegal, not just
unimplemented — the gen-6 dispatch contract says "explicit model, never
inherit," so there is no legal way to express "top-tier work runs on whatever
the harness is set to."

Wanted (Russell, 2026-07-10): set the harness to opus and the workflow's
top-tier work runs on opus; set it to fable and it runs on fable — for
projects that use the workflow but aren't a fable-class challenge.

## Fix

Support a `primary: session` sentinel in `model_routing`:

- **Semantics**: tiers mapped to `session` are dispatched by omitting the
  model param (inherit the session model). All other tiers keep explicit ids.
- **Contract carve-out** (WORKFLOW template wording): "never inherit" becomes
  "never inherit, except tiers mapped to `session`, which must inherit."
- **Audit resolution**: `spine audit routing` resolves `session` against the
  main loop's actual model from the transcript JSONL it already parses, and
  reports the resolution in the per-task table (e.g. `primary (session →
  claude-opus-4-8)`) — visible, never silent.
- **Floor warning**: when the resolved session model maps below `routine` in
  the repo's own tier table, the audit warns (advisory, not blocking). An
  accidental haiku session must not silently become the "primary" that runs
  the final whole-branch review — the one invariant the routing design exists
  to protect is that the quality ceiling cannot erode invisibly.
- Silent-descent detection is unchanged for routine / mechanical / fallback.
- Escalation grammar is unaffected (tier→tier, not id-based).

Surfaces: WORKFLOW template routing block (template patch or gen bump), audit
resolution logic + fixtures (session-mapped tier: clean resolve, below-routine
warn), CONTEXT.md vocabulary line for the sentinel.

Sequencing: overlaps I006's surfaces — fold into the fleet sweep or land
immediately after it, so the sweep isn't re-run for a template wording change.
Do NOT land before praxis I001: that build is the acceptance exercise for the
gen-6 contract as shipped.
