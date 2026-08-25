---
title: "i101-merged-brief-attribution-live-on-main"
created: 2026-08-24
handoff_ordinal: 16
---

# Handoff — i101-merged-brief-attribution-live-on-main (2026-08-24)

## Context

Session 2026-08-24. Grilled **I101** (dispatch-brief attribution), wrote the PRD +
ADR 0020, dispatched a fresh codex team, and merged its branch. Also filed **I107**
after the required lane surfaced a real gate defect.

The grill overturned the ticket. I101 said to read the brief from disk; measured
against the corpus the ticket itself cites, that recovers **0 of 27** —
`.superpowers/` is gitignored so no brief was committed, and the
`spine-wt-local-harness` worktree was removed when that effort shipped. The ticket
ids survive in the lead's transcript, inside the heredoc bodies I090's C1 round
strips. Path-keyed recovery from those recorded writes scores **25 of 27**. ADR 0020
records the rejection of disk reads on evidence.

## State (verify before relying)

- `main` = `3c17020`, **pushed**; `origin/main` in sync. Sits on merge `7e020cb`
  (`--no-ff` of `i101-brief-attribution`, 10 commits, 41 files, +948/-46).
- Lane at final HEAD: `maipipe run full` **#25 passed @3c17020**.
- Gates at `7e020cb`: `maipipe run full` **#24 passed**; gofmt/vet clean;
  `SPINE_REQUIRE_MAIPIPE=1 make test` green (18 packages, 0 FAIL).
- `spine doctor` exits **1** with two long-standing D4 notes on
  `docs/issues/README.md` and `docs/adr/README.md`. Pre-existing — verified identical
  at `25eb985`. Earlier handoffs called this "exit 0"; that was a fish
  `$status`-after-a-pipe artifact. Read exit codes unpiped.
- `spine audit stages` exit 0; cursor now tracks effort
  `i101-dispatch-brief-attribution` through `docs[x]`.
- `~/bin/spine` rebuilt at `7e020cb`. **No maipipe daemon restart is needed** after
  `make install` — stages exec `spine` from PATH per run (verified: the daemon
  refused `stop-if-idle` with another project's work admitted, and run #21 still
  picked up the rebuilt binary). This corrects a standing gotcha.
- Tickets fixed: I101. Open: I102 (low), I105 (low, no code), I106 (docs), **I107
  (med, new)**. Next free id: **I108**.
- `../spine-wt-2` (`codex-wt-2`) still stale — owner's. cmux: `workspace:45`
  "spine: codex lead" is done and idle in `workspace_group:3`; owner closes it.

## Next steps

1. **Next ticket: I107** — the only remaining med, and fully specified from this
   session's live evidence (`spine gate` analysis classes panic with a raw
   `gcimporter` stack trace when the installed binary predates the Go toolchain).
   Below it: I102 (low — unify team-spawn pairing across flavors; I101 deliberately
   left the claude/codex pairing divergence alone), I105 (low, no code), I106 (docs).
2. **Untracked `README.md` at the repo root** is the owner's, requested in a separate
   session — not a stray from the team run. Leave it alone; it is not this effort's
   to commit or delete.
3. Owner decisions still open from prior sessions: `maipipe gate approve-definition
   fad20a5e…` (would start a baseline; never run in this repo), removing
   `../spine-wt-2`, and closing cmux `workspace:45` ("spine: codex lead", done and
   idle — auto-mode denies `cmux close-workspace`).

## Gotchas

- **Read exit codes unpiped.** fish reports the *last* pipeline command's `$status`,
  so `spine doctor 2>&1 | tail; echo $status` prints tail's 0 and hides a real
  failure. Use `cmd >/dev/null 2>&1; echo $status` or `bash -c '…; echo $?'`.
- If `gates/dead-code-callgraph` or `gates/deferred-cleanup-errcheck` exit 2 with a
  `go/types` / `gcimporter` panic ("export data version N greater than maximum
  supported"), that is **I107**: the installed spine binary predates the Go toolchain.
  `make install` fixes it. It fails on docs-only commits too, so it reads as a defect
  in whatever change is in flight.
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves,
  docs-only included. Commit before running the lane.
- `implement` stage evidence comes from a `I<id>: … done` line in
  `.superpowers/sdd/progress.md`, which is **gitignored** — a team worktree's ledger
  dies with the worktree. Re-append the per-ticket evidence on main before
  `spine cursor tick implement`, or the tick refuses.
- Start the effort cursor (`spine cursor start`) *before* writing any handoff, or
  `spine handoff new` embeds the previous effort's block and derivation goes
  `blocking`. Never hand-edit a cursor block.
- Stage explicit paths only; `.DS_Store` and `README.md` stay untracked unless the
  owner decides otherwise.

<!-- spine:cursor -->
effort: i101-dispatch-brief-attribution
prd: docs/specs/2026-08-24-i101-dispatch-brief-attribution-design.md
tickets: I101
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
