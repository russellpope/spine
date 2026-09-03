# Fable 5.1 remap rollout: retired mirrors refuse dispatch

Status: ready-for-agent
Date: 2026-09-02
Ticket: I128
Precedent: `docs/specs/2026-08-10-estate-default-claude-routine-remap-design.md` (I063),
`docs/specs/2026-09-02-claude-primary-fable-5-1-remap-design.md` (68aa28f, retroactive)

## Problem Statement

Commit 68aa28f moved the estate's `claude.primary` default from
`claude-fable-5` to `claude-fable-5-1` and put the old id in the row's
history. That is the correct model-table shape (ADR 0011), and `spine update`
refreshes an inherited mirror correctly. But the rollout has four gaps, each
reproduced against the HEAD binary on 2026-09-02:

1. Every fleet checkout whose WORKFLOW.md still mirrors `claude-fable-5`
   (14 primary checkouts and 8 worktrees under `~/Projects/github.com`) fails
   `spine model validate claude primary` with exit 1 `retired-model`. The
   claude-team and codex-team skills run that command as their dispatch
   preflight with stderr discarded and, on any non-zero exit, tell the
   operator to rebuild spine. `spine doctor` on such a repo says only D2
   "behind template generation".
2. A mirror pinning the retired id at a non-default effort
   (`claude.primary: claude-fable-5 @ xhigh`, which the gen-9 to 10 effort
   migration minted) is an override. `spine update --write` preserves it
   verbatim, validate refuses it as retired, and the refusal's only remedy is
   the update that changes nothing.
3. Host routing configs match the resolved id byte-exactly. A host file that
   lists only `claude-fable-5` makes every refreshed repo's primary tier
   unreachable, and doctor's D16 message names no remedy. The I072 fixtures
   still list only the old id.
4. Several render locks assert a substring of the new id, so they pass with
   the table reverted. The gen-13 to 14 lock checks "row changed implies
   itemized" but not the converse, the sanctioned-row allowlist has no
   negative control, and ten per-generation locks each carry a copy of the
   same skip block over a static allowlist rather than their own itemized
   refreshes. `modelDefaultDivergence` flags a gen-0 `model_default:` set to
   any historical primary id as a divergence.

## Solution

Split by urgency, all in one effort:

1. **Rollout.** A new doctor finding, D18, names a retired mirrored id per
   (harness, tier) with the update remedy. The team-skill preflights in
   deepthought distinguish "spine missing or too old" (command absent or exit
   2) from a policy refusal (exit 1) and relay validate's stderr verbatim,
   which already names `spine update --dir R --write`. The fleet sweep runs
   at deploy with the shipped binary.
2. **Retired override.** `spine update` migrates an override whose id is a
   historical id of its harness to that id's successor, keeping the effort
   and alternate the repo chose, and itemizes it as a refresh of kind
   "retired override". The validate remedy is then correct as printed.
3. **Host config.** Byte-exact matching stays. D16's unreachable message
   gains the host-file remedy and, when the host lists a historical id of
   the requested lineage, says so. Fixtures carry the current id. The remap
   checklist (retroactive PRD) names host configs as a sweep target.
4. **Tests.** Exact-row locks; the converse assertion; a negative control on
   the sanctioned-row helper; one shared helper for the ten skip blocks that
   admits a mirror-row diff line only when that lock's own report itemizes
   it; `model_default:` retires quietly for any historical primary id.

## Grill record

Self-answered from the ticket, the code, and live repros; every answer the
ticket did not settle is marked **assumption** for the owner to challenge.

- Q1 Sweep vs per-repo. Fact: the 14 stale primary checkouts all have a
  clean WORKFLOW.md; 8 have other uncommitted work; 8 stale checkouts are
  worktrees on feature branches. **Assumption:** write the refresh in the 14
  primary checkouts, commit nothing in other repos (the maipipe and
  maikanban precedent), skip worktrees. The handoff lists what was written.
- Q2 Preflight fix location. Fact: the blocks live in deepthought
  (`skills/claude-team`, `skills/codex-team`), guarded by
  `skills/lib/test-model-validation-preflight.sh`. **Assumption:** one
  deepthought commit referencing spine I128, extending that script with the
  refusal arm; no deepthought ticket.
- Q3 Doctor code. Fact: D17 is taken (pin evidence). D18, severity warn
  (doctor already exits 1 on warn), one finding per retired (harness, tier),
  message naming the id, its successor, and the update remedy. D2 still
  fires; it is true.
- Q4 Stuck override. Options: migrate keeping effort, or tell the operator
  to edit. Chosen: migrate. Launch policy already forbids every historical
  id byte-exactly, so a historical-id override can never launch; preserving
  it preserves a dead value. The effort and alternate were the deliberate
  half. This touches I063's pair-aware decision only on the update side:
  the resolver still classifies `claude-sonnet-5 @ low` as Override; update
  now migrates it to `claude-opus-5 @ low` as a retired-override refresh
  instead of preserving it. **Assumption** flagged for the owner.
- Q5 Successor lookup. The successor of a historical id is the current id
  of the row whose history lists it, same tier first, else first tier in
  `model.Tiers` order. Cross-tier only arises for hand-edited mirrors.
- Q6 Host config. Byte-exact stays (I051 active-ID contract, I072/I074
  design). No host config exists on this machine, so D16 is verified by
  fixture tests only. **Assumption:** a message change plus fixtures is the
  whole fix; no history-aware matching.
- Q7 Lock consolidation. Replace the static `modelRefreshMirrorRows`
  allowlist with a helper that takes the lock's own `ModelRefreshes` and
  admits a `+`/`-` mirror-row line only when its key and value match an
  itemized Old or New (bare gen ≤9 tier keys map through the shared dotted
  resolver; trailing comments are stripped). `modelRefreshDiffLines` goes
  with it.
- Q8 AC4 wording. The locks pin the literal `claude-fable-5-1`, not a value
  derived from the table, so reverting the table fails them.
- Q9 Generation bump. None: no template content changes. Message and logic
  changes only.
- Q10 Retroactive PRD. Written in this effort as a separate docs commit; its
  spec-review of 68aa28f's diff runs in this effort's review stage and is
  recorded in I128.
- Q11 ADR. None. The retired-override rule is reversible data handling and
  refines ADR 0011's "editing a value makes it an override" without
  reversing it; a dated note on ADR 0011 points at I128. CONTEXT.md gains
  the term.

## User Stories and Acceptance Criteria

1. As an operator running claude-team on a stale fleet repo, I see the
   preflight refuse with validate's own message naming
   `spine update --dir R --write`, not "rebuild spine".
2. As an operator with no spine binary, or one predating `model validate`,
   I still see the rebuild message (exit 2 or command not found).
3. As a repo owner running `spine doctor` on a stale mirror, I see D18 naming
   the retired id, its successor, and the update remedy, alongside D2.
4. As a repo owner with `claude.primary: claude-fable-5 @ xhigh`, running the
   printed remedy leaves `claude.primary: claude-fable-5-1 @ xhigh`, the plan
   itemizes it as a retired-override refresh, and validate then exits 0.
5. As a repo owner with a retired id at the default effort, update still
   itemizes an inherited refresh exactly as today.
6. As a repo owner with an override on a current id (any effort) or an
   unrelated id, update preserves it verbatim exactly as today.
7. As a repo owner with a retired id and an edited alternate (pi cells),
   update keeps the alternate and migrates only the id.
8. As a host owner whose routing-host.json lists only the retired id, doctor
   D16 names the host file as the remedy and says the listed id is retired.
9. As a maintainer reverting `models/defaults.json` primary to
   `claude-fable-5` with empty history, the render locks fail.
10. As a maintainer, every per-generation lock admits a mirror-row diff only
    when its own report itemizes that change; a fabricated row line is
    rejected (negative control).
11. As a gen-0 repo owner with `model_default: claude-fable-5`, the retired
    key retires quietly with no divergence message.
12. As the fleet owner, the 14 primary stale checkouts mirror
    `claude-fable-5-1` after deploy and validate exits 0 in each.
13. As a reader of `docs/specs/`, the 68aa28f remap has a design/plan pair
    and I128 records its spec-review.

Acceptance criteria are I128's five, mapped: AC1 = stories 1, 3; AC2 = 4;
AC3 = 8 plus fixtures; AC4 = 9; AC5 = 13.

## Implementation Decisions

- **Retired override migration lives in update's model-routing pass**, next
  to the inherited refresh, not in the resolver. The resolver's provenance
  classification is unchanged (I063 pair-aware history holds). Update asks
  the model package whether the override's id is historical for its harness
  and, if so, which current id succeeds it; `HistoricalIDs` already exists,
  a successor lookup is added beside it.
- **Itemization.** The migration is reported as a `ModelRefresh` with a
  kind marker distinguishing retired-override from inherited; the CLI plan
  prints it as `model refresh (retired override)`. The migrated value keeps
  the repo's own spelling of effort and alternate; only the id token is
  replaced. It is not reported as a preserved override.
- **Gen-9 to 10 effort migration** runs after the retired-override step so a
  bare `primary: claude-fable-5` with a customized top-level effort still
  mints `claude-fable-5-1 @ xhigh`.
- **D18** is a new doctor check, one finding per (harness, tier) whose
  strictly resolved on-disk id is historical and not current for that
  harness, the same predicate launch validation applies. Path WORKFLOW.md,
  severity warn. It runs only when WORKFLOW.md exists and resolves.
- **D16 message** for the unreachable arm appends the host-file remedy;
  when the host's model list for that harness contains a historical id of
  the requested tier's lineage, the message says that id is retired.
- **Preflight (deepthought).** Capture validate's exit code; 127/2 or
  command-not-found keeps today's rebuild text; exit 1 prints
  `claude-team: launch validation refused: <stderr>. No worker spawned.`
  The same shape for codex-team. The guard script gains a refusal arm using
  a fake `spine` that exits 1 with a retired-model line.
- **`modelDefaultDivergence`** treats a value equal to any id
  `claude.primary` ever shipped (current or historical) as never a
  deliberate divergence.
- **Locks.** A single `sanctionedRefreshLine(line, refreshes)` helper in the
  update test package replaces `isModelRefreshDiffLine` and both static
  allowlists. Each generation lock passes its own WORKFLOW.md report's
  refreshes. The gen-13 to 14 lock adds the converse: every itemized refresh
  corresponds to an actual before/after key change and the primary refresh
  is present.
- **Fleet sweep** is a deploy-stage action with the installed binary:
  `spine update --dir R --write` per listed checkout, then
  `spine model --dir R validate claude primary` as the proof.

## Testing Decisions

- Behavior tests at the existing seams: `update.Run` on temp repos
  (modelrouting_test.go prior art), `doctor.runWithHostPath` with a written
  host file (i072_host_test.go prior art), the compiled-CLI plan output
  (main_test.go prior art), and the generation locks.
- Negative controls: an override on a current id at xhigh is preserved;
  `sanctionedRefreshLine` rejects a row not in the report; D18 is silent on
  a refreshed repo and on a current-id override; D16 has no retired hint
  when the host lists no historical id.
- Deepthought's guard script exercises both preflight arms with fake
  binaries.
- Live verification against the installed binary: the jarvis stale mirror
  and the scratch stuck-override copy from the grill repro, before and
  after.

## Out of Scope

- History-aware matching in host config resolution or launch validation.
- Committing in fleet repos, touching worktrees, or the maipipe and
  maikanban working trees.
- A template generation bump or ADR.
- Editing captured generation fixtures.
- Changing which models the table ships.
