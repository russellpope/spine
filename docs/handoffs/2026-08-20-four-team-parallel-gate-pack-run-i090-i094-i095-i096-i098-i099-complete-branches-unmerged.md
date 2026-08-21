---
title: "Four-team parallel gate-pack run: I090 I094 I095 I096 I098 I099 complete, branches unmerged"
created: 2026-08-20
handoff_ordinal: 8
---

# Handoff — Four-team parallel gate-pack run: I090 I094 I095 I096 I098 I099 complete, branches unmerged (2026-08-20)

## Context

Four SDD teams run in parallel, one per git worktree, dispatched via claude-team on cmux
(implementer + reviewer slot each; the controller drove all four loops). Gate per task: **blind**
review (brief + diff, never the implementer's report) → fix rounds → scoped re-review →
whole-branch review on the most capable model → one fix wave → one scoped re-review.

Seven tickets closed — I090, I094, I095, I096, I098, I099, plus an ADR 0017 carry-forward task.
Four filed: I101–I104. **Note the cursor block below is for the previous effort
(local-harness-conventions, I079–I087); this run was ticket-driven, not PRD-driven, and did not
open a new cursor effort.**

## State (verify before relying)

`main` is unchanged at `983f1a3`; **nothing merged, nothing pushed**. Working tree carries only
untracked files: this handoff, `I095`–`I099` tickets, the region-ownership research doc, `.DS_Store`.

Four complete branches, each with its own worktree:

| Branch | Worktree | Tickets | Head |
|---|---|---|---|
| `i090-audit-routing` | `../spine-wt-i090` | I090 | `b70b21e` |
| `i094-maikanban-slug` | `../spine-wt-i094` | I094 | `c5fe3e1` |
| `i099-contract-drift` | `../spine-wt-i099` | I099 | `7ab4cae` |
| `i095-i096-i098-gatepack` | `../spine-wt-gatepack` | I095, I096, I098 | `241ff45` |

**Integration verified**, not assumed: all four merged onto `983f1a3` in `/private/tmp/spine-integ`
(branch `integration-trial`) with **zero conflicts**; `gofmt` clean, `go vet` clean, `make test`
green across 19 packages with `SPINE_REQUIRE_MAIPIPE=1` so the maipipe-dependent controls actually
ran. Cross-branch semantics checked: ticket ids I090–I104 unique, ADR 0016 `Superseded by 0017`,
0016's "carried forward into the ADR that supersedes this one" promise **true** in the merged tree.

`maipipe run full --wait` → run #8 `@983f1a3` **passed**, all 7 stages executed — but that pins
`main`, so **it does not cover the branch work.** Re-run against the merge commit.

Per-branch SDD ledgers (34 rulings with reasoning, gitignored, **not yet deleted — read before
cleaning up**): `<worktree>/.superpowers/sdd/plan-*/progress.md`.

## Next steps

1. **Merge decision (owner).** Fast-forward `main` to `integration-trial`, or merge the four
   branches directly onto `main`. Then re-run `maipipe run full --wait` so gate evidence covers the
   shipped tree, and push (origin is at `ab204e5`, two behind before any merge).
2. **Owner calls, all blocking nothing but worth settling:**
   - **`docs/adr/README.md` contradicts itself** — a broad "the only permitted edit is the status
     flip" sentence vs a narrow "reversing or amending a *decision*". The repo practises the narrow
     reading (ADR 0013, 0016's I091 amendment). Two reviewers found it independently; refused twice
     inside these tickets as out of scope. Until tightened, every amendment re-litigates it.
   - **I104** — should the hand-rolled TOML scanner exist? ADR 0001 blocks a library, but the
     alternative nobody weighed is *no scanner*: require `maipipe` on PATH when `gate_pack` is set.
     Removes ~400 lines and the whole over/under-refusal surface.
   - **I094 item 3** — `workflow-init` SKILL.md note in the deepthought repo; outside every worktree.
   - **Release notes**: D12's warn changes `spine doctor`'s exit code fleet-wide (verified no CI lane
     goes red); `spine update` now rejects a `maipipe.toml` lacking top-level `schema`.
3. **I097** (gate-pack opt-out) is still untracked and unstarted — it was blocked on the I095 call,
   which is now settled as reading (A), so it is unblocked.
4. Cleanup once merged: `git worktree remove` the four worktrees, delete `integration-trial` and
   `/private/tmp/spine-integ`, close the cmux `spine` group's four `sdd-*` workspaces (owner closes;
   auto-mode denies workspace closes).

## Gotchas

- **Commit tickets before branching.** I095–I099 were untracked at the branch point and therefore
  invisible in every worktree. Consequences: one team imported its own requirements mid-task,
  another built from the brief alone, a third filed **colliding ticket ids** (I095/I096, renumbered
  to I101/I102). The ledger is a directory with no allocation lock.
- **`cmux send` silently truncates** long messages pasted into an already-running claude. A fix
  message lost two Important findings; the worker reported them unseen rather than guessing. Send a
  **file pointer** to a worker that is already running. Launch-time dispatches are safe — they
  interpolate through the shell (`"$(cat file)"`).
- **Workers die quietly.** A cmux claude can be suspended (`fish: Job 1 … has stopped`), leaving the
  pane at a prompt with no report — indistinguishable from "still thinking" without reading the
  screen. A commit-watching monitor also silently missed a landed commit. Three detection failures
  this session: verify directly, never trust one signal.
- **Hand-written handoffs break derivation.** Writing `docs/handoffs/*.md` by hand omits the
  `spine:cursor` block and flips `spine cursor` to `derivation: blocking`. Always
  `spine handoff new "Topic"` (flags before topic), then fill sections — never hand-edit the cursor
  block. This session did it wrong first and had to regenerate.
- **`--include=*.go` fails unquoted** in fish; quote it (`"--include=*.go"`) or use `bash -c`.
- `fable` hit 100% quota mid-run; weekly reached ~85%.

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
