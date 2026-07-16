---
id: I028
title: Story-11 closure — objectstudio/maipipe WORKFLOW reconciliation, 4 uncommitted sweep repos, old I006
severity: med
status: fixed
affects: [fleet]
blocked-by: []
execution-mode: inline
tier: primary
review-tier: n/a
---

## Problem

The gen-8 sweep (I023) left the PRD's story 11 ("no repo remains a gap") partially open: objectstudio WORKFLOW.md still gen 6 (26 unrecognized local edits incl. the deliberate vfb gate — doctor D4 warn is its standing state) and maipipe WORKFLOW.md still gen 7 (unrecognized local edits, recorded in the I023 sweep table); hbmview, home-lab-admin, moo-clone, praxis have gen-8 content on disk but uncommitted (pre-existing dirty AGENTS.md, no-mixing rule). Old I006 (model-routing fleet sweep + memory supersede) is partially subsumed by I023.

## Fix

Owner-in-the-loop (hence inline/HITL): rule on each unrecognized WORKFLOW.md edit (fold into gen 8 by hand like the objectstudio gen-6 precedent, or supersededLines them in a future gen), commit the four dirty repos after eyeballing the pre-existing AGENTS.md changes, close or amend old I006, and supersede stale fleet-generation memories in Open Brain.

## Resolution

Owner ruled and executed 2026-07-16 (all four rulings on the recommended path):

- **objectstudio** WORKFLOW.md hand-folded gen 6 → 8 (e200c05): version stamp + stage-cursor section inserted; vfb pre-handoff gate and `functional_harness: framebuffer` preserved verbatim. Doctor D4 warn remains its standing state.
- **maipipe** WORKFLOW.md hand-folded gen 7 → 8 (931a52e, on `feat/i021-events-wait-sse` — live Codex build in that repo, single-file pathspec commit): Codex model remap (`gpt-5.6-sol` default, annotated opus fallback) preserved verbatim.
- **Four dirty-sweep repos committed**, sweep content only (`spine update` dry-run confirmed byte-exact gen-8 output first; pre-existing dirt untouched per no-mixing): hbmview 3eae927, home-lab-admin d99769f, moo-clone 35d1849, praxis 892d157 (incl. its untracked machine-owned AGENTS.md).
- **Old [I006](I006-fleet-sweep-memory-supersede.md) closed as subsumed**; stale Open Brain fleet-generation memories superseded in a dated writeback.

Fleet state after closure: 17/17 repos at gen 8, zero sweep residue. Nothing pushed — pushes stay owner-gated. Story 11 fully closed.
