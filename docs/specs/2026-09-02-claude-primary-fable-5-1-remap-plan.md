# Estate default Claude primary remap to Fable 5.1 — plan

**Date:** 2026-09-02 · **Commit:** 68aa28f · **Ticket:** none at ship time (recorded by I128)

Retroactive record of the sequence 68aa28f followed, written 2026-09-03.

## Build sequence

1. Change the embedded Claude primary entry to `claude-fable-5-1`, move
   `claude-fable-5` to the row's history, and replace the exact-id alias.
2. Mirror the change into the gen-13 model-table fixture.
3. Update model, update, audit, and CLI test expectations (the template test
   was missed; see the design's Testing Decisions); add the pair-aware
   history test for the primary row.
4. Couple the gen-13 to 14 lock to the report's itemized refresh; add the
   row's old and new values to the text allowlist the gen-9 to 10 and 11 to
   12 locks consult.
5. Refresh spine's own WORKFLOW.md through `spine update --write`.
6. Full suite, maipipe full lane, install both binaries, push.

## Stage mapping

The commit ran no cursor effort; I128 supplies the review and verify
stages retroactively (spec-review against the paired design, recorded in
I128).

## Risks and controls

- A displaced id left out of history makes every fleet mirror read as an
  override and never refresh (controlled by the pair-aware history test).
- The fleet was not swept in the same session; the hazards that produced
  are I128.
