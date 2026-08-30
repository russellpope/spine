---
title: "open-ledger-batch-codex"
created: 2026-08-29
handoff_ordinal: 35
---

# Handoff — open-ledger-batch-codex (2026-08-29)

Audience: the codex team lead. This doc is authoritative for the batch; read
it before anything else, then `AGENTS.md` for repo conventions and
`WORKFLOW.md` for the stage/gate contract and model routing.

## Context

The spine ledger (`docs/issues/`) holds 18 open tickets after I119/I120
shipped. This batch hands **17 of them** to a codex worker team — everything
open except **I112** (owner-parked with the openweights programme; it is a
definition decision, not build work — do NOT pick it up). Every listed
ticket's `blocked-by` is satisfied: I070, I071, I107, I110 are all
`status: fixed`. Two intra-batch edges remain: **I073 and I077 are blocked
by I072** — sequence them after it.

Read each ticket file before dispatching it; the files are the specs.
Ticket files: `docs/issues/I0NN-*.md`. Conventions for closing them
(front-matter `status`, `commits`, Resolution section): see
`docs/issues/README.md` and the freshly closed
`docs/issues/I120-adr-new-supersedes-parses-zero-padded-ids-as-octal.md`
as a worked example.

## State (verify before relying)

- main = `d1f3c10`, pushed (origin/main == main), maipipe lane verdict
  passed at exactly that SHA (run #55). Working tree clean except one
  untracked stray: `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`
  — another session's file; never stage or delete it.
- Binaries `~/bin/spine` and `~/.local/bin/spine` both at `d1f3c10`.
- The cursor block below is I119's, terminal at `handoff[<]` — that effort
  is closed. Start your own cursor per effort/batch with `spine cursor
  start`; `spine` is the sole legal cursor writer (`start|tick|here|set`),
  never hand-edit, and never tick the handoff stage.
- Next free issue id: **I121**.

## The batch (17 tickets)

Severity-ordered; sizes are hints, the ticket files are binding.

1. **I111** (high) — derive audit flavor from the observed model id, not
   the transcript source. Touches `internal/audit/`. The D28 hazard named
   in its body is real; read its Related section.
2. **I051** (med) — fail-closed pre-dispatch model validation: forbidden
   tokens, unmapped-route refusal.
3. **I050** (med) — verify convention: approved-untested as a first-class
   acceptance state.
4. **I072** (med) — host config schema + preference/constraint precedence.
   Do this EARLY: it unblocks I073 and I077.
5. **I073** (med, after I072) — flavor → harness rename migration.
6. **I074** (med) — audit routing verdicts for heterogeneous dispatches.
7. **I075** (med) — effort as a first-class dispatch parameter.
8. **I078** (med) — discarded-dispatch record grammar (audit routing stops
   reading abandoned prototypes as silent descent).
9. **I066** (med) — Claude-as-harness-for-3P-models wayfinder map.
   Research/doc deliverable, not code.
10. **I076** (low) — routing-yield forward build: REVIEW record line +
    `spine yield` verb.
11. **I077** (low, after I072) — eval evidence feeding equivalence-pin
    ratification.
12. **I007** (low) — `primary: session` sentinel for the primary tier.
13. **I032** (low) — scope the tickets-typo hint to the issues row;
    decouple truncation test from cap.
14. **I093** (low) — gate-pack dogfood follow-ups (4 subitems; note prior
    handoffs flagged items 3–5 as owner-call — implement only what the
    ticket marks actionable, surface the rest).
15. **I102** (low) — unify team-spawn worker-prompt pairing
    (`teamspawn.go` vs `codex.go` recognizers).
16. **I105** (low) — opencode custom-subagents research to inform pi vs
    opencode worker choice. Research/doc deliverable.
17. **I108** (low) — `spine doctor` advisory for toolchain skew between
    installed binary and PATH Go.

## What's next / definition of done

- Per ticket: TDD (failing test first), full `go test ./...` green,
  ticket closed (status/commits/Resolution), CHANGELOG.md entry when the
  behaviour is consumer-visible.
- Mandatory gates per `WORKFLOW.md`: grill + verify. For feature-shaped
  tickets (I050, I051, I072, I073, I074, I075, I076, I078) a PRD/spec in
  `docs/specs/` before implementation; verify = fresh-context reviewer
  against the ticket/spec, requirements-attack first (check the spec
  itself for contradictions before judging the diff — surface with a
  proposed resolution, never silently resolve).
- Ship: `maipipe run full --wait` must pass at the exact final SHA before
  push. Push works over ssh only if keys are restored; otherwise the
  https workaround:
  `git -c credential.helper= -c credential.helper='!gh auth git-credential' push https://github.com/russellpope/spine.git main:main`
  then the matching `fetch origin` with the same helper to refresh
  tracking refs.
- Deploy after ship: `make install` (writes `~/bin/spine`) AND
  `cp ~/bin/spine ~/.local/bin/spine` — the SessionStart hook runs the
  `~/.local/bin` copy; both must be refreshed.
- Completion report: `.superpowers/sdd/team-report.md` (git-ignored) —
  per-ticket disposition (shipped SHA / blocked-why / surfaced-for-owner).

## Task breakdown hints

- I072→I073→(I077) is the only ordered chain; everything else is
  independent and parallelizes.
- I111, I074, I078 all touch `internal/audit/` — serialize or assign to
  one worker to avoid merge churn.
- I007 and I075 both touch model/dispatch surfaces (`internal/model/`,
  routing docs) — check overlap before parallel dispatch.
- I066 and I105 are pure research/writing — cheap workers, no code gates,
  but still land as committed docs (`docs/research/` or per-ticket
  guidance).

## Gotchas

- **Flags-first CLI (I119):** every `spine` subcommand requires flags
  before positionals; `spine gate` grammar is
  `gate [--dir D] <pack>[@<v>] <check>`. Violations exit 2 naming the rule.
- A bare `spine gate --dir . go@1 tskip` at repo root exits 1 by design
  (two dogfood t.Skip fixtures); the lane sets `SPINE_GATE_TSKIP_ALLOW`.
  Not a regression.
- Under fish, `$status` after a pipe reports the LAST pipeline command —
  read exit codes unpiped. (herdr exit codes are honest when unpiped.)
- Stage explicit paths only; never `git add -A` / `git add .`. Two
  sessions have shared branches here before — check `git log` before
  assuming your commits are alone.
- Scratch/dispatch files stay in git-ignored `.superpowers/sdd/`.
- Secrets: none in-repo; gh auth is already logged in with repo scope.

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
