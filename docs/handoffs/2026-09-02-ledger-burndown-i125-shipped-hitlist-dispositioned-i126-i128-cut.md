---
title: "ledger burndown: I125 shipped, hitlist dispositioned, I126-I128 cut"
created: 2026-09-02
handoff_ordinal: 39
---

# Handoff — ledger burndown: I125 shipped, hitlist dispositioned, I126-I128 cut (2026-09-02)

## Context

Autonomous session (owner absent) running the two efforts the 2026-09-02
remap handoff queued. Effort one shipped I125: a ticket file closed by the
ledger lifecycle (`status: fixed` plus a SHA-shaped `commits:` token) is now
a **closure record** and evidences the implement stage, OR'd with the
progress-ledger scan. Effort two dispositioned the fusion-harness borrow
hitlist and adopted the note as this effort's research record. The grill
was self-answered from the ticket and code; every answer the ticket did not
settle is marked "assumption" in the PRD's grill record for the owner to
challenge.

This effort is closed at its terminal `handoff[<]`; do not tick it. The next
session starts a new effort with `spine cursor start --force`.

## State (verify before relying)

- Commits since 68aa28f, all cite I125: 7a54c3c (PRD, plan, glossary),
  a171265 (fix), 5251346 (spec-review corrections, ticket closed), 995760e
  (code-review fix: first-wins frontmatter keys, closed fence required for
  closure records), 7f76879 (docs: I126, I127, I128, research disposition,
  prior handoff), then this handoff's commit. `git log --oneline 68aa28f..`.
- Gates: `go vet && go test ./...` green; maipipe run #81 passed at 5251346;
  the final lane at this handoff's SHA is recorded in the ledger before
  push. Spec review (opus, ESCALATION recorded) 3 minor findings fixed;
  independent verify (fable) PASS on all four criteria with a load-bearing
  negative control; code review (fable) APPROVE with 1 medium + 1 low, both
  fixed. Functional walk: scratchpad `ft.fish`, 5 cases, 0 failures.
- Live proof: the 68aa28f binary refused `spine cursor tick implement` on
  this effort (I125 had no ledger line); the a171265 binary ticked it from
  the closure record alone. I125's implement evidence remains closure-only.
- Ledger after this session: I105 open (owner decision), I112 parked,
  I126 (dispatch-brief templates), I127 (install skill + LICENSE), I128
  (Fable 5.1 remap rollout hazards) open. I125 fixed.
- Binaries: `make install` (~/bin) and `~/.local/bin/spine` are refreshed
  from the final SHA after the lane; `spine audit stages` prints the
  renamed label `implement evidence`.

## Next steps

1. **Owner rulings, in priority order.** (a) I128: 20 of ~28 fleet checkouts
   still mirror `claude-fable-5` and fail the claude-team dispatch
   preflight as `retired-model`; the remedy is `spine update --dir R
   --write` per repo, not a rebuild. Decide sweep vs per-repo, and whether
   host configs get the new id. (b) I105: adopt OpenCode for the constrained
   worker lane (research recommendation) or fund a Pi-extension ticket; the
   ADR is written only on that ruling. (c) The PRD's "assumption" rows
   (SHA-shaped commits, symmetric present-unticked, unconditional rule text,
   label rename).
2. Cut the non-spine hitlist tickets: pi child driver (maipipe), writer
   lease (maipipe + maikanban), ACK fan-in (claude-team skill).
3. I126 and I127 are grill-ready; I127 needs the LICENSE choice first.

## Gotchas

- The Bash tool runs zsh, not fish; fish's builtin `printf` treats `--` as
  its format string (auto-memory `bash-tool-runs-zsh-not-fish`).
- `/code-review high 68aa28f` reviewed commit 68aa28f itself, not the diff
  since it; pass a diff range or run a targeted reviewer for "since X".
- The stop hook demands `maipipe run full --wait` after every commit,
  including docs-only ones; batch commits, then run the lane once.
- A closure record needs a closed `---` fence and an unquoted
  `status: fixed`; `commits:` must hold a hex token of 7–40 chars (a
  `[see PR #12]` value still blocks a ticked implement, by design).
- `docs/issues/README.md` is template-managed; the `commits` bullet was not
  edited to avoid a generation bump. The convention note lives in
  CHANGELOG, CONTEXT.md, and the stages package doc.

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
