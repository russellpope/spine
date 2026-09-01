---
title: "open-ledger-batch-final-ship"
created: 2026-09-01
handoff_ordinal: 36
---

# Handoff — open-ledger-batch-final-ship (2026-09-01)

## Context

The expanded 21-ticket open-ledger batch is product-complete. I112 remains
excluded and untouched; I105 remains open at its explicit owner-decision
boundary with research complete and no product action; owner-authored I125 is
open, ticket-only, and out of batch. I007 is superseded. Every other included
ticket is fixed with durable commit and Resolution evidence.

I051's controlled Deepthought phase is current at `7650d84`. I073's exact
generation-14 candidate migrated all 20 primaries without force; the 19
external commits remain current/ancestral and all generated states converge.
I076 is fixed pending batch ship after exhaustive primary review and separate
verification.

## State (verify before relying)

- Pre-handoff tracked candidate: `d37ee1c1601db948d90f679017ba2194a868cd27`.
- Comparison base / origin main: `d1f3c101928bf15772b100b160a5474fb1fdb2a2`.
- Whole-branch primary re-review: PASS at `d37ee1c`.
- Independent primary acceptance re-verification: VERIFIED at `d37ee1c`.
- Full unscoped routing window: exit 0. I051, pre-base I120, and I122 are
  `escalated-with-reason`; no blocking row remains.
- Source-built full tests, full race, vet, native/Linux/Windows builds, update
  convergence, configured go@1 gates, hostile matrices, external Deepthought
  tests, and all-20 fleet state checks pass.
- No `.superpowers` path is tracked. The sole worktree item is the protected
  untracked `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`.
- Installed binaries are deliberately pre-deploy and are refreshed only after
  the exact final SHA passes the batch lane.

## Next steps

1. With this `verify[<]` snapshot current, run source-built `spine doctor` and
   `spine audit stages`; resolve any blocker rather than force-ticking.
2. Advance `verify` through the `spine cursor` sole writer, create the final
   current-cursor ship handoff, and commit handoff documentation only.
3. Pin that exact final SHA and run `maipipe run full --wait`. A later commit
   invalidates the verdict and requires a new run.
4. On PASS, push the same SHA through the gh HTTPS credential-helper workaround
   and fetch origin with the matching helper.
5. Deploy that SHA with `make install`, copy `~/bin/spine` to
   `~/.local/bin/spine`, and verify both hashes/version revisions.
6. Update `.superpowers/sdd/team-report.md`, set herdr pane metadata DONE, send
   the completion notification, and leave the workspace open for owner review.

## Gotchas

- Do not stage or delete the protected research note.
- Do not pick up I112, implement I125, or turn I105's research into a product
  choice.
- The final lane is the only ship verdict; prior informational/pre-candidate
  lanes do not count.
- `spine gate` and all commands remain flags-first.
- The owner closes the herdr workspace; agents never do.

<!-- spine:cursor -->
effort: open-ledger-batch
prd: docs/specs/2026-08-29-open-ledger-batch-design.md
tickets: I111,I051,I050,I072,I073,I074,I075,I078,I066,I076,I077,I007,I032,I093,I102,I105,I108,I121,I122,I123,I124
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[<] ship[ ] deploy[ ] docs[ ] handoff[ ]
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
