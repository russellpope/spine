---
title: "I104 option B and I097 complete on reviewed stacked branches"
created: 2026-08-21
handoff_ordinal: 10
---

# Handoff — I104 option B and I097 complete on reviewed stacked branches (2026-08-21)

## Context

This is the completed, reviewed stacked delivery for I104 and I097. The
authoritative prior handoff is
`docs/handoffs/2026-08-20-i104-drop-toml-scanner-and-i097-gate-pack-opt-out-codex.md`;
the combined PRD is
`docs/specs/2026-08-20-i104-i097-maipipe-preflight-design.md`.

I104 is the first fixed range on `i104-drop-toml-scanner`, ending at
`2e56006`. I097 is the reviewed descendant stack on
`i097-gate-pack-opt-out`; approved code HEAD is `47e869e`.

I104 chose option B, recorded in ADR 0018: no spine-side TOML scanner remains
(including its scanner-only tests); `maipipe` on PATH is required before spine
touches a gate-pack `maipipe.toml`. A missing binary skips that file while
other updates can apply; a present binary is the sole candidate validator. I096
carries the dated note that its structural half was removed.

I097 makes clearing `gate_pack` a safe operation. It aggregates outside-region
`gate-go`/`mutation-go` compositions into a plan refusal naming every owning
pipeline and stage; otherwise it plans marker-inclusive region deletion. The
doctor reports stale/invalid regions, including unknown pins. Its header reader
is intentionally narrow: a single-line, quote-aware TOML table-path and
assignment reader, with maipipe remaining the grammar authority.

Blind review found and the fix waves closed the plan-report refusal surface,
TOML spelling/quoted-path/bare-key/escape boundaries, doctor behavior under
missing maipipe and damaged markers, one-time maipipe resolution, operational
validator classification, and PRD/ticket wording. Final primary review ends
**APPROVED**.

## State (verify before relying)

- `SPINE_REQUIRE_MAIPIPE=1 make test`, `gofmt -l .`, `go vet ./...`,
  `git diff --check`, and `spine audit routing` passed at the approved code.
  Before this handoff existed, stage audit was pending only its stale snapshot;
  this generated handoff completes that workflow evidence.
- `maipipe run full --wait` run #6 is pinned to `47e869e` and passed all 7/7
  stages: `fast/vet`, `fast/test`, `gates/binary-hygiene`,
  `gates/dead-code-callgraph`, `gates/deferred-cleanup-errcheck`,
  `gates/gitignore-control`, and `gates/tskip`.
- During the run, `main` advanced externally to `e4d0e41` through unrelated
  I105. These branches were based on `64f88cf`; integrate carefully so I105 is
  preserved.

## Next steps

- Do not push or merge from this worktree. The owner should merge/rebase I104
  first, then the stacked I097 branch, preserving unrelated I105 on main.
- Release-note facts: D12's warn finding changes `spine doctor`'s exit code;
  `spine update` rejects a `maipipe.toml` missing top-level `schema`; I104
  requires maipipe on PATH before gate-pack `maipipe.toml` can be touched.

## Gotchas

- Do not replace the targeted header reader with a general TOML parser. It is
  deliberately limited to single-line stage headers and keys; maipipe owns
  full grammar validation.
- I097's preflight/composition refusal guarantee is pre-write: it leaves all
  pending files untouched. It is not a rollback promise for later filesystem
  write failures.
- Preserve the generated cursor block below; `spine` is its sole writer.
<!-- spine:cursor -->
effort: i097-gate-pack-opt-out
prd: docs/specs/2026-08-20-i104-i097-maipipe-preflight-design.md
tickets: I097
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
