---
id: I026
title: tickets: grammar — bare single id unwritable; unresolvable values degrade silently
severity: med
status: fixed
affects: [cursor, audit, template]
blocked-by: []
execution-mode: subagent-driven
tier: routine
review-tier: routine
---

## Problem

Two coupled gaps from the gen-8 final review (requirements-attack 3 + M5): (1) the published grammar `tickets: I0NN-I0MM | prefix I0` cannot express a single-ticket effort (`I001` doesn't resolve; `I001-I001` works but is undocumented); (2) an unresolvable `tickets:` value silently degrades the issues+implement evidence rules to not-judged with no operator-visible signal.

## Fix

Accept a bare id (and document same-endpoint ranges); surface unresolvable values as a Notes entry naming the bad value (per the no-cursor notes pattern) — conservative non-blocking, but visible. Update the grammar reference in internal/cursor Grammar + the gen-8 template section together (they must stay verbatim-identical); template content change bumps generation or rides the next gen per update policy — flag that choice to the owner at dispatch.

## Resolution

Fixed 2026-07-16 (derivation-polish batch, main @ fdad11c, 0667a76 + 109e207): bare ids resolve (`tickets: I001`), same-endpoint ranges documented; unresolvable values surface as a non-blocking warning naming the value (audit stages, doctor D9, and — post fix wave — spine cursor). Owner ruled the template change bumps generation 8→9 (ADR 0004), paired with I027's template text; verbatim-identity of Grammar const and template locked by TestCursorGrammarVerbatimInTemplate; gen-8 repos regenerate cleanly (supersededLines + gen8to9 test trio; FT6 dry-run proof on a real gen-8 repo). Fleet sweep to gen 9 deliberately NOT included — follow-up at the owner's call. Missing-ids detail discoverability follow-up: [I029](I029-ticked-missing-names-missing-ids.md).
