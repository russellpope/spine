---
title: "ledger burndown closed: I125 shipped, I105 ruled, I126-I129 open"
created: 2026-09-02
handoff_ordinal: 40
---

# Handoff — ledger burndown closed: I125 shipped, I105 ruled, I126-I129 open (2026-09-02)

## Context

Closes the `ledger-burndown` effort. Supersedes handoff 39
(`2026-09-02-ledger-burndown-i125-shipped-hitlist-dispositioned-i126-i128-cut.md`),
which stays the reference record for the I125 build: review findings, the
grill's flagged assumptions, and the live before/after proof. This handoff
adds the owner's I105 ruling and points the next session at I128.

I125 shipped: a closed ticket file (`status: fixed` plus a SHA-shaped
`commits:` token, closed `---` fence, unquoted status) is a closure record
and evidences the implement stage, OR'd with the progress-ledger scan. I105
is ruled: fund the Pi parity extension (I129) so Pi stays a viable second
worker harness; OpenCode remains the lane available now.

This effort sits at its terminal `handoff[<]`; never tick it. The next
session starts a fresh effort with `spine cursor start --force`.

## State (verify before relying)

- main at 1faf2b8 plus this handoff's commit; the final lane runs at that
  SHA before push. Check `git rev-parse HEAD origin/main`.
- Commits this session, oldest first: 7a54c3c, a171265, 5251346, 995760e,
  7f76879, 9ddf718, 1faf2b8. maipipe runs #81 (@5251346) and #82 (@9ddf718)
  passed; the handoff-commit lane is recorded in the ledger.
- Both binaries (`~/bin/spine`, `~/.local/bin/spine`) are rebuilt after the
  final lane; `spine version` must print the handoff-commit SHA.
- Ledger: I125 fixed, I105 fixed. Open: I128 (med, Fable 5.1 remap rollout
  hazards), I126 (low, dispatch-brief templates), I127 (low, install skill
  and LICENSE), I129 (low, Pi parity extension; owner-external code), I112
  (low, parked). `spine doctor` exits 0 with only the D4 info line.
- Fleet: maipipe (branch i221-panel-yank) and maikanban (main) carry
  uncommitted one-line WORKFLOW.md refreshes of `claude.primary`; leave
  them to their own agents. About 20 other checkouts still mirror
  `claude-fable-5` (I128 item 1).

## Next steps

1. Next effort: I128, the rollout fix. It is the only medium and it is
   live: every unrefreshed fleet repo fails the claude-team dispatch
   preflight as `retired-model` with a misleading "rebuild spine" remedy.
   Grill the split in the ticket's Fix section (sweep vs per-repo refresh,
   the stuck-override remedy, host-config coupling, the vacuous substring
   locks) and decide whether the retroactive PRD for 68aa28f is part of it
   or a separate doc commit.
2. Then I126 and I127 (I127 needs the LICENSE choice first), or I129 if the
   owner wants the Pi extension started in its own repo.
3. Non-spine hitlist tickets still to cut by the owner: pi child driver
   (maipipe), writer lease (maipipe + maikanban), ACK fan-in (claude-team).

## Gotchas

- The stop hook demands `maipipe run full --wait` after every commit,
  docs-only included; batch commits and run the lane once at the final SHA.
- The Bash tool runs zsh, not fish; fish's builtin `printf` treats `--` as
  its format string. Auto-memory `bash-tool-runs-zsh-not-fish`.
- `/code-review high <sha>` reviews that commit, not the diff since it.
- Superseding a terminal `handoff[<]` cursor needs `spine cursor start
  --force`; ticking implement on a fresh effort will show present-unticked
  for any range ticket that already has a ledger line or closure record.
- `docs/issues/README.md` is template-managed: editing it means a
  generation bump. Model-table edits need the displaced id in the row's
  history and `internal/model/testdata` gen-13 fixture mirrored.
- PICKUP.md and the scratchpad `ft.fish` are untracked; keep them out of
  commits.

<!-- spine:cursor -->
effort: ledger-burndown
prd: docs/specs/2026-09-02-i125-closure-implement-evidence-design.md
tickets: I125,I105
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
