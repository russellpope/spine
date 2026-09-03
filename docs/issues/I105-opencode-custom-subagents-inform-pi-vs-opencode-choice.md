---
id: I105
title: "opencode supports bench-defined custom subagents (fixed prompt, restricted tools, inherited model+variant) — weigh in pi vs opencode worker decisions"
severity: low
status: fixed
commits: [c06a896]
affects: [I102, I129]
blocked-by: []
execution-mode: inline
tier: n/a
effort:
risk-triggers: []
review-tier: n/a
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

## Related

Research completed 2026-08-29 in
[`docs/research/2026-08-29-opencode-pi-worker-dispatch.md`](../research/2026-08-29-opencode-pi-worker-dispatch.md).
It verifies that OpenCode already provides the constrained, inspectable worker
contract needed by ladderbench, while Pi needs an owner-maintained extension to
match it. The material choice remains owner-dependent, so this ticket stays
open: adopt OpenCode for the constrained worker lane now, or fund a scoped Pi
extension that implements the listed parity controls. I102 remains unchanged.

## Resolution

Owner ruling 2026-09-02: fund the Pi extension, so Pi remains a viable
second worker harness if it becomes the chosen one, rather than letting the
harness be decided by default. OpenCode stays the constrained worker lane
available now; Pi reaches parity through the owner-maintained extension
scoped in **I129** (immutable agent definitions, parent allowlist, depth
counter, job ids and cancellation, persisted parent-child record with
normalized usage, and a field-by-field parity test against OpenCode's
persisted worker record). No spine code changes; I102 is unchanged. No ADR:
the ruling is recorded here and in I129, and it is reversible by closing
I129 as wontfix.
