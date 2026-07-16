---
id: I027
title: Derivation polish batch — M4, M9, M11
severity: low
status: fixed
affects: [audit, update, template, skills]
blocked-by: []
execution-mode: subagent-driven
tier: mechanical
review-tier: routine
---

## Problem

Accumulated Minors from the gen-8 build triaged to follow-up: (M4) deriveHandoff conflates an I/O error on docs/handoffs with "no handoffs exist" in the finding detail; (M9) gen8ContentLines doc-block hand-edit detection asymmetry deserves a one-line maintainer note; (M11) the doctor-advises half of the I014 backstop is unstated in both the gen-8 template handoff-rule line and the /handoff skill section.

## Fix

Three small edits: split the deriveHandoff detail message on error vs absent; comment near gen8ContentLines; one clause in the template section + /handoff skill (template text change rides the next generation bump — do not bump for this alone; pair with I026's).

## Resolution

Fixed 2026-07-16 (derivation-polish batch, main @ fdad11c, 89d4a07): M4 deriveHandoff detail splits I/O-error from no-handoffs; M9 maintainer note on the gen8ContentLines asymmetry; M11 doctor-advises clause added to the gen-9 template handoff-rule line (riding I026's bump, no extra generation) and the user-level /handoff skill (also synced to "missing/stale" + gen 9 in the fix wave).
