---
id: I065
title: "Estate-wide: docs/issues/README.md carries pre-convention stock text the updater no longer recognizes"
severity: low
status: open
affects: [update]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Found live during the 2026-08-10 estate sweep (I063 deployment): in at least
8 estate repos (ccq, home-lab-admin, jarvis, notetui, observability_notes,
pure-automation, ultima-dci-edition, deepthought, praxis, hbmview, moo-clone),
`spine update -write` reports

    skipped docs/issues/README.md — unrecognized local edits (use --force to drop):
      - `tier` — primary | routine | mechanical | fallback; the model tier the work is dispatched at

The "local edit" is an OLD stock line — the tier bullet before the
`tier: n/a` exemption text extended it — so the updater misreads superseded
stock text as owner customization and skips the file every sweep, exiting 1
and leaving the README a generation behind indefinitely. Spine itself had the
same condition and was reconciled by hand on 2026-08-09 (D4 reconcile commit
`0249f87`); the estate never was.

## Fix

Either add the superseded stock line(s) to the updater's known-stock set
(the `internal/update` recognized-strings tables — gen5to6/gen9to10 tests
are the pattern) so the file refreshes cleanly, or run a one-time estate
reconcile adopting the current stock text per repo. Prefer the known-stock
fix: it's the general cure for stock-text drift and makes future sweeps
exit 0.
