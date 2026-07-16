---
id: I009
title: audit routing — no Codex transcript adapter; routing gate is toothless on Codex-driven builds
severity: high
status: open
affects: [I008]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Codex became a first-class harness on 2026-07-10 (gen 7: spine emits AGENTS.md,
workflow skills symlinked into `~/.codex/skills`), but `spine audit routing`
only reads Claude Code's transcript format: `<dir>/*.jsonl` session records
with Task-dispatch tool-use entries linked to subagent transcripts by
toolUseID. Codex sessions live elsewhere in a different shape.

Consequence, hit live prepping the maipipe v1 build (Codex-driven, first of
its kind): every ticket degrades to `no-transcript` (warn, by design — see
`internal/audit/audit.go` "degrade, never fail"). The verify gate does not
block, but it verifies nothing — silent tier descent on a Codex build is
undetectable. The audit's enforcement purpose ("auditability is the
enforcement layer", ADR 0010 / CONTEXT.md routing-purpose) silently vanishes
on exactly the builds least familiar to review.

Known facts about the Codex side (verified 2026-07-10 on Codex 0.144.1):

- Sessions are JSONL at `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl`.
- Session records carry `"model":"<id>"` (verified: probe runs recorded
  `gpt-5.6-terra` / `gpt-5.6-luna` faithfully — no silent fallback).
- Unknown: how Codex `multi_agent` subagent dispatches are recorded and how a
  dispatch correlates to its subagent's model evidence (the toolUseID-link
  equivalent). Needs a live subagent-driven Codex session to inspect.
- maipipe's `WORKFLOW.md` `model_routing` is remapped to
  sol/terra/luna + claude-opus-4-8 fallback, so mapped-id matching must handle
  a mixed-provider map (the alias/substring rule in audit.go should already).

## Fix

Add a Codex transcript reader alongside the Claude Code one: discover the
session dir (respect `CODEX_HOME`, default `~/.codex/sessions`), parse rollout
JSONL into the same dispatch/agent evidence structures, and correlate
dispatches to tickets by the existing ticket-id-token contract. Date-sharded
layout may help scope transcripts per build and mitigate the I008 cross-build
collision class. Format is undocumented/unstable — same degrade-never-fail
posture as the Claude reader. Research the multi_agent recording shape first;
if subagent model evidence turns out not to be recoverable, the honest
fallback is a distinct verdict (e.g. `no-subagent-evidence`) rather than a
generic `no-transcript`.
