---
title: "i107-built-unmerged-moving-to-herdr"
created: 2026-08-25
handoff_ordinal: 18
---

# Handoff — i107-built-unmerged-moving-to-herdr (2026-08-25)

## Context

Session 2026-08-24/25. Grilled **I107**, wrote ADR 0021 + the spec pair, dispatched
a codex team on cmux, and the team finished. **The branch is built but unmerged
and independently unreviewed.** Work now moves to **herdr**.

The grill overturned two parts of the ticket as filed, both recorded in ADR 0021
and in the ticket's own "Grilled 2026-08-24" section:

- "Exit 2 is right" was half wrong. Today's 2 is the Go runtime's exit status for
  an unrecovered panic, not `gate.Run`'s misconfiguration code — `Run` never
  finishes. The missing results document is the same accident. The fix makes both
  guarantees real rather than preserving them.
- The ticket's "optionally cheaper and stronger" `runtime.Version()` preflight is
  **rejected**, not deferred. The panic keys on the export-data *format* version,
  which does not move every Go release, so the comparison would refuse working
  setups. Detection keys on the importer actually failing.

Also filed this session: **I108** (deferred `spine doctor` toolchain-skew
advisory) and **I109** (cursor-block scanning matches a marker quoted in prose).

## State (verify before relying)

- `main` = **`5cfcb2f`**, **pushed**, `origin/main` in sync. Tree clean, no
  untracked files. `README.md` is now committed; `.DS_Store` is gitignored.
- Lane on main: `maipipe run full` **#30 passed @5cfcb2f**.
- Branch **`i107-gate-panic-misconfiguration`** = **`ff7e71f`**, 6 commits,
  branched from `f930083` (so it does *not* contain main's `531efbc` handoff
  commit or anything after — merging brings both sides together). Diff vs its
  base: 4 files, +296/-5 — `internal/gate/gate.go` (+19), `internal/gate/load.go`
  (+37), and two new test files (`load_panic_test.go` +170,
  `run_panic_test.go` +75). **Not merged. Not pushed.**
- Worker branches still present: `codex-wt-i107-loader` (`079c8bb`),
  `codex-wt-i107-run` (`46aa59f`). Their worktrees are already removed.
- `spine cursor`: effort `i107-gate-panic-misconfiguration`, `implement[x]`,
  now at `functional-test[<]`. **`implement` was ticked this session** on the
  documented evidence at `.superpowers/sdd/progress.md:265` (`I107: gate panic
  misconfiguration done` plus four evidence lines). Derivation goes clean again
  with this document.
- `spine doctor` exits **1** on two long-standing D4 notes
  (`docs/issues/README.md`, `docs/adr/README.md`). **Pre-existing** — verified
  identical at `25eb985`. Not a regression, do not "fix" it.
- `~/bin/spine` built at `7e020cb` with Go 1.27.0; machine toolchain 1.27.0.
  **Rebuild after merging** — the installed binary predates this fix.
- Open tickets: **I107** (in-progress, branch built), I102 (low), I105 (low, no
  code), I106 (docs), I108 (low), I109 (med, new). Next free id: **I110**.
- cmux workspaces still open in `workspace_group:3`: `59` (lead), `60`–`64`
  (workers), all idle. `workspace:6` (`spine: sdd-workers`) is the group anchor
  and pre-existing. **The owner closes workspaces by hand** — auto-mode denies
  `cmux close-workspace`.
- `../spine-wt-2` (`codex-wt-2`) still stale — owner's, untouched.
- herdr integrations: `claude: current (v8)`, `codex: current (v8)` — both ready.
  `pi` is outdated (v6 < v8) and `opencode` outdated (v9 < v10); neither is
  needed for this work.

## Next steps

1. **The gate that has not run: independent review.** The codex lead
   *self*-reviewed. `WORKFLOW.md` makes `verify` a fresh-context verifier against
   the PRD, not self-review, and `/spec-review` of the finished diff against the
   PRD is mandatory. Run both from a session with no build context.
2. **Check the one declared deviation first.** The lead replaced Task 5's
   negative control (widen D39's classifier, expect the constraint test to fail)
   with "a temporary panic at the actual `p.Error` return", arguing ordinary
   compile errors never reach the panic classifier. Task 5's whole purpose is
   proving the recover does not swallow a genuine type-check failure — the one
   constraint in the PRD and in ADR 0021. Decide whether the substitute proves
   the same thing. If it does not, that control has to be redone.
3. Then, per `WORKFLOW.md` stage order: tick `functional-test`, `review`,
   `verify`; `git merge --no-ff i107-gate-panic-misconfiguration`;
   `make install`; `maipipe run full --wait`; delete the two `codex-wt-i107-*`
   branches; CHANGELOG entry (this is consumer-visible — gate output changes);
   tick `ship`/`deploy`/`docs`; push.
4. Not yet routed: **I109** has empty `execution-mode`/`tier`/`review-tier`. Fill
   before dispatching it. I108 likewise.

## Task breakdown hints

- The whole diff is four files and reads in ten minutes. Start at
  `internal/gate/gate.go` — `recoverRunPanic` plus two `func(){ defer … }()`
  wrappers around the `fn`/`rfn` invocations. Then `internal/gate/load.go` for
  the classifier and the two message classes.
- Decisions to check the diff against are **D38–D44** in the design doc. The
  constraint that matters is D44 plus the "genuine type-check failures unchanged"
  line — everything else is message shape.
- `docs/specs/2026-08-24-i107-gate-panic-misconfiguration-plan.md` lists the
  negative control each task owed. Cross-check the claimed controls against it.

## Gotchas

- **Read exit codes unpiped.** In fish, `cmd 2>&1 | tail; echo $status` prints
  *tail's* status and hides a real failure. Use `cmd >/dev/null 2>&1; echo $status`
  or `bash -c '…; echo $?'`. This produced two false "exit 0" claims in an
  earlier session.
- **Never write the literal spine cursor marker in handoff prose.** The parser
  locates the block with a substring scan over the whole file, so a marker quoted
  mid-sentence — even inside backticks — hijacks the block and
  `spine audit stages` exits 1 with a wall of quoted prose. Hit live on
  2026-08-24; filed as **I109**. Refer to it by name, never by marker.
- **A gate stage exiting 2 with a `gcimporter` panic is I107 itself**, not your
  change. `make install` clears it. It fires on docs-only commits too.
- **A live team can move the primary checkout's branch under you.** The codex
  lead worked in `/Users/ldh/Projects/github.com/spine` directly and switched it
  to the feature branch; a commit made without re-checking landed on the wrong
  branch and had to be cherry-picked to `main`. **Run `git rev-parse
  --abbrev-ref HEAD` immediately before every commit while a team is live.**
- **No `maipipe daemon` restart** is needed after `make install` — stages exec
  `spine` from PATH per run.
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves,
  docs-only included. Commit first, then run the lane.
- `.superpowers/sdd/progress.md` is **gitignored** — a team worktree's ledger
  dies with the worktree. Per-ticket evidence must land where the tick will see
  it before `spine cursor tick implement`.
- Run `spine cursor start` **before** `spine handoff new`, or the doc embeds the
  previous effort's block and derivation goes blocking. `spine handoff new` will
  not overwrite an existing filename.
- **Stage explicit paths only** — never `git add -A` or `git add .`.
- Codex classifies read-only commands into a stricter sandbox, where connecting
  to `~/.local/state/cmux/cmux.sock` fails with `EPERM`. That makes
  `frontend-preflight.sh` self-report `plain` from inside a codex pane even
  though cmux is fully functional and the lead's own `cmux new-workspace` calls
  (escalated) succeed. **Cosmetic — it did not change what the team did.** Do not
  "fix" the preflight on the strength of that symptom.
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @ low`.

<!-- spine:cursor -->
effort: i107-gate-panic-misconfiguration
prd: docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md
tickets: I107
stages: grill[x] prd[x] issues[x] implement[x] functional-test[<] review[ ] verify[ ] ship[ ] deploy[ ] docs[ ] handoff[ ]
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
