# Cursor hygiene batch (I113 + I114 + I115) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the cursor `tickets:` grammar a comma-list form so batch
efforts run judged (I114, retiring spine's live D9 warn), close the
`NonCanonical` closing-fence asymmetry (I113), and harden the D13 per-ticket
checks (I115) — with I114's template change carrying the gen-bump rule's
binding one-liner and the estate sweep.

**Architecture:** Two lanes. Lane A (worker terra, serial): I114 then I113,
both in `internal/cursor` (+ `internal/stages` for I114's resolver and
`internal/update` for the superseded line). Lane B (worker luna, parallel):
I115 in `internal/doctor` only. Lead: sol. Reviews claude-side per task at
routine tier; final whole-branch review fable-5 @ high with the
requirements-attack step. Never claude-sonnet-5 (substitute claude-opus-5 @
low).

**Tech Stack:** Go standard library only (ADR 0001).

**Spec:** `docs/specs/2026-08-28-cursor-hygiene-batch-design.md`

## Global Constraints

- Tiers: I114/I115 routine, I113 mechanical; review-tier routine per task;
  all commits cite their ticket id.
- Ledger keys dogfooded, **lead is sole writer**: `batch:`
  `2026-08-28-chyg#1` (I114) / `#2` (I113) / `#3` (I115); `workspace:`
  absolute path while the worktree exists, cleared at close; `commits:` in
  the close commit. Workers never touch ticket frontmatter.
- `spine cursor` is the only cursor writer; never write the literal cursor
  marker in prose; never tick the handoff stage.
- Every negative control must be **observed red** (command + output
  recorded). A prescribed control is a hypothesis, not a fact — run both
  arms.
- `Block()` and everything `spine cursor` emits are untouched (I113
  constraint).
- No partial resolution of a comma-list, ever: malformed element, duplicate,
  or whitespace ⇒ the whole value is unresolvable (I114 constraint).
- Stage explicit paths only — never `git add -A`. Batch commits so one
  `maipipe run full --wait` covers them. Read exit codes unpiped (fish).

### Task 1 (lane A): I114 — comma-list tickets grammar

**Files:**
- Modify: `internal/stages/stages.go` (`resolveTicketIDs`,
  `unresolvableTicketsNote`, package doc)
- Modify: `internal/cursor/cursor.go` (`Grammar` text, package doc)
- Modify: `internal/stages/stages_test.go`, cursor tests

**Interfaces:**
- Produces: `resolveTicketIDs` resolves `I0NN,I0MM[,...]` (each element a
  bare id matching the existing bare-id rule) to exactly those ids, order
  preserved, no dedup — duplicates make the whole value unresolvable.
- The unresolvable note's grammar summary becomes
  `I0NN | I0NN,I0MM[,...] | I0NN-I0MM | prefix <str>`.

- [ ] **Step 1: Failing tests** (stages seam, ticket-grammar pattern):
  `I065,I106` resolves to exactly `[I065 I106]`; a three-element list
  resolves; `I065, I106` (space) unresolvable; `I065,I065` (duplicate)
  unresolvable; `I065,nope` (malformed element) unresolvable; empty element
  (`I065,,I106`, trailing `I065,`) unresolvable; the note names the comma
  form.
- [ ] **Step 2: Verify red.** Record command + output.
- [ ] **Step 3: Implement the minimum.** Split on `,` only when the raw
  value contains one; every element must match the bare-id rule; reject on
  any duplicate or non-matching element.
- [ ] **Step 4: Verify green**, then `gofmt`, `go vet`.
- [ ] **Step 5: Negative control.** Revert the resolver hunk (keep tests);
  observe the new tests red; restore.

### Task 2 (lane A): I114 — template grammar line + gen-bump rule

**Files:**
- Modify: `templates/current/WORKFLOW.md.tmpl` (grammar line + authoring
  note), `internal/update/update.go` (`supersededLines`), WORKFLOW.md
  (spine's own, via `spine update`)
- Modify: updater migration tests

**Interfaces:**
- Produces: the template's tickets line reads
  `tickets: I0NN | I0NN,I0MM[,...] | I0NN-I0MM | prefix I0`; the outgoing
  line joins `supersededLines` in the same change; the template gains the
  binding one-liner: any content-changing template edit appends its
  predecessors' dropped lines to the superseded set in the same change.
- No `templates/VERSION` bump (dhyg ruling, precedent d78f6ee).

- [ ] **Step 1: Failing test** (migration-fixture pattern): a WORKFLOW
  frozen at the outgoing grammar line refreshes cleanly with zero
  unrecognized lines.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement**: template line + authoring note + superseded
  entry; run `spine update --dir .` to refresh spine's own WORKFLOW.md.
- [ ] **Step 4: Verify green.**
- [ ] **Step 5: Negative control, both arms**: (a) remove the superseded
  entry — the migration fixture goes red; (b) a genuine local edit in the
  fixture still skips the file (guard not over-broad). Record both.

### Task 3 (lane A, after Task 2): I113 — closing-fence canonical span

**Files:**
- Modify: `internal/cursor/cursor.go` (`fence` third offset, `scanFences`,
  the `NonCanonical` comparison)
- Modify: cursor tests (I109 family)

**Interfaces:**
- Produces: the compared span ends at the close-fence line's end, excluding
  its terminator (`len(content)` when none). `Block()` unchanged.

- [ ] **Step 1: Failing tests**: closing fence + trailing spaces ⇒
  `NonCanonical` true, no findings; same with tabs; byte-canonical block at
  EOF without trailing newline ⇒ `NonCanonical` false.
- [ ] **Step 2: Verify red** (the EOF case should already be green — record
  which arms are red; if the EOF case is red before the change, that is a
  finding, stop and report).
- [ ] **Step 3: Implement**: carry the line-end offset on `fence` from
  `scanFences`; compare `content[open.start:closeLineEnd]`.
- [ ] **Step 4: Verify green**; existing opening-fence and CRLF tests still
  green and no noisier.
- [ ] **Step 5: Negative control.** Revert the comparison hunk; new tests
  red; restore.

### Task 4 (lane B, parallel): I115 — D13 hardening

**Files:**
- Modify: `internal/doctor/tickets.go` + its tests

**Interfaces:**
- Produces: guard comment on the frontmatter parser naming the
  no-comment-stripping divergence and why; `workspace:` values that are not
  absolute warn (`filepath.IsAbs`) and are **not** stat'd; frontmatter
  values are used with surrounding single/double quotes stripped; fence-less
  tickets pinned silent.

- [ ] **Step 1: Failing tests**: relative `workspace:` ⇒ absolute-path warn,
  no existence warn regardless of CWD; absolute-and-missing ⇒ existence
  warn (unchanged); `batch: "2026-08-28-chyg#1"` (quoted) ⇒ no malformed
  warn; quoted `workspace:` handled likewise; fence-less ticket ⇒ zero D13
  findings.
- [ ] **Step 2: Verify red** (fence-less case expected already green — it
  pins existing behavior; record which arms are red).
- [ ] **Step 3: Implement the minimum** + the guard comment.
- [ ] **Step 4: Verify green**; existing D13 tests (including
  malformed-batch-on-fixed-ticket negative control) still green.
- [ ] **Step 5: Negative control.** Revert the quote-stripping hunk; quoted
  cases red; restore.

### Task 5 (lead): merge, functional test, reviews

- [ ] Merge lane B into lane A's result (disjoint files; expect clean).
- [ ] `go test ./...`, `gofmt -l`, `go vet ./...` — exit 0, record output.
- [ ] Functional pass: `make install`; on a scratch repo, a cursor with
  `tickets: I113,I115` derives with evidence judged; `spine doctor` clean
  on a healthy fixture.
- [ ] Final whole-branch review: fable-5 @ high, requirements-attack step
  first (attack this spec for internal contradictions; surface with
  proposed resolutions, never silently resolve).

### Task 6 (lead): ship, deploy, sweep

- [ ] ff-merge to main; `maipipe run full --wait` green at the merge SHA.
- [ ] `make install`; record sha256 prefix.
- [ ] Estate sweep, dhyg checklist shape (per-repo `pre/-write/post/doctor`
  exits, logs under /tmp/sweep-*.out): ccq, home-lab-admin, jarvis,
  notetui, observability_notes, pure-automation, deepthought, hbmview.
  Residual-skip repos (praxis, moo-clone, ultima) untouched; note ultima's
  WORKFLOW stays one grammar-line stale. Sweep commits local-only.
- [ ] Ledger close per ticket: status fixed, `commits:` written,
  `workspace:` cleared — lead only.
- [ ] Dogfood proof: spine's own `spine doctor` — D9 gone; exit 0 with the
  adr info note (record command + output, unpiped).
