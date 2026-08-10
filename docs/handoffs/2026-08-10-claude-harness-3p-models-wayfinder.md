---
title: "claude-harness-3p-models-wayfinder"
created: 2026-08-10
handoff_ordinal: 4
---

# Handoff — claude-harness-3p-models-wayfinder (2026-08-10)

## Context

Reference doc for the next session's task: a **wayfinder-style brainstorm** of
a new spine feature batch around **using Claude Code as a harness for
3rd-party models**. The owner said "brainstorm" but means /wayfinder — the
output should be a wayfinder map issue plus child tickets in `docs/issues/`
(convention: "Wayfinding operations" in `docs/issues/README.md` — map issue
labelled `wayfinder:map`, children carry `parent:`), not a PRD yet.

### What spine already has in this space (read before ideating)

- **Tier→model contract**: artifacts reference tiers, never ids (ADR 0010);
  the id table resolves in spine keyed by flavor (ADR 0011,
  `models/defaults.json`, `spine model <flavor> <tier>`, `MirrorRows()` →
  WORKFLOW.md mirror rows). Two flavors exist today: `claude`, `codex`.
  WORKFLOW.md §Model routing explicitly anticipates "new model families,
  local models, other providers".
- **Audit**: `spine audit routing` correlates dispatch records + Claude/Codex
  transcripts to ticket tiers; verdicts include escalated-with-reason,
  silent-descent (blocking), unmapped-dispatch, exempt (`tier: n/a`).
  A third-party-model harness only counts if the audit can read its evidence.
- **Escalation/fallback grammar** in WORKFLOW.md; fallback tier exists partly
  for refusal rerouting.
- **Local models**: the `/model-eval` skill + `docs/evals/` convention +
  mutation battery (I053–I056, shipped) already evaluate local models; the
  model table was designed to hold local ids (I033).
- **Fresh precedent**: the sonnet-5 ban → `claude.routine: claude-opus-5 @
  low` shipped estate-wide today (I063) — the first live exercise of
  swapping a model out from under a tier, history preserved in the table.
- **Untracked but relevant**: `docs/research/2026-08-05-routing-yield-feasibility.md`
  (routing-yield feasibility research, sits uncommitted in the tree) and the
  I052 routing-yield feedback charter ticket. Likely seed material — read
  both before the brainstorm; decide whether the research doc should finally
  be committed as part of this effort.

### Threads worth pulling in the brainstorm (starters, not conclusions)

- Claude Code can run 3rd-party/local models via env/config (e.g.
  ANTHROPIC_BEDROCK/VERTEX, proxies, OpenAI-compatible gateways to local
  models) — what would spine need so a *claude-flavored harness running a
  non-Anthropic model* stays first-class: flavor vs harness split? a third
  axis (harness=claude-code, model-family=X)?
- Transcript attribution: audit routing currently reads per-turn model ids
  from Claude/Codex session formats — what do proxied 3rd-party ids look
  like in Claude Code transcripts, and does the resolver/table need alias
  rows for them?
- Tier semantics when the pool is heterogeneous (a local 70B as mechanical?
  routing-yield feedback loop from eval records into table changes?).
- Fleet/estate story: per-repo overrides already work (objectstudio,
  maikanban); what's the estate default story for a mixed-provider table?

## State (verify before relying)

- main at `48d1a25`, pushed, clean except untracked `.DS_Store` and the
  routing-yield research doc. `spine doctor`/`audit stages`/`audit routing`
  all exit 0; live binary current (`make install` run today, embeds the
  I062 ordinal tiebreak + I063 opus-5-low default).
- Estate: sweep complete — 17/18 repos resolve `claude routine →
  claude-opus-5`. Pending: **maipipe** unswept (codex team I190–I193 was
  active in it; sweep = `spine update -write` + WORKFLOW.md commit + verified
  push after they finish), **praxis** sweep commit local atop a ~70-commit
  origin/main backlog (owner pushes), **hbmview/moo-clone** sweep commits on
  branch/no-remote.
- Open spine tickets: I064 (cursor-marker literal in prose breaks handoff
  parser), I065 (estate issues-README stale stock text defeats sweep).
  Both small, neither blocks the brainstorm.

## Next steps

1. Read the two seed docs (routing-yield research + I052), then run the
   wayfinder brainstorm; publish map + tickets to `docs/issues/`.
2. Whatever the map decides, the build path afterward is the standard gate
   chain: grill → PRD → tickets → dispatch per WORKFLOW.md routing.

## Gotchas

- Sole-writer cursor rule estate-wide; never quote the literal cursor opening
  marker in handoff prose (I064 — the parser latches onto it).
- After any cursor write, `audit stages` blocks until a fresh
  `spine handoff new` — ratified as intended behavior (ADR 0014).
- Auto-mode classifier: multi-repo batch scripts and some compound
  git chains get denied nondeterministically — decompose to per-repo
  single commands; hand `cmux close-workspace` to the owner.
- Bash tool shell behaves zsh-like (`$status` works, `(cmd)` substitution in
  double quotes does not); multi-repo scripts go in the scratchpad and run
  via `bash file.sh` — but expect the classifier to deny script-driven
  git/write batches; only read-only survey scripts pass reliably.

<!-- spine:cursor -->
effort: i063-estate-remap
prd: docs/specs/2026-08-10-estate-default-claude-routine-remap-design.md
tickets: I063
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
