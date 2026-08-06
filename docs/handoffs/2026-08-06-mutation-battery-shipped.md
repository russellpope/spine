# Handoff — mutation-battery, SHIPPED (2026-08-06)

Full effort completed in one day: grill → PRD → tickets I053–I056 → claude-team
build on cmux → fable-5 whole-branch review → fix wave → verify → ship. All
gates green; all owner decisions ratified in-session.

## Stage cursor

<!-- spine:cursor -->
effort: mutation-battery
prd: docs/specs/2026-08-06-mutation-battery-design.md
tickets: I053-I056
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->

## What shipped, where

| Repo | State | Content |
|---|---|---|
| spine | main at **86cf362** (ff from 2d15c2e) | `docs/mutation-battery-checklist.md`, ADR 0013 (amended in place, R2), PRD pair, tickets I053–I056, grilled research doc + reproduction evidence (`docs/research/2026-08-06-mutation-battery-repro/`), grill-entry handoff |
| deepthought | main at **4c06342** (ff from 1b7ad89) | `/model-eval` skill: battery runner (`mutate.py` w/ report_only + dual rates, `sites.sh`, `run-battery.sh` manifest-driven), battery + DNR skill sections, DNR template + Laguna example. Live immediately via `~/.claude/skills/model-eval` symlink |
| local-model-evaluation | **be49c9e** on owner branch `laguna-s-2.1-hf-q8_0-mixed` | 5 mutation specs beside the eval records (B7/B8 `report_only: true`), Laguna record carries the AC4 battery block (raw 5/8, scorable 5/6, distinct-cause summary) |

**Neither main is pushed** (spine ahead of origin by 3, deepthought by 43+ — the
latter mostly pre-existing). Pushing is the owner's call.

## The convention, as adopted

Agent-assisted instrument for `/model-eval`; **reporting gate** — record presence
required by skill process, **no threshold**, no doctor check. Record = per-class
verdict matrix (report-only marks) + one-line distinct-cause summary, riding the
eval record's **Audit/Rescore body** (R2; ADR 0013 records this). Scalar quoted =
killed/valid-scorable; raw rate disclosed alongside (R1). Classes 8/9
report-only; 2/10 CANDIDATE; fixture strength = reviewer instruction. Historical
n=5 table stays raw-denominated. DNR block: generated from prior round's
verified criteria, prepended as section 0 of every remediation dispatch.

## Open items (tracked, none blocking)

- **B7/B8 harness question** — filed as future work against the eval repo
  (prescribed `simulator.Test` style can't reach client-construction/teardown).
- **Classes 2/10** graduate from CANDIDATE when a wired tree probes them.
- **Threshold** revisited only after cause-annotated matrices accumulate.
- **WORKFLOW.md `claude.routine` still maps `claude-sonnet-5`** — owner banned
  sonnet-5 mid-run (substitute opus-5@low); dispatches recorded ESCALATION
  lines. Remap the table if the ban is permanent.
- **PICKUP.md** still untracked and not in `.gitignore` (one-line fix, owner's).
- Haiku-tier pane workers get **no auto permission mode** (documented platform
  gate, all providers) — mechanical-tier cmux dispatches need prompt-babysitting
  or an opus-5@low bump.

## Process notes for successors

- Run record: effort ledger `.superpowers/sdd/progress.md` (per-ticket evidence
  lines + ESCALATION records); task-level detail in
  `.superpowers/sdd/2026-08-06-mutation-battery-plan/` (dispatches, reports,
  reviews — kept as audit trail per estate convention, gitignored).
- `spine audit stages` exit 0, derivation clean. `spine audit routing` exit 0 —
  pane workers leave no Agent-tool transcripts; dispatch files (also copied to
  flat `dispatch-I0NN-*.md` names) + ledger records are the evidence.
- The one Important post-review defect (ADR 0013 carrying pre-R2 wording) was a
  cross-task staleness: the ADR was committed before the review that amended the
  spec it encodes. Worth a checklist item in future multi-ticket efforts: after
  any owner-ratified spec amendment, grep already-completed deliverables for the
  superseded wording.
