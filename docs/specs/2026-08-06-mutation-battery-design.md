# Mutation battery — design (PRD)

**Date:** 2026-08-06 · **Effort:** mutation-battery · **Stage:** prd
**Input:** `docs/research/2026-08-05-behavioural-mutation-battery.md` (GRILLED
2026-08-06, primary tier; all decisions ratified by owner — see its *Grill outcomes*)
**Evidence:** n=5 kill-rate table reproduced from clean copies, 0 no-site / 0
build-err (`docs/research/2026-08-06-mutation-battery-repro/combined.txt`)

## Problem

The estate's verify gate accepts a green test suite as evidence. That signal is
provably blind: qwen-3.6-27b ships a green, zero-skip, `-race`-clean suite with real
coverage that detects **zero of eight** behaviour changes — over a binary whose own
eval record says it cannot log in. A behavioural mutation battery discriminates where
pass/fail cannot (frontier 25%, mutation-remediated 62%, broken 0%).

## What ships (decision summary, from the grill)

The battery is an **agent-assisted instrument** consumed by the `/model-eval` loop,
adopted as a **reporting gate**: eval records must *carry* the battery result; there
is **no pass threshold**. Sites are authored per tree (`sites.sh` candidates → agent
writes the literal spec → `mutate.py` `NO-SITE`/`BUILD-ERR` rows validate
mechanically). The record carries the per-class verdict matrix plus a required
one-line distinct-cause summary for survivors; the scalar, when quoted, is
killed / valid-scorable-probes.

## Deliverable 1 — battery convention

- **D1a — checklist document** at spine `docs/mutation-battery-checklist.md`: the 10
  runnable classes with provenance marks exactly as grilled — classes 8/9
  `[report-only]` (excluded from scored denominator), classes 2/10 `[CANDIDATE]`
  (no probe data; graduate when a wired tree runs them) — plus the record format,
  reporting rules (build-breakers excluded and disclosed), and the fixture-strength
  **reviewer instruction** (former class 11, not a battery entry).
- **D1b — runner bundled with the `/model-eval` skill**: `mutate.py`, `sites.sh`,
  and the batch flow move into the skill's files; the skill gains instructions for
  the site-authoring loop (candidates → literal spec → standalone validation, where
  a `NO-SITE` row means the spec is wrong, not the tree) and for the reporting
  requirement including the cause annotation.
- **D1c — record convention**: battery result rides the eval record's
  Audit/Rescore **body** — **zero spine code**, no schema change, consistent with
  ADR 0007's opaque-record stance. (Amended 2026-08-06, owner-ratified reviewer
  finding R2: the `/model-eval` skill's ledger contract fixes the `stage`
  vocabulary and defines `score` as the rubric total, so the matrix cannot live
  in those fields.) Presence is required by the skill's process; a `spine doctor`
  check is out of scope unless a threshold is ever adopted. The runner emits
  **both** a raw rate over all valid probes and the scorable rate
  (killed / valid-scorable, `report_only` probes excluded); the record quotes the
  scorable one (amendment R1).
- **D1d — ADR 0013**: records the packaging decision (checklist in `docs/`, never
  `templates/` per ADR 0004; runner outside spine; record rides opaque score;
  no enforcement code).
- **D1e — artifact relocation**: per-tree specs (`scripts/*.json`,
  `scripts/specs/*.json`) move beside the eval records they describe in the
  consuming repo's `docs/evals/` area; reproduction evidence is preserved under
  spine `docs/research/`.

## Deliverable 2 — do-not-regress block

A generated block prepended to every remediation dispatch in the `/model-eval` loop,
listing what the previous round verified working (file:line + proving probe/test),
closing with both fixed rule lines: "Breaking one of these costs more than any fix
below gains." / "Report any that you must break, and why, before you break it."
(RA1 amendment 2026-08-06 — the research doc's two-line close is normative.) Ships as a template +
dispatch-prep instruction in the same skill. Evidence: two self-inflicted
regressions in the corpus (ornith-35b r1 latent ordering bug, r2 firing `t.Skip`;
Laguna r2 gains "paid for by a regression").

## Non-goals (explicitly out of scope)

- AST-based site discovery, or any unattended/24×7 operation (no named consumer).
- A pass threshold, and any `spine doctor` enforcement of the record field.
- Making B7/B8 gradable — the harness-style question is **filed as future work
  against the eval repo**, not built here.
- Generic mutation tooling (go-mutesting et al.); the class taxonomy is the point.
- Spine binary changes of any kind.

## Acceptance criteria

1. Checklist doc exists in spine `docs/` with all provenance marks intact; ADR 0013
   committed; neither touches `templates/` or spine Go code (`git diff --stat`
   shows docs-only for D1a/D1d).
2. `/model-eval` skill contains the runner and can execute the battery end-to-end on
   a wired tree from its new home, reproducing a known **raw** kill rate (the n=5
   table's rates are raw 8-probe rates; the scorable rate is additionally emitted —
   amendment R1).
3. **Negative control (load-bearing guard):** corrupting one spec string produces a
   `NO-SITE` row and the run reports it excluded — proving the literal-match guard
   actually fires from the skill-bundled path.
4. Skill instructions produce a record containing the full matrix + cause line for
   at least one real eval record (format check, not a threshold check). *(Amended
   2026-08-06, reviewer finding R4: not satisfiable inside I054's file scope —
   evidence for this criterion is collected at the effort's verify stage against a
   live eval run.)*
5. DNR template exists in the skill with the dispatch-prep instruction; a sample
   generated block for the Laguna round history renders correctly.
6. Spine `scripts/` no longer carries per-tree specs after relocation; research-doc
   references updated; reproduction evidence retained under `docs/research/`.
