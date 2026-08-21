---
title: "I104 drop TOML scanner and I097 gate-pack opt-out codex"
created: 2026-08-20
handoff_ordinal: 9
---

# Handoff — I104 drop TOML scanner and I097 gate-pack opt-out codex (2026-08-20)

## Context

Codex team lead handoff. Primary repo: `/Users/ldh/Projects/github.com/spine` (Go CLI, stdlib
only — ADR 0001). Conventions: `AGENTS.md`, `WORKFLOW.md` (profile `library-cli`), tickets in
`docs/issues/`, ADRs in `docs/adr/` (`docs/adr/README.md` — tightened 2026-08-20: the *decision*
is immutable; citation/typo/dated-note edits are allowed in place).

Read first, in order:
1. `docs/issues/I104-should-the-hand-rolled-toml-scanner-exist.md` — **owner chose (B)** on
   2026-08-20: drop the scan, require `maipipe` on PATH to plan/write `maipipe.toml` when
   `gate_pack` is set, record as an ADR.
2. `docs/issues/I097-gate-pack-opt-out-leaves-the-region-running.md` — unblocked (I096 shipped;
   I095 settled as ADR 0017 reading (A), pure projection).
3. `docs/adr/0017-*.md`, `docs/adr/0015-*.md`, `docs/adr/0001-*.md`.
4. `docs/handoffs/2026-08-20-four-team-parallel-gate-pack-run-*.md` — how the previous run was
   gated (blind review → fix rounds → whole-branch review); repeat that rigor.

## State (verify before relying)

- `main` = `4774e22` (origin at `e3da336`, one doc commit behind; push is fine). Clean tree.
- Gate: `maipipe run full --wait` → run #9 `@e3da336` **passed**. `SPINE_REQUIRE_MAIPIPE=1 make
  test` green, 19 packages.
- Code anchors (verified 2026-08-20):
  - Scanner to delete: `internal/update/maipipecheck.go` (416 lines; `maipipeOnPath`,
    `checkMaipipeContent`, `checkStructure`, `tomlScan`…) and `maipipecheck_test.go` (527 lines).
    Sole callers: `internal/update/update.go:200` (`structuralOnly := !maipipeOnPath()`) and
    `:207` (`checkMaipipeContent`). Keep `maipipeOnPath` and the `maipipe validate` half.
  - Opt-out site: `internal/update/gatepack.go:151` `planMaipipe` — empty/unknown pack returns
    `ok=false` and leaves an existing region alone.
  - Doctor silence: `internal/doctor/doctor.go:184` `gatePackCheck` returns nil on empty pack.
  - Repo-owned composition that a naive splice breaks: `maipipe.toml:68` (`pipeline = "gate-go"`
    inside `full`/`gates`); comment at `:71-78` explains the deliberate absence of an audit lane.

## Next steps

Two tickets, sequential is safest (I097's removal plan runs through the pre-write gate that I104
reshapes). Suggested split:

**I104 (B)** — one worker, routine tier:
1. `spine adr new --dir . "maipipe on PATH is a precondition for touching maipipe.toml when
   gate_pack is set"` (flags before title). Decision: no spine-side TOML scan; `maipipe validate`
   is the only pre-write check; without the binary, `spine update` **refuses to plan/write
   `maipipe.toml`** (plan names the skipped file and the reason) and applies the rest. Cite ADR 0001
   (unchanged) and I104.
2. Delete `maipipecheck.go` + test; rewire `update.go:200-215`: binary absent → `maipipe.toml`
   is skipped with a reason line in the plan, the rest of the plan applies, exit 0; binary
   present → `maipipe validate` as today. Either way the plan says which pre-flight ran (I104 AC 4).
3. Negative controls: PATH without maipipe → plan shows the skip and the file is untouched;
   `maipipe.toml` with a deliberately broken candidate + binary present → refusal still fires.
4. Resolve I104 and note in I096 that the structural half is gone.

**I097** — one worker, routine tier, after I104 merges:
1. Out-of-region reference scan before removal (I097 Fix step 1): stages outside the markers
   composing `gate-go` or `mutation-go` → refuse, name pipeline/stage, file untouched.
2. No references → plan region deletion (markers included) as an ordinary `spine update` diff.
3. `gate_pack: go@99` with an existing region → doctor finding, not silence.
4. ACs in the ticket; the spine-layout fixture must reproduce `composes unknown pipeline "gate-go"`
   when the refusal is removed (load-bearing control).

Both: `gofmt -l . ; go vet ./... ; SPINE_REQUIRE_MAIPIPE=1 make test`, then `maipipe run full
--wait` at the committed SHA. Work on branches off `main`; do not push to `origin/main` —
leave branches for the owner to merge. Write the completion report to
`.superpowers/sdd/team-report.md` (gitignored).

Release-notes facts to carry into whatever release note ships next (no CHANGELOG file exists;
record in the report): D12's warn changes `spine doctor`'s exit code fleet-wide; `spine update`
rejects a `maipipe.toml` lacking top-level `schema`; after I104(B), `spine update` needs `maipipe`
on PATH to touch `maipipe.toml`.

## Gotchas

- **Handoffs**: `spine handoff new "Topic"` (flags before topic), then fill sections. Never
  hand-write one or edit the `spine:cursor` block — it flips `spine cursor` to `blocking`.
- **maipipe reads `maipipe.toml` at the pinned commit** — commit before running a lane. A stop hook
  runs `maipipe run full --wait` when code changed since the last verified run.
- **Shell is fish**: quote `"--include=*.go"` or run through `bash -c`.
- **Commit tickets before branching** — untracked files are invisible in worktrees and caused an
  id collision last run. Next free ticket id is **I105**.
- `cmux send` silently truncates long messages into a running agent — send a file pointer. Workers
  can die quietly (`fish: Job 1 … has stopped`) looking exactly like "still working"; verify
  directly.
- Stage explicit paths only (never `git add -A`); every fix needs a negative control; verify
  against the live artifact, not source edits.
- Owner ban: never route a worker to `claude-sonnet-5`; resolve tiers via `spine model`.

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
