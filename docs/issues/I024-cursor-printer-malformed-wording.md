---
id: I024
title: spine cursor prints "derivation: clean" on a malformed cursor
severity: low
status: fixed
affects: [cli, cursor]
blocked-by: []
execution-mode: subagent-driven
tier: mechanical
review-tier: routine
---

## Problem

On a cursor with grammar findings (zero parsed stage rows), `spine cursor` prints `derivation: clean` while `spine audit stages` on the same repo blocks with "malformed cursor" — incoherent wording across the two exposures (found by the gen-8 confirmation pass, pre-existing to the 28c0608 fix). Doctor D9 likewise never surfaces grammar-level CursorFindings.

## Fix

When HasCursor && len(CursorFindings) > 0, `spine cursor` prints something like `derivation: n/a (cursor malformed)` instead of clean (still exit 0), and doctor D9 gains a warn finding for grammar problems. Keep audit stages behavior as shipped.

## Resolution

Fixed 2026-07-16 (derivation-polish batch, main @ fdad11c): `spine cursor` prints `derivation: n/a (cursor malformed)` on grammar findings (still exit 0) and doctor D9 warns with the findings (1008ed9); the final-review fix wave additionally kept the handoff detail visible under the malformed header and taught the printer to surface `Report.Notes` warnings (1441a78).
