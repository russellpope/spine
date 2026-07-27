---
id: I049
title: Codex discovery pruning — cheap pre-filter before full JSONL parse
severity: med
status: open
affects: [audit, I009]
blocked-by: [I041]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Filed at I041 review (finding I2): I009's verified facts say "Discovery
needs date/token pruning, not a full-dir parse" (951+ session files; the
I041 reader's full walk measured ~12–14s live), and no ticket owned it —
the I041 report's claim that I043 owns it was checked and found false.

Add a cheap pre-filter to codex session discovery so the audit does not
fully parse files that cannot contribute evidence. Sound basis: every
evidence path (spawn_agent task_name, team spawn command text, opening
user message) requires some audited ticket token to appear as a literal
byte string in the file, so a raw-bytes token scan is a safe pre-filter.
Any pre-filter error falls back to the full parse (degrade-never-fail);
pruning must be invisible in Report content — wall time only.

## Acceptance criteria

- [ ] Discovery pre-filters files (raw token scan or equivalent) before JSONL parse
- [ ] All audit fixtures produce byte-identical Reports with pruning on
- [ ] Pre-filter failure degrades to full parse, never an error
- [ ] Live store walk time drops materially (informal measurement quoted in report)
- [ ] `go test ./...` green

## Blocked by

- I041
