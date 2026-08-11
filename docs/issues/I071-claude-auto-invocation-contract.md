---
id: I071
title: claude-auto invocation contract — selecting model and effort per dispatch
severity: med
status: in-progress
affects: [workflow, fleet]
blocked-by: []
labels: [wayfinder:research]
parent: I066
assignee: codex-team
---

## Question

Driving a non-Anthropic model through Claude Code changes the invocation
itself: dispatches will invoke `claude-auto` with arguments rather than bare
`claude`. Pin down the contract: which arguments/env select the model and the
effort per dispatch, per gateway path in use (work custom gateway, local
OpenAI-compatible endpoints)? What must the dispatch transports — cmux/herdr
claude-team skills, plain SDD Agent-tool dispatches — pass through so that the
declared (harness, model, effort) of
[I069](I069-attribution-declare-then-confirm.md) is exactly what the
invocation runs? Deliverable: the invocation matrix + the list of skill/config
touchpoints that must change.
