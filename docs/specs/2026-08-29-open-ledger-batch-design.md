---
title: "Open-ledger batch coordination PRD"
tickets: I111,I051,I050,I072,I073,I074,I075,I078,I066,I076,I077,I007,I032,I093,I102,I105,I108,I121,I122,I123,I124
created: 2026-08-29
status: active coordination PRD
---

# Open-ledger batch design

## Purpose

This PRD coordinates the expanded 21-ticket `open-ledger-batch` named by the live
cursor. Each ticket remains its own source of requirements. This document does
not approve an unresolved design or replace a ticket-level PRD.

I112 is deliberately excluded. It is an owner-parked OpenWeights definition
decision, not batch build work.

## Scope and order

Work follows this severity order, subject to the serialization rules below:

1. High: I111.
2. Medium: I051, I050, I072, I073, I074, I075, I078, I066, I121, I122.
3. Low: I076, I077, I007, I032, I093, I102, I105, I108, I123, I124.

The owner added I121-I124 on 2026-08-30. They have no dependency edges and
join the same final review, exact-SHA lane, and single ship as the original
17 tickets.

The only required dependency chain is I072 -> I073 and I072 -> I077. I072's
approved [host-routing PRD](2026-08-29-host-routing-config-design.md) and
[plan](2026-08-29-host-routing-config-plan.md) define the host schema and
precedence. I073 and I077 wait for the I072 ticket to be implemented and
verified, not merely for its design documents to exist.

Feature-shaped tickets require their own approved PRD before implementation:
I050, I051, I072, I073, I074, I075, I076, I078, I123, and I124. I121 and I122
are bounded audit defects whose ticket acceptance contracts drive TDD. I050's
[PRD](2026-08-29-approved-untested-acceptance-design.md) and
[plan](2026-08-29-approved-untested-acceptance-plan.md), plus I072's
[PRD](2026-08-29-host-routing-config-design.md) and
[plan](2026-08-29-host-routing-config-plan.md), are committed. I051's
[PRD](2026-08-29-predispatch-model-validation-design.md) and
[plan](2026-08-29-predispatch-model-validation-plan.md) are committed and its
Spine phase is verified; the authorized deepthought phase is in progress. The
owner accepted I075's declared-only contract and I074's dependent verdict
contract on 2026-08-30. Later feature PRDs link back here only for batch
coordination.

## Current bounded results

These tickets have committed, bounded results and are not reopened by this
batch PRD:

- I111: implementation `0723251`, closure `a7ee899`.
- I050: implementation/correction through `a353f98`, closure `9f7c46f`.
- I078: implementation/correction through `3113ee6`, closure `25bc380`.
- I032: implementation `2e75d5e` and `910e421`, closure `1d7786b`.
- I102: implementation `35808b3`, closure `78ceeb1`.
- I108: implementation `3eae6e8`, closure `72749d9`.

I105's research note is committed in `c06a896`, but its material choice is
explicitly owner-dependent and the ticket remains open. I072's detailed PRD
and plan are committed in `b963eb9`; implementation, review, and verification
remain open. I066 remains open until its dependent wayfinder decisions land.

## Coordination rules

- Serialize audit work that shares attribution or verdict code: I111, I078,
  I072's audit boundary, I121, I122, and I074. Check I007 and I075 together
  before dispatch because both touch model and dispatch resolution.
- I051's eight controlled deepthought launch-site changes are owner-authorized
  and require their own independent deepthought review/verification followed
  by cross-repository final review. Do not edit installed caches.
- I073's named generation-14 fleet sequence is owner-ratified once I072 passes
  at an exact independently verified SHA. It still uses exact-candidate,
  review-first, no-force, stop-on-first-failure controls.
- Owner decisions remaining outside implementation are I112's axis definition
  and I105's OpenCode-versus-Pi choice. I077 is ruled BOTH mechanisms,
  warn-only; I093 is disposed through I123/I124 with no I125.
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
| I111, I051, I050, I072, I073, I074, I075, I078, I076, I077, I007, I032, I093, I102, I108, I121, I122, I123, I124 | Ticket PRD where required, focused failing test before code, task review at the ticket's routed review tier, fresh-context verify against the ticket/PRD. |
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
`~/.local/bin/spine`, and verify both copies identify the shipped SHA. Execute
only externally scoped work authorized by its ticket contract; I051's named
deepthought phase and I073's named fleet sequence are already authorized, while
host-file provisioning is not implied. The final handoff lists each ticket's
shipped SHA or blocker; it does not convert owner-pending work into completion.
