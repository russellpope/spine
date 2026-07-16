---
id: I028
title: Story-11 closure — objectstudio/maipipe WORKFLOW reconciliation, 4 uncommitted sweep repos, old I006
severity: med
status: open
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
