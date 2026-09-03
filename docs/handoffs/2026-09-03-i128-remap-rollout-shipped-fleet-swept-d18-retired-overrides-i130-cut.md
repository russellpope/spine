---
title: "i128 remap rollout shipped: fleet swept, D18, retired overrides, I130 cut"
created: 2026-09-03
handoff_ordinal: 41
---

# Handoff — i128 remap rollout shipped: fleet swept, D18, retired overrides, I130 cut (2026-09-03)

## Context

Closes the `i128-remap-rollout` effort. Supersedes handoff 40
(`2026-09-02-ledger-burndown-closed-i125-shipped-i105-ruled-i126-i129-open.md`).

I128 shipped: the Fable 5.1 remap (68aa28f) no longer strands the fleet.
Doctor D18 names a retired mirrored id with its successor and the update
remedy; `spine update` migrates a retired override (`claude-fable-5 @
xhigh`) to `claude-fable-5-1 @ xhigh` as an itemized "retired override"
refresh, so validate's printed remedy is now sufficient; D16 names the host
file and a retired listed id; the render locks pin exact rows and all ten
generation locks sanction only what their own report itemizes. The
deepthought preflights (claude-team, codex-team, handoff-to-codex) relay
validate's refusal verbatim instead of "rebuild spine". The 68aa28f remap
has its retroactive design/plan pair with a spec-review record. The 14
stale primary checkouts were swept at deploy.

Grill was self-answered (owner absent); assumptions are flagged in the
PRD's grill record for the owner to challenge, chiefly Q1 (sweep written,
not committed, worktrees skipped) and Q4 (retired overrides migrate rather
than being preserved, a refinement of I063's pair-aware rule on the update
side only).

This effort sits at its terminal `handoff[<]`; never tick it. The next
session starts a fresh effort with `spine cursor start --force`.

## State (verify before relying)

- main at d423270 plus this handoff's commit; pushed after the final lane.
  Check `git rev-parse HEAD origin/main`. Commits this effort, oldest
  first: 4e1dee0 (PRDs), 5acb7f7 (fix), aa1d551 (review fixes, I130),
  d423270 (ticket closure). maipipe full #84 passed @5acb7f7, #85
  @d423270; the handoff-commit lane is recorded in the progress ledger.
- Both binaries (`~/bin/spine`, `~/.local/bin/spine`) built from d423270;
  rebuild both after the handoff commit so `spine version` prints it.
- Deepthought: 0a39e20 on main (preflight fix, guard script 103/0), NOT
  pushed; main is 6 commits ahead of origin there. Its WORKFLOW.md carries
  the uncommitted one-line sweep refresh. Push and commit are the owner's.
- Fleet after the sweep: ai_infra_notes, ai-virt-framebuffer, ccq,
  deepthought, hbmview, home-lab-admin, jarvis, moo-clone, notetui,
  objectstudio, observability_notes, obsidian-ep-vault, praxis,
  pure-automation each carry an uncommitted one-line WORKFLOW.md refresh
  (`claude.primary: claude-fable-5-1`), validate exit 0, D18 silent.
  maipipe and maikanban still carry their earlier uncommitted refresh.
  Skipped: the 8 worktrees (`maipipe-wt-*`, `ladderbench-wt-m1a`), which
  refresh when their branches rebase or their agents run update; the
  ultima checkouts pin opus-5 and are unaffected.
- Ledger: I128 fixed. Open: I126 (low, dispatch-brief templates), I127
  (low, install skill and LICENSE; needs the LICENSE choice), I129 (low,
  Pi parity extension; owner-external code), I130 (low, trailing comment on
  a mirror row makes update skip the file, so the retired remedy loops),
  I112 (parked). `spine doctor` on spine exits 0 with only D4 info once
  this handoff is committed.
- Scratch copies used for repros live in this session's scratchpad only.

## Next steps

1. Owner: challenge the flagged grill assumptions if wanted (PRD grill
   record, Q1/Q2/Q4/Q6); commit the sweep refreshes in the fleet repos as
   their agents get to them; push deepthought.
2. Next effort candidates: I130 is small and closes I128's last gap
   (comment-preserving migration); I126 and I127 are grill-ready; I129 if
   the owner wants the Pi extension started in its own repo.
3. Non-spine hitlist tickets still to cut by the owner: pi child driver
   (maipipe), writer lease (maipipe + maikanban), ACK fan-in (claude-team).

## Gotchas

- The stop hook demands `maipipe run full --wait` after every commit; a
  fresh-context verifier that reverts `models/defaults.json` for a
  negative control must finish before the lane runs (it did; wait on the
  agent, do not run them concurrently).
- `spine cursor tick implement` needs a progress-ledger line
  `I128: … done` before it accepts; write the line first.
- The Bash tool runs zsh, not fish; `grep --include=*.go` must be quoted.
- `/code-review high <sha>` reviews that commit, not the diff since it.
- Doctor codes now run to D18; D17 was already pin evidence. Doctor
  messages are pinned by tests only where noted (D16's old string was not).
- Generation-lock sanctioning is report-driven now: a new remap needs no
  allowlist edit, but a lock whose fixture change is not itemized by
  update fails until update itemizes it.
- Deepthought's preflight blocks are checked by
  `skills/lib/test-model-validation-preflight.sh`; its old-binary arm greps
  for "predates model validate", "make install", "no worker spawned", so
  keep those strings in the rebuild branch.
- PICKUP.md and the scratchpad stay untracked; stage explicit paths only.

<!-- spine:cursor -->
effort: i128-remap-rollout
prd: docs/specs/2026-09-02-i128-fable-5-1-remap-rollout-design.md
tickets: I128
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
