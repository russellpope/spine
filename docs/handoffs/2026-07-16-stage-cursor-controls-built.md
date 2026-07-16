# Handoff Reference — stage-cursor controls built, verify pending owner (2026-07-16)

## Stage cursor

<!-- spine:cursor -->
effort: stage-cursor-controls
prd: docs/specs/2026-07-15-stage-cursor-controls-design.md
tickets: I018-I023
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[<] ship[ ] deploy[ ] docs[ ] handoff[ ]
<!-- /spine:cursor -->

(Verbatim `spine cursor` output at handoff time reported `derivation: blocking` on exactly one finding — the 2026-07-10 handoff predating this convention lacked a cursor block. THIS document carries the block, clearing it; re-run `spine audit stages` to confirm exit 0.)

## State

- Branch `feat/stage-cursor-controls` at **28c0608**, 9 commits over base 3cb6d48 (main). All tests green (14 packages), routing audit exit 0 (session-scoped transcripts per I008 workaround), confirmation pass MERGEABLE.
- Fleet: 17/17 repos processed to gen 8 — 11 clean commits, ultima clean via supersededLines (18a3ba1, no --force), 2 WORKFLOW.md skips + 4 uncommitted-dirty repos → [I028](../issues/I028-story11-closure-fleet-residue.md).
- Estate: SessionStart hook live in ~/.claude/settings.json (absolute path, gen-8 binary installed at /Users/ldh/bin/spine); /handoff skill carries the mandatory stage-cursor section; deepthought doc at docs/reference/claude-session-hooks.md (untracked).

## Why (key decisions + rationale)

Map [I010](../issues/I010-wayfinder-map-stage-skipping-controls.md) indexes decisions I011–I017 (grilled with owner 2026-07-15). Final review + acceptance sim proved the original ultima incident is now mechanically caught in both directions (present-unticked and ticked-missing), and caught the malformed-cursor gate bypass — fixed in 28c0608 (blocking in audit stages only; spine cursor/doctor stay advisory).

## Open questions & risks

- Owner morning gate (verify stage): /spec-review of 3cb6d48..28c0608 against the PRD; live hook demo (open any swept repo in a fresh session — cursor should appear in context); then ship call (merge feat/stage-cursor-controls → main; push only on owner's word).
- Follow-ups filed: [I024](../issues/I024-cursor-printer-malformed-wording.md) printer wording, [I025](../issues/I025-effort-matched-handoff-block.md) effort-matched handoff block, [I026](../issues/I026-tickets-grammar-single-id-unresolvable.md) single-id grammar + unresolvable surfacing, [I027](../issues/I027-derivation-polish-batch.md) polish batch, [I028](../issues/I028-story11-closure-fleet-residue.md) story-11 closure (HITL).
- deepthought side (uncommitted, owner's call): docs/issues/I001 spec-path drift ticket, docs/reference/claude-session-hooks.md.
- Bookkeeping notes: commit 87fdc86 swept in the pre-existing I009 ticket file (content-appropriate, kept); old I006 closure question inside I028.

## Gotchas & hard-won lessons

- Reviewer-above-tier dispatches need their ESCALATION record AT DISPATCH TIME — the routing audit flagged all five post-hoc, second build running this lesson.
- `.superpowers/` gitignore: every testdata fixture under a `.superpowers/` path must be `git add -f`'d and verified via `git ls-files` — bit two tasks.
- Cross-build routing audit: always `--transcripts` a session-scoped dir (spine I008), else other builds' dispatches collide on ticket ids.
- audit-stages blocking rule keys on CursorFindings non-empty (any grammar finding blocks), not zero-stages — pinned by the confirmation pass.
