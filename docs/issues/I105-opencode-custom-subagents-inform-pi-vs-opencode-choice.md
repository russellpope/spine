---
id: I105
title: "opencode supports bench-defined custom subagents (fixed prompt, restricted tools, inherited model+variant) — weigh in pi vs opencode worker decisions"
severity: low
status: open
affects: [I102]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Note from the ladderbench (benchmark v2) design session, 2026-08-20. Verified against
opencode 1.18.19 (binary strings + https://opencode.ai/config.json schema + docs):

- `agent.<name>` in `opencode.json` accepts `mode: "subagent"`, `prompt` (string or
  `{file: ./x.md}`), `tools: {...}` / `permission: {...}`, `model`, `variant`,
  `temperature`, `top_p`, `steps`/`maxSteps`, `hidden`. Custom names are allowed
  (`additionalProperties: AgentConfig`).
- A subagent with **no** `model` inherits the parent's provider+model **and** reasoning
  `variant`; a subagent that pins its own `model` drops the parent's variant.
- `permission.task` globs restrict which subagents a parent may spawn
  (`{"*": "deny", "ladder-worker": "allow"}`). `subagent_depth` defaults to 1 (no
  grandchildren); background subagents need `OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=true`.
- Runtime control surface: `opencode serve --port N` exposes `POST /session/:id/prompt_async`
  (inject a message, returns 204), `POST /session/:id/abort`, and SSE `GET /event` with
  `session.idle` / `session.status` / `message.part.updated`. `opencode run --session <id>
  --attach http://host:port "…"` is the CLI equivalent.
- Child sessions carry `parent_id` and `agent` in the sqlite db; per-session token columns
  are cumulative.

ladderbench defines a `ladder-worker` subagent (one block per worker, fixed prompt,
parent restricted to it) as part of its instrument. If spine/maipipe worker dispatch is
choosing between pi and opencode for local-model workers, opencode already has the
worker-definition, restriction, and observability primitives; pi's equivalents should be
checked before the decision is made.

## Fix

Evaluate and record (ADR or note in the team-spawn design) whether opencode's custom
subagent + serve/event surface changes the pi-vs-opencode choice for local-model worker
dispatch. No code change implied by this issue.
