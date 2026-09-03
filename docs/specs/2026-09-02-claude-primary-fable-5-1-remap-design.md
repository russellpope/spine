# Estate default Claude primary remap to Fable 5.1

Status: shipped (retroactive record, written 2026-09-03 under I128)
Date: 2026-09-02
Commit: 68aa28f
Ticket: none at ship time; recorded by I128
Precedent: `docs/specs/2026-08-10-estate-default-claude-routine-remap-design.md` (I063)

## Problem Statement

Anthropic released Claude Fable 5.1 (`claude-fable-5-1`) as the successor
to Fable 5. Spine's embedded model table still shipped `claude.primary` as
`claude-fable-5`, so every new scaffold and every inherited fleet mirror
kept dispatching primary-tier work to the superseded model, and the
`fable` alias resolved to it.

## Solution

Change only the `claude.primary` entry in the embedded model table:

- current id `claude-fable-5-1` at the tier's default effort (no explicit
  effort, as before);
- current aliases: the exact new id and the `fable` shorthand;
- history: `claude-fable-5` with omitted effort, the pair as it shipped.

The existing resolver and update path do the migration: an inherited
`claude-fable-5` mirror at the default effort is classified as inherited
and refreshed to `claude-fable-5-1` with an itemized old/new entry; the same
id at another effort is an override and is preserved (I063's pair-aware
history rule, unchanged).

## User Stories and Acceptance Criteria

1. A new scaffold emits `claude.primary: claude-fable-5-1`.
2. Resolution without repository context returns Claude primary as
   `claude-fable-5-1` at the tier default effort.
3. A repository mirroring `claude-fable-5` at the default effort is
   inherited, refreshed, and itemized.
4. A repository mirroring `claude-fable-5 @ low` remains an override.
5. `claude-fable-5` is discoverable only as an exact historical id; the
   `fable` alias describes the new id.
6. Launch validation refuses `claude-fable-5` as `retired-model` for a
   mirror that still carries it, and refuses `--expect claude-fable-5`.
7. Generation locks admit the primary mirror-row change only when the
   update report itemizes it as a refresh.
8. Spine's own WORKFLOW.md is refreshed through the real update path, not
   by hand, and is byte-stable on a second run.
9. The gen-13 model-table fixture mirrors `models/defaults.json`.
10. `go test ./...` and the maipipe full lane pass.

## Implementation Decisions

- **Plain sweep refresh; no generation bump.** As I063: the mirror renders
  from the table and update refreshes independently of generation.
  `templates/VERSION` stays 14; no ADR.
- **History is pair-aware.** The old id ships in history with omitted
  effort; the current entry carries no effort either, so the tier default
  governs both sides.
- **Locks sanction, not rewrite.** Captured generation fixtures keep their
  literal `claude-fable-5` rows; the sanctioned-row allowlist in the update
  test package admits the primary row's old and new values.
- **Fleet rollout is a deploy action** outside the commit: install both
  binaries, then `spine update --write` per repo.

## Remap checklist (the part 68aa28f did not do)

Recorded here so the next remap has it. Each item is a rollout hazard
I128 found after the fact:

1. Sweep every fleet mirror, or accept that launch validation refuses
   dispatch on each unrefreshed one until it is swept.
2. Overrides pinning the retired id at another effort need the
   retired-override migration (I128) or a hand edit.
3. Host routing configs match ids byte-exactly; add the new id to each
   host file's model list.
4. Render locks must pin the exact new id, not a substring of it.
5. Team-skill preflights relay validate's refusal, so operators see the
   update remedy rather than a rebuild instruction.

## Testing Decisions

- Embedded-default expectations updated in the model, template, update,
  audit, and CLI tests.
- A focused pair-aware history test for the primary row (shipped pair
  inherited, unshipped low pair override).
- Generation locks (9 to 10, 11 to 12, 13 to 14) extended so the primary
  row change is admitted only as an itemized refresh.
- Full suite and the maipipe full lane (run #80) at 68aa28f.

## Out of Scope

- Any other tier or harness.
- The rollout hazards above (I128).
- Template prose or structure.

## Spec-review record

Reviewed under I128 on 2026-09-03 against this design; result recorded in
I128's Resolution.
