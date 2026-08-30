---
id: I050
title: "Verify convention: approved-untested as a first-class acceptance state"
severity: med
status: fixed
commits: [7e94222, 4850488, fae360f, 6950089, c55faf3]
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

## Resolution

Fixed 2026-08-29 in `7e94222`, `4850488`, `fae360f`, `6950089`, and
`c55faf3`. The shared `internal/acceptance` scanner owns candidate detection,
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

Strict TDD recorded intended red failures for the scanner, D15 adapter,
stage-audit adapter, and generation migration, plus checked-state, severity,
blocking, and local-edit negative controls. Focused packages, an uncached full
`go test ./...`, `go vet ./...`, `git diff --check`, current-binary update,
and compiled-CLI acceptance cases passed. The repository has no `make verify`
target; current doctor/routing/stages commands retain their pre-existing
repository advisories and blockers. Per the controller's dispatch, a separate
fresh primary review follows this worker report.
