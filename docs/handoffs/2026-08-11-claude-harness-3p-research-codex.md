---
title: "claude-harness-3p-research-codex"
created: 2026-08-11
handoff_ordinal: 5
---

# Handoff — claude-harness-3p-research-codex (2026-08-11)

## Context

Codex-team dispatch: resolve two **wayfinder research tickets** from the I066
map ("Claude Code as a harness for 3rd-party models"). Deliverables are
research answers appended to the tickets, not code.

Read first, in order (all paths repo-relative to /Users/ldh/Projects/github.com/spine):

1. `docs/issues/I066-claude-harness-3p-models-wayfinder-map.md` — the map;
   Destination + Decisions so far are ratified, do not relitigate.
2. `docs/issues/I070-proxied-model-ids-in-claude-transcripts.md` — ticket 1.
3. `docs/issues/I071-claude-auto-invocation-contract.md` — ticket 2.
4. `docs/issues/README.md` §Wayfinding operations — the resolve convention:
   append `## Resolution`, set `status: fixed`, index the gist in the map's
   Decisions so far.
5. `WORKFLOW.md` §Model routing — the tier/record grammar the answers feed.

Relevant source for I070: `internal/audit/` (transcript parsing for
`spine audit routing`), `internal/model/` + `models/defaults.json` (id table,
alias-row question). `spine audit routing` currently reads per-turn model ids
from Claude/Codex session formats.

## State (verify before relying)

- Repo: /Users/ldh/Projects/github.com/spine, branch main @ `40aba1d`, pushed,
  clean except untracked `.DS_Store` (leave it).
- `spine doctor`, `spine audit stages`, `spine audit routing` all exit 0 as of
  dispatch.
- I070/I071 frontmatter already set `status: in-progress`,
  `assignee: codex-team` (committed with this doc).
- **This host** (verified at dispatch): `claude-auto` is NOT on PATH — it is
  the owner's work-laptop wrapper (Claude Code + custom gateway → GPT models).
  Locally present: `ollama` (/usr/local/bin/ollama, only nomic-embed-text
  pulled) and LM Studio CLI (`~/.lmstudio/bin/lms`). No `ANTHROPIC_BASE_URL`/
  Bedrock/Vertex env overrides active. Claude Code session transcripts live
  under `~/.claude/projects/<escaped-cwd>/*.jsonl`.

## Next steps

- **I070** — determine what model-id strings land in Claude Code transcripts
  when a non-Anthropic model answers via a gateway/proxy. Empirical lane
  available on this host: serve a small open-weight model through an
  OpenAI-compatible endpoint (lms or ollama; pulling a small chat model is
  acceptable), point a throwaway Claude Code session at it via env
  (`ANTHROPIC_BASE_URL`/auth token per Claude Code docs), run one trivial
  turn, then inspect the session `.jsonl` for per-turn model ids. Compare
  against what `internal/audit` parses. Answer: observed-id shape, stability,
  whether `models/defaults.json` needs alias rows, and whether the observed id
  supports the confirm half of declare-then-confirm (I069).
- **I071** — pin down the invocation contract for selecting model + effort
  per dispatch through Claude Code. `claude-auto` itself is not on this host:
  document the general contract from what IS verifiable here (`claude --help`
  model/env surface, settings.json model keys, env overrides), produce the
  invocation matrix (harness invocation × model selection × effort selection)
  and the touchpoint list (cmux/herdr claude-team skills at
  `~/Projects/github.com/deepthought/skills/`, dispatch transports), and
  **explicitly flag** every cell that needs work-laptop verification of
  `claude-auto` by the owner — flagged unknowns are acceptable; guessed
  contracts are not.
- Resolve both tickets per the convention (Resolution + status + map index),
  commit, push.

## Task breakdown hints

- Two workers, one ticket each; I070 and I071 are independent — parallel is
  fine. Reviewer per codex-team convention; reviewers run a
  requirements-attack on the ticket text itself first (contradictions surfaced
  with proposed resolution, never silently resolved).
- Findings style: match `docs/research/2026-08-05-routing-yield-feasibility.md`
  — verified commands/paths cited, no claims from memory. State the exact
  command run and paste the relevant output as evidence.
- If the empirical lane for I070 is blocked (no model pullable, endpoint
  refused), degrade explicitly: answer from transcript-format source reading
  plus a documented repro recipe for the owner, and say so in the Resolution.

## Gotchas

- **Cursor sole-writer rule**: the cursor block at the bottom of this file is
  spine-owned. Never edit it, never reproduce its literal opening marker in
  any prose you write (the parser latches onto it — I064). Cursor state
  changes only via `spine cursor` verbs — you should not need any.
- Commit hygiene: stage explicit paths only, never `git add -A`/`.`;
  `.DS_Store` and `.superpowers/sdd/` scratch stay untracked.
- The two tickets are the scope. Adjacent map tickets (I072–I077) are
  blocked/unclaimed — do not start them; your resolutions unblock I072/I074/
  I075 for later sessions.
- Effort semantics context for I071: Anthropic effort levels are
  low/medium/high/xhigh/max; table-cell pin notation precedent is
  `claude-opus-5 @ low` (see WORKFLOW.md mirror rows).

<!-- spine:cursor -->
effort: i063-estate-remap
prd: docs/specs/2026-08-10-estate-default-claude-routine-remap-design.md
tickets: I063
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
