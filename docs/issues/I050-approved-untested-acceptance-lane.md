---
id: I050
title: "Verify convention: approved-untested as a first-class acceptance state"
severity: med
status: open
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
