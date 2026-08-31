---
id: I074
title: Audit routing verdicts for heterogeneous dispatches
severity: med
status: open
affects: [audit]
blocked-by: [I070]
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

Extend `spine audit routing` for declare-then-confirm
([I069](I069-attribution-declare-then-confirm.md)): the dispatch declares
(harness, model, effort); the audit confirms against the transcript. What is
the verdict vocabulary — confirmed; declared-vs-observed mismatch (the new
silent-descent analogue for proxied pools); unconfirmable (transcript carries
no usable id for a known combo — distinct from silent-descent and from
unmapped-dispatch)? Does effort get confirmed too, or declared-only? Verdicts
must respect host-pinned equivalences ([I068](I068-host-scoped-availability-and-tier-pins.md)):
a routine ticket running the host's pinned gpt-5.6-sol is conformant, not an
escalation. Blocked on [I070](I070-proxied-model-ids-in-claude-transcripts.md)
for what observed ids exist to confirm against.

## Implementation notes

I074 implementation is in review. It consumes only complete I075 declarations
and validated I072 host-local observed IDs, compares final pinned targets, and
requires complete event identity plus a linked worker event. No current
transcript format proves observed effort, so supported records remain
`observed-effort=-` and `unconfirmable`; the confirmed/mismatch matrix is
exercised through the narrow internal evidence seam. This ticket remains open
for fresh review and independent verification.
