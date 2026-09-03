---
title: "fable-5-1-remap-and-ledger-burndown"
created: 2026-09-02
handoff_ordinal: 38
---

# Handoff — fable-5-1-remap-and-ledger-burndown (2026-09-02)

## Context

The open-ledger batch is closed at its terminal `handoff[<]` stage. This
handoff does two things: it records the post-batch Fable 5.1 remap that
shipped today, and it refreshes the cursor snapshot so the D9 stale-snapshot
block clears. The next session opens a new effort, not this one.

Two efforts follow, in order. First, burn down the open ledger (three tickets
remain). Second, take in the research notes and decide what to borrow from
the fusion-harness survey.

## State (verify before relying)

- HEAD `68aa28ffb51eac886863a0cfbc0d4c266e0426df` on main, pushed; origin/main
  matches. Run `git rev-parse HEAD origin/main` to confirm.
- 68aa28f remaps the claude primary tier: `models/defaults.json` ships
  `claude-fable-5-1` with `claude-fable-5` in the row's history. Full test
  suite green; `maipipe run full --wait` passed as run #80 at 68aa28f.
- Both installed binaries (`~/bin/spine`, `~/.local/bin/spine`) were built
  from 68aa28f. Check with `spine model claude primary` (prints
  `claude-fable-5-1`).
- Fleet: maipipe (branch `i221-panel-yank`) and maikanban (main) each carry
  an uncommitted one-line WORKFLOW.md refresh of `claude.primary`. Left for
  their own agents to commit; do not touch from here.
- Open ledger, after this batch: I125 (med, implement derivation ignores
  closed ticket files), I105 (low, opencode custom subagents vs pi choice),
  I112 (low, openweights axis value; parked with the openweights programme
  on the second laptop).
- Untracked and unrelated: `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`
  (status: proposed, no tickets cut). It is the input to effort two.

## Next steps

1. Effort one, ledger burndown: `spine cursor start` a new effort scoped to
   I125 and I105 (I112 stays parked). I125 is the substance: accept ticket
   file closure (`status: fixed` plus non-empty `commits:`) as implement
   evidence, OR'd with the progress-ledger scan, and rescope the I032 typo
   hint when the issues row proves the ids resolve. Run the mandatory gates
   (`/grill-with-docs` -> `/to-spec`, `/spec-review`, verify) per WORKFLOW.md.
2. Effort two, research intake: read the fusion-harness hitlist, grill the
   five items, cut tickets for what survives, and either commit the note as
   the effort's research record or drop it.

## Gotchas

- The stop hook runs the maipipe gate: any commit after a verified lane
  needs `maipipe run full --wait` again before push.
- `handoff[<]` is terminal; never tick it. The D9 stale-snapshot warning is
  cleared by `spine handoff new`, not by a tick.
- Model-table changes: add the displaced id to the row's `history` or every
  fleet mirror reads as an override and never refreshes. The gen-13 fixture
  under `internal/model/testdata` must mirror `models/defaults.json`.
- Generation locks in `internal/update` sanction mirror-row changes only
  when itemized as a refresh (`modelRefreshMirrorRows` in
  `modelrouting_test.go`); add new ids there, do not edit captured fixtures.
- Stage explicit paths only; PICKUP.md and the research note stay out of
  commits unless the effort adopts the note.

<!-- spine:cursor -->
effort: open-ledger-batch
prd: docs/specs/2026-08-29-open-ledger-batch-design.md
tickets: I111,I051,I050,I072,I073,I074,I075,I078,I066,I076,I077,I007,I032,I093,I102,I105,I108,I121,I122,I123,I124
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[<]
<!-- /spine:cursor -->

## Checkpoint (newest): 003-dogfood-the-shipped-local-harness-conventions-on-spine-itsel.md

<!-- spine:checkpoint:facts -->
touched:
- internal/update/gatepack.go
- internal/gate/results.go
- internal/gate/mutate.go
- maipipe.toml
- WORKFLOW.md
gate: pass
sha: 265efc9ede4c229f135c38b558bfe722ec918427
effort_recommended: medium
written: 2026-08-19T16:31:36Z
<!-- /spine:checkpoint:facts -->

### Prior narrative (model-authored, not evidence)

## Task

Dogfood the shipped local-harness conventions on spine itself (deepthought handoff 2026-08-19 §1a–h) and close the cross-repo follow-through (§2).

## Conclusions

- go@1 pack is self-enabled on spine (I089); five classes + mutation-go pass under maipipe at the pinned commit.
- First live maipipe seam found four defects, all fixed: region TOML grammar + schema (I091); results line 0 / file "." / severity "warn" and battery env leak (I092).
- Bake-off positive control: hygiene classes catch committed binaries on 3/3 arms (docs/research/2026-08-19-…).
- Checkpoint round-trip, model alternate provenance, routing blind spot (I090) verified; minor follow-ups in I093.
- Cross-repo: maipipe I201 filed; deepthought spine PRD amended; /model-eval runs the binary.

## Next moves

- Owner: push spine (main ahead of origin, unpushed since 2132d89); close herdr team workspace; remove worktree spine-wt-local-harness.
- Owner call on I093 items 3–5 (unconfigured-class stages, --force scoping, D11 value tamper).
- Phase 1 continues: `/grill-with-docs` in maipipe with deepthought's maipipe execution-floor PRD.
