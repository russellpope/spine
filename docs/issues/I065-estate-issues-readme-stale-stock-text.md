---
id: I065
title: "Estate-wide: docs/issues/README.md carries pre-convention stock text the updater no longer recognizes"
severity: low
status: fixed
batch: 2026-08-27-dhyg#1
commits: [6f306a2]
affects: [update, I106]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
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

## Resolution

Fixed 2026-08-27 (batch 2026-08-27-dhyg#1, commit 6f306a2, spec
docs/specs/2026-08-27-doctor-hygiene-batch-design.md). The known-stock fix:
the full-history audit of templates/current/issues-README.md found THREE
retired lines — the pre-`superseded` `status` bullet (retired d78f6ee), the
pre-I046 `tier` bullet this ticket names (retired c55ffb3), and the
pre-`review-tier: n/a` `review-tier` bullet (retired 3dacdde) neither ticket
nor spec had named — all added verbatim to the updater's superseded-lines
set, with gen-migration tests per historical render plus a negative control.
The gen-bump rule is now stated as binding in the set's doc comment. Spine's
own README refreshed in the same commit; `spine update --dir .` and
`spine doctor` both exit 0. The 11-repo estate sweep is the batch's
deploy-stage checklist item, not this ticket's code.
