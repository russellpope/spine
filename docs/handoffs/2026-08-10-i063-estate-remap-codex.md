---
title: "i063-estate-remap-codex"
created: 2026-08-10
handoff_ordinal: 2
---

# Handoff — i063-estate-remap-codex (2026-08-10)

## Context

You are the codex team lead for the **I063 build** in
`/Users/ldh/Projects/github.com/spine`. Read first, in order:

1. `docs/issues/I063-estate-default-claude-routine-remap.md` — the ticket,
   ratified by the owner 2026-08-10 (sonnet-5 ban permanent, build approved).
2. `AGENTS.md` + `WORKFLOW.md` — dispatch contract, escalation grammar, model
   routing (tiers only; resolve ids via `spine model`).
3. Prior art: `docs/issues/I035-refresh-rule-model-keys.md` (refresh rules for
   model keys), `docs/adr/0011-model-table-resolves-in-spine-keyed-by-flavor.md`
   and `docs/adr/0004-templates-compile-into-the-binary-with-a-single-integer-generation.md`
   (whether this needs a generation bump is a decision this build must make
   and record).

## State (verify before relying)

- main at `17a7bb7`, even with origin/main. Untracked `.DS_Store` and
  `docs/research/2026-08-05-routing-yield-feasibility.md` — leave both alone.
- `spine doctor`, `spine audit stages`, `spine audit routing` exit 0;
  `go test ./...` green. Live binary `~/bin/spine` deployed from `17a7bb7`'s
  tree (I062 ordinal mechanism live — this handoff carries `handoff_ordinal`).
- Where the change lives: the estate default table is
  `models/defaults.json` — `flavors.claude.routine` is `claude-sonnet-5`
  (banned). `MirrorRows()` in `internal/model/` renders it into
  `{{MODEL_ROUTING_ROWS}}` at `internal/tmpl/tmpl.go:111`; `spine update`'s
  refresh rules decide whether existing estate WORKFLOW.md files pick up the
  new default (`internal/update/`, see the stock-row strings there).
- Spine's own WORKFLOW.md already carries the owner override
  `claude.routine: claude-opus-5 @ low` — that row must survive the change
  as an override (or become identical to the new default; either way
  `spine update` must stay idempotent on it).

## Next steps

1. `spine cursor start` for the i063 effort (sole-writer rule; expect `audit
   stages` to block after any cursor write until your shipped handoff — ADR
   0014 says that rhythm is ratified, account for it, don't fight it).
2. Build: `flavors.claude.routine` becomes `claude-opus-5` with effort `low`
   (grammar precedent: codex.primary carries `"effort": "xhigh"`); move
   `claude-sonnet-5` to that entry's `history` (precedent: claude.fallback
   carries opus-4-8 history). Decide alias handling and record it.
3. Decide and record: generation bump vs plain sweep-refresh for estate
   pickup (per ADR 0004 + I035 refresh rules); record the decision in the
   ticket Resolution, and in an ADR if the mechanism changes.
4. Tests: MirrorRows round-trip (`internal/model/model_test.go` has the I036
   coverage test), update/refresh-rule tests (`internal/update/`), full
   `go test ./...`. Note some existing tests assert the literal
   `claude-sonnet-5` stock strings — updating those expectations is in scope;
   gutting what they guard is not.
5. Ship per convention: explicit-path commits, doctor + both audits green,
   ticket Resolution, `spine handoff new` shipped handoff. No push; binary
   install is the owner's deployment action.

## Gotchas

- NEVER hand-edit any spine cursor comment block — and never quote its
  literal opening marker in handoff prose either; the parser latches onto the
  quoted marker and reds out doctor/stages (live-found, filed as I064).
- Historical `claude-sonnet-5` strings in old tickets, handoffs, and gen
  migration fixtures (`internal/update/testdata/`, gen5to6/gen9to10 tests)
  are records, not live config — do not sweep-edit history.
- The auto-mode permission classifier denies compound rebase+push chains and
  workspace closes — surface exact commands to the owner instead.
- Do not touch `PICKUP.md` or the untracked research doc.

<!-- spine:cursor -->
effort: i062-handoff-tiebreak
prd: docs/specs/2026-08-06-cursor-writes-design.md
tickets: I062
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
