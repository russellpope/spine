---
id: I059
title: "Cursor writes: canonical-form gate in audit stages + doctor advisory"
severity: med
status: open
affects: [cli, audit, doctor, cursor]
blocked-by: [I057]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## Parent

PRD: docs/specs/2026-08-06-cursor-writes-design.md. Glossary: CONTEXT.md
"Stage cursor" (canonical form, sole-writer rule).

## What to build

Enforcement teeth for the sole-writer rule: a valid-but-non-canonical cursor
block is evidence of an illegal hand edit. Parse → re-serialize → diff; on
mismatch `spine audit stages` blocks and `spine doctor` advises — the same
posture as malformed-cursor findings (2026-07-16 amendment). The finding text
names the remediation: any cursor verb (or a no-op `set`) rewrites the block
canonically — which is why this ticket sits behind I057.

## Acceptance criteria

- [ ] New fixtures in the existing audit/doctor testdata style:
      valid-but-non-canonical block → audit blocking finding, doctor advisory
- [ ] Canonical block (as any I057 verb emits) passes both untouched
- [ ] Malformed-block behavior unchanged (still blocking, distinct finding)
- [ ] Finding text points at the verb-based remediation
- [ ] `go test ./...` green

## Blocked by

- [I057] — the remediation path the finding names must exist before the gate
  can flag.
