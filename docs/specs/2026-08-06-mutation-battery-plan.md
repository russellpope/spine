# Mutation battery — plan

**Date:** 2026-08-06 · **Effort:** mutation-battery · pairs with
`2026-08-06-mutation-battery-design.md`

## Ticket breakdown

| Ticket | Deliverable | Tier | Notes |
|---|---|---|---|
| I053 | D1a + D1d — checklist doc + ADR 0013 | routine | docs-only in spine; provenance marks are requirements, not prose |
| I054 | D1b + D1c — runner into `/model-eval` skill + record convention | routine | includes acceptance-criteria 2–4 (end-to-end run, negative control, record format) |
| I055 | D2 — do-not-regress template + dispatch-prep instruction | routine | sample block generated from Laguna round history |
| I056 | D1e — spec/evidence relocation + reference updates | mechanical | spine `scripts/` sheds eval artifacts; evidence preserved in `docs/research/` |

Order: I053 and I054 are independent; I055 depends on I054 (same skill files);
I056 last (relocation after the runner's new home exists).

## Stage mapping

- **implement:** I053–I056 via subagent-driven development, tiers as annotated.
- **functional-test:** acceptance criteria 2–3 (battery runs from skill home;
  `NO-SITE` negative control fires). The battery itself is the harness here;
  `functional_harness: cli` applies to spine, not to these docs/skill artifacts.
- **review/verify:** `/spec-review` of the finished diff against the design doc;
  verify includes `spine audit stages` + `spine audit routing` clean, and re-running
  the reproduction batch from the skill-bundled runner (rates must match
  `results-20260806-090020`).
- **ship:** commit spine docs (+ relocations) with explicit paths; skill changes
  land in the skill's own home (estate-owned, outside this repo's git).
- **docs/handoff:** research doc cross-references finalized; handoff records the
  filed future-work note against the eval repo (B7/B8 harness question).

## Risks

- The skill lives at `~/.claude/skills/model-eval/` — outside spine's git. Verify
  steps must check the live skill dir, not assume repo state covers it.
- Literal-replacement brittleness is a feature (self-detecting via `NO-SITE`) but
  only if the exclusion is *disclosed* — the reporting rule must survive edits.
- `local-model-evaluation` eval trees are never modified; relocation targets the
  records/specs area only.
