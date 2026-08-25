---
title: "i107 shipped, openweights specs filed"
created: 2026-08-25
handoff_ordinal: 19
---

# Handoff — i107 shipped, openweights specs filed (2026-08-25)

## Context

Session 2026-08-25. Ran the gate that had not run on **I107** — independent
review — then merged and shipped it. Also filed two **openweights** design docs
the owner dropped in the repo root, and turned the spine-side one into tickets.

I107 arrived built but unmerged and *self*-reviewed by the codex lead. Both
missing gates ran cold, from a session with no build context: a spec-review of
the finished diff against the PRD, and a fresh-context verifier that re-ran the
plan's negative controls itself rather than accepting the team's claims.

That was worth doing. The team's one **declared deviation was upheld**, but for a
reason the team had not given — and the re-run found a **second** inert control
nobody had noticed.

## State (verify before relying)

- `main` = **`bacafdd`**, **pushed**, `origin/main` in sync. Tree clean.
- Lane on main: `maipipe run full` **#32 passed @bacafdd**.
- `~/bin/spine` rebuilt by `make install` at the merge commit `612980b`
  (`vcs.revision=612980bd8642…`). It now contains the I107 fix.
- **I107 is `status: fixed`**, merged `--no-ff` at `612980b`, with a Resolution
  section carrying the live-artifact evidence. Cursor: every stage `[x]` through
  `docs`, `handoff` current.
- Branches `i107-gate-panic-misconfiguration`, `codex-wt-i107-loader` and
  `codex-wt-i107-run` are **deleted**; their content was confirmed present on
  `main` first (each worker branch was a strict subset). Review worktree removed.
- `spine audit routing` exits **0** — no silent descent. 42 team spawns are
  recognised-but-unattributed, which is the long-standing no-ticket-token case.
- `spine doctor` exits **1** on the same two pre-existing D4 notes
  (`docs/issues/README.md`, `docs/adr/README.md`). **Pre-existing, not a
  regression** — verified identical before any of this session's work.
- New tickets **I110** and **I111** (openweights); **I108**/**I109** now routed.
  Next free id: **I112**.
- `../spine-wt-2` (`codex-wt-2`) still stale — owner's, untouched.
- cmux workspaces `59`–`64` in `workspace_group:3` still idle. The **owner closes
  workspaces by hand** — auto-mode denies `cmux close-workspace`.

## Next steps

1. **I110 first, then I111** — the openweights work, in that order. I110 is
   additive data (a flavor row) and unblocks deepthought's `/openweights-team`
   capability check. I111 is behavioural and blocked by I110.
2. **I111 carries a hazard that passes every existing test.** The audit's
   repo-qualification predicate (D28, from I047) is gated on a record's flavor
   being `claude`. Tag open-weights records `openweights` and they fall out of
   that gate and start claiming tickets they should not. The condition has to
   test *transcript source*, not flavor — the two were the same thing before this
   change and are not afterwards. The ticket makes the D28 regression test and
   the flavor-literal audit sweep named deliverables.
3. Run `spine cursor start` for the new effort **before** any `spine handoff new`.
4. deepthought has `docs/specs/2026-08-25-openweights-team-design.md` filed but
   **uncommitted** — the owner's to commit. That work is inert until I110 ships
   *and the fleet binary is rebuilt and reinstalled*.

## Gotchas

- **The plan's negative controls were wrong in three places, and two of them
  looked fine.** Task 5's ("widen D39's classifier") leaves its target test green
  — a genuine type-check failure returns via `p.Error` from `go list -e` before
  any importer runs, so no panic fires and the classifier is never consulted.
  D44 already said exactly this; nobody connected it. Task 2's mutates
  `runGoVersionCommand`, the very seam both target tests stub, so it can never
  redden. **A prescribed control is a hypothesis, not a fact — run both arms.**
- **Four docs named `$SPINE_GATE_RESULTS`, which does not exist.** The real
  constant is `MAIPIPE_RESULTS` (`internal/gate/results.go:14`). The name is
  plausible because a `SPINE_GATE_*` family *does* exist. Corrected at `39a6972`.
- **Read exit codes unpiped.** In fish, `cmd 2>&1 | tail; echo $status` prints
  *tail's* status. Use `cmd >/dev/null 2>&1; echo $status` or `bash -c '…; echo $?'`.
- **Never write the literal cursor marker in handoff prose.** The parser finds
  the block by substring scan over the whole file, so a marker quoted
  mid-sentence — even in backticks — hijacks it and `spine audit stages` exits 1.
  Refer to it by name only. Filed as **I109**.
- **Ticking stages makes the previous handoff a stale snapshot**, which blocks
  `spine audit stages` and warns D9 in doctor until `spine handoff new` runs.
  Expected, not a defect.
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves,
  docs-only included. **Commit first, then run the lane** — batch commits so one
  lane run covers them.
- No `maipipe daemon` restart is needed after `make install`; stages exec `spine`
  from PATH per run.
- A gate stage exiting 2 with a `gcimporter` panic **was** I107 and is now fixed
  in the installed binary. If it reappears, the binary is stale again.
- **Stage explicit paths only** — never `git add -A`.
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @ low`.

<!-- spine:cursor -->
effort: i107-gate-panic-misconfiguration
prd: docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md
tickets: I107
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
