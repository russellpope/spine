---
id: I052
title: "Charter: measured yield per (flavor, tier) as routing-table feedback"
severity: low
status: fixed
affects: [audit]
blocked-by: []
execution-mode: subagent-driven
tier: primary
effort:
risk-triggers: []
review-tier: primary
---

## Provenance (captured 2026-08-03)

Feature-mined from agentflow's `usage yield` (accepted-result yield per task
class) — see the evaluation at
`maipipe:docs/research/2026-08-03-agentflow-steal-list.md`.

## Problem

The routing-purpose decision (2026-07-09) justifies tiers by principle
(quality ceiling first; down-route only provably mechanical work) and enforces
them by audit. There is no measured feedback: nothing tells the estate whether
routine-tier implementers actually produce accepted-first-pass work at a rate
that justifies the down-route, or whether a tier change (e.g. a new shipped
default) moved outcomes. Tier-table changes are argued, never measured.

## Charter (feasibility first — do not build from this ticket)

Assess whether per-ticket outcome data can be harvested from artifacts the
conveyor already leaves — `.superpowers/sdd/` dispatch/review/rereview files,
ESCALATION records, `spine audit routing` verdicts, ticket status history —
into a per-(flavor, tier) yield view: accepted-first-pass rate, rework rounds,
escalation frequency. Key questions:

- Is "accepted first pass" reliably derivable from existing dispatch/review
  file pairs, or does it need a new record at review time?
- Where does the view live — a `spine` verb over one repo, `--fleet` like
  `handoff latest`, or out of scope for spine entirely?
- Sample-size honesty: per-tier counts are small; the output must carry counts
  and refuse conclusions below a floor, not print bare percentages.

Output of this charter is a feasibility note (docs/specs or a research note)
with a build/no-build recommendation, not an implementation.

## Resolution

(2026-08-10) Charter answered by
`docs/research/2026-08-05-routing-yield-feasibility.md` (committed with the
I066 wayfinder map). Verdict: **build-differently** — retrospective harvesting
over dispatch/review/rereview files is unreliable (demonstrated filename false
negatives, flavor structurally absent from files, no historical cell clears an
honest sample floor). The forward-looking half — one review-time REVIEW record
line in the ledger grammar + `spine yield` (per-repo and `--fleet`, counts
always, rates only above a floor) — is mapped as
[I076](I076-routing-yield-review-record-and-yield-verb.md) under the
[I066](I066-claude-harness-3p-models-wayfinder-map.md) map, amended to key on
actual model id for heterogeneous pools. Escalation frequency was already
derivable from existing ESCALATION/FALLBACK records with no new artifact.
