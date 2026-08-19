---
id: I081
title: "Handoff embeds newest checkpoint; doctor advisory for checkpoints"
severity: med
status: fixed
affects: [handoff, doctor, checkpoint]
blocked-by: [I080]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md`
(Checkpoint — handoff and doctor decisions).

## What to build

A session that ends with checkpoints on disk keeps its forward intent in
the committed handoff: `spine handoff new` embeds the newest checkpoint
after the cursor block whenever the working home is non-empty — the facts
region verbatim, the model region under a fixed heading "Prior narrative
(model-authored, not evidence)". `spine doctor` gains one advisory finding
covering: malformed facts region, non-canonical (byte-drifted) facts region,
ordinal gaps in the working home, and `.superpowers/` not being gitignored.
Advisory only — no `audit stages` change in this ticket.

## Acceptance criteria

- [ ] `handoff new` with checkpoints present embeds the newest one in the
      spec'd shape; without checkpoints the handoff is unchanged (negative
      control).
- [ ] Doctor fires on a hand-mutated facts block and on an ordinal gap;
      does not fire on canonical checkpoints (negative control).
- [ ] Doctor advises when `.superpowers/` is unignored; silent when ignored.
- [ ] Tests at the CLI seam per the spec.

## Blocked by

- I080 (checkpoint document + commands).

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commit 3444832).
`spine handoff new` embeds the newest checkpoint after the cursor block when
the working home is non-empty (`internal/handoff/checkpoint.go`): heading
`## Checkpoint (newest): NNN-<slug>.md`, facts region with its markers
(interior bytes unchanged), then `### Prior narrative (model-authored, not
evidence)` + model region; facts-only → "narrative: missing — reconstruct
intent from facts"; a malformed newest checkpoint never fails the handoff;
without checkpoints the handoff is unchanged. Doctor D11 (advisory, `warn`,
no audit-stages effect; `internal/doctor/checkpoint.go`): malformed or
non-canonical facts region per file, ordinal gaps, `.superpowers/` not
gitignored (only when it exists; silent outside a git repo). Negative
controls at the CLI seam. Ruling: the spec's two heading spellings resolve to
the Implementation Decisions form. Review clean, no fix round.
