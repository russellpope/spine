---
id: I009
title: audit routing — no Codex transcript adapter; routing gate is toothless on Codex-driven builds
severity: high
status: closed
affects: [I008]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

> Design: `docs/specs/2026-07-26-codex-audit-design.md` (grilled 2026-07-25/26;
> D20–D28, ADR 0012). Verified format facts below, dated 2026-07-25.

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

Verified 2026-07-25 (codex cli_version 0.145.0, against moo-clone M4b I024
ground truth — lead `gpt-5.6-sol` session 019f97f6-a862-7302-9488-d564f82c43f1,
workers `gpt-5.6-terra`, expected verdict: match):

- Line kinds via top-level `type`: `session_meta` (1), `turn_context` (N),
  `event_msg`, `response_item`, `world_state`. The per-turn model lives in
  `turn_context.payload.model`; `session_meta.payload.model` is present but
  NULL — never read tier from session_meta.
- Threads form a tree, but no walking is needed to find a tree's members:
  `session_meta.payload.id` is the thread's own id, `payload.session_id` is
  the ROOT thread's id (shared by every file in the tree), and
  `payload.parent_thread_id` is the immediate parent. Filter on session_id.
- Guardian auto-review threads report a FAKE model: `thread_source:
  "subagent"`, `source: {"subagent":{"other":"guardian"}}`, and
  `turn_context.payload.model: "codex-auto-review"` on every turn. Naively
  read, a ticket scores as routed to a review model — confident wrong answers,
  worse than no-transcript. Exclude by source/thread_source.
- Real codex-native subagents are distinguishable from guardians:
  `source: {"subagent":{"thread_spawn":{parent_thread_id, depth, agent_path,
  agent_nickname, ...}}}` and carry true models.
- herdr/cmux team workers are NOT codex-native subagents: each is a separate
  top-level session (`thread_source: "user"`, parent null). The lead's
  transcript records the dispatch as `response_item`/`function_call`
  (`exec_command`) whose arguments carry the explicit model —
  `herdr agent start moo-clone-worker1 --kind codex --pane wM:p2 -- -m
  gpt-5.6-terra` — followed by `herdr agent prompt <name>
  "$(<.superpowers/sdd/<date>-<milestone>-i024-codex/dispatch-task-*.md)"`.
  This is the Claude Task-dispatch analog (explicit model + ticket-token
  linkage), if a dispatch-record path is wanted alongside session evidence.
- Lead sessions are orchestration-tier (`sol`) and mention the ticket token
  heavily (96× in the I024 lead); counting the lead's model as ticket evidence
  would misjudge every ticket. Same rule as the Claude reader: orchestrator
  models are never ticket evidence — but leads and workers are BOTH top-level
  `thread_source: "user"` sessions, so the reader needs another discriminator
  (dispatch-record linkage, or the worker's opening user message being the
  dispatch brief).
- Ticket-token matching MUST be repo-scoped: praxis and maipipe carry their
  own I024 tickets, and 24+ sessions outside moo-clone matched the I024 token
  on the same days (their own builds plus quoted transcript history in
  guardian threads — the literal string `gpt-5.6-terra` also appears inside
  quoted `response_item` text). But cwd == repo is too narrow: cmux codex-team
  workers run in worktree cwds like `/private/tmp/maipipe-codex-team-i083`.
  `session_meta.payload.git` carries only `{commit_hash, branch}` — no remote
  URL — so repo identity needs cwd + worktree heuristics or git correlation.
- M4a's transcripts survive: I008/I012/I015/I020/I022 tokens hit 48–74
  moo-clone-cwd session files each across 2026-07-17..24, so the fix is
  testable against cmux-run history, not just herdr's.
- Scale: 951 session files total (2026-07-02..25); 63 reference moo-clone
  since 07-24 alone. Discovery needs date/token pruning, not a full-dir parse.

Verified 2026-07-27 (I048 live acceptance — surprises the synthetic fixtures
missed, found by running the finished reader against the real store):

- `exec_command` function_call arguments are a JSON-string-encoded object
  whose command text lives under key **`cmd`** (NOT `command`); sibling keys
  observed: `workdir`, `yield_time_ms`, `max_output_tokens`,
  `sandbox_permissions`, `justification`, `prefix_rule`.
- The FIRST user-role message of a session is an INJECTED preamble, not the
  operator's prompt: `# AGENTS.md instructions for <cwd>` (96 of the 120
  most recent sessions) or `<recommended_plugins>` (19/120). Hook prompts
  (`<hook_prompt hook_run_id=…>`) are also injected as user-role messages
  later in the stream. The dispatch brief is the first user message NOT
  shaped as an injected preamble — discriminator: first line starts with
  `# AGENTS.md instructions` or matches an angle-bracket tag opener
  (`^<[a-z_]+[ >]`). If every user message is injected-shaped, there is no
  opening brief (degrade: contributes nothing).
- Real M4b worker briefs DO carry the ticket token in their first line
  ("You are the fresh PRIMARY-tier I035 finisher…") — the D21 first-line
  rule holds on real dispatch data.
- `thread_source` is serialized compact (`:"subagent"`, no space) in
  619/953 files with zero spaced variants (recorded at I049 review; the
  pruning exemption's marker relies on this external serializer behavior).
- **`session_meta.payload.source` is POLYMORPHIC**: a plain JSON string
  (`"cli"`) on top-level user sessions, an object
  (`{"subagent":{"other":"guardian"}}` / `{"subagent":{"thread_spawn":{…}}}`)
  only on subagent threads. A reader that types it as an object fails to
  unmarshal every top-level session's meta line (observed live: 410
  malformed-meta warnings across the 953-file store — every lead and worker
  invisible). Parse it polymorphically; a string source means a plain
  top-level session. Other payload fields observed with non-string types:
  `base_instructions` (object), `context_window` (object), `git` (object).
- **cmux codex-team leads dispatch via `custom_tool_call`, not
  `function_call`.** A third call shape: `response_item` with
  `payload.type: "custom_tool_call"`, `name: "exec"`, and `input` = a plain
  STRING holding script text that itself invokes
  `tools.exec_command({"cmd":` cmux send --surface <ref> <prompt> && cmux
  send-key --surface <ref> enter `,…})`. The observed maipipe cmux lead
  (2026-07-21) carries 60 such calls and ZERO recognizable
  herdr/cmux-agent function_calls — so an orchestrator latch reading only
  function_call cmd args misses cmux leads entirely, and the lead's own
  sol+terra turns attribute to every primary ticket its kickoff's first
  line names (manufactured blocking verdicts, observed live). Worker
  models inside these scripts are template-built (`${JSON.stringify(…)}`)
  and NOT reliably extractable — cmux cluster evidence remains the worker
  scan plus D26's records-at-source; the latch, however, MUST scan
  custom_tool_call input text for team-dispatch markers (`herdr|cmux agent
  start|prompt`, `cmux send --surface`), non-anchored (script-embedded).

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

## Closed 2026-07-27 (codex-audit effort, I048 live acceptance)

The codex reader shipped (I040-I049, feat/codex-audit): discovery with
CODEX_HOME/default + --codex-sessions override, dispatch records
(spawn_agent + herdr team spawns), spawned-thread actuals by root id,
D21 worker scan (first-line, injected-preamble-aware, multi-token-ambiguous),
guardian exclusion, D22 cwd-or-commit scoping, unattributed-transcript
verdict with source naming, --since/--session, token-scan pruning.
Live evidence (real ~/.codex store, 953 files):

    I024    routine  gpt-5.6-terra              match     (herdr dispatch records + worker sessions)
    I021    primary  gpt-5.6-sol                match
    I008..I015 (M4a) gpt-5.5                    unmapped-dispatch  (honest history: id never declared)

Guardian threads contribute nothing anywhere (zero codex-auto-review
strings). Four wire-shape surprises found live and recorded above as dated
2026-07-27 facts (cmd key; injected preambles; polymorphic source;
custom_tool_call cmux leads) — the degrade-never-fail posture held for all
four: every failure direction was missed evidence, never a false blocker.
