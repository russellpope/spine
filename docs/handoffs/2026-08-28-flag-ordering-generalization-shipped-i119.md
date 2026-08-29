---
title: "flag-ordering generalization shipped (I119)"
created: 2026-08-28
handoff_ordinal: 34
---

# Handoff — flag-ordering generalization shipped (I119) (2026-08-28)

## Context

Session 2026-08-28 (third of the day). Picked I119 from the batch-shipped
handoff's candidates, grilled it in two rounds (Q1–Q11, all ratified on
recommendations), ran it solo inline with full gates and TDD.

Shipped: no spine subcommand silently discards input. `parseArgs` in
cmd/spine/main.go owns Parse + I116 ordering guard + exact arity + usage
for all 24 FlagSet sites; the 24-entry ordering-sweep table in
strictargs_test.go doubles as the conversion checklist. Unknown cursor
sub-subcommands error naming start|tick|here|set (the filing repro
`spine cursor show --dir X` — wrong repo, exit 0 — now exits 2); flag-only
cursor invocations keep the exit-0 hook contract (doc comment amended).
**`spine gate` grammar flipped to flags-first**
(`gate [--dir D] <pack>[@<v>] <check>`); no estate caller passed flags to
gate, so only its 30 in-repo test call sites moved. First-position `--`
exemption, trailing `--force`, and version/help leniency all preserved and
pinned. Spec-review (fresh sub-agent, requirements-attack first): 0 scope
creep; C1–C3 contradictions amended into the PRD; its two real catches
(docs/mutation-battery-checklist.md example, model `--` exemption arm)
fixed in 0b42006.

Guidance sweep: README + checklist fixed in-repo; NO out-of-repo edits
needed — no skill, memory file, or hook teaches the retired "silently
ignores" workaround. This handoff retires the last living
flags-before-positionals gotcha: every subcommand now works or names the
rule.

## State (verify before relying)

- main = `0b42006`, ff-merged, lane-verified at exactly that SHA (maipipe
  full #3; #1 passed @d39398d). NOT yet pushed — ssh still keyless; use
  the gh-https workaround (memory: estate-push-gh-https-workaround). The
  docs/handoff commit on top of 0b42006 needs its own
  `maipipe run full --wait` + push.
- I119 status fixed, `batch: 2026-08-28-flagorder#1`, commits + Resolution
  written. **I120 filed by ANOTHER session** (8d38c19, session_012xpA…:
  `adr new --supersedes 0011` parses as octal via flag.Int base-0,
  observed in maikanban) — it landed on the i119 branch mid-session and
  merged with it. Open, unassigned. Next free id: I121.
- Binaries: `~/bin/spine` AND `~/.local/bin/spine` both rebuilt at
  0b42006 (`build: …0b42006c0e69…+dirty` — dirty is the other session's
  untracked docs/research file, not code). ~/.local/bin was stale at a
  July build and is what the SessionStart hook hardcodes; its pre-I114
  tickets-grammar warning disappears next session.
- Untracked stray: docs/research/2026-08-26-fusion-harness-borrow-hitlist.md
  (other session's — leave it).

## Next steps

1. Push main (docs commit + this handoff) via the gh-https workaround;
   run the lane at the final HEAD first.
2. **I120** (adr octal --supersedes) is small and freshly filed — natural
   next pick; the other session may claim it.
3. Standing from before: I112 decision (openweights axis, parked), I111
   D28 hazard, I072/I102/I105 routing work, hbmview feat/header-redesign
   fate, deepthought /openweights-team, ssh key restore.

## Gotchas

- `spine gate` is flags-first now. The old `gate go@1 tskip --dir X` form
  errors naming the rule. Bare maipipe run lines unchanged.
- A bare `spine gate --dir . go@1 tskip` at spine's root exits 1 — the two
  dogfood t.Skip fixtures; the lane sets SPINE_GATE_TSKIP_ALLOW. Not a
  regression.
- Stop re-teaching flag ordering in handoffs: a violation now names the
  rule at the point of failure on every subcommand (I116 -> I119).
- Two sessions shared this branch today (8d38c19). Check `git log` before
  assuming your commits are alone on a branch; stage explicit paths only.
- Never tick the handoff stage; `spine cursor` sole writer; exit codes
  unpiped under fish.

<!-- spine:cursor -->
effort: flag-ordering-generalization
prd: docs/specs/2026-08-28-flag-ordering-generalization-design.md
tickets: I119
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
