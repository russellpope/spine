# Stage-cursor controls (template gen 8) — design

Source: wayfinder map [I010](../issues/I010-wayfinder-map-stage-skipping-controls.md) (decisions I011–I017, grilled 2026-07-15). Companion plan: [2026-07-15-stage-cursor-controls-plan.md](2026-07-15-stage-cursor-controls-plan.md). Triage: ready-for-agent.

## Problem Statement

Workflow stages (grill → prd → issues → implement → …) can be silently skipped. On 2026-07-15 an ultima-dci-edition resume handoff named an abbreviated path ("grill-with-docs → to-spec then SDD build"), dropping the `issues` stage, and implementation began before /to-tickets. The stage cursor lived only in prose handoffs; no tooling tracked stage progression. The repo-local fix (a hand-written "Stage cursor (consistency rule)" section in ultima's WORKFLOW.md) covers one repo, blocks that repo's template updates as an unrecognized local edit, and depends on the model honoring skill text — the control class that already failed. Meanwhile the fleet is split across template gens 5/6/7, so no template-level control reaches every repo.

## Solution

Make stage-skipping mechanically resisted in every spine repo: a structured, machine-parseable stage cursor in each effort's SDD ledger becomes spine-owned convention (template gen 8); a shared derivation engine judges the cursor against on-disk artifacts, exposed blocking as `spine audit stages` (verify gate, beside `audit routing`) and advisory in doctor; a new read-only `spine cursor` subcommand prints the cursor + verdict; one global SessionStart hook injects it at session start; /handoff embeds `spine cursor` output verbatim with mechanical backstops; and a fleet sweep brings all 17 repos to gen 8, ultima last via the sanctioned `supersededLines` reconciliation.

## User Stories

1. As a repo owner, I want the stage cursor to be a spine-owned template convention, so that every repo gets the same protection without hand-written local edits.
2. As a session agent, I want the stage cursor injected into context at session start, so that "where are we" never depends on my memory or a prose handoff.
3. As a session agent resuming from a handoff, I want the handoff to carry the cursor verbatim, so that an abbreviated prose path cannot hide a skipped stage.
4. As the verify gate, I want `spine audit stages` to exit non-zero on cursor/artifact mismatch, so that a skipped stage blocks completion the same way silent tier descent does.
5. As a repo owner running doctor, I want an advisory stage-consistency check, so that I can see drift early without doctor becoming a gate.
6. As an agent mid-effort, I want a ticked stage with a missing artifact to block, so that a cursor that lies (claims prd done with no PRD on disk) is caught.
7. As an agent mid-effort, I want present artifacts with an unticked stage to block, so that a stale cursor (tickets exist but issues unticked) is caught.
8. As an owner of a repo not mid-effort, I want a missing `progress.md` to produce only a warning, so that dormant or non-SDD repos aren't blocked by a control aimed at active efforts.
9. As a multi-effort repo owner, I want the cursor block to name its effort's PRD and ticket set, so that derivation judges exactly this effort's artifacts, not any PRD that ever existed.
10. As the /handoff author, I want to paste `spine cursor` output rather than paraphrase, so that the resume prompt's stage picture is generated, not remembered.
11. As the fleet owner, I want all 17 repos on gen 8 after one sweep, so that no repo remains a gap and the next adherence audit doesn't reproduce the skew finding.
12. As the ultima-dci-edition owner, I want plain `spine update --write` to upgrade my hand-edited WORKFLOW.md cleanly, so that the pioneering local fix doesn't strand the repo off the template train.
13. As a future collaborator, I want the cursor format specified in WORKFLOW.md, so that skills and humans write it identically and the parser accepts what the docs promise.

## Implementation Decisions

- **Shared derivation engine, two exposures** (I011): one engine derives the true stage from on-disk artifacts and compares to the ledger cursor. `spine audit stages` blocks (non-zero exit) on mismatch; doctor gains an advisory check on the same engine. Doctor stays read-only-advisory; audit remains the blocking family.
- **Structured cursor block anchors the effort** (I012): the cursor lives at the top of `.superpowers/sdd/progress.md` as a machine-parseable block: effort name, PRD path, ticket-id set or prefix, and the stage checklist (a single `stages:` line, one token per WORKFLOW.md stage: `[x]` done, `[<]` you-are-here — exactly one among non-done stages, `[ ]` pending; amended 2026-07-16 to the shipped grammar, owner-ratified). Derivation is bidirectional and judges only the anchored artifacts. No `progress.md` ⇒ warn-only. The exact grammar is specified in the gen 8 WORKFLOW.md section and is the single format both the parser and skills honor.
- **`spine cursor` subcommand** (I013): read-only; prints the parsed cursor plus the advisory derivation verdict; exits zero even on mismatch (surfacing, not gating). It is the single primitive the hook and /handoff call.
- **Global SessionStart hook** (I013): one hook in the user-global harness settings; when the cwd is a spine repo with a cursor, inject `spine cursor` output into context. No per-repo harness config.
- **/handoff hardening** (I014): the handoff skill must run `spine cursor` and embed its output verbatim — prose paraphrases of "what's next" are defined as incomplete. Backstops: doctor advisory + `audit stages` blocking check that the newest handoff document contains a cursor block whenever a cursor exists.
- **Template gen 8** (I012/I014/I017): WORKFLOW.md gains the spine-owned "Stage cursor (consistency rule)" section (adapted from ultima's) including the cursor grammar and the handoff rule; template generation bumps to 8; ultima's hand-written section lines are added verbatim to the update machinery's `supersededLines` so plain update recognizes and replaces them.
- **Fleet sweep** (I015): all 17 spine repos updated to gen 8 (dry-run review → write → commit per repo); ultima-dci-edition last, via the supersededLines path; `--force` with an owner-reviewed diff is the fallback if unexpected drift appears.
- **Gates for this build itself** (I016): tickets in spine's ledger with routing annotations; overnight subagent-driven execution with ESCALATION/FALLBACK records; `spine audit routing` at verify; morning /spec-review against this PRD + owner verification.

## Testing Decisions

- Test external behavior only: command exit codes, printed verdicts, and file outcomes — never parser internals. Carve-out (amended 2026-07-16, owner-ratified): white-box regression tests are permitted for documented internal heuristics where the defect class cannot be pinned through the CLI seam (e.g. the `implementEvidence` negation cases).
- Amendment (2026-07-16, owner-ratified, from final review): cursor grammar findings BLOCK `spine audit stages` (a malformed cursor must not pass the gate); they remain advisory in `spine cursor` and doctor.
- **Engine + commands**: fixture-repo trees under the audit/doctor testdata pattern (prior art: `internal/doctor/doctor_test.go`, `internal/audit` tests) exercising: clean cursor passes; ticked-stage-missing-artifact blocks; artifacts-present-unticked-stage blocks; no `progress.md` warns without blocking; newest handoff lacking a cursor block blocks when a cursor exists; `spine cursor` prints and exits zero on mismatch.
- **Migration**: gen7→8 in the existing per-generation update-test seam (prior art: `internal/update/gen4to5_test.go`), including a fixture of ultima's real WORKFLOW.md proving plain update upgrades it with no unrecognized lines (prior art: `internal/update/hbmview_test.go`).
- **Hook**: thin shell wrapper, no CI seam; verified live during morning verify (owner-approved seam decision).
- Zero new seams: everything tests at the spine CLI command/package boundary.

## Out of Scope

- Per-repo harness hooks for collaborators not sharing the global settings (revisit if spine repos gain outside collaborators).
- Stale-`template_version` on-touch nudges (mooted by the full sweep).
- Fleet-wide enforcement of /spec-review and audit-routing *frequency* — a separate adherence gap; fresh effort if pursued.
- Any change to the `audit routing` contract or the ESCALATION/FALLBACK grammar.

## Further Notes

- The build must capture ultima's WORKFLOW.md section lines **verbatim** before templating, or the supersededLines reconciliation silently misses.
- `spine adopt`/`init` seed the cursor convention for new repos automatically once the template carries it; mid-flight efforts adopt on their next /handoff cycle.
- Wayfinder efforts don't carry an SDD ledger; the cursor governs SDD build efforts. A wayfinder map reaching its to-spec handoff starts the cursor at `grill✓ prd←`.
