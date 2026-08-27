---
id: I045
title: Record-wins fallback reading of shared-id tokens (ADR 0012)
severity: high
status: fixed
affects: [audit]
blocked-by: [I040]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: primary
---

## What to build

ADR 0012 / design D25. FALLBACK-record consultation moves before tier
resolution: when the audited ticket carries a FALLBACK record and an
observed token's candidate tiers include fallback, the token resolves as
fallback and judges escalated-with-reason (advisory). Without a record, the
ordered reading stands and real descent still judges silent-descent.

This is the shared-id edge the flavor table sanctions deliberately (codex
routine/fallback share one id): a properly recorded refusal-rerun must never
be a standing false blocker, and an unrecorded off-tier dispatch must never
hide behind a lateral fallback interpretation.

## Acceptance criteria

- [ ] Shared-id token on an above-tier ticket WITH a FALLBACK record → escalated-with-reason, reason quoted
- [ ] Same fixture WITHOUT the record → silent-descent (blocking), unchanged
- [ ] Token resolving only to fallback keeps existing behavior (record → escalated-with-reason; annotation `tier: fallback` → match; neither → unexplained-fallback)
- [ ] ESCALATION-record semantics untouched; all prior blocking scenarios pass unchanged except where ADR 0012 says otherwise
- [ ] `go test ./...` green

## Blocked by

- I040

## Resolution — closed 2026-08-26 (ledger reconciliation)

Shipped; never closed. Two independent pieces of evidence:

- **ADR 0012** is ratified and carries this ticket's decision by name:
  `docs/adr/0012-fallback-records-excuse-the-fallback-reading-of-shared-id-tokens.md`.
- The record-wins reading is implemented and documented in
  `internal/audit/audit.go` (see the `FALLBACK` clauses around lines 21 and
  71-76: "without a FALLBACK record the ordered reading … does not manufacture
  descent"), and asserted by `internal/audit/audit_test.go:153-158`.

Closed transitively by **I048** (`fixed` 2026-07-27), which lists this ticket in
`blocked-by`.
