---
title: "Codex team delivered I104 B and I097 on stacked branches, unmerged"
created: 2026-08-21
handoff_ordinal: 10
---

# Handoff — Codex team delivered I104 B and I097 on stacked branches, unmerged (2026-08-21)

## Context

Session 2026-08-20/21. Landed the four-team gate-pack run (I090/I094/I095/I096/I098/I099) on
`main` via fast-forward; took the owner calls (ADR README tightened; **I104 → option B**; I097
started); then handed I104(B)+I097 to a codex team via `/handoff-to-codex`. The codex team finished:
two reviewed, stacked, **unmerged** branches. Brief:
`docs/handoffs/2026-08-20-i104-drop-toml-scanner-and-i097-gate-pack-opt-out-codex.md`.

## State (verify before relying)

- `main` = `e4d0e41` (I105 docs, committed outside this session); origin at `64f88cf` — **one commit
  unpushed**. Gate: run #10 `@64f88cf` passed; e4d0e41 is docs-only but unverified by the hook.
- Branches (both from `64f88cf`; worktrees still open):
  - `i104-drop-toml-scanner` @ `2e56006` — `../spine-wt-i104`. ADR 0018; `internal/update/maipipecheck.go`
    + test deleted; no-maipipe path skips `maipipe.toml` with a named preflight skip, exit 0.
  - `i097-gate-pack-opt-out` @ `9fd1cb2` — `../spine-wt-i097`, stacked on 2e56006. Opt-out refuses when
    out-of-region stages compose `gate-go`/`mutation-go`; else plans marker-inclusive deletion; doctor
    reports stale/damaged regions. Single-line composition reader, deliberately narrow.
- Codex verification at 9fd1cb2: gofmt/vet clean, `SPINE_REQUIRE_MAIPIPE=1 make test` 19 pkgs green,
  `spine audit routing`/`stages` exit 0, `maipipe run full` #7 passed. Final primary review APPROVED.
  Report: `.superpowers/sdd/team-report.md` (gitignored). Team handoff:
  `../spine-wt-i097/docs/handoffs/2026-08-21-i104-option-b-and-i097-complete-on-reviewed-stacked-branches.md`.
- `git merge-tree --write-tree main i097-gate-pack-opt-out` → **clean** (checked 2026-08-21).
- cmux `spine` group `workspace_group:3`: lead `workspace:33`, workers `workspace:34/35/36` still open.
- Previous run's SDD rulings ledgers (34 rulings) archived, not durable:
  `/private/tmp/claude-501/-Users-ldh-Projects-github-com-spine/2a272890-211b-40b2-94c7-dbb9e16fc9aa/scratchpad/sdd-ledgers/`.

## Next steps

1. `git merge --no-ff i097-gate-pack-opt-out` onto main (brings I104 along; I104 is its ancestor).
   Commit, `maipipe run full --wait` at the merge commit, push (origin is two+ behind).
2. Cleanup after merge: `git worktree remove ../spine-wt-i104 ../spine-wt-i097`; delete the two
   branches; owner closes cmux workspaces 33–36 (auto-mode can't).
3. Release notes (no CHANGELOG exists — decide where they live): D12 warn changes `spine doctor`
   exit code fleet-wide; `spine update` rejects `maipipe.toml` without top-level `schema`; with
   `gate_pack` set, `spine update` needs `maipipe` on PATH to touch `maipipe.toml` (ADR 0018);
   clearing `gate_pack` now refuses external consumers or removes the region.
4. Open tickets worth a look next: I103 (pack attribution not pinned to go@N), I101, I102, I105.
5. Optional: move the archived ledgers somewhere durable before the scratchpad is reaped.

## Gotchas

- `spine handoff new "Topic"` only (flags before topic); never hand-edit the `spine:cursor` block.
- maipipe pins `maipipe.toml` at the committed SHA — commit before any lane; a stop hook demands
  `maipipe run full --wait` whenever HEAD moved past the last verified SHA (even docs-only).
- fish shell: quote `"--include=*.go"` or use `bash -c`; heredocs and `echo ===` in chained
  commands break — use the Write/Edit tools for file content.
- Commit tickets before branching — untracked files are invisible to worktrees (caused an id
  collision on 2026-08-20). Next free ticket id: **I106**.
- `cmux send` truncates long messages into a running agent; send file pointers. Workers die quietly;
  verify via report file + branch + pane, never one signal.
- Stage explicit paths only; every fix needs a negative control; owner ban on `claude-sonnet-5`.

<!-- spine:cursor -->
effort: local-harness-conventions
prd: docs/specs/2026-08-18-local-harness-conventions-design.md
tickets: I079-I087
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
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
