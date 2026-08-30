---
id: I078
title: Discarded-dispatch record grammar so audit routing stops reading abandoned prototypes as silent descent
severity: med
status: fixed
commits: [29fbfe0, 48022aa, 22facd1, e5752b1, 6e2e79d, 4de34ff, 6ca5428, 8a64449, 0e43da5, 30d91be, 951f8e5, 5df47c3, 5cf8f68, 3bc182f, 3bf5c0e, 3113ee6]
affects: [audit]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier:
---

## Problem

`spine audit routing` classifies any lower-tier transcript attributed to a
higher-tier ticket as `silent-descent`, which fails the verify gate. There is
no record grammar to truthfully classify a **discarded dispatch** — a
prototype or exploratory run at a lower tier whose output was thrown away and
reimplemented at the required tier.

Observed 2026-08-14 in maikanban I014 (primary-tier acceptance ticket): the
team lead ran a routine-tier prototype, discarded it, recorded
`ESCALATION I014 routine->primary reason: discarded routine prototype and
reimplementation at required primary tier` in its dispatch ledger at the time,
and landed only primary-tier reviewed work. The audit still exited 1 with
`silent-descent` — transcripts are immutable and no retrospective record can
cure the classification, so the owner had to grant a manual one-time waiver to
close verification despite fully honest contemporaneous recording.

Related known gap (same audit, different shape): herdr claude worker
transcripts read as `unattributed-transcript` (seen in maikanban I004,
2026-08-11) — evidence exists in retained dispatch files but the audit cannot
attribute it.

## Proposal

Add a record grammar the audit recognizes, e.g. a ledger line
`DISCARDED <ticket> tier:<tier> reason:<text>` written at (or after) dispatch
time, that reclassifies a matching lower-tier transcript from `silent-descent`
to `discarded-with-reason` (audit passes, classification preserved in output —
analogous to the existing escalated-with-reason handling for `FALLBACK`).
Guardrails: a `DISCARDED` record must not suppress descent findings for
transcripts whose work reached the merged range; if the audit can cheaply
correlate transcript output with landed diffs, refuse the reclassification on
overlap.

## Acceptance

- [x] A lower-tier prototype transcript under a higher-tier ticket with a matching
  `DISCARDED` record audits as passing `discarded-with-reason`, not
  `silent-descent`.
- [x] The same transcript without the record still fails as `silent-descent`
  (negative control).
- [x] Documented in the audit section of the workflow docs.

## Resolution

Fixed 2026-08-30. A `DISCARDED` record excuses only the exact ticket, source,
session, dispatch event, and resolved tier. Cross-source or cross-session
collisions, quoted or malformed records, duplicate or ambiguous declarations,
and identity-less root-only Codex workers remain unexcused. Template generation
13 publishes the grammar and preserves I050's retained wording.

Routine final re-review passed at `3113ee6`; independent verification passed at
integrated `04f9ea4`, after confirming the later I072 changes do not overlap
I078 paths. Both `go@1` checks passed, and maipipe `main #64` passed at
`04f9ea4`. Repository-wide routing and stage audits have unrelated blockers and
are not claimed green here.
