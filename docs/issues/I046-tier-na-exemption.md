---
id: I046
title: tier n/a — explicit routing exemption for pre-convention tickets
severity: low
status: open
affects: [audit]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Design D27. A ticket may declare `tier: n/a` to opt out of routing judgment,
mirroring the existing `review-tier: n/a` convention. The audit reports such
tickets as exempt — distinct from unannotated, never judged, never warned.
An empty tier stays loud: absence is a gap, n/a is a decision. The ticket
template gains the hint.

## Acceptance criteria

- [ ] `tier: n/a` ticket reports exempt; no unannotated noise, no judgment
- [ ] Empty tier still reports unannotated exactly as today
- [ ] Unknown tier values still report unannotated-with-detail as today
- [ ] Template documents the exemption
- [ ] `go test ./...` green

## Blocked by

- None — can start immediately.
