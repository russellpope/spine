---
title: "I103 merged, pack pin live on main"
created: 2026-08-22
handoff_ordinal: 14
---

# Handoff — I103 merged, pack pin live on main (2026-08-22)

## Context

Session 2026-08-21/22. Merged I104 (B, ADR 0018) + I097, wrote `CHANGELOG.md`, grilled and
specced I103 (ADR 0019, **pack pin** in `CONTEXT.md`), dispatched it to the existing codex
team (reused idle lead, workspace 33), then merged the team's reviewed branch. Prior handoffs
today: `…codex-team-delivered-i104-b-and-i097…`, `…i104-b-and-i097-merged-changelog-added…`,
`…i103-pack-pin-attribution-codex.md` (lead brief), `…i103-pack-pin-attribution-complete-on-
reviewed-branch.md` (team's).

## State (verify before relying)

- `main` = this handoff's commit on `b417d73` (merge `--no-ff` of `i103-pack-pin` @ `8688d9a`,
  6 commits, 25 files). Pushed through `b417d73`; this docs commit gated and pushed after.
- Gates at `b417d73`: gofmt/vet clean, `SPINE_REQUIRE_MAIPIPE=1 make test` green,
  `maipipe run full` #18 passed, `maipipe gate --wait` **green** with all five `gates/*`
  stages executed under the pinned `spine gate go@1 <check>` run lines. Definition hash
  `fad20a5e3a954ee45ab04b334f1e381e2170bc6b`, baseline `no_baseline` — this repo has never
  run `maipipe gate approve-definition`, so none was demanded. Owner call whether to start.
- `~/bin/spine` rebuilt via `make install` at `b417d73` and the maipipe daemon restarted
  (`stop-if-idle` → `start`) so stages run the new binary — required, the old binary exits 2
  on `go@1`.
- I103 `status: fixed`; I104, I097 fixed. Worktrees/branches for all three removed.
  `../spine-wt-2` (`codex-wt-2`, clean) still stale. cmux: lead 33 gone; no `spine: workers`
  panes listed — the group may be empty, check before reusing.
- `CHANGELOG.md` `Unreleased` now carries I094, I104/ADR 0018, I103/ADR 0019, I097 entries.
- Next free ticket id: **I107**.

## Next steps

1. Remaining open tickets: I101 (med: audit routing can't attribute file-delivered briefs),
   I102 (low: unify team-spawn pairing), I105 (low: opencode note, no code), I106 (docs:
   maikanban keys + doctor tolerance). I101 is the next med.
2. Owner: decide on `maipipe gate approve-definition fad20a5e…` (starts a baseline); remove
   `../spine-wt-2`; close any leftover cmux workspaces.
3. Other adopters of go@1 take the region rewrite on their next `spine update --write` and
   must rebuild spine first (old binary rejects `go@1` as an unknown pack → every gate stage
   exits 2). Worth a line in each adopter's next handoff.

## Gotchas

- After merging a branch whose handoff doc is newest, `spine cursor` reports
  `derivation: blocking` (stale effort in that doc's cursor block). Fix: `spine handoff new`.
  Happened twice today; never hand-edit the block.
- Deploy order for any change to what the region's run lines invoke: `make install` →
  `maipipe daemon stop-if-idle && maipipe daemon start` → `maipipe run full --wait`.
- `cmux send` to an idle codex pane: trailing `\r` doesn't submit; send a bare `"\n"` second.
- maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves, docs-only too.
- fish shell: quote globs / use `bash -c`; stage explicit paths only; `.DS_Store` stays
  untracked; owner ban on `claude-sonnet-5`.

<!-- spine:cursor -->
effort: local-harness-conventions
prd: docs/specs/2026-08-18-local-harness-conventions-design.md
tickets: I079-I087
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
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
