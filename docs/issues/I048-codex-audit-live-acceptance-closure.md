---
id: I048
title: Codex audit — live acceptance against the estate, close I008/I009
severity: high
status: open
affects: [audit, I008, I009]
blocked-by: [I041, I042, I043, I044, I045, I046, I047, I049]
execution-mode: inline
tier: primary
effort: xhigh
risk-triggers: []
review-tier: n/a
---

## What to build

Design "Live acceptance" + calibration notes. Run the finished audit against
the real session store and the real estate — this is the step that validates
synthetic fixtures against the undocumented format. Inline justification:
requires the live ~/.codex session store and operator-machine git state; no
per-task review cycle, verify-stage gates apply.

Expected honest outcomes (from the 2026-07-25 ground-truth investigation):

- moo-clone I024 → match (lead sol excluded, workers terra attributed)
- moo-clone M4a I008–I015 → unmapped-dispatch (gpt-5.5 was never a declared
  id; history reported honestly, non-blocking)
- moo-clone I021/I022 → match (sol on primary tickets)
- guardian threads contribute nothing anywhere
- praxis and maipipe audits unaffected by moo-clone's tokens and vice versa
- `tier: n/a` applied to moo-clone's pre-convention tickets kills the
  unannotated noise; empty-tier tickets stay loud

Close I008 and I009 with evidence (audit output quoted in the tickets),
update the issue ledger, and record any format surprise the live run
exposes as a new dated fact in I009 before closing.

## Acceptance criteria

- [ ] Live run on moo-clone matches every expected outcome above, or each deviation is explained and ratified
- [ ] Live run on praxis and maipipe shows no moo-clone contamination
- [ ] A deliberately mis-scoped run (praxis tokens vs moo-clone) produces no false blocking verdict
- [ ] I008 and I009 closed with quoted evidence; ledger updated
- [ ] Fleet handoff notes the new flags and verdicts for operators

## Blocked by

- I041, I042, I043, I044, I045, I046, I047
