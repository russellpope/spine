---
id: I004
title: /to-tickets skill — assignment step + tier vocabulary
severity: med
status: fixed
affects: []
blocked-by: []
parent: [I001]
labels: [ready-for-agent]
execution-mode: subagent-driven
tier: routine
effort: medium
risk-triggers: []
review-tier: routine
---

## Parent

I001 — implements the /to-tickets half of the declared-intent layer of
`docs/specs/2026-07-09-model-routing-design.md`.

## What to build

A /to-tickets run annotates every ticket it emits with execution-mode,
tier, effort override (when deviating from the tier default), risk
triggers, and review-tier, using the CONTEXT.md glossary vocabulary. The
legacy Model field vocabulary (opus | sonnet | haiku) is retired in favor
of tier names. Where the work's shape demands parallel orchestration
(unknown-size discovery, cross-cutting audits, N-perspective verification),
the skill recommends ultracode — the owner's ticket approval is the opt-in.

## Acceptance criteria

- [ ] Skill instructions require the five annotations per ticket, defined by reference to the glossary
- [ ] Legacy opus/sonnet/haiku vocabulary removed from the skill's templates
- [ ] Ultracode recommendation criteria stated (shape-based, per D2)
- [ ] Both publish forms (local tickets file and real tracker) carry the annotations

## Blocked by

None — can start immediately.
