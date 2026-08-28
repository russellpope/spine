---
title: "cursor-hygiene-batch-codex"
created: 2026-08-28
handoff_ordinal: 27
---

# Handoff — cursor-hygiene-batch-codex (2026-08-28)

## Context

You are the codex team lead for the **cursor-hygiene batch (I113 + I114 +
I115)** in `/Users/ldh/Projects/github.com/spine`. Everything is grilled,
specced, and planned — your job is execution, not design.

Read first, in order:
1. `docs/specs/2026-08-28-cursor-hygiene-batch-plan.md` — the task-by-task
   plan (6 tasks, TDD steps, negative controls). This is your work order.
2. `docs/specs/2026-08-28-cursor-hygiene-batch-design.md` — the PRD; every
   grill ruling is recorded there. Do not re-litigate settled rulings.
3. Ticket files: `docs/issues/I113-*.md`, `I114-*.md`, `I115-*.md` — each
   carries a `## Rulings` section (I114/I115) binding the implementation.
4. `AGENTS.md` at the repo root for repo conventions.

The batch in one line: I114 adds a comma-list `tickets:` grammar form (plus
the WORKFLOW template's grammar line + superseded entry + gen-bump authoring
note); I113 widens the NonCanonical compared span to the end of the closing
cursor fence's line; I115 hardens the D13 per-ticket doctor checks (guard
comment, IsAbs warn, quote stripping, fence-less test).

**Lanes (plan §Architecture):** Lane A serial — I114 (plan Tasks 1–2) then
I113 (Task 3), both touch `internal/cursor`/`internal/stages`/`internal/update`.
Lane B parallel — I115 (Task 4), `internal/doctor` only. Then lead: Task 5
(merge, functional test, reviews) and Task 6 (ship, deploy, estate sweep —
the sweep is owner-visible; if your environment cannot reach the estate
repos, stop after ship and report, do not fake it).

**Ledger keys — lead is SOLE writer:** `batch:` `2026-08-28-chyg#1` (I114),
`#2` (I113), `#3` (I115); `workspace:` absolute path while a worktree
exists, cleared at close; `commits:` in the close commit. Workers never
touch ticket frontmatter.

**Reviews:** per-task at routine tier; the final whole-branch review is
claude-side (fable-5 @ high) and includes the requirements-attack step —
if you cannot dispatch a claude reviewer, leave `review[ ]` unticked and
report; do not substitute a codex self-review for it. Never route anything
to claude-sonnet-5 (substitute claude-opus-5 @ low effort).

## State (verify before relying)

- `main` = **`2ea670c`**, clean (untracked only:
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md` — not yours;
  `PICKUP.md` — scratch, never stage either).
- Lane: `maipipe run full` **#46 passed @`2ea670c`**.
- Cursor: effort `cursor-hygiene-batch`, `tickets: I113-I115` (contiguous
  range — resolves under the CURRENT grammar, so this effort runs judged),
  at `implement[<]`. Change cursor state ONLY via `spine cursor tick <stage>`
  / `spine cursor here <stage>`.
- `spine doctor` after this doc exists: exit 0 expected, with one info note
  (`docs/adr/README.md` hand-authored — leave it).
- I116/I117 were filed this session (spine-model flag order; implement-tick
  message). **Open, NOT in this batch** — do not pick them up.
- Test baseline: `go test ./...` green at `2ea670c`; `gofmt -l` empty;
  `go vet ./...` clean.

## Next steps

1. Build the worker team per the lanes above (one worker for lane A serial,
   one for lane B; size to the work, not bigger).
2. Execute plan Tasks 1–4 test-first exactly as written — each Step 1/2
   writes failing tests and records them red BEFORE implementing; each
   Step 5 negative control must be **observed red** with command + output
   recorded. A control asserted from reasoning does not count.
3. Task 5: merge lanes, full battery (`go test ./...`, `gofmt -l`,
   `go vet ./...`), functional pass with a rebuilt binary (`make install`),
   reviews as specified above.
4. Task 6: ff-merge to main, `maipipe run full --wait` at the merge SHA,
   ledger close per ticket. The estate sweep and pushing are owner-visible
   steps — report them ready rather than improvising access.
5. Write the completion report to `.superpowers/sdd/team-report.md`:
   per-task verdicts, commands + output for every verification and negative
   control, SHAs, and anything left for the owner.

## Gotchas

- **`Block()` and everything `spine cursor` emits are untouched** (I113 is
  about what the guard compares, never what the tool writes).
- **No partial resolution of a comma-list** — malformed element, duplicate,
  or internal whitespace ⇒ the whole value unresolvable (I114 rulings).
- **No `templates/VERSION` bump** for I114's template change (dhyg ruling,
  precedent `d78f6ee`); the outgoing grammar line joins `supersededLines`
  in the SAME change.
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD
  moves, docs-only included — **batch commits** so one run covers them.
- **Stage explicit paths only** — never `git add -A` / `git add .`.
- Ledger resolution lines must start with the ticket id and contain
  `done`/`complete` as a whole word, or they are silently not implement
  evidence (I117 documents the trap — it is not fixed yet).
- `spine` subcommands want **flags BEFORE positionals** (`spine model
  --effort codex primary`, not trailing flags) — trailing flags exit 2
  with bare usage (I116, unfixed).
- **Never tick the `handoff` stage**; `handoff[<]` is terminal for a
  session, recover with `spine cursor here handoff`.
- Never write the literal cursor open/close marker text in prose or
  reports; refer to it as "the cursor block".
- Under fish, read exit codes unpiped — `$status` after a pipe reports the
  last pipeline command.
- Worktrees: `dirname "$(git rev-parse --path-format=absolute
  --git-common-dir)"` is the primary repo path; `--show-toplevel` lies
  inside a worktree.

<!-- spine:cursor -->
effort: cursor-hygiene-batch
prd: docs/specs/2026-08-28-cursor-hygiene-batch-design.md
tickets: I113-I115
stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] deploy[ ] docs[ ] handoff[ ]
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
