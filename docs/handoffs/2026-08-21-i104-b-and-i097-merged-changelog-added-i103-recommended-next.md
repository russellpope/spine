---
title: "I104 B and I097 merged, CHANGELOG added, I103 recommended next"
created: 2026-08-21
handoff_ordinal: 11
---

# Handoff — I104 B and I097 merged, CHANGELOG added, I103 recommended next (2026-08-21)

## Context

Session 2026-08-21 (continuation). Merged the codex team's two reviewed, stacked branches —
I104 option B (drop the hand-rolled TOML scanner, ADR 0018) and I097 (gate_pack opt-out) —
onto `main`, gated, pushed, cleaned up, and added the project's first `CHANGELOG.md` carrying
the consumer-visible behaviour changes that earlier tickets had deferred "to the release note".
Previous handoff: `docs/handoffs/2026-08-21-codex-team-delivered-i104-b-and-i097-on-stacked-branches-unmerged.md`.

## State (verify before relying)

- `main` = `9d63621` (CHANGELOG) on `81891fa` (merge, `--no-ff`); **pushed**, origin in sync.
- Gate: `maipipe run full` #12 `@81891fa` passed; #13 `@9d63621` passed. At `81891fa`:
  gofmt clean, `go vet` clean, `SPINE_REQUIRE_MAIPIPE=1 make test` 18 pkgs ok (+`models`, no tests).
- I104 and I097 tickets: `status: fixed` (set by the codex team, on the merged branches).
- Worktrees `../spine-wt-i104` and `../spine-wt-i097` removed; branches `i104-drop-toml-scanner`
  and `i097-gate-pack-opt-out` deleted. `../spine-wt-2` (`codex-wt-2`, clean, no commits past
  main) is stale from an earlier run — left for the owner.
- cmux `spine` group `workspace_group:3` (lead 33, workers 34–36) still open; owner closes.
- `CHANGELOG.md`: root, Keep-a-Changelog, `Unreleased` section — D12 doctor exit code (I094),
  maipipe as sole grammar authority + maipipe-on-PATH precondition (I104/ADR 0018), opt-out
  refusal/removal + D10 stale/damaged (I097). Every claim traced to a named test.
- No cursor effort opened: this was ticket-driven merge/docs work, not a PRD'd effort.

## Next steps

1. **Recommended next ticket: I103** (med, unblocked) — `gate_pack: go@1` freezes the class
   list but not the attribution string; findings would be coded `go@2/<check>` once go@2 ships.
   Fix direction is in the ticket (render `SPINE_GATE_PACK` into the region, `gate.Code`/
   `PackID` honour it, refuse unknown packs). Free to fix now, expensive later. Needs the owner
   for `/grill-with-docs` → `/to-spec` (mandatory PRD gate), then `/handoff-to-codex`.
   Open question for the grill: the render change bumps `definition_hash` fleet-wide — does
   the I098 added-stage notice need a "same stages, changed env" case?
2. Then I101 (med: audit routing can't attribute file-delivered briefs), I102 (low: unify
   team-spawn pairing), I105 (low: opencode subagents note, no code).
3. Owner: close cmux 33–36; decide on `../spine-wt-2`.
4. Optional: archived SDD rulings ledgers from the gate-pack run are still only in a scratchpad
   (see previous handoff) — move or let them go.

## Gotchas

- A merged branch can bring in a handoff doc newer than yours whose cursor block names a
  branch-local effort; `spine cursor` then reports `derivation: blocking`. Fix is
  `spine handoff new "…"` (never hand-edit the block) — that is why this doc exists as well.
- maipipe pins `maipipe.toml` at the committed SHA — commit before any lane; the stop hook
  demands `maipipe run full --wait` whenever HEAD moved (docs-only too).
- fish shell: quote `"--include=*.go"` or use `bash -c`; Write/Edit for file content.
- Commit tickets before branching — untracked files are invisible to worktrees. Next free
  ticket id: **I106**.
- Stage explicit paths only; every fix needs a negative control; owner ban on `claude-sonnet-5`.
- Owner wants `/handoff-to-codex` for new implementation work; `cmux send` truncates long
  messages — send file pointers; auto-mode can't `cmux close-workspace`.

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
