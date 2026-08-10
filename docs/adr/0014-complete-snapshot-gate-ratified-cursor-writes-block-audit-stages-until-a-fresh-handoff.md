---
id: "0014"
title: "complete-snapshot gate ratified: cursor writes block audit stages until a fresh handoff"
status: Accepted
date: 2026-08-10
---

# 0014: complete-snapshot gate ratified: cursor writes block audit stages until a fresh handoff

## Context

The cursor-writes effort's final-review correction (I059 canonical-form gate,
PRD `docs/specs/2026-08-06-cursor-writes-design.md`) made `spine audit stages`
compare the full cursor snapshot in the newest handoff against live cursor
state. Consequence: ANY cursor write (`spine cursor start/tick/here/set`)
blocks the audit until a fresh `spine handoff new` embeds a matching snapshot.
Flagged to the owner 2026-08-07 as possibly too strict mid-effort — a
one-ticket loosening was on the table (estate-maintenance pickup handoff,
next-step 2). The I062 build ran through the gate end-to-end on 2026-08-09
without friction: the verifier accounted for the expected mid-effort block,
and cutting the shipped handoff cleared it.

## Decision

Owner ratified the gate as-is on 2026-08-10: mid-effort cursor writes
blocking `audit stages` until a fresh handoff snapshot exists is intended
behavior, not a defect. No loosening ticket will be filed.

## Consequences

- The block-until-fresh-handoff state during an effort is expected and must
  be accounted for (not "fixed") by verifiers and successor sessions.
- Handoff docs and skills should keep stating it as normal rhythm.
- Reversing this requires a new ADR superseding this one.
