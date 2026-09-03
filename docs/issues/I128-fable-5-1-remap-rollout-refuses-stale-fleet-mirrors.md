---
id: I128
title: "Fable 5.1 primary remap (68aa28f) refuses dispatch on every unrefreshed fleet mirror as retired-model, with no rollout signal, a stuck override path, and host-config coupling"
severity: med
status: fixed
commits: [4e1dee0, 5acb7f7, aa1d551]
affects: [I051, I072, I063]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## Problem

Findings from a primary-tier code review of commit 68aa28f (the
`claude.primary` remap to `claude-fable-5-1`, shipped 2026-09-02 without a
ticket, spec pair, or spec-review record — the only `models/defaults.json`
commit in history without one). Each was reproduced with the HEAD binary.

1. **Rollout window with no signal.** Moving `claude-fable-5` into the
   primary row's history makes `spine model validate claude primary` exit 1
   `retired-model` on every fleet repo whose WORKFLOW.md still mirrors
   `claude-fable-5` (20 of ~28 checkouts under `~/Projects/github.com` at
   review time; only maipipe and maikanban were refreshed). The claude-team
   skill runs that validate as its dispatch preflight and, on non-zero exit,
   tells the operator to rebuild spine — the actual remedy is
   `spine update --dir R --write`. `spine doctor` on such a repo reports only
   D2 "behind template generation" and never names the retired id.
2. **Stuck override.** A mirror that pins the retired id at a non-default
   effort (`claude.primary: claude-fable-5 @ xhigh`, which the gen-9 to 10
   migration itself minted) is refused as retired-model; the refusal's
   remedy (`spine update --write`) preserves the override verbatim and
   changes nothing, so the repo can never validate without a hand edit the
   message does not describe.
3. **Host-config coupling.** `applyHostConfig` matches the resolved id
   byte-exactly with no history awareness. A `routing-host.json` that still
   lists only `claude-fable-5` makes every refreshed repo's primary tier
   unreachable (exit 2, doctor D16), and the retired-model remedy moves
   stale repos into that state. Nothing in the commit, handoff, or remap
   precedent mentions updating host configs; the I072 doctor fixture still
   ships with only the old id.
4. **Test precision.** Several render locks assert
   `strings.Contains(x, "claude-fable-5")`, a substring of the new id, so
   they pass vacuously after the remap. The gen-13 to 14 pristine lock checks
   "row changed => itemized" but not the converse, and the
   `modelRefreshMirrorRows` text exemption has no negative control; the
   ten per-generation lock skip blocks should share one helper that
   sanctions a mirror-row diff only when that lock's own `ModelRefreshes`
   itemizes it (as gen13to14 already does). `modelDefaultDivergence`
   compares a retired `model_default:` against the resolved primary without
   the row's history, so a gen-0 repo hand-set to the old id is now skipped
   as a divergence instead of retiring quietly (low; one-time `--force`).

## Fix

Split by urgency:

1. Rollout (now): refresh the remaining fleet mirrors (`spine update --dir R
   --write` per repo, or a fleet sweep), fix the claude-team preflight
   message to name the real remedy, and give doctor a finding that names a
   retired mirrored id distinctly from generation lag.
2. Stuck override: make the retired-model remedy correct for overrides
   (either migrate a historical-id override to the current id keeping its
   effort, itemized as a refresh, or say "edit the override" in the message).
3. Host config: decide whether history-aware matching belongs in
   `applyHostConfig` or whether host configs are refreshed alongside
   mirrors; update the I072 fixture and the remap precedent docs either way.
4. Tests: replace the substring locks with `hasRow` on the exact id, add the
   converse assertion to the gen-13 to 14 lock, add a negative control for
   the text exemption, and consolidate the skip blocks.

Retroactively, the remap itself needs its PRD pair and spec-review record
(the I063 routine remap shows the shape).

## Acceptance criteria

- [x] A fleet repo mirroring the retired id fails the dispatch preflight with a message naming `spine update --dir R --write`, and `spine doctor` names the retired mirrored id.
- [x] A repo with `claude.primary: claude-fable-5 @ xhigh` can reach a validating state by following the printed remedy alone.
- [x] A host config listing only the retired id either resolves the current primary or is reported by doctor with the host-config remedy, and the I072 fixture carries the current id.
- [x] Reverting `models/defaults.json` primary to `claude-fable-5` with empty history fails the render locks it currently passes.
- [x] `docs/specs/` carries a design/plan pair for the 68aa28f remap and the ticket records its spec-review.

<!-- Record an approved-without-test exception using the exact grammar in WORKFLOW.md's Acceptance exceptions section. -->

## Resolution

Shipped 2026-09-03 in the `i128-remap-rollout` effort (PRD
`docs/specs/2026-09-02-i128-fable-5-1-remap-rollout-design.md`; inline
execution, justified in the plan: one successor predicate drives the update
migration, doctor, locks, and the preflights). Test-first throughout.

- **Rollout.** Doctor D18 names a retired mirrored id per (harness, tier)
  with its successor and `spine update --dir R --write`, using the strict
  launch selection so it fires exactly where validate refuses; D2 still
  reports the generation lag. The three deepthought preflight blocks
  (claude-team, codex-team, handoff-to-codex; deepthought 0a39e20) relay a
  validate refusal verbatim, keep the rebuild text for a missing or old
  binary (exit 127 or an unknown-command/usage line), and relay any other
  failure as a configuration error; the guard script covers all three arms
  (103 pass). The fleet sweep ran at deploy; the handoff lists the checkouts
  written.
- **Retired override.** `spine update` migrates an override whose id is a
  historical id of its harness to the successor (`model.SuccessorID`),
  keeping the effort and alternate, itemized as
  `model refresh (retired override)`; `claude.primary: claude-fable-5 @
  xhigh` becomes `claude-fable-5-1 @ xhigh` and validates. The resolver's
  I063 pair-aware classification is unchanged; ADR 0011 carries a dated
  note and CONTEXT.md the term.
- **Host config.** Byte-exact matching stays (I051 contract). D16's
  unreachable message names the host file and, when the host lists a
  historical id of the lineage, says it is retired. Both host fixtures carry
  `claude-fable-5-1`. Not live-verifiable here (no host file on this
  machine); covered by tests.
- **Tests.** Exact-row locks replace the three substring locks; the gen-13
  to 14 lock checks the converse; all ten generation locks share
  `sanctionedRefreshLine`, which admits a mirror-row diff only when that
  lock's own report itemizes it, with a negative control; the static
  allowlists are gone. `modelDefaultDivergence` retires any historical
  primary id quietly. Negative control run twice (build and verify): the
  table reverted to Fable 5 with empty history fails 16 tests.
- **Retroactive PRD.** `docs/specs/2026-09-02-claude-primary-fable-5-1-remap-{design,plan}.md`
  record 68aa28f. Spec-review of 68aa28f against that design (primary,
  blind, 2026-09-03): (a) four items the first draft claimed but the commit
  lacked (template test, itemization-coupled locks for gen 9-10 and 11-12,
  historical-id discoverability and refusal tests); (b) two behaviours the
  draft omitted (the gen 11-12 blanket skip; the gen 9-10 minted override
  wording); (c) two wording mismatches. All corrected in the design and
  plan (aa1d551); the remap checklist was confirmed accurate on all five
  items.
- **Reviews of this diff.** Spec-review (primary, blind): pass with two
  minor mismatches and one spec omission, fixed. Code review (primary):
  approve spine; two deepthought findings (exit 2 misread as old binary,
  `echo` escapes), fixed; the trailing-comment mirror row that still
  defeats the printed remedy is pre-existing and cut as I130. Fresh-context
  verification (primary): PASS on AC1-AC4 under execution with negative
  controls; AC5 completed by this record.
- Lane: maipipe full #84 passed at 5acb7f7; the final lane number is in
  the effort handoff.
