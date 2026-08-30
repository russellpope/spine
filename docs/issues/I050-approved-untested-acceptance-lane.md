---
id: I050
title: "Verify convention: approved-untested as a first-class acceptance state"
severity: med
status: open
commits: [7e94222, 4850488, fae360f, 6950089, c55faf3, 3761f4f, 8f8db36, f741224]
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

- [ ] Template generation ships the convention; `spine update` propagates it.
- [ ] A well-formed waived item passes doctor; a reason-less one warns
      (negative control).
- [ ] The grammar is documented next to the ESCALATION grammar it mirrors.

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
