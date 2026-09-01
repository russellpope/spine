---
id: I076
title: Routing-yield forward build — REVIEW record line + spine yield verb
severity: low
status: fixed
commits: [de0c935, 6ba2092, e7d96f5, a7ef427, 6b3d5f4, ca2245b, 9aa6c2f, baaea09, b73d344, 0b49e72, a50bcc2, a9b9394, 2560c64, b6839c0, c1c8ab8, e21019b, 0fb43a5, 70801d1, 6cb9822]
affects: [audit, cli, workflow]
blocked-by: []
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

Build the forward-looking half recommended by
`docs/research/2026-08-05-routing-yield-feasibility.md` (which resolved
[I052](I052-routing-yield-feedback-charter.md)): one review-time record line in
the progress-ledger grammar, spine parsing records only (never filenames), and
`spine yield` (per-repo + `--fleet`) that always prints counts and refuses
rates below a floor (n≥20 low-confidence, n≥40 stated rate). Heterogeneous
amendment: the proposed line keyed `<flavor>/<tier>` must instead carry the
**actual model id** (and harness, post-[I073](I073-flavor-to-harness-rename-migration.md)
naming) — per-family yield is precisely the evidence the equivalence pins of
[I068](I068-host-scoped-availability-and-tier-pins.md) need. Escalation
frequency needs no new record (derivable from existing ESCALATION/FALLBACK
lines).

## Accepted design

The accepted I076 design is
[the 2026-08-30 routing-yield review-record PRD](../specs/2026-08-30-routing-yield-review-record-design.md)
with its [paired implementation plan](../specs/2026-08-30-routing-yield-review-record-plan.md).
It records explicit task-gate REVIEW lines keyed by canonical harness, actual
model ID, and effective tier, plus a separate final-review series that never
changes a task-rate denominator. It does not infer outcomes from filenames or
transcripts and introduces no rating rule.

## Owner-directed batch-lane amendment (2026-08-31)

The original I073 prerequisite and its standalone exact-SHA maipipe rationale
are preserved in the accepted design and plan as historical context. The owner
has superseded that standalone-lane requirement only: I073 is now satisfied by
fixed product SHA `46b2324`, the PASS all-20 primary fleet result, closure
`dcb1c3e`, and a fresh primary post-fleet PASS. Canonical `harness`
terminology and its bounded compatibility surface are therefore available to
I076; the product contract still has no `flavor` alias.

I076 Task 1 may pass on that durable evidence. This ticket remains open for its
gated tail. After focused and full review, an independent verification, and the
final requirements attack pass, I076 closes as `fixed, pending batch ship`.
The owner-directed batch-final exact-SHA `maipipe run full --wait` runs only
after every included ticket is fixed, or is recorded blocked by a concrete
owner decision or dependency,
following the final whole-branch review, routing audit, and fresh handoff. No
standalone I073 or I076 lane is required or claimed. The batch-final lane is
the ship verdict; any post-lane commit invalidates it and requires a rerun at
the new exact SHA.

## Resolution

Fixed 2026-09-01 at exact product SHA `6cb9822`. I076 adds the strict,
column-zero REVIEW grammar and read-only `spine yield` command for one selected
repository or an immediate-primary fleet. Task outcomes aggregate only by
canonical harness, opaque actual model ID, and effective implementer tier;
final outcomes remain a separate rate-free series. Exact duplicates warn and
deduplicate, conflicts and malformed evidence are excluded, task rounds must
be contiguous and terminate in acceptance, and rework remains attributed to
the round-one route.

The report implements permanent `n=0,19,20,39,40` boundaries: rate refused /
insufficient below 20, percentage / low-confidence at 20–39, and percentage /
stated at 40 or more. Text and JSON share typed zero-evidence state, sorted
cells/repositories/diagnostics, counts on refusal, isolated peer retention,
and the documented exit-0/1/2 contract. Existing exact model-tier ESCALATION
and FALLBACK records contribute report-wide counts only; effort and malformed
forms do not create cell attribution.

Selected progress ledgers and fleet discovery are descriptor-rooted,
no-follow, identity-revalidated reads. The implementation binds the fleet
parent, each child, `.git` eligibility, `.superpowers`, `sdd`, and
`progress.md` through observation, open, read, and final revalidation; symlink
and different-object replacements fail closed, same-object controls pass, and
child failures retain valid peers with bounded diagnostics. The command reads
no transcript, ticket, model table, Git history, host configuration, sibling
ledger, or user-home evidence and writes no file. Raw malformed text,
conditions, reviewer reasons, outside paths, and ambient secrets are never
printed or inferred.

Three primary requirements-review rounds drove hostile corrections for
incomplete sequences, all Unicode whitespace and quotation classes,
zero-evidence JSON parity, selected-ledger and fleet TOCTOU binding, and
multi-round conflict determinism. The final fresh primary review and separate
independent primary verifier both passed exact SHA `6cb9822`. Evidence included
focused/full/full-race/vet/format/diff gates, native/Linux/Windows builds,
guard mutations, exhaustive quote/whitespace matrices, 200 fresh conflict
processes with byte-identical text/JSON, 2,309 hostile assertions across 495
fresh CLI processes, and unchanged fixture hashes. All eight acceptance
criteria passed.

I073's canonical harness prerequisite is satisfied by product SHA `46b2324`,
the all-20 fleet PASS, closure `dcb1c3e`, and fresh post-fleet primary PASS.
Per the owner-directed open-ledger sequence, I076 is now fixed pending batch
ship; no standalone I076 lane is claimed. The sole exact-SHA
`maipipe run full --wait` remains the later batch-final ship verdict after the
whole-branch review, routing audit, and fresh handoff.
