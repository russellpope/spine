---
title: "Open-ledger batch coordination PRD"
tickets: I111,I051,I050,I072,I073,I074,I075,I078,I066,I076,I077,I007,I032,I093,I102,I105,I108
created: 2026-08-29
status: active coordination PRD
---

# Open-ledger batch design

## Purpose

This PRD coordinates the 17-ticket `open-ledger-batch` named by the live
cursor. Each ticket remains its own source of requirements. This document does
not approve an unresolved design or replace a ticket-level PRD.

I112 is deliberately excluded. It is an owner-parked OpenWeights definition
decision, not batch build work.

## Scope and order

Work follows this severity order, subject to the serialization rules below:

1. High: I111.
2. Medium: I051, I050, I072, I073, I074, I075, I078, I066.
3. Low: I076, I077, I007, I032, I093, I102, I105, I108.

The only required dependency chain is I072 -> I073 and I072 -> I077. I072's
approved [host-routing PRD](2026-08-29-host-routing-config-design.md) and
[plan](2026-08-29-host-routing-config-plan.md) define the host schema and
precedence. I073 and I077 wait for the I072 ticket to be implemented and
verified, not merely for its design documents to exist.

Feature-shaped tickets require their own approved PRD before implementation:
I050, I051, I072, I073, I074, I075, I076, and I078. At HEAD `5d2825e`, I050's
[PRD](2026-08-29-approved-untested-acceptance-design.md) and
[plan](2026-08-29-approved-untested-acceptance-plan.md), plus I072's
[PRD](2026-08-29-host-routing-config-design.md) and
[plan](2026-08-29-host-routing-config-plan.md), are committed. I051's
PRD/plan work is active but is not committed evidence at this snapshot. I075
has a recommendation, not owner approval. Later feature PRDs link back here
only for batch coordination.

## Current bounded results

These tickets have committed, bounded results and are not reopened by this
batch PRD:

- I111: implementation `0723251`, closure `a7ee899`.
- I032: implementation `2e75d5e` and `910e421`, closure `1d7786b`.
- I102: implementation `35808b3`, closure `78ceeb1`.
- I108: implementation `3eae6e8`, closure `72749d9`.

I105's research note is committed in `c06a896`, but its material choice is
explicitly owner-dependent and the ticket remains open. I072's detailed PRD
and plan are committed in `b963eb9`; implementation, review, and verification
remain open. I066 remains open until its dependent wayfinder decisions land.

## Coordination rules

- Serialize I111, I074, and I078. They share audit behavior and must land in
  reviewed order. Check I007 and I075 together before dispatch because both
  touch model and dispatch resolution.
- Cross-repository changes need an explicit owner decision and their own
  repository review. I051's team-skill work belongs in the deepthought source
  repository after the spine binary contract ships. Do not edit installed
  caches. I066 and I105 are documentation/research results, not permission to
  change another repository.
- Fleet changes are a deploy decision, never an incidental implementation
  step. Template generation, estate sweeps, host-file provisioning, and
  installed-binary rollout must name their affected fleet and owner approval.
- Owner decisions remain outside implementation: I112's axis definition;
  I075's raw effort recommendation; I077 pin-ratification evidence posture;
  I093 items 3 through 5; and I105's OpenCode-versus-Pi choice. A worker may
  surface a decision with evidence, but may not silently choose it.
- I072 host files hold local capability constraints and owner-ratified pins.
  They do not turn repository mirrors into host-specific configuration.

## Gates and acceptance

Every implementation ticket keeps the `grill -> PRD -> issues -> implement
-> functional-test -> review -> verify` path from `WORKFLOW.md`. Code tickets
use TDD: record a failing focused test, implement the smallest change, run
focused tests, then run `go test ./...`. I066 and I105 follow their
source-backed documentary path and retain grill, review, and fresh verify;
they do not gain a product PRD, issues stage, or code-TDD requirement.

| Ticket group | Required ticket gate |
| --- | --- |
| I111, I051, I050, I072, I073, I074, I075, I078, I076, I077, I007, I032, I093, I102, I108 | Ticket PRD where required, focused failing test before code, task review at the ticket's routed review tier, fresh-context verify against the ticket/PRD. |
| I066, I105 | Source-backed document, scope review against the ticket, fresh-context verify that it makes no product claim or owner decision beyond the evidence. |

The final batch spec review compares the finished diff against this PRD and
every applicable ticket PRD. It must attack conflicting requirements before
accepting the diff. `spine audit routing` runs at verify with the required
transcripts. Routing records name tiers, never model IDs.

## Ship and deploy contract

Ship only after every included ticket is either fixed with its exact commits or
recorded as blocked with a concrete owner decision or dependency. Run
`maipipe run full --wait` at the exact final main SHA, record that SHA and the
lane result, then push that same SHA. A later commit requires another full run.

After ship, deployment is explicit: run `make install`, copy `~/bin/spine` to
`~/.local/bin/spine`, and verify both copies identify the shipped SHA. Any
fleet sweep, host routing-file provision, or external skill rollout is a
separate owner-authorized deploy item with its own evidence. The final handoff
lists each ticket's shipped SHA or blocker; it does not convert owner-pending
work into completion.
