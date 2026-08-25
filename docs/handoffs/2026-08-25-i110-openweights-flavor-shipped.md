---
title: "i110 openweights flavor shipped"
created: 2026-08-25
handoff_ordinal: 20
---

# Handoff — i110 openweights flavor shipped (2026-08-25)

## Context

Session 2026-08-25, second effort. **I110 is shipped**: spine now declares an
`openweights` flavor, so `spine model openweights <tier>` resolves and the
downstream `/openweights-team` capability check passes. Built inline — the spec
fixed the exact table contents, which `WORKFLOW.md` lists as a justified inline
exception (verbatim pre-specified diff).

The goal driving this is getting an open-weights team runnable **on a second
laptop**. I110 is the spine half of that; the deepthought half is not built yet.

## State (verify before relying)

- `main` = **`8807542`**, tree clean. **Not yet pushed at the time of writing —
  verify with `git status --short --branch`.**
- Lane: `maipipe run full` **#34 passed @`41eda16`**; a further run covers the
  two docs commits after it.
- `~/bin/spine` rebuilt at the I110 commit. `spine model openweights primary`
  exits **0**.
- **I110 `status: fixed`.** Cursor: every stage `[x]` through `docs`, `handoff`
  current. **I111 is open and unstarted.**
- `spine doctor` exits **1** on the same two long-standing D4 notes
  (`docs/issues/README.md`, `docs/adr/README.md`). Pre-existing, not a
  regression — unchanged across both efforts this session.
- Next free ticket id: **I112**.
- deepthought: `docs/specs/2026-08-25-openweights-team-design.md` is filed but
  **uncommitted**, and the `/openweights-team` skill itself does not exist yet.

## Next steps

1. **The second laptop needs `git pull` + `make install`.** spine is a
   fleet-wide binary; the flavor lives in the binary, not in a repo. Its first
   `spine update` in any repo will show the mirror reflow described below.
2. **Build `/openweights-team` in deepthought** — that is the actual critical
   path to a team running on Kimi/DeepSeek/GLM. I110 only unblocks its preflight.
   Its design doc is the filed-but-uncommitted spec above. Note the design's own
   warning: `herdr agent start --kind claude` cannot point at `claude-auto`, so
   herdr workers need `herdr pane run`, and whether herdr *detects* a pane-run
   agent is **UNVERIFIED** and gated on a one-time manual spike.
3. **I111** (derive audit flavor from the observed model id) is independent of
   getting a team running — it only fixes whether `spine audit routing` judges
   those runs correctly. Safe to defer past the laptop handoff. It carries the
   D28 hazard plus two guards inherited from I110's review.

## Gotchas

- **`spine model` wants flags BEFORE positionals.** `spine model --effort
  openweights primary` works; `spine model openweights primary -effort` prints
  usage and exits 2. Easy to misread as a broken flavor.
- **The `implement` tick needs a ledger line that starts with the ticket id AND
  contains `done`/`complete` as a whole word** (`implementDoneWordRe` in
  `internal/stages/stages.go`). A line reading "I110: … declared" is silently
  not evidence; the tick fails with "every id is missing; check it for a typo",
  which points at the tickets field rather than at the wording.
- **Adding a flavor whose name is longer than any existing one reflows every
  repo's `model_routing` block.** The mirror pads its key column to the longest
  `flavor.tier` key. Whitespace only, but it broke six tests — three of them in
  *setup* ("fixture line not found to replace"), which reads as a broken fixture
  rather than as the reflow it is. Fixtures now match rows by key, so this
  should not recur; if it does, look at the padding first.
- **Never tick the `handoff` stage.** It makes the handoff doc it describes a
  stale snapshot and blocks `spine audit stages`. `handoff[<]` is the terminal
  state; recover from an accidental tick with `spine cursor here handoff`.
- **`spine cursor start` refuses while an effort is mid-flight** — pass
  `--force` to supersede. Run it BEFORE `spine handoff new`.
- **Never write the literal cursor marker in prose.** The parser finds the block
  by substring scan, so a quoted marker hijacks it. Filed as **I109**.
- Read exit codes unpiped: in fish, `cmd 2>&1 | tail; echo $status` reports
  *tail's* status.
- The maipipe stop hook demands `maipipe run full --wait` whenever HEAD moves,
  docs-only included. Batch commits so one lane run covers them.
- **Stage explicit paths only** — never `git add -A`.
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @ low`.

<!-- spine:cursor -->
effort: i110-openweights-flavor
prd: docs/specs/2026-08-25-openweights-flavor-spine-design.md
tickets: I110
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[<]
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
