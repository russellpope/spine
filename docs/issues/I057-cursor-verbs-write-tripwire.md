---
id: I057
title: "Cursor writes: four verbs + write-time tripwire"
severity: med
status: open
affects: [cli, cursor]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## Parent

PRD: docs/specs/2026-08-06-cursor-writes-design.md (all decisions owner-ratified
2026-08-06 grill). Glossary: CONTEXT.md "Stage cursor" section.

## What to build

The sole-writer rule's tooling: `spine cursor start | tick <stage> |
here <stage> | set` mutating the cursor's working home canonically, so a
session never hand-edits the block. Demoable by driving a fixture effort
through its whole lifecycle from the command line.

- `start --effort <name> [--prd <path>] [--tickets <range>]`: seeds all-pending
  stages, marker on first stage, empty `prd:`/`tickets:` unless prefilled.
  Refuses (non-zero) while any stage in the existing block is unticked;
  `--force` supersedes an abandoned effort.
- `tick <stage>`: marks done. Runs the stage's derivation rule (the audit's
  own): absent artifacts → non-zero exit with the audit's finding text, no
  write; `--force` writes anyway (the audit remains the authority — a forced
  tick still fails it later). Stages with no derivation rule tick freely.
  Marker semantics: ticking the marker-holder advances `[<]` to the next
  unticked stage, dropped when none remain; ticking elsewhere leaves it.
- `here <stage>`: places the marker explicitly; on a done stage reverts it to
  current (this is the regression path — there is no untick verb).
- `set --prd <path> | --tickets <range>`: field edits; a no-op `set`
  normalizes a block to canonical form.
- All verbs re-serialize the whole block canonically on every write. Bare
  `spine cursor`, `--quiet`, and the SessionStart hook are untouched. No
  grammar change; empty `prd:`/`tickets:` stay legal. Direct writes — no
  dry-run mode (precedent: adr new / handoff new).

## Acceptance criteria

- [ ] Fixture-repo tests at the CLI command boundary (stages/audit/doctor
      testdata style): exit codes, printed findings, resulting file bytes only
- [ ] `tick` without artifacts refuses with text identical to the
      `audit stages` finding for that stage; `--force` writes
- [ ] Marker advance / drop-when-complete / stay-on-non-marker-tick and
      `here`-on-done regression all covered
- [ ] `start` guard: refuses mid-flight, `--force` supersedes; seeded block
      is canonical and parses clean
- [ ] A messy-but-valid hand-written block is rewritten canonically by a
      no-op `set`
- [ ] `go test ./...` green

## Blocked by

- None — can start immediately.
