---
id: I051
title: "Fail-closed pre-dispatch model validation: forbidden tokens, unmapped-route refusal"
severity: med
status: open
affects: [model, skills-preflight]
blocked-by: []
execution-mode: subagent-driven
tier: primary
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## Provenance (captured 2026-08-03)

Feature-mined from agentflow's model policy — see the evaluation at
`maipipe:docs/research/2026-08-03-agentflow-steal-list.md`. Agentflow enforces
model policy **before launch**, fail-closed: exact `(provider, model, role,
effort)` routes, plus an explicit forbidden-tokens denylist (`auto`, bare
`opus`/`sonnet`/`haiku`, vendor "auto" tiers) that blocks a launch even when a
human typed the model directly. Nothing unlisted spawns.

## Problem

Spine's enforcement is post-hoc: `spine audit routing` catches silent descent
from transcripts *after* the tokens are burned. I038 put `spine model <flavor>
<tier>` in the team-skill dispatch path and added a presence check plus a
grep-style regression test against literal model ids — but nothing at dispatch
time refuses a bad resolution: a hand-edited mirror override containing a
retired id, a bare-token model (`opus`, `auto`) that a vendor CLI would
resolve unpredictably, or a dispatch that bypasses `spine model` entirely and
is only caught days later by audit (maipipe I060 being the live example).

## What to build

- A validation verb — shape decided at PRD time, e.g.
  `spine model validate <flavor> <tier> <model-id>` or a `--validate` mode —
  that exits non-zero when the id is not the resolved route for that key
  (inherited or override) or matches a forbidden-token denylist.
- The denylist ships in `models/defaults.json` alongside the table (bare-token
  and vendor-auto patterns; content decided at PRD time).
- The shared team-skill preflight calls it fail-closed before every spawn:
  no spawn on refusal, error names the key and the offending id.
- `spine audit routing` and this check stay one vocabulary: a dispatch that
  passed pre-validation should never be `unmapped-dispatch` at audit for the
  same reason (link the verdict definitions).

## Open questions for the PRD grill

- Does validate read the repo mirror, the embedded table, or both (override
  provenance says both — decide precedence wording)?
- Should refusal have an escape hatch, and if so does it reuse the ESCALATION
  record grammar rather than a flag? (Leaning: yes, a recorded escalation is
  the only bypass — consistent with silent-descent being the crime.)
- Effort validation in scope or a follow-up?

## Acceptance criteria (sharpened at PRD time)

- [ ] Valid route passes; retired-default id, unknown id, and each
      forbidden-token class refuse with a named reason (negative controls).
- [ ] Team-skill preflight blocks a spawn on refusal end-to-end.
- [ ] Regression test: the preflight path cannot spawn with a literal model id
      that skipped validation.
