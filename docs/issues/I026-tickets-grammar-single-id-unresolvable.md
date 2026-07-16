---
id: I026
title: tickets: grammar — bare single id unwritable; unresolvable values degrade silently
severity: med
status: open
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
