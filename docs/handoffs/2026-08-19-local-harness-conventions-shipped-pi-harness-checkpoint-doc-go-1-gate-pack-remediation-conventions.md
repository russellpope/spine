---
title: "local-harness-conventions shipped: pi harness, checkpoint doc, go@1 gate pack, remediation conventions"
created: 2026-08-19
handoff_ordinal: 6
---

# Handoff — local-harness-conventions shipped: pi harness, checkpoint doc, go@1 gate pack, remediation conventions (2026-08-19)

## Context

Effort `local-harness-conventions` (spec `docs/specs/2026-08-18-local-harness-conventions-design.md`,
plan `…-plan.md`, ADRs 0015/0016, tickets I079–I087) — the spine half of
deepthought's weak-local-model harness map (I023). Grilled 2026-08-18 in the
deepthought session; built 2026-08-18/19 by a claude-team on herdr (lead
fable-5 @ high, routine implementers opus-5 @ low, primary review on I080/I085
and the final whole-branch pass). Branch `local-harness-conventions` merged to
main as `2132d89` (no-ff, 28 commits). Team report with all 35 rulings:
`~/Projects/github.com/spine-wt-local-harness/.superpowers/sdd/team-report.md`
(worktree kept as evidence; delete after reading).

Shipped: `pi` harness rows + `alternate` cell (`spine model --alternate pi
<tier>`, mirror `alt:` suffix); `spine checkpoint new|latest|list` + reload
preamble + handoff embedding + doctor D11; `spine gate go <check>` ×8 + `mutate`
(go@1, results contract, positive-control pairs, stdlib only); template gen 11
(`gate_pack*` WORKFLOW keys, `maipipe.toml` gate-go/mutation-go region, doctor
D10, `docs/remediation/` scaffold); remediation hitlist/round templates +
round-budget advisory in `audit stages`.

## State (verify before relying)

- main `2132d89`+handoff commit; `~/bin/spine` reinstalled (`spine version` →
  gen 11); `make test` green (17 pkgs); `spine doctor` → only pre-existing D4
  (`docs/issues/README.md`, I065); `spine audit stages` clean after this
  handoff; `spine audit routing` → `no-transcript` for I079–I087 (see gotchas).
- Cursor: all stages `[x]` (snapshot below). Primary ledger
  `.superpowers/sdd/progress.md` carries the per-ticket evidence block.
- Not pushed. Worktree `spine-wt-local-harness` still registered.

## Next steps

1. **Dogfood before self-enabling**: `spine gate go tskip --dir .` reports 4
   real skips; `deferred-cleanup-errcheck` 2 sites (internal/audit/audit.go,
   internal/gate/binaryhygiene.go); `dead-code-callgraph` flags
   `model.TierDefaultEffort`. Fix, then set `gate_pack: go@1` in spine's own
   WORKFLOW.md and `spine update --write`.
2. **Ticket: audit routing blind to herdr/cmux team spawns** — it parses only
   Task/Agent tool-use blocks; claude-team dispatches are Bash `herdr agent
   start … --model …`. Until fixed, claude-team runs show `no-transcript`.
3. Owner data check: pi `alternate` equals primary/fallback (qwen @ xhigh) —
   spec-mandated; change when a second model is served. D11's ".superpowers/
   unignored" arm is `warn` (reviewer reservation: consider `info`).
4. Cross-repo follow-through (deepthought session owns): maipipe cross-product
   ticket (region tolerated in `maipipe.toml`, `spine` on daemon PATH, `code`
   field, pipeline names); deepthought PRD amendments section; `/model-eval`
   skill to call `spine gate go mutate`. Then Phase 1 maipipe grill.
5. Deferred-minor ride list: `spine-wt-local-harness/.superpowers/sdd/2026-08-18-local-harness-conventions-plan/progress.md`
   (`Task N: minor (deferred)` lines) → one follow-up ticket.

## Gotchas

- `spine model` flags go BEFORE args: `spine model --alternate pi routine`.
  Tier names are primary/routine/mechanical/fallback — there is no `review`
  tier; pi efforts are `low|medium|xhigh` only.
- `spine cursor tick` refuses without per-ticket evidence lines in the
  primary ledger — append evidence below the cursor block (never inside),
  then tick. Tick `handoff` BEFORE `spine handoff new` so the snapshot
  matches.
- Gate config env convention: `SPINE_GATE_<UPPER_KEY>`; `gate_pack_disabled:
  [mutate]` drops the whole `mutation-go` pipeline (a composing lane would
  dangle). `tools/go.mod` false-positives under `binary-hygiene` (go@2).
- go@2 binary cannot render a go@1 pin (reports unknown pack, rewrites
  nothing) — bump pins deliberately.

<!-- spine:cursor -->
effort: local-harness-conventions
prd: docs/specs/2026-08-18-local-harness-conventions-design.md
tickets: I079-I087
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
