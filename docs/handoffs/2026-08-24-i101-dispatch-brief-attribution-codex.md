---
title: "i101-dispatch-brief-attribution-codex"
created: 2026-08-24
handoff_ordinal: 15
---

# Handoff — i101-dispatch-brief-attribution-codex (2026-08-24)

## Context

Primary repo: `/Users/ldh/Projects/github.com/spine`. Conventions live in `AGENTS.md`
and `WORKFLOW.md` (unified workflow, profile `library-cli`).

Your job is to execute **I101 — dispatch-brief attribution** against the approved
PRD. The grill is done; the design decisions are settled and recorded. Do not
re-open them — implement them.

Read first, in this order:

1. `docs/specs/2026-08-24-i101-dispatch-brief-attribution-plan.md` — the 5-task
   plan you execute. Task-by-task, TDD, with named negative controls.
2. `docs/specs/2026-08-24-i101-dispatch-brief-attribution-design.md` — decisions
   D29–D37 and the testing decisions.
3. `docs/adr/0020-dispatch-brief-attribution-reads-the-lead-s-transcript-never-the-file-on-disk.md`
   — why disk reads are rejected outright.
4. `docs/issues/I101-audit-routing-attribution-from-brief-file.md` — note the
   **Amended 2026-08-24** block: the ticket's original "read the file from disk"
   fix was measured at 0 of 27 and overturned. The amended text is authoritative.
5. `CONTEXT.md` → "Audit evidence" for **dispatch brief**, **attribution**,
   **dispatch record**; `internal/audit/teamspawn.go`'s header comment for I090's
   C1 ruling, which this work narrows but does not overturn.

The one-line summary: attribute a team spawn whose brief was delivered by
`$(cat <path>)` using the heredoc write the lead's own transcript recorded —
first line supplies the ticket token, whole body may satisfy D28 repo
qualification — and never open the referenced file.

## State (verify before relying)

- `main` = `a46aa0c`, pushed; `origin/main` in sync. Tree clean except an
  untracked `.DS_Store` (leave it).
- `a46aa0c` I107 ticket · `ebc73f4` I101 PRD (ADR 0020, spec pair, CONTEXT.md,
  ticket amendment) · `25eb985` prior session's tip.
- Gate green at HEAD: `maipipe run full` **#22 @a46aa0c passed**, 7/7 stages
  (`fast/vet`, `fast/test`, `gates/binary-hygiene`, `gates/dead-code-callgraph`,
  `gates/deferred-cleanup-errcheck`, `gates/gitignore-control`, `gates/tskip`).
- `gofmt -l .` empty, `go vet ./...` clean, `SPINE_REQUIRE_MAIPIPE=1 make test`
  green across 18 packages.
- `spine doctor` exits **1** with two long-standing D4 notes on
  `docs/issues/README.md` and `docs/adr/README.md`. This is pre-existing —
  verified identical at `25eb985`. Do not "fix" it; do not let it read as your
  regression. (Earlier handoffs called this "exit 0"; that was a fish `$status`
  artifact after a pipe.)
- `~/bin/spine` was rebuilt this session at `a46aa0c` with Go 1.27.0.
- Open tickets: **I101** (yours), I102 (low), I105 (low, no code), I106 (docs),
  I107 (med, filed today — do not fix it here). Next free id: **I108**.

## Next steps

1. Branch `i101-brief-attribution` in a worktree off `a46aa0c`.
2. Execute the plan's Tasks 1–5 in order. Each task is TDD: failing test →
   verify red → minimum implementation → verify green → gofmt → negative control
   demonstrated failing, then restored.
3. Task 5 is live verification: `make install`, then run `spine audit routing`
   from the repo root with **no** `--transcripts`, and record how many of the 27
   local-harness spawns are now attributed and which of I079–I087 changed
   verdict. The design predicts 25 of 27.
4. Expect and *report* the verdict churn ADR 0020 names: reviewer spawns ran
   `claude-fable-5 @ high` against `tier: routine` tickets, so several
   `unattributed-transcript` lines become `escalated-no-reason`. That is the fix
   working. Do not silence it, do not edit tickets to make it go away.
5. Full branch verification per the plan's Verification section, then a review
   round (I101 is `review-tier: primary`), then write the completion report.

## Gotchas

- **Under-attribution is always the acceptable failure.** A wrong attribution
  certifies unrouted work as compliant. If a reference will not resolve, leave
  the spawn in the unmatched list. Never guess.
- The audit must never invoke a shell and never open a referenced brief path.
  Path handling is string work only (expand recorded `NAME=value`, absolutize
  against session cwd, `path.Clean`).
- I090's `recognizeTeamSpawns` negative-control var is the pattern to mirror for
  `recognizeBriefFiles`. Keep I090's existing tests green and untouched.
- Stdlib only (ADR 0001). No third-party dependencies.
- Stage **explicit paths** only — never `git add -A` / `git add .`. Review
  `git status --short` before each commit. `.superpowers/` and `.DS_Store` stay
  out.
- `spine cursor` is the only cursor writer; never hand-edit a `spine:cursor`
  block. Handoffs are created with `spine handoff new` (flags before the topic).
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves,
  docs-only commits included. Commit before running the lane.
- If `gates/dead-code-callgraph` or `gates/deferred-cleanup-errcheck` exit 2 with
  a `go/types` / `gcimporter` panic, that is **I107**, not your change: the
  installed spine binary predates the Go toolchain. Fix with `make install`. No
  maipipe daemon restart is needed — stages exec `spine` from PATH per run.
- fish is the interactive shell: quote globs (`"--include=*.go"`) or use
  `bash -c`. Use file-write tools, not shell heredocs, for file content.

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
