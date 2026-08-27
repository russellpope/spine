---
id: I114
title: "Cursor tickets: grammar cannot express a two-ticket batch — add a comma-list form"
severity: low
status: open
affects: [I026]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Found 2026-08-27 starting the doctor-hygiene batch (I065 + I106), the first
two-ticket effort. The cursor `tickets:` grammar resolves three forms (I026):
a bare id, an inclusive numeric range, and `prefix <str>`. A batch of two
non-adjacent tickets fits none of them:

- `I065,I106` fails to resolve — `spine cursor` and `spine audit stages` emit
  the unresolvable-tickets note and degrade the issues/implement evidence
  rules to not-judged for the whole effort.
- `I065-I106` resolves, but to all 42 intermediate ids, so the evidence rules
  would judge the effort against ~40 tickets it never touched.

The degradation is visible (the note names the bad value) but real: the first
multi-ticket batch runs with its ticket evidence unjudged.

## Fix

Add a comma-list form (`I0NN,I0MM[,...]`, each element a bare id) to the
grammar in `internal/cursor` (cursor.Grammar and the package doc) and to
`resolveTicketIDs` in `internal/stages`. Update the unresolvable-tickets note
text to name the new form. Tests follow the I026 pattern: a two-element list
resolves to exactly those ids; a list with a malformed element stays
unresolvable as a whole (no partial resolution).
