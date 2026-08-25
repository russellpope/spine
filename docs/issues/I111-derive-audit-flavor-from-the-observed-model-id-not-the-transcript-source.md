---
id: I111
title: "derive audit flavor from the observed model id, not the transcript source"
severity: high
status: open
affects: [I110]
blocked-by: [I110]
execution-mode: subagent-driven
tier: primary
effort:
risk-triggers: [cross-task-integration, plan-flagged-ambiguity]
review-tier: primary
---

## Problem

D15 derives flavor from the **transcript source**. That holds for codex, which
has its own session layout and tags its records `codex` directly. It does not
hold for open-weights models: those sessions run the ordinary `claude` CLI (via
a wrapper that passes `--model` through), so their transcripts land in the same
`~/.claude/projects` layout as real Claude sessions. `transcriptFlavor` ignores
its argument and returns the constant `"claude"` for that whole source.

Every open-weights dispatch would therefore be judged against the `claude` tier
table, where its model id does not appear — a wrong verdict on exactly the runs
(weak open models) where the routing gate matters most.

## Fix

Extend D15 so a record's flavor comes from the model id observed in the
transcript, with the transcript source retained as the tiebreaker. This is an
**extension, not a reversal**: D15's own final clause already says "where a
model id is declared under more than one flavor, the transcript-derived flavor
decides" — it reasons about model ids and uses source only to break ties. Today
the tie is trivial because one source yields one flavor; this makes the
unambiguous case authoritative and leaves the tiebreaker intact.

Replace the single per-source stamp with per-record derivation:

- Id declared under exactly one flavor in the resolved table -> that flavor.
- Id declared under more than one flavor -> fall back to transcript-source
  derivation (D15's existing tiebreaker).
- Id declared under no flavor -> preserve today's behaviour **exactly**. Do not
  invent a new failure mode.

Resolve a third tier mapping for `openweights` alongside the existing claude and
codex mappings, following the same shape.

**Do not rebuild what exists.** The architecture already supports per-record
flavor: the evidence-token type carries a flavor field, the tier mappings are
keyed by flavor so each token resolves within its own flavor, and the codex
effort already proved the seam by tagging its records at read time. The only
thing that changes is *where a claude-layout record's tag comes from*.

## The hazard — D28 must not silently stop applying

**This is the most important paragraph in this ticket.**

The audit contains a match predicate gated on a record's flavor being `claude`,
which enforces D28 (I047): a claude dispatch claims a ticket only if it *also*
references the audited repo or its session shows cwd evidence inside it. Codex
records are exempt because they are hard-scoped to the repo upstream.

Open-weights records come from the claude-layout source and have identical cwd
and description semantics — they need that same gate. But the condition is
written in terms of *flavor*, so the moment these records are tagged
`openweights`, **they fall out of the D28 check and begin claiming tickets they
should not.**

The fix is small: the condition must test "this record came from the
claude-layout transcript source", not "this record's flavor is claude". The two
were the same thing before this change and are not afterwards.

**This failure passes every existing test.** No current test has an
open-weights record, so nothing goes red; the damage shows only as wrong
verdicts in the field. The regression test below is mandatory.

**Generalisation — part of the deliverable, not incidental cleanup.** Before
implementing, grep for every comparison against a flavor literal and classify
each as genuinely flavor-dependent or actually source-dependent. Record the list
and the verdict for each in the completion report.

## Tests

Direct prior art exists in the audit suite (one flavor's id is invisible within
another; a token resolves within its own flavor; a transcript mixing claude and
codex evidence is judged per flavor). Model the new tests on those.

1. A dispatch on an open-weights id, in a claude-layout transcript, is judged
   against the `openweights` table — the core of the change.
2. A dispatch on a Claude id in the same transcript is still judged against the
   `claude` table.
3. A single transcript containing both is judged **per token, not per source**.
   Direct analogue of the existing mixed claude/codex test; highest-value case.
4. **D28 still applies to open-weights records.** A record whose model id is
   open-weights and whose text and cwd do *not* qualify for the audited repo
   must **not** claim the ticket. Without this the hazard ships silently.
5. An id declared under two flavors still resolves via the source tiebreaker.
6. An unrecognised id behaves exactly as before.

Run the full suite, not the targeted tests — the change touches a file with
substantial recent churn.

### Two guards inherited from I110's review gate

Both were surfaced by I110's cold review, are out of scope for a data-only
change, and land here because this ticket is where they become load-bearing.

7. **Enforce id/alias disjointness across flavors.** Story 5 says the ids
   "stay disjoint", and this ticket's whole derivation rests on it — the spec
   states outright that if a future edit points any `openweights` tier at a
   `claude-*` id, the core assumption breaks and the tiebreaker path becomes
   load-bearing. Disjointness was verified to hold as shipped (a scan of every
   id, alias and history entry across all four flavors found no collision) but
   **nothing fails if someone breaks it.** Add a table-level test that scans
   ids + aliases + history across every flavor and fails on any cross-flavor
   collision. This is the guard that keeps the unambiguous case unambiguous.
8. **Make the per-flavor resolution test fail on an unasserted flavor.**
   `TestResolve_NoRepoContext_ReturnsDefaultsForEveryFlavorTier` ranges over a
   hardcoded `want` map, so a flavor can ship with no resolution assertions at
   all — `pi` is in that state today. Drive the check from `Flavors()` and fail
   when the table ships a flavor the map does not cover, then backfill `pi`.

## ADR

Record the derivation change as an ADR at **0022 or higher** (0021 is taken by
the gate-panic decision). It must state that it extends D15 rather than
replacing it, name the tiebreaker as retained, and state that source and flavor
are no longer the same axis. Note that the disjointness of the openweights ids
is what makes id-derived flavor unambiguous: if a future change points any
`openweights` tier at a `claude-*` id, this ticket's core assumption breaks and
the tiebreaker path becomes load-bearing.

## Related

- Spec: `docs/specs/2026-08-25-openweights-flavor-spine-design.md` (Change 2).
  The spec deliberately carries **no line numbers** — upstream moved every
  location originally cited, including the expression inside the D28 predicate.
  Anchor on function names, D-numbers, and behaviour.
- Blocked by **I110**: there is nothing to derive until the flavor exists.
- Blast radius: `spine` is a fleet-wide binary and `spine audit routing` is a
  blocking verify gate for every repo that uses it. This edits the code that
  decides whether work is allowed to ship, and its failure mode is a wrong
  verdict rather than a crash.
