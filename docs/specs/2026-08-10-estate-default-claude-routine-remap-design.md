# Estate default Claude routine remap

Status: ready-for-agent
Date: 2026-08-10 (owner-ratified in I063)
Ticket: I063

## Problem Statement

Spine's embedded estate model table still ships `claude.routine` as
`claude-sonnet-5` at the routine tier's default medium effort, although the
owner permanently banned that model and selected `claude-opus-5 @ low` as its
replacement. New scaffolds therefore emit a banned default, and existing
repositories that still carry the inherited Sonnet row cannot refresh until
the table records both the new current pair and the old shipped pair.

Spine's own `WORKFLOW.md` already carries `claude-opus-5 @ low`. That row must
remain byte-stable under `spine update`; unrelated repository overrides must
also remain untouched.

## Solution

Change only the `claude.routine` entry in the embedded model table:

- current pair: `claude-opus-5` with explicit effort `low`;
- current aliases: the Opus 5 exact id and `opus` shorthand;
- shipped history: `claude-sonnet-5` with omitted effort, preserving the pair
  as it actually shipped at the routine default effort (`medium`).

The existing resolver and update path then do the migration: the historical
Sonnet pair resolves as inherited and refreshes to the current Opus pair;
values that match no shipped pair remain overrides. `MirrorRows` renders the
new row as `claude.routine: claude-opus-5 @ low` because low differs from the
routine tier's default effort.

## User Stories and Acceptance Criteria

1. A new scaffold emits `claude.routine: claude-opus-5 @ low` and never emits
   Sonnet 5 as the current routine default.
2. Resolution without repository context returns Claude routine as Opus 5 at
   low effort.
3. A repository carrying the old shipped Sonnet 5 routine row at its shipped
   medium effort is classified as inherited, refreshed to Opus 5 at low, and
   receives an itemized old/new model-refresh entry.
4. A repository carrying an unrelated Claude routine value remains an
   override and is reported rather than overwritten.
5. A repository already carrying `claude-opus-5 @ low`, including spine
   itself, is unchanged by a write update and remains unchanged on a second
   run.
6. `MirrorRows` still covers and round-trips every flavor/tier and explicitly
   renders the low-effort suffix for Claude routine.
7. Historical Sonnet 5 remains discoverable only as an exact historical id;
   current aliases describe Opus 5, not the displaced default.
8. A generation 5-9 repository with a customized top-level `effort:` value
   still migrates that choice to every Claude entry, including routine. The
   new routine default suffix must not cause the customized effort to be
   skipped or discarded; `xhigh` becomes Opus 5 at xhigh and `medium` remains
   a real routine override because the new shipped pair is low.
9. Existing generation-migration fixtures remain historical records. Their
   literal Sonnet rows are not sweep-edited; only sanctioned expected diff
   allowances change if the new table refresh makes that necessary.
10. Focused tests and `go test ./...` pass.

## Implementation Decisions

- **Plain sweep refresh; no generation bump.** The current generation already
  renders `{{MODEL_ROUTING_ROWS}}` from the embedded table, and
  `applyModelRouting` resolves inherited/current/override values independently
  of the on-disk template generation. No template structure or prose changes,
  so `templates/VERSION` remains 10 under ADR 0004.
- **No new ADR.** This uses ADR 0011's accepted table/history and refresh
  mechanism without changing it. The generation decision is recorded here
  and in I063's Resolution.
- **History is pair-aware.** The Sonnet history entry omits effort because it
  shipped at the routine default (`medium`). The new current entry explicitly
  stores `low`; treating history as a bare id at the new effort would wrongly
  classify `claude-sonnet-5 @ low` as inherited.
- **Aliases follow the current default.** Replace the routine entry's Sonnet
  aliases with Opus aliases. Historical recognition remains exact-id-only, as
  already required by the model table contract.
- **Customized legacy effort wins.** The gen 5-9 top-level `effort:` migration
  must replace a table-rendered per-entry suffix as well as add a missing one.
  Compare the migrated effort with the current entry's resolved default pair,
  not merely the tier-wide default. A customized `xhigh` therefore produces
  `claude-opus-5 @ xhigh`, customized `medium` remains an override, and an
  inherited/default repository still receives `claude-opus-5 @ low`.
- **No estate sweep in this build.** The binary change makes normal
  `spine update` sweeps correct. Installing the binary and sweeping other
  repositories remain owner deployment actions.

## Testing Decisions

- Update embedded-default expectations and add a focused historical-pair test
  for the old routine row.
- Strengthen the MirrorRows round-trip coverage with an explicit Claude
  routine row assertion.
- Add update coverage at generation 10 for inherited refresh/itemization,
  unrelated override preservation, and already-current idempotence.
- Retain and strengthen gen 5-9 customized-effort coverage so a table default
  that already contains `@ low` is replaced by the migrated choice rather than
  silently bypassing it.
- Keep legacy fixtures unchanged. If strict migration diff locks observe the
  table-driven routine refresh, extend only the centralized sanctioned
  model-refresh diff allowlist.
- Run focused model/update tests, the full Go suite, formatting/diff checks,
  doctor, stage audit at the expected cursor state, and routing audit.

## Out of Scope

- Changing tier-default effort values.
- Changing any Claude tier other than routine.
- Editing historical tickets, handoffs, or migration fixtures merely because
  they contain `claude-sonnet-5`.
- Template prose/structure changes or a template-generation migration.
- Installing the built binary, sweeping the wider estate, pushing commits, or
  modifying `PICKUP.md`.
