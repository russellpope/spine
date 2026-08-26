---
title: "openweights docs and the i112 axis question"
created: 2026-08-25
handoff_ordinal: 21
---

# Handoff — openweights docs and the i112 axis question (2026-08-25)

## Context

Session 2026-08-25. Three things landed: **I107 shipped** (gate panic becomes a
contractual misconfiguration exit), **I110 shipped** (spine declares an
`openweights` model-table flavor), and the documentation pass that followed
I110 surfaced a domain-model conflict now filed as **I112**.

The programme goal behind all of it is running an agent team on open-weights
models (Kimi K3 / DeepSeek V4 Pro / GLM 5.2) on a **second laptop**. spine's
half of that is done. The deepthought half — the `/openweights-team` skill
itself — **does not exist yet** and is the real critical path.

### Why I112 exists (the part worth reading)

CONTEXT.md ratified **harness** (I067) as the model table's first axis, defined
as *"the execution vehicle that runs a dispatch"*, and drew an explicit
boundary: reachability of a model from a host is *"never a harness"*, with
`"local flavor"` named as the term to avoid because *"local is a property of
where a model is served, not a harness."*

`openweights` fails that definition. Those dispatches run the **ordinary Claude
Code binary** through a wrapper that passes `--model` through — it is the claude
harness pointed at other models, which is exactly what the definition excludes.
`pi` is not a precedent: it is its own driver binary, so it genuinely is an
execution vehicle.

The strongest evidence is that **I111 has to exist at all.** Open-weights
sessions land in `~/.claude/projects` *because they are Claude Code sessions*,
so deriving the axis from the transcript source misfiles every one. Having to
abandon source-derivation for one value of an axis says that value does not
belong on that axis. I112 lays out three options (ratify a widened definition /
rename / split into (harness, model-family)) and is explicitly the owner's call.
Nothing blocks on it — resolution and dispatch work today.

## State (verify before relying)

- `main` = **`4ded51a`**. **Verify push state with `git status --short --branch`**
  — it was one commit ahead of `origin/main` while this was being written.
- Lane: `maipipe run full` **#35 passed @`89fa69e`**; a further run covers
  `4ded51a`.
- `~/bin/spine` rebuilt at the I110 commit. `spine model openweights primary`
  exits **0** — the downstream skill's fatal capability check passes.
- Cursor: effort `i110-openweights-flavor`, every stage `[x]` through `docs`,
  `handoff` current, **derivation clean**.
- Ticket state: **I107 fixed**, **I110 fixed**. Open: **I111** (unstarted, the
  behavioural half), **I112** (new, unrouted, owner decision), I108, I109, I102,
  I105, I106. Next free id: **I113**.
- `spine doctor` exits **1** on two long-standing D4 notes
  (`docs/issues/README.md`, `docs/adr/README.md`). **Pre-existing, not a
  regression** — identical before any work this session. Do not "fix" it.
- deepthought: `docs/specs/2026-08-25-openweights-team-design.md` is filed but
  **uncommitted**, and `/openweights-team` is not built.

## Next steps

1. **Second laptop: `git pull && make install`.** spine is a fleet-wide binary
   and the flavor lives in the binary, not in a repo. That machine's first
   `spine update` in any repo will show a whitespace-only reflow of the
   `model_routing` block (owner-approved; see Gotchas).
2. **Build `/openweights-team` in deepthought** — the actual critical path.
   Before trusting its herdr path, run the design's own **manual spike**:
   `herdr agent start --kind claude` cannot be pointed at `claude-auto`, so
   workers need `herdr pane run`, and whether herdr *detects* a pane-run agent
   is marked **UNVERIFIED** in the spec. If detection fails, the herdr playbook
   needs redesign and the cmux path ships alone. Spike it early.
3. **I111** when convenient. Independent of getting a team running — it only
   fixes whether `spine audit routing` judges those runs correctly. Carries the
   D28 hazard (below) plus two guards inherited from I110's review.
4. **I112** is a decision, not a build. Read it and choose.

## Gotchas

- **I111's D28 hazard passes every existing test.** The audit's
  repo-qualification predicate is gated on a record's flavor being `claude`.
  Tag open-weights records `openweights` and they fall out of that gate and
  start claiming tickets they should not. No current test has an open-weights
  record, so nothing goes red — the damage is wrong verdicts in the field. The
  condition must test *transcript source*, not flavor.
- **`spine model` wants flags BEFORE positionals.** `spine model --effort
  openweights primary` works; `spine model openweights primary -effort` prints
  usage and exits 2, which reads like a broken flavor.
- **The `implement` tick needs a ledger line that starts with the ticket id AND
  contains `done`/`complete` as a whole word** (`implementDoneWordRe`,
  `internal/stages/stages.go`). "I110: … declared" is silently not evidence, and
  the failure message blames the tickets field instead of the wording.
- **Never tick the `handoff` stage.** It makes the handoff doc it describes a
  stale snapshot and blocks `spine audit stages`. `handoff[<]` is terminal;
  recover with `spine cursor here handoff`.
- **`spine cursor start` refuses while an effort is mid-flight** — `--force` to
  supersede. Run it BEFORE `spine handoff new`.
- **Adding a flavor whose name is longer than any existing one reflows every
  repo's `model_routing` block** (the mirror pads to the longest key).
  Whitespace only, but it broke six tests — three in *setup*, reading as broken
  fixtures rather than as the reflow. Fixtures now match rows by key.
- **Never write the literal cursor marker in prose** — the parser finds the
  block by substring scan, so a quoted marker hijacks it. Filed as **I109**.
- **A prescribed negative control is a hypothesis, not a fact.** Three of
  I107's plan-authored controls could not discriminate, and two of those looked
  fine (stayed green). Run both arms.
- Read exit codes unpiped — fish reports the *last* pipeline command's status.
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
