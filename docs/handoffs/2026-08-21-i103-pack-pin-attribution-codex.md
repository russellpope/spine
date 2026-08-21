---
title: "I103 pack pin attribution codex"
created: 2026-08-21
handoff_ordinal: 12
---

# Handoff — I103 pack pin attribution codex (2026-08-21)

## Context

Codex team lead brief for **I103** — the pack pin freezes the check-class list (I098) but not
the attribution string; fix it now while one pack exists. Grilled and specced 2026-08-21 with
the owner; the design is settled, do not re-open it. Conventions: `AGENTS.md` (Codex twin of
CLAUDE.md), `WORKFLOW.md` (`library-cli` profile, `model_routing` tiers — reference tiers, never
ids; owner ban on `claude-sonnet-5`).

**Read first, in order:**
1. `docs/specs/2026-08-21-i103-pack-pin-attribution-design.md` — problem, stories, decisions.
2. `docs/specs/2026-08-21-i103-pack-pin-attribution-plan.md` — five tasks, red/green steps,
   negative controls. This is the work order.
3. `docs/adr/0019-the-pack-pin-rides-the-stage-run-line-not-an-env-var.md` — the carrier
   decision and why; the ticket's env-var suggestion is superseded.
4. `docs/issues/I103-pack-attribution-is-not-pinned.md` — ticket, acceptance criteria,
   dated Decision note.
5. `CONTEXT.md` §Gate pack — **pack pin** glossary entry; use the term.

## State (verify before relying)

- Primary repo: `/Users/ldh/Projects/github.com/spine`, branch `main` = `cca65c5` (spec pack),
  origin in sync. Gate: `maipipe run full` #16 `@cca65c5` passed. Tree clean but `.DS_Store`.
- I104 (ADR 0018) and I097 are **merged** at `81891fa`; their worktrees/branches are gone.
  `../spine-wt-2` (`codex-wt-2`) is a stale clean worktree — leave it.
- I103 frontmatter: `execution-mode: subagent-driven`, `tier: routine`, `review-tier: routine`,
  `status: open`. Next free ticket id: **I107** (I106 was filed today by another session).
- Code anchors (line numbers at `cca65c5`): `internal/gate/gate.go:25-40` (`PackName`,
  `PackVersion`, `PackID`, `Code`), `:148-205` (`packClasses` table, `PackClassesFor`,
  `PackIDs`); `internal/update/gatepack.go:102` (`packClassesFor` seam), `:120-140` (stage
  render), `:233` (`stageDelta` → `StagesAdded/Removed`), `:600-625`
  (`unrecognizedRegionLines` run-line recognition); `cmd/spine/main.go:654` (`gateUsage`).
  Prior-art tests: `internal/update/gatepin_test.go` (seam stub, stage delta),
  `internal/gate/gate_test.go:47-60` (unknown check exits 2), `cmd/spine/main_test.go` gate cases.
- Exit vocabulary: 1 = findings, 2 = misconfiguration. Unshipped pin and out-of-pin class are
  both exit 2, no findings document.

## Next steps

1. Branch `i103-pack-pin` from `cca65c5` in a worktree (`../spine-wt-i103`). Commit the ticket
   state first if you touch it — untracked files are invisible to worktrees.
2. Execute the plan task-by-task (subagent-driven; workers get an explicit model tier and the
   `I103` token in every dispatch — `spine audit routing` checks this at verify).
3. Task 2 step 7 rewrites spine's own `maipipe.toml` (spine is a go@1 adopter). Commit it
   with the code **before** any maipipe lane; maipipe pins the file at the committed SHA.
4. Verify per plan Task 5: gofmt, `go vet ./...`, `SPINE_REQUIRE_MAIPIPE=1 make test`,
   `spine update` dry-run clean, `maipipe run full --wait` at the final commit. Paste commands
   and output in the report. Then `/spec-review` of the diff against the design doc (mandatory
   gate), final primary-tier review, and `spine audit routing` / `spine audit stages` exit 0.
5. Record: I103 Resolution + `status: fixed`; story 23 and I098 note per plan Task 4.
6. Report to `.superpowers/sdd/team-report.md`; write a team handoff with
   `spine handoff new --dir <worktree> "<topic>"`. **Do not merge or push** — leave the
   reviewed branch for the owner, trigger-flash your surface.

## Gotchas

- `spine handoff new` only; never hand-edit a `spine:cursor` block (flips derivation to
  blocking). Cursor changes go through `spine cursor …` only.
- maipipe's stop hook demands `maipipe run full --wait` whenever HEAD moves past the last
  verified SHA, docs-only included. Commit before running lanes.
- fish shell on this host: quote `"--include=*.go"` or use `bash -c`; no heredocs in chained
  commands.
- Stage explicit paths only — never `git add -A`/`.`; `.DS_Store` stays untracked.
- Every fix needs a negative control (the plan names one per behaviour); state the exact
  command and paste the output.
- `cmux send` truncates long messages — dispatch workers with file pointers. Workers die
  quietly: verify via report file + branch + pane, never one signal.
- Reports ≤ ~1500 words; Write/Edit tools for file content.

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
