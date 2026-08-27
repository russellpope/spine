---
id: I046
title: tier n/a — explicit routing exemption for pre-convention tickets
severity: low
status: fixed
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

## Resolution — closed 2026-08-26 (ledger reconciliation)

Shipped; never closed. The `n/a` tier exemption is implemented in
`internal/audit/audit.go`.

Closed by **I048** (`fixed` 2026-07-27), whose live acceptance exercised this
exemption directly and recorded the result: "tier: n/a applied to moo-clone
I001-I007 -> 7 exempt rows, post-convention I100-I106 stay loud". That is
precisely this ticket's intended behaviour — pre-convention noise silenced,
post-convention tickets still enforced.
