---
id: I002
title: spine audit routing — audit core + CLI
severity: med
status: open
affects: []
blocked-by: []
parent: [I001]
labels: [ready-for-agent]
execution-mode: subagent-driven
tier: primary
effort: high
risk-triggers: [plan-flagged-ambiguity]
review-tier: primary
---

## Parent

I001 — implements the audit layer of `docs/specs/2026-07-09-model-routing-design.md`.

## What to build

Running `spine audit routing` in a scaffolded repo prints a per-task table —
declared tier → actual model(s) → verdict — sourced from the repo's
WORKFLOW.md tier mapping, ticket annotations, escalation records in the
build ledger, and the harness's per-project transcript records. The owner
gets routing ground truth at the verify stage for zero tokens.

Verdicts: match · escalated-with-reason (advisory) · silent-descent
(blocking) · unmapped-dispatch (warn) · no-transcript (warn). The boundary
is a pure function from (repo, transcript records) → report; the CLI is a
thin printer per the existing two-level stdlib dispatch.

## Acceptance criteria

- [ ] Clean fixture (annotations match transcript) → all-match report, exit 0
- [ ] Escalation with recorded reason → advisory verdict, exit 0
- [ ] Dispatch below annotated tier with no recorded reason → silent-descent, exit non-zero
- [ ] Model id absent from the repo's tier mapping → unmapped-dispatch warning, exit 0
- [ ] Missing transcript dir or malformed JSONL → warning verdicts, never a failure (parser rot must not fail builds)
- [ ] Unannotated tickets reported as unannotated, not judged
- [ ] Package tests cover all of the above against fixture repos + fixture transcript JSONL (prior art: adopt/doctor testdata pattern)

## Blocked by

None — can start immediately.

## Risk-trigger note

Transcript JSONL is an undocumented internal format and verdict semantics
encode D3/D9 judgment — primary-tier implementation and review.
