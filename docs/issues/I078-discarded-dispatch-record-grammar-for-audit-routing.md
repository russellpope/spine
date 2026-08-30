---
id: I078
title: Discarded-dispatch record grammar so audit routing stops reading abandoned prototypes as silent descent
severity: med
status: open
commits: [29fbfe0, 48022aa]
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

- A lower-tier prototype transcript under a higher-tier ticket with a matching
  `DISCARDED` record audits as passing `discarded-with-reason`, not
  `silent-descent`.
- The same transcript without the record still fails as `silent-descent`
  (negative control).
- Documented in the audit section of the workflow docs.
