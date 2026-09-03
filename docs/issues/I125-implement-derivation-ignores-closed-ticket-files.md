---
id: I125
title: "implement derivation ignores closed ticket files — verified-fixed tickets derive ticked-missing without a progress-ledger line, with a misleading typo hint"
severity: med
status: fixed
commits: [7a54c3c, a171265]
affects: [I019, I029, I032, I117]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: primary
---

## Problem

The implement stage's only evidence source is `.superpowers/sdd/progress.md`
(`implementEvidence` in `internal/stages/stages.go`): a `<id>: … done|complete|completed`
whole-word line per ticket. The ticket files themselves — the ledger of record, whose
`status: fixed` and `commits:` fields the batch lifecycle defines as written at close —
contribute nothing. An effort that closes its tickets properly but never writes progress-md
lines derives `implement (ticked-missing)` and **blocks**, on tickets whose implementation
is verifiable on disk.

Observed live in maikanban on 2026-08-28 (context-navigation effort, gen-11 derivation;
recorded in `docs/handoffs/2026-08-28-context-navigation-closeout-and-gate-flake-fixes.md`):
I047/I048 both existed with `status: fixed` and `commits: [a9ddea5]`, `spine audit stages`
reported "2/2 ticket file(s) present" on the issues row — and the implement row still blocked
as ticked-missing because `.superpowers/sdd/progress.md` had no `I047:`/`I048:` lines at all
(confirmed 2026-08-31: the file has `implementation complete` lines for every other closed
effort, none for I047/I048). `tick implement` had to be `--force`d on verified reality.

Two aggravations beyond the block itself:

1. **Contradictory rows in one derivation.** The issues row proves the ids resolve to real,
   closed ticket files; the implement row claims the same ids are "missing". The reader is
   told both in the same output.
2. **The typo hint fires on the wrong cause.** With zero anchored lines, the I032 hint
   (`tickets: "…" resolved but every id is missing; check it for a typo`) points at the
   `tickets:` value — but the issues row has already proven the value correct. I117 fixed
   the adjacent wording case (anchored lines without a done-word); this no-line-at-all case
   still gets the typo message whenever ticket files exist.

This is not the I029/I117 messaging class: the verdict itself is a false block, hit whenever
an effort's close discipline lives in the ticket files rather than the SDD progress ledger.
The gap persists at HEAD (verified against `implementEvidence` at 37ddb6f, gen-13).

## Fix

Proposal, in two parts (part 1 is the substance; part 2 stands alone if part 1 is declined):

1. Accept ticket-file closure as implement evidence: a resolved ticket file with
   `status: fixed` and a non-empty `commits:` list evidences implement for that id, OR'd
   with the existing progress-ledger done-word scan. The `commits:` field is written by the
   team lead in the close commit (per `docs/issues/README.md` lifecycle) — a stronger
   implementation record than the progress-md heuristic the code itself calls a stand-in
   ("there is no authoritative on-disk artifact for 'implemented'" — with `commits:` there
   now is one). `wontfix`/`superseded` stay non-evidence: they close without implementing.
2. Scope the typo hint with the issues-row facts: when the ticket files for the anchored ids
   exist, the `tickets:` value is proven good — say the real rule (no progress-ledger
   implement line for the id(s)) instead of suggesting a typo, mirroring how I117 split the
   anchored-no-done-word case.

## Acceptance criteria

- [x] A cursor with implement `[x]`, no progress-md lines for its ids, and ticket files with
      `status: fixed` + non-empty `commits:` derives implement as match, not ticked-missing.
- [x] Negative control: same shape with `status: open` (or `fixed` with empty `commits:`)
      still derives ticked-missing — the new path is load-bearing, not a blanket pass.
- [x] Negative control: `wontfix`/`superseded` ticket files do not evidence implement.
- [x] With ticket files present for every anchored id and zero progress-md lines, the
      ticked-missing detail names the missing-progress-line rule and does not emit the
      typo hint.

<!-- Record an approved-without-test exception using the exact grammar in WORKFLOW.md's Acceptance exceptions section. -->

## Resolution

Shipped 2026-09-02 in the `ledger-burndown` effort (PRD
`docs/specs/2026-09-02-i125-closure-implement-evidence-design.md`).

- Part 1 as proposed: a ticket file with `status: fixed` and a `commits:`
  value naming at least one SHA-shaped token (hex, 7–40 chars) is a closure
  record and evidences implement for its id, OR'd with the progress-ledger
  scan. Placeholders (`[pending]`), empty or absent lists, `open`,
  `in-progress`, `wontfix`, and `superseded` are non-evidence. Closure
  records feed both derivation directions, like ledger lines.
- Part 2, corrected: the typo hint had already left the implement row at
  68aa28f (I032 passes an empty tickets value for that row), so the Problem
  sentence "still gets the typo message" was stale when this shipped. What
  remained, and what criterion 4 tests, is naming the rule: a zero-evidence
  implement miss with no anchored ledger line now says no progress-ledger
  implement line and no closure record for the id(s). The row label is now
  `implement evidence`.
- Live proof on this repo: the 68aa28f binary refused `spine cursor tick
  implement` for this ticket (no ledger line); the a171265 binary ticked it
  from the closure record alone.
- Residuals: a fixed ticket whose `commits:` names no SHA still blocks a
  ticked implement; a multi-line YAML `commits:` block reads as empty; a
  range or prefix cursor spanning long-closed tickets sees present-unticked
  on a pending implement, as it already did for their ledger lines.
