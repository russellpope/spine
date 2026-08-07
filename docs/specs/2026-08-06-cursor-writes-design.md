# Cursor writes — spine as sole writer of the stage cursor

Status: ready-for-agent
Date: 2026-08-06 (grilled same day; decisions owner-ratified in-session)
Glossary: CONTEXT.md "Stage cursor" section (added this grill)

## Problem Statement

Sessions get lost adjusting the stage cursor. The cursor is the single source
of truth for "where are we," but every mutation — ticking a stage, moving the
here-marker, starting a new effort's block, copying the block into a handoff —
is a hand edit of markdown performed by a model, usually under context
pressure at exactly the moments (stage transitions, handoffs) when attention
is scarcest. The parser and audit catch the resulting malformations and stale
copies, but only *after* the mistake, at audit/handoff time, when diagnosing
and repairing the cursor burns the very context the cursor exists to protect.
The ADR 0013 defect in the mutation-battery effort (a committed deliverable
carrying superseded wording) was this class: cross-doc staleness from a
hand-performed copy step.

## Solution

Spine becomes the sole legal writer of the cursor (sole-writer rule, glossary).
Four small verbs mutate the working home; `spine handoff new` captures the
committed snapshot automatically; stage-completion claims hit a write-time
tripwire that runs the same derivation rules the audit uses, so mistakes fail
at the moment of the write instead of at handoff. Hand edits become detectable
(canonical form check) and illegal. The model's job reduces to *deciding*
state changes; spine owns *expressing* them.

## User Stories

1. As a session agent, I want `spine cursor tick <stage>` to mark a stage done and advance the here-marker, so that a stage transition is one idempotent command instead of a hand edit with marker bookkeeping.
2. As a session agent, I want `tick` to refuse when the stage's artifacts are absent (per the audit's derivation rules), so that I discover an unearned tick at write time, not at handoff.
3. As a session agent, I want the refusal text to be the same finding text `spine audit stages` would emit, so that I learn one vocabulary of failure.
4. As an effort owner, I want `--force` on `tick`, so that a legitimately unconventional stage completion is never hard-blocked — while the audit remains the final authority.
5. As a session agent, I want `spine cursor start --effort <name>` to seed a canonical all-pending block with the marker on the first stage, so that new efforts begin from a well-formed cursor with zero transcription.
6. As an effort owner, I want `start` to refuse over a mid-flight cursor (with `--force` to supersede an abandoned effort), so that a confused session cannot silently destroy the record of a half-done effort.
7. As a session agent, I want `--prd` and `--tickets` prefill flags on `start`, so that fields known at effort start are captured immediately.
8. As a session agent, I want `spine cursor set --prd <path>` / `--tickets <range>`, so that mid-effort scope changes (a ticket added, a PRD renamed) are field edits, not block rewrites.
9. As a session agent, I want `spine cursor here <stage>` to place the marker explicitly, so that non-linear moves (review kicks work back to implement) have a first-class expression.
10. As a session agent, I want `here` on a done stage to revert it to current, so that regression is expressible without a separate untick verb.
11. As a handoff author, I want `spine handoff new` to embed the current cursor block automatically, so that the committed snapshot can never go stale against the working home by transcription error.
12. As a handoff author, I want `handoff new` without a cursor present to scaffold with a note rather than fail, so that wayfinder efforts and pre-cursor repos keep working.
13. As an auditor, I want `audit stages` to block on a valid-but-non-canonical cursor block, so that hand edits — now illegal — are detected and named.
14. As a repo maintainer, I want `doctor` to advise (not block) on the same non-canonical condition, so that health checks stay readable while the gate stays in the audit.
15. As a session agent, I want any verb (or a no-op `set`) to rewrite the block canonically, so that recovering from a flagged hand edit is trivial.
16. As a resuming session, I want the bare `spine cursor` read command and the SessionStart hook unchanged, so that resume ergonomics are untouched by the write path.
17. As a fleet repo owner, I want the gen 10 WORKFLOW.md text to state the sole-writer rule and the automatic-embed rule, replacing the "embed the verbatim output" instruction, so that no repo declares two contradictory procedures.
18. As a fleet repo owner, I want the full-estate sweep in this effort, so that no repo half-believes hand-editing is still legal.
19. As a skill maintainer, I want the handoff and handoff-to-codex skills updated in the same effort, so that the last hand-writing instructions die the day the rule lands.
20. As a subagent implementer, I want the verbs to write directly (no dry-run mode), so that a stage transition is not a two-step ceremony.
21. As a future collaborator, I want the cursor vocabulary in CONTEXT.md and the grammar unchanged, so that existing repos need no migration and humans and tools keep speaking one language.
22. As an effort owner, I want a forced tick to still fail the later audit if artifacts never appear, so that `--force` defers the reckoning and never waives it.

## Implementation Decisions

- **Sole-writer rule** (owner-ratified): spine is the only legal writer of the
  cursor. Hand-editing the block is a workflow violation. Enforcement is the
  canonical form check, not perfect detection — a byte-canonical hand edit is
  undetectable and acceptably harmless.
- **Verb surface** (owner-ratified): exactly four verbs on the existing
  `cursor` subcommand — `start`, `tick <stage>`, `here <stage>`, `set`.
  No untick (`here` + re-tick covers regression); no stage add/remove (the
  stage list belongs to WORKFLOW.md).
- **Marker semantics**: `tick` of the marker-holding stage advances the
  marker to the next unticked stage, dropping it when none remain (matching
  the shipped-effort shape); `tick` of a non-marker stage leaves the marker
  alone; `here` on a done stage reverts it to current.
- **Write-time tripwire** (owner-ratified): `tick` runs the target stage's
  derivation rule (the audit's own); absent artifacts → non-zero exit with
  the audit's finding text, no write. `--force` writes anyway. Stages with no
  derivation rule tick freely. The audit remains the single authority; the
  tripwire is an early copy of it, never a second opinion.
- **Dual-home resolution** (owner-ratified): verbs write the working home
  only. `spine handoff new` embeds the current parsed block into the file it
  creates. Committed snapshots are historical and never retro-mutated. The
  existing audit check (newest snapshot matches working home) is the
  consistency gate between homes.
- **`start` guard** (owner-ratified): refuses while any stage in the existing
  block is unticked; `--force` supersedes. Seeds: all stages pending, marker
  on first stage, `prd:`/`tickets:` empty (legal early-stage values, per the
  existing grammar and live precedent), prefill flags accepted.
- **Canonical form enforcement** (owner-ratified): parse → re-serialize →
  diff. Mismatch blocks `audit stages`, advises in `doctor` — the same
  posture as malformed-cursor findings (2026-07-16 amendment).
- **Gen 10 template bump + full fleet sweep** (owner-ratified): the
  machine-owned WORKFLOW.md stage-cursor section gains the sole-writer rule
  and verb references, and **supersedes** (not augments) the "MUST embed the
  verbatim output of `spine cursor`" handoff-rule line — flagged at grill as
  a contradiction if left alongside. Sweep precedent: gen 8 (I015), gen 9
  (I029).
- **Deepthought skills** (owner-ratified): `handoff` and `handoff-to-codex`
  (the only two skills referencing the cursor, verified by sweep) change to:
  embed happens automatically via `spine handoff new`; mutate only via the
  verbs.
- **Grammar and read path unchanged**: no cursor format change, no migration;
  bare `spine cursor`, `--quiet`, and the SessionStart hook untouched.
- **Direct writes**: verbs write without a dry-run stage — precedent `adr
  new` / `handoff new`; dry-run-by-default is the convention for bulk
  rewrites (`update`/`adopt`), not single-block mutations.

## Testing Decisions

- External behavior only: command exit codes, printed findings, and resulting
  file bytes — never parser/serializer internals. (Carve-out for documented
  internal heuristics exists from 2026-07-16 but is not expected to be
  needed here.)
- **Zero new seams.** All four verbs test at the CLI command boundary against
  fixture-repo trees (prior art: `internal/stages`, `internal/audit`,
  `internal/doctor` testdata patterns): tick with/without artifacts;
  `--force`; marker advance/drop/stay; `here`-on-done regression; `start`
  guard and `--force` supersede; `set` field edits; canonical rewrite of a
  messy-but-valid block; refusal texts matching audit finding texts.
- Handoff embed at the existing `handoff new` command seam: fixture with a
  cursor → created file carries the block verbatim; fixture without → note,
  exit zero.
- Canonical form check as new fixtures in the existing audit/doctor testdata
  style: valid-but-non-canonical block → audit blocks, doctor advises.
- Gen 9→10 in the per-generation update seam (prior art:
  `internal/update` genNtoN+1 tests), proving the superseded handoff-rule
  line is replaced, not duplicated.
- Skills edits and fleet sweep: no code seam; live verification per estate
  convention (run the real commands in a real repo before claiming done).

## Out of Scope

- Any cursor grammar change or migration of existing cursor blocks.
- Derivation rules for stages that currently lack them (grill,
  functional-test, review, verify, ship, deploy, docs, handoff-completion) —
  the tripwire only mirrors rules the audit already has.
- Auto-ticking stages from observed artifacts (spine deciding state, not just
  expressing it) — the model decides, spine writes.
- Changes to `audit routing`, the ESCALATION grammar, or the model table.
- Wayfinder ledger conventions beyond the existing "cursor governs SDD build
  efforts" note.
- Retro-editing committed snapshots in prior handoffs, under any flag.

## Further Notes

- The `internal/cursor` package already carries parse and canonical
  serialization (`StagesLine`); the writer is parse → mutate → re-serialize →
  splice, plus a thin command layer.
- Empty `prd:`/`tickets:` are deliberately legal (live precedent: the
  mutation-battery grill-entry snapshot) — `start` relies on this.
- The wayfinder note stands: a wayfinder map reaching its to-spec handoff
  starts the cursor at grill-done, prd-current — expressible as `start` +
  `tick grill`.
- Sequencing note for the plan: land the verbs and embed before the gen 10
  text that mandates them, and update the skills in the same window as the
  sweep — the estate should never instruct a procedure its tooling can't yet
  perform.
- Process-note carryover from mutation-battery applies here: after any
  owner-ratified amendment mid-build, grep completed deliverables for
  superseded wording (the gen 10 supersession line is itself this lesson
  applied).
