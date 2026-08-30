---
id: I050
title: "Verify convention: approved-untested as a first-class acceptance state"
severity: med
status: fixed
commits: [7e94222, 4850488, fae360f, 6950089, c55faf3, 3761f4f, 8f8db36, f741224, 0547817, fc20dba, a353f98]
affects: [templates, doctor]
blocked-by: []
execution-mode: subagent-driven
tier: primary
effort:
risk-triggers: []
review-tier: primary
---

## Provenance (captured 2026-08-03)

Feature-mined from agentflow's acceptance-matrix schema — see the evaluation at
`maipipe:docs/research/2026-08-03-agentflow-steal-list.md`. Agentflow's
acceptance rows carry a lane vocabulary that includes **`approved-untested`** —
a first-class, honest way to record "we did not verify this and someone chose
to accept it anyway" — and a `waived` status that requires both a note and an
approval reference (their implementation additionally requires an
independently authenticated approval record, on the argument that a ledger can
name an approver but cannot authenticate one).

## Problem

The estate's verify-stage convention is binary in practice: acceptance
criteria are checked or the effort isn't done. Reality occasionally includes
deliberate owner-approved exceptions (a criterion waived at review, a check
deferred to a follow-up ticket). Today those live as prose in handoffs or
review notes — invisible to `spine audit stages`, `spine doctor`, and anyone
reading the ticket later, and indistinguishable from a criterion that was
silently skipped.

## What to build

- A convention (issue-template + WORKFLOW.md wording, template generation
  bump) for recording a waived/approved-untested acceptance item **on the
  ticket itself**: the unchecked box stays unchecked, annotated with a
  structured marker carrying a reason and an approval reference (who/where —
  review note, handoff, or session), dated.
- Grammar decided at PRD time; candidates: a checkbox annotation line
  (`- [ ] … — WAIVED <date> by <ref>: reason`) mirroring the ESCALATION
  record style in `.superpowers/sdd/progress.md`.
- Doctor/audit posture: a waived item with the full record is clean; a waived
  marker missing reason or approval reference is a warning. Absence of any
  waived markers changes nothing — the common case stays zero-cost.

## Open questions for the PRD grill

- Does this ride the stage-cursor evidence rules (verify stage) or stay
  ticket-local?
- Whether spine should distinguish waived-item counts in `audit stages`
  output, and at which severity.
- Authentication: the estate has no HMAC seam; is "approval ref points at a
  dated artifact" sufficient? (Probably yes — record the decision.)

## Acceptance criteria (sharpened at PRD time)

- [x] Template generation ships the convention; `spine update` propagates it.
- [x] A well-formed waived item passes doctor; a reason-less one warns
      (negative control).
- [x] The grammar is documented next to the ESCALATION grammar it mirrors.

## Prior implementation and reopening

The first implementation landed 2026-08-29 in `7e94222`, `4850488`,
`fae360f`, `6950089`, and `c55faf3`. The shared `internal/acceptance` scanner
owns candidate detection,
the exact single-line grammar, acceptance-section scope, dated Markdown
reference validation, symlink containment, ticket discovery, and counts.
References establish local provenance only: spine does not authenticate the
approver or resolve the fragment.

Doctor D15 scans tickets at every status and emits one `warn` per malformed
candidate. `audit stages` scans only cursor-resolved ticket IDs, prints one
warning per invalid record and an optional valid/invalid summary, and leaves
`Report.Blocking()` unchanged. The no-marker audit path is byte-identical to
the pre-I050 output.

Template generation 12 adds the grammar to `WORKFLOW.md`, an empty acceptance
section to new tickets, and a semantic pointer to the issue-ledger README.
The captured generation-11 migration is additive and idempotent, preserves the
knowledge profile's no-issue-ledger manifest, and refuses local edits in all
five touched managed files without force.

The ticket was prematurely closed by `4b04184` before its required fresh
primary review and independent verification. The fresh primary review failed
with five findings: relative roots suppress scanning, the default Scanner
limit hides long and later lines without surfacing read errors, failures are
not fully aggregated, valid bare/tab-delimited H1/H2 headings do not end the
acceptance section, and the ledger/gate state was advanced too early. I050 is
therefore open again, and every acceptance criterion remains unchecked until
a later fresh primary re-review and a different independent verifier approve
the corrected exact SHA.

Correction commits so far are `3761f4f` (reopen the ticket and amend the
binding design/plan before product edits) and `8f8db36` (relative-root
normalization, unlimited line reading with surfaced errors, deterministic
multi-failure aggregation, complete H1/H2 boundary recognition, and compiled
CLI regressions), plus `f741224` (compiled hostile outside-root symlink
acceptance). Focused affected-package tests and repeated aggregation/path
controls passed after their recorded red runs. Full repository verification,
fresh re-review, and independent verification remain pending; this is not a
closure resolution.

The different fresh primary re-review at exact SHA `3a0a46a` failed with three
additional findings. The parser accepts ASCII or Unicode whitespace in the
approver and full reference token, selects the first bare sentinel substring
instead of the unique exact structural marker, and leaks pre-ID failures from
unknown unscoped tickets into cursor-scoped audit output. The re-review also
found contradictory no-marker/read-error promises. The amended binding design
and plan now require zero candidates plus zero applicable scan errors for byte
compatibility, fail closed by frontmatter identity in `ScanTicketIDs`, allow
sentinel bytes in criterion text, reject multiple structural markers, and name
package, doctor, audit, and compiled-CLI regressions for all three corrections.
I050 stays open, every acceptance criterion stays unchecked, and fresh
re-review plus independent verification remain pending.

The batch gate rerun after I032's layout correction also found a loop-scoped
`defer f.Close()` in ticket discovery. This correction records the gate red,
closes each ticket at the end of its own scan iteration, preserves scan-error
reporting, and reruns both the go@1 `dead-code-callgraph` and
`deferred-cleanup-errcheck` classes before its code commit.

Second-correction commits are `0547817` for the binding contract amendment and
`fc20dba` for whitespace-free tokens, structural marker selection, fail-closed
identity scoping, per-iteration close/error handling, and package/doctor/audit/
compiled-CLI regressions. Focused and full local verification passed before
the code commit. Fresh primary re-review, independent verification, maipipe at
the eventual exact review SHA, and ticket closure remain pending.

The deterministic-order correction is `a353f98`. It places the rule-2
`criterion is required` failure before the rule-3 multiple-structural-marker
failure and keeps both before date and later fields. The exact combined
package regression and doctor, audit, and compiled-CLI output regressions were
observed red before the aggregation-only reorder and pass afterward. I050
remains open and unchecked pending a fresh primary re-review, a different
independent verifier, final maipipe, and ticket closure.

## Resolution

Generation 12 ships the ticket-local `APPROVED-UNTESTED` grammar and the
additive, idempotent generation-11 migration. Doctor D15 scans every ticket and
warns on malformed records while accepting complete records. `audit stages`
scans cursor-resolved tickets, reports valid and invalid counts when present,
and keeps acceptance warnings nonblocking. The generated workflow, issue
template, and ledger documentation carry the convention, and `spine update`
propagates it without overwriting local edits.

The correction history above records the premature closure and the fixes for
root handling, unbounded lines and read errors, failure aggregation, heading
boundaries, hostile references, token whitespace, exact marker selection,
identity scoping, per-file cleanup, and deterministic error order. The fresh
primary re-review and different independent primary verifier both passed at
exact SHA `42eba0a`. Both required go@1 gates reported no findings, and
maipipe run HEAD #3 passed at `42eba0a`.

This closes I050 only. Later batch commits still require the final batch
exact-SHA lane. No push or binary install is claimed.
