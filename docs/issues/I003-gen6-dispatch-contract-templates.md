---
id: I003
title: Gen-6 templates — dispatch contract, ticket annotation fields, gen5→6 migration, ADR
severity: med
status: open
affects: []
blocked-by: [I002]
parent: [I001]
labels: [ready-for-agent]
execution-mode: subagent-driven
tier: routine
effort: medium
risk-triggers: [plan-flagged-ambiguity]
review-tier: primary
---

## Parent

I001 — implements the declared-intent and dispatch-discipline layers of
`docs/specs/2026-07-09-model-routing-design.md`.

## What to build

`spine init` emits a WORKFLOW.md carrying the full dispatch contract from
the spec: four tiers with the tier→model-id mapping, tier-default efforts,
escalate-freely-with-reason / silent-descent-fails rule, reviewer floor +
the four named risk triggers, fallback semantics (proactive security-framed
pre-routing + reactive orchestrator-mediated re-dispatch with ledger record
and push notification; the word "auto" removed; standalone security_routing
key folded in), execution-mode defaults (subagent-driven/ultracode default,
inline as justified exception), plan-gated ultracode opt-in, and the
verify-stage requirement to run `spine audit routing`. The ticket template
gains optional annotation fields (execution-mode, tier, effort,
risk-triggers, review-tier). `spine update` migrates gen-5 repos forward.
The tiers-not-ids + estate-owned-contract ADR is recorded.

## Acceptance criteria

- [ ] Scaffold tests assert the gen-6 contract content: four tiers present, "auto" absent, effort defaults, escalation rule, verify-stage audit line
- [ ] Ticket template contains the annotation fields; plain bug issues remain valid without them
- [ ] gen5→6 migration test follows the established genNtoM pattern and carries a gen-5 fixture forward correctly
- [ ] ADR records tiers-not-ids + estate-owned placement (hard to reverse, surprising, real trade-off)
- [ ] Contract text uses CONTEXT.md glossary vocabulary exclusively

## Blocked by

- I002 — a shipped generation must not reference a command that does not exist.

## Risk-trigger note

Contract wording propagates fleet-wide on sweep — routine transcription
implementer, primary-tier review per the floor+trigger rule.
