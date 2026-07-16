---
id: I027
title: Derivation polish batch — M4, M9, M11
severity: low
status: open
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
