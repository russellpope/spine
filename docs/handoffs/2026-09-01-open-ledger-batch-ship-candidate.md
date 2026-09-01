---
title: "open-ledger-batch-ship-candidate"
created: 2026-09-01
handoff_ordinal: 37
---

# Handoff — open-ledger-batch-ship-candidate (2026-09-01)

## Context

Final ship-candidate handoff for the expanded open-ledger batch. The preceding
ordinal-36 handoff records the complete implementation, external fleet, review,
verification, routing, and disposition narrative. This snapshot exists to bind
the current `ship[<]` cursor immediately before the single batch-final lane.

## State (verify before relying)

- Pre-handoff tracked HEAD: `d37ee1c1601db948d90f679017ba2194a868cd27`.
- Whole-branch primary re-review: PASS at `d37ee1c`.
- Independent acceptance re-verification: VERIFIED at `d37ee1c`.
- Source-built doctor: exit 0 with D4 information only.
- Source-built stage audit: exit 0, PRD 1/1, issues 21/21, implement 21/21,
  current handoff snapshot nonblocking before the verify tick.
- Full unscoped routing audit: exit 0; no blocking verdict. I051, I120, and
  I122 are retained as `escalated-with-reason` with immutable evidence.
- Source-built dead-code, deferred-cleanup, binary-hygiene,
  gitignore-control, and configured tskip gates: no findings.
- No tracked `.superpowers` path or build/cache artifact exists. The protected
  research note remains the only unrelated untracked file.
- The commit containing this handoff is the intended exact lane SHA; replace
  the pre-handoff HEAD above with `git rev-parse HEAD` after commit when
  invoking the lane, without editing this file.

## Next steps

1. Commit only ordinal-36 and ordinal-37 handoff documents.
2. Confirm clean tracked state and pin the resulting exact SHA.
3. Run `maipipe run full --wait` once at that SHA. Stop on any finding.
4. On PASS, push exactly that SHA with the gh HTTPS helper and fetch origin.
5. Install exactly that SHA to both `~/bin/spine` and
   `~/.local/bin/spine`, then verify hashes/revisions.
6. Finish the ignored team report and herdr completion signals; leave the
   workspace open.

## Gotchas

- Any commit after the lane invalidates its verdict.
- Do not stage/delete the research note or change I105/I112/I125 scope.
- Installed copies are intentionally stale until the lane passes.

<!-- spine:cursor -->
effort: open-ledger-batch
prd: docs/specs/2026-08-29-open-ledger-batch-design.md
tickets: I111,I051,I050,I072,I073,I074,I075,I078,I066,I076,I077,I007,I032,I093,I102,I105,I108,I121,I122,I123,I124
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[<] deploy[ ] docs[ ] handoff[ ]
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
