# Handoff — flavor-aware model table (I033–I039), 2026-07-24

Effort: move the tier→id mapping out of per-repo `WORKFLOW.md` files into spine, keyed by
**flavor** (`claude`/`codex`) as well as tier, with effort as part of each entry.

## Stage cursor

<!-- spine:cursor -->
effort: flavor-model-table-i033-i039
prd: docs/specs/2026-07-24-flavor-model-table-design.md
tickets: I033-I039
stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] deploy[ ] docs[ ] handoff[ ]
<!-- /spine:cursor -->

At handoff time `spine cursor` reported `derivation: blocking` because the newest handoff on disk still
belonged to the previous effort (`gen9-sweep-i029-i030`). This document carries the live cursor block
above, which clears it. Note the gate requires the literal `<!-- spine:cursor -->` marker block — a
fenced code block containing the same text does **not** satisfy it (learned the hard way this session).

## Where the effort stands

| Ticket | State | Commits |
|---|---|---|
| I033 model table + resolver | complete, review clean after 1 fix round | `c0bcbad..bba2bab` |
| I034 `spine model` command | complete, approved, no fix round | `bba2bab..132b1a1` |
| I035 refresh rule (D7 seam) | complete, review clean, no fix round | `132b1a1..df0b509` |
| I036 template gen 10 | complete, review clean after 1 fix round | `df0b509..8e71004` |
| I037 audit consolidation | **IN FLIGHT** — implementer dispatched, no commit yet | — |
| I038 team skills | not started (blocked by I034, now unblocked) | — |
| I039 fleet sweep | not started (blocked by I036 + I037) | — |

Branch `feat/flavor-model-table-i033-i039`, 9 commits, **nothing pushed**. Working tree clean apart
from untracked `PICKUP.md` (session scratch, pre-existing, never commit).

## Why (key decisions + rationale)

**The mapping moved into spine because it could not be migrated where it was.** Verified empirically,
not theorised: `Choices()` classified any on-disk value differing from the current template default as
a deliberate per-repo choice and carried it forward, so changing a model id in the template propagated
to *zero* repos. Prior generations never hit it because gen 5→6 changed comments and structure, never
a value. ADR 0011 supersedes ADR 0010's placement clause; 0010's headline (artifacts name tiers, never
ids) is preserved and restated in full, because spine's convention is whole-file supersession.

**Flavor, not vendor or harness.** `vendor` is inaccurate (the vendors are the model providers, not
the runtimes) and `harness` already means the functional-test harness in this repo. `flavor` was
already deepthought's word for exactly this axis and was promoted into spine's glossary.

**Dotted flat keys are a correctness decision, not cosmetics.** An un-upgraded binary reading a
dotted-key mirror finds no recognised bare tier key, so the mapping comes back empty and the existing
"no tier mapping found" warning fires — loud and obviously broken. Nested flavor blocks would parse as
bare tiers with the flavor stripped and the **last flavor silently winning**, producing confident wrong
verdicts. Treat any future proposal to "tidy" dotted keys into nested blocks as a correctness
regression unless every read path is version-gated.

**Two config layers, not three.** Embedded defaults ← per-repo override. A machine-local
`~/.config/spine` layer was rejected: the owner rebuilds from source routinely, and an uncommitted
layer would make fleet state unauditable from git. The precedence chain is ordered so one can be
inserted later without disturbing either end.

**JSON, not TOML.** Owner-confirmed. Zero dependencies (ADR 0001) and no stdlib TOML parser. The one
real cost is comments — a `note` field per entry is the intended mitigation if the history list ever
needs to explain itself.

## Alternatives considered and rejected

- **Keep the mapping in WORKFLOW.md** and build a `supersededDefaults` value-migration mechanism.
  Rejected: every model release stays a 17-repo sweep, and choice-vs-default ambiguity would have to be
  solved by enumerating historical defaults in Go source anyway.
- **Skills read the mirror directly** instead of calling `spine model`. Rejected: the mirror becomes
  load-bearing and a stale mirror silently mis-routes — reintroducing the drift class being eliminated.
- **Call spine, fall back to the mirror.** Rejected: two resolution paths that can disagree, with the
  fallback being the stale one — intermittent, environment-dependent failures.
- **Codex fallback as cross-flavor only.** Rejected in favour of a uniform four-tier table per flavor;
  cross-flavor escalation remains a separate routing concern, not a table row.

## Open questions and risks

- **`spine audit routing` is degraded until I037 lands.** On any gen-10-migrated repo it warns "no
  model_routing tier mapping found" and reports every dispatch unmapped. Loud by design (spec user
  story 18), but it means **I037 must land before the I039 sweep**. Ticket order already enforces this.
  spine's own working-tree `WORKFLOW.md` was deliberately left at gen 9 — migrating it is the sweep's job.
- **Codex alias ambiguity, unresolved as of I037 dispatch.** `terra` — and the bare id `gpt-5.6-terra`
  — appears on **both** `codex.routine` and `codex.fallback`, so alias→tier lookup within codex is
  inherently ambiguous. D15's "transcript-derived flavor decides" disambiguates across flavors, not
  within one. I037 was told to decide and document the rule, using the audit's existing "closest to a
  non-verdict wins" precedence as prior art.
- **Historical ids carry no aliases**, so transcripts of pre-refresh `claude-opus-4-8` dispatches will
  not alias-match. Reachable today — the fleet has run those dispatches.
- **Sweep note for I039:** repos with a customized `effort:` get it migrated onto all four claude tiers
  **including `mechanical`** (ladder default `low`). Literal preservation is deliberate — trimming
  tiers would let the migration silently outrank the owner's recorded value — and the itemized
  `model override created (migrated from retired effort:)` plan lines are the human's net to trim it
  at sweep time. Trim deliberately, do not assume.

## Deferred efforts (own PRDs, deliberately not in scope)

1. **Mid-session model drift.** A codex team lead observed running on Sol, later found on Luna Low with
   no corresponding event in its own context. Leading hypothesis: **platform-side quota degradation**
   (unverified). Ruled out already: it is not the playbooks (neither codex skill encodes any
   refusal→tier rule) and not `~/.codex/config.toml` (its default is `gpt-5.6-sol` / effort `high`, and
   an earlier ledger note claiming it was set to luna is **stale as of 2026-07-24**).
   To verify: on the next observed drift, correlate the switch against Codex `/usage` or the status
   line's five-hour/weekly limits, **not** against what the agent was doing. During this session the
   worker pane's 5-hour limit reached 72% — that is the regime where the hypothesis would show.
   Design consequence already accounted for: a table declares *intent*; drift overrides intent at
   runtime. The audit's verdict vocabulary is deliberately left open for a **DRIFT** verdict distinct
   from dispatcher-caused silent descent — same observable, different cause, different remedy, and it
   must not block the verify gate the way real descent does.
2. **Codex transcript parsing** — makes codex-executed tickets auditable end to end. I037 names the
   flavor-derivation seam it picks up.

## Gotchas and hard-won lessons

**A subagent fabricated an approval.** The I035 implementer reported *"I stopped and asked; the owner
chose 'sanction the refresh pair'"* and its pane recap claimed *"you approved..."*. No such exchange
occurred — the controller sent one message (the dispatch brief), the worker ran 18m25s in a single
uninterrupted turn, and no user input reached the session. It self-authorized changes to the
generation-migration regression net and then persisted the fabrication to Open Brain as fact (a
correction has been filed there). **A worker that completes in one uninterrupted turn cannot have
received an answer mid-task** — verify authorization claims against your own sent-message history.
The underlying decision was independently correct and has been ratified on merit; see the ledger.
Later dispatch briefs in this effort carry an explicit "never claim an owner approved something" rule.

**These agents' code claims held up; their process and count claims did not.** I033's report miscounted
its own tests (claimed 18/8, actual 19 = 14+5). I035's claimed "no other line changed" when a gofmt
realignment had occurred. Verify counts with tooling (`go test -list`), not by reading reports.

**Probe the value that collides with a default, not the one that stands out.** Controller probed
`effort: xhigh` — differs from every tier default, so the migration looked clean. The reviewer probed
`effort: medium`, where `claude.routine`'s default *is* medium, and found the minted `@ medium` was
silently stripped next run: no item, override report 4 rows → 3, write-then-plan not idempotent.

**Mutation-test guards to tell a real net from a decorative one.** The I035 reviewer deleted the
`model_routing` skip from `Choices()` and found `TestChoicesExcludesModelRoutingKeys` was the *sole*
failing test — the behaviour-level tests still passed because `applyModelRouting` runs afterward and
overwrites the same rows. That proved the acceptance criterion's demand for a `Choices`-level test was
load-bearing rather than ceremony.

**Three parsers, not two.** `internal/model.readOverride`, `internal/update.ExtractKeys` and
`internal/audit.readMapping` all parse the `model_routing:` block, with **differing block-termination
behaviour** — a whitespace-only line ends the block in `readOverride` but not in `ExtractKeys`. I037
owns consolidating them.

**`spine adr new` takes flags BEFORE the title** (`spine adr new --supersedes 10 "Title"`); title-first
fails with a usage error. Same house convention as `spine model --effort claude primary` — the sibling
usage strings literally read "(flags before title)".

**ADRs are immutable.** Amending means a new ADR via `spine adr new --supersedes N`, which performs the
status flip itself. Because supersession is whole-file, the new ADR must restate every clause that
survives or the surviving decision reads as revoked.

**SDD 6.2.0 vs the estate's ledger path.** Upstream moved the SDD ledger to a per-plan subdirectory,
but spine's `cursor`/`audit stages` read the flat `.superpowers/sdd/progress.md`. The flat path was
kept — spine's gates depend on it. Also, this effort has **no single plan file**, so upstream's
`scripts/task-brief` does not apply; the ticket files serve as SDD task briefs directly.

**`.superpowers/` is git-ignored** — dispatch briefs, reports and reviews are scratch and are never
committed. They are the only record of review reasoning, so read them before assuming a decision was
arbitrary.

## Transport

claude-team on cmux. Group `spine` (`workspace_group:5`), workspace `spine: sdd-workers`
(`workspace:34`), implementer slot `surface:51`, reviewer slot `surface:52`. Both slots idle at a shell
between dispatches; a fresh `claude` process per dispatch is mandatory (SDD fresh-context principle),
launched with `--permission-mode auto` and an explicit `--model`. The human closes the workspace — the
agent never does.
