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
   (Pre-existing mechanism; the commit adds no primary-row test for it.
   Evidence: the I128 live repro on 2026-09-02, recorded in that design.)
7. The gen-13 to 14 lock admits the primary mirror-row change only when
   the update report itemizes it as a refresh; the gen-9 to 10 and 11 to
   12 locks admit it through the static text allowlist, without an
   itemization check. (I128 later coupled all ten locks to their reports.)
8. Spine's own WORKFLOW.md is refreshed through the real update path, not
   by hand. (Evidence: the one-line WORKFLOW.md diff in the commit and the
   session handoff; the commit adds no self-repo byte-stability test.)
9. The gen-13 model-table fixture mirrors `models/defaults.json`.
10. The gen-9 to 10 effort migration mints `claude-fable-5-1 @ xhigh` for
    a customized top-level effort, so a legacy repo never mints an override
    on the retired id.
11. `go test ./...` and the maipipe full lane pass. (Evidence: run #80 in
    `docs/handoffs/2026-09-02-fable-5-1-remap-and-ledger-burndown.md`, not
    the commit.)

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

- Embedded-default expectations updated in the model, update, audit, and
  CLI tests. The template test was not touched: its substring assertion on
  `claude-fable-5` passed vacuously against the new id until I128 replaced
  it with an exact-row check.
- A focused pair-aware history test for the primary row (shipped pair
  inherited, unshipped low pair override).
- The gen-13 to 14 lock couples the primary row change to the report's
  itemized refresh; the gen-9 to 10 and 11 to 12 locks were extended with a
  text allowlist entry for the row's old and new values (I128 replaced the
  allowlist with report-coupled checks in all ten locks).
- Full suite and the maipipe full lane (run #80) at 68aa28f, recorded in
  the session handoff.

## Out of Scope

- Any other tier or harness.
- The rollout hazards above (I128).
- Template prose or structure.

## Spec-review record

Reviewed under I128 on 2026-09-03 (primary, blind) against the first draft
of this design. The review found the draft overstated the commit: it
claimed template-test coverage and itemization-coupled locks for gen 9-10
and 11-12 that 68aa28f did not have, omitted the gen 11-12 blanket skip and
the gen 9-10 minted-override wording, and carried two wording mismatches.
The user stories and testing decisions above were corrected to what the
commit evidences; the remap checklist was confirmed accurate on all five
items. Full record in I128's Resolution.
