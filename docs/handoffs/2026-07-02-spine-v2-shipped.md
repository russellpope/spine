---
title: spine v2 shipped
created: 2026-07-02
---

# Handoff — spine v2 shipped (2026-07-02)

## Context

spine v2 designed, planned, built, reviewed, and live-accepted in one session (full superpowers
spine: brainstorm → spec → plan → subagent-driven execution → final whole-branch review → live
acceptance). Spec `docs/specs/2026-07-02-spine-v2-design.md`, plan `-plan.md`, execution ledger
`.superpowers/sdd/progress.md` (gitignored).

## State (verify before relying)

- spine main @ f67fc93 (FF from build/v2, 24 commits, branch deleted). **NOT pushed** — origin
  (`git@github.com:russellpope/spine.git`) still has no main.
- `~/bin/spine` = generation 2. New surface: `adopt` (dry-run/exit 1/--write, stamped-profile-wins,
  diffs in dry-run, pure `--json` + `pending`), `handoff new|list|latest [--fleet DIR]`,
  `eval new|add-run|list`, `adr list --json`, doctor D1-profile-aware + D7 (evals) + D8 (handoff
  naming). Profiles + swift/infra/knowledge with per-profile manifest (knowledge: no
  harness-interface/issues). ADRs 0006–0009 (0009 = hand-authored `docs/adr/README.md` preserved
  as user-owned; `--force` converts).
- Live fleet state, all committed locally in each repo, nothing pushed anywhere:
  praxis 42acc16 (go-service; hand ADR index byte-preserved; CLAUDE invariants intact),
  moo-clone c43d486 (swift), home-lab-admin 7abb4f1 (infra), obsidian-ep-vault 6ff7b4b
  (knowledge), ccq 114d4f0 (gen 2 + ADR status casing + docs/handoffs dir), hbmview 77a9a09
  (gen 2), local-model-evaluation 8a7c40e (docs/evals/ govmomi eval + 8 backfilled run records:
  4 audited, 4 rescored; `spine eval list` renders scores; D7 clean).
- Post-conditions verified live on all four adopts: adopt-write 0 / doctor 0 / update no-op 0 /
  re-adopt idempotent 0.

## Next steps

- **/model-eval skill** (Russell's stated next build): drives wire→audit→score→compare→remediate→
  rescore against the now-machine-readable `docs/evals/` convention (ADR 0007 seam: spine owns
  schema, skill owns process; stage vocabulary = run-template body sections).
- Remaining fleet adopts post-ship, one `spine adopt` each as repos get touched: ultima-dci-edition,
  notetui, jarvis, ai-virt-framebuffer, ai_infra_notes, observability_notes, pure-automation
  (+ local-model-evaluation itself if wanted). Same gate: dry-run → review → --write.
- Push spine to origin when Russell says (first push; also praxis push uses remote `github`).
- v3 ledger (in `.superpowers/sdd/progress.md`): fsutil exclusive-create (TOCTOU), adr.tmpl.md
  scalar quoting + octal-id quirk (template edit = generation bump), preserve-heuristic vs future
  adr-README template evolution, fleet age_days UTC off-by-one, `handoff list` path column,
  eval list header, update-output silent on preservation, deferred minors list.

## Gotchas

- **Never edit existing `templates/current|gen0` content within a generation** — the ccq fixture
  test (`TestGen1To2IsStampOnly`) fails the build if a gen bump is not stamp+marker-only.
- Hand-authored `docs/adr/README.md` is user-owned (ADR 0009): update shows it as up-to-date,
  doctor D4-info; `--force` is the only conversion path and REGENERATES (destroys) it.
- adopt follows a valid stamped profile over detection; conflicting `--profile` is a hard error.
- local-model-evaluation has pre-existing uncommitted model-tree modifications (predate this
  session) — left untouched, only docs/evals/ was committed.
- Review-loop yield this session: 6 plan-authored defects caught (meta.Parse colon widening,
  handoff Stat-guard overwrite path, fleet error-branch coverage, adr --json empty-ledger, ADR
  0007 YAML title, adopt --json prose) + 1 Critical spec-level gap (praxis ADR index) found only
  by the final whole-branch review's acceptance simulation. Per-task reviews alone would have
  shipped it.
