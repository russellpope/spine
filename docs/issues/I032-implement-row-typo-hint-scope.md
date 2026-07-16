---
id: I032
title: Scope the all-missing tickets-typo hint to the issues row; decouple truncation test from cap
severity: low
status: open
affects: [I029]
blocked-by: []
execution-mode:
tier: mechanical
effort:
risk-triggers: []
review-tier:
---

## Problem

Two Minors from the gen9-sweep-i029-i030 final review (2026-07-16), accepted as non-gating and deferred here:

1. **Misleading typo hint on the implement row** (`internal/stages/stages.go:328-330`, call site `:276`). I029's
   all-missing hint (`— tickets: "<value>" resolved but every id is missing; check it for a typo`) fires for ANY
   all-missing set, including implement's ledger-evidence set. Reproduced: issues row `match 2/2` (tickets: value
   demonstrably correct) while the implement row still says "check it for a typo" — the real cause is absent
   ledger dispatch records, and the hint points the reader at the wrong artifact. Verdict/blocking behavior
   correct; text only. Partly ticket-inherited (I029's Fix directed the hint generically into judgeSet for both
   stages; its motivating scenario only ever exercised the both-rows-all-missing case).

2. **Truncation test couples to the cap value** (`internal/stages/stages_test.go:434-459`).
   `TestTickedMissingTruncatesLongMissingSet` hardcodes a 7-ticket range and asserts `!Contains("I007")`;
   raising `maxNamedMissingIDs` above 6 silently flips the test's premise.

## Fix

1. Pass `ticketsRaw` only at the issues call site (`judgeSet(s.State, implPresent, ids, "", "ledger implement
   evidence")`) so the typo hint appears only where the missing artifacts are the ticket files themselves —
   or gate the hint on the sibling issues row's evidence. Keep ids named on both rows.
2. Derive the truncation fixture's range size from `maxNamedMissingIDs` (or pin the coupling with a comment
   and an exact-boundary case).
