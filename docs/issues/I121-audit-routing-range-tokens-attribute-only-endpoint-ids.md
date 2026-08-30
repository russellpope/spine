---
id: I121
title: "audit routing: a ticket-range token attributes only its endpoint ids"
severity: med
status: fixed
commits: [fca3ced, 13859a1, 0abd5b3, c8d88cf, b4cd4a1]
affects: []
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

A dispatch whose first line carries a ticket RANGE (e.g. "tickets I051-I056", the same range
grammar `spine cursor` accepts) is attributed by `spine audit routing` to only the two endpoint
ids — the literal substrings `I051` and `I056` — leaving the interior ids (`I052`–`I055`)
reported as `no-transcript` even though the dispatch covered them all.

Observed live in maikanban on 2026-08-29 (`spine audit routing --since 2026-08-28`): a fix-wave
dispatch opening "Fix all 10 confirmed findings … (tickets I051-I056)" produced `match` rows
for I051 and I056 and `no-transcript` for I052–I055, misreporting four tickets whose evidence
existed.

## Fix

Expand a range token `I0NN-I0MM` found in an attributable first line into its full id list
before matching, using the same range grammar the cursor accepts (which I114 notes cannot yet
express a two-ticket batch — resolve consistently with whatever that lands). Tests: a
transcript with "I051-I056" in its first line attributes all six ids; a hyphenated non-range
string does not over-attribute (negative control).

## Related

- I114 — cursor tickets range grammar limits (shared grammar decision).
- maikanban `.superpowers/sdd/progress.md` ROUTING AUDIT FACTS entry, 2026-08-29.

## Resolution

Fixed 2026-08-30 at final I121 product SHA `b4cd4a1`. Shared ticket-reference
parsing now applies the cursor-compatible inclusive range grammar through
bounded arithmetic membership across Claude, Codex, workflow, and discovery
paths. It rejects every endpoint of chained, partial, or surrounding-hyphen
forms while preserving the exact `dispatch-task-I###.md` compatibility
carrier, lowercase Codex behavior, and later-message near misses. A fresh
primary review and a different independent primary verifier passed hostile
cross-source fixtures plus full/race/vet/build gates at `b4cd4a1`. The single
batch-final exact-SHA maipipe lane remains the ship gate.
