---
title: "cursor hygiene batch shipped"
created: 2026-08-28
handoff_ordinal: 29
---

# Handoff — cursor hygiene batch shipped (2026-08-28)

## Context

The cursor-hygiene batch is implemented, reviewed, verified, and locally
shipped on `main`. I114 adds exact comma-list ticket resolution and propagates
the grammar through the generation-11 WORKFLOW template and updater. I113
widens the canonical comparison through the closing fence line. I115 hardens
D13 frontmatter handling for absolute workspaces, quoted values, parser
divergence, and fence-less tickets.

## State (verify before relying)

- Final implementation head: `0f4adc8`; ticket-close commit: `a3f371b`.
- `maipipe run full --wait` run `main #48` passed at `a3f371b`.
- Fresh primary verification passed uncached tests, vet, format, `make test`,
  four negative controls, a new installed-CLI scratch acceptance, stage audit,
  routing audit, and doctor checks.
- I113, I114, and I115 are `fixed`; their `commits:` lists are recorded and
  their live `workspace:` keys are cleared.
- The stage cursor is at deploy. Ship is complete; deploy is intentionally
  left current because the estate sweep is owner-visible and outside this
  workspace's write scope.

## Next steps

1. Owner: run the eight-repo estate sweep from the plan for ccq,
   home-lab-admin, jarvis, notetui, observability_notes, pure-automation,
   deepthought, and hbmview, recording pre/write/post/doctor exits.
2. Leave praxis, moo-clone, and ultima untouched; ultima remains one grammar
   line stale until its genuine local edit is reconciled.
3. Owner: push spine and any approved local-only sweep commits, then close the
   herdr team workspace.

## Gotchas

- No estate repository was modified and nothing was pushed.
- The unrelated untracked research file
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md` remains untouched.
- Final review deferred only wording and test-fixture cleanup: align the
  canonical prefix example with diagnostic metasyntax on a future template
  edit, correct the stale grammar-home citation, and rename the synthesized
  migration fixture's "frozen" wording.
- Never tick the handoff stage. The generated block below is the current
  deploy-stage snapshot.

<!-- spine:cursor -->
effort: cursor-hygiene-batch
prd: docs/specs/2026-08-28-cursor-hygiene-batch-design.md
tickets: I113-I115
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[<] docs[ ] handoff[ ]
<!-- /spine:cursor -->

## Checkpoint (newest): 003-dogfood-the-shipped-local-harness-conventions-on-spine-itsel.md

<!-- spine:checkpoint:facts -->
touched:
- internal/update/gatepack.go
- internal/gate/results.go
- internal/gate/mutate.go
- maipipe.toml
- WORKFLOW.md
gate: pass
sha: 265efc9ede4c229f135c38b558bfe722ec918427
effort_recommended: medium
written: 2026-08-19T16:31:36Z
<!-- /spine:checkpoint:facts -->

### Prior narrative (model-authored, not evidence)

## Task

Dogfood the shipped local-harness conventions on spine itself (deepthought handoff 2026-08-19 §1a–h) and close the cross-repo follow-through (§2).

## Conclusions

- go@1 pack is self-enabled on spine (I089); five classes + mutation-go pass under maipipe at the pinned commit.
- First live maipipe seam found four defects, all fixed: region TOML grammar + schema (I091); results line 0 / file "." / severity "warn" and battery env leak (I092).
- Bake-off positive control: hygiene classes catch committed binaries on 3/3 arms (docs/research/2026-08-19-…).
- Checkpoint round-trip, model alternate provenance, routing blind spot (I090) verified; minor follow-ups in I093.
- Cross-repo: maipipe I201 filed; deepthought spine PRD amended; /model-eval runs the binary.

## Next moves

- Owner: push spine (main ahead of origin, unpushed since 2132d89); close herdr team workspace; remove worktree spine-wt-local-harness.
- Owner call on I093 items 3–5 (unconfigured-class stages, --force scoping, D11 value tamper).
- Phase 1 continues: `/grill-with-docs` in maipipe with deepthought's maipipe execution-floor PRD.
