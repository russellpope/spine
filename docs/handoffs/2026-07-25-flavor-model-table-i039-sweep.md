# Handoff Reference — flavor-aware model table: I039 sweep + close-out (2026-07-25)

Effort: move the tier→id mapping out of per-repo `WORKFLOW.md` files into spine, keyed by **flavor**
(`claude`/`codex`) as well as tier, with effort as part of each entry. **Six of seven tickets are done.**
This document carries what the sweep needs and the reasoning that would be expensive to rediscover.

## Stage cursor

<!-- spine:cursor -->
effort: flavor-model-table-i033-i039
prd: docs/specs/2026-07-24-flavor-model-table-design.md
tickets: I033-I039
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[<] deploy[ ] docs[ ] handoff[ ]
<!-- /spine:cursor -->

`go run ./cmd/spine cursor` reports `derivation: clean` as of this writing. The literal marker block above
is what satisfies `spine audit stages` — a fenced code block containing the same text does **not**.

## Where the effort stands

| Ticket | State | Commits |
|---|---|---|
| I033 model table + resolver | complete, clean after 1 fix round | `c0bcbad..bba2bab` |
| I034 `spine model` command | complete, approved, no fix round | `bba2bab..132b1a1` |
| I035 refresh rule (D7 seam) | complete, clean, no fix round | `132b1a1..df0b509` |
| I036 template gen 10 | complete, clean after 1 fix round | `df0b509..8e71004` |
| I037 audit consumes resolver | complete, clean after 1 fix round | `3edb7a9..a2f2df6` |
| I038 team skills (deepthought tree) | complete, clean after 1 fix round | dt `9554cb1..95c1f4b` + spine `59f3b08` |
| **I039 fleet sweep** | **not started** | — |

Two repos, two branches, **nothing pushed**:
- spine `feat/flavor-model-table-i033-i039` — 14 commits, tree clean but for untracked `PICKUP.md`.
- deepthought `feat/i038-team-skills-resolve-through-spine` — 2 commits off `main`. Untracked
  `agents*.zip` / `agents 2/` are pre-existing junk, not ours; keep them out of any commit.

## The sweep's actual shape (measured 2026-07-25, not assumed)

35 scaffolded directories carry `template_version`: **17 primary repos + 18 worktrees**, all at gen 9.

**16 of the 17 are stock and migrate mechanically.** Verified by dry-run against `ccq`: `template_version`
9→10, the dotted `model_routing:` mirror renders, and exactly **one** itemized refresh —
`model refresh (inherited): model_routing.claude.fallback: claude-opus-4-8 -> claude-opus-5`. Every one of
the 16 carries byte-identical stock routing values, so expect that same single refresh in each.

**maipipe is the exception and the only real judgment call in I039.** It is the one repo with genuine
overrides — `primary: gpt-5.6-sol`, `routine: gpt-5.6-terra`, `mechanical: gpt-5.6-luna` (remapped
2026-07-10 for the Codex-driven build) plus `fallback: claude-opus-4-8` commented "cross-harness: runs in
Claude Code, not dispatchable from Codex". **`spine update` refuses to write it**, naming that fallback
comment: `skipped WORKFLOW.md — unrecognized local edits (use --force to drop)`. The guard works — do not
`--force` past it.

The substantive problem it protects you from: gen ≤9 had no flavor axis, so I036's migration maps bare tier
keys onto **`claude.*`** rows. Applied blindly to maipipe that yields `claude.primary: gpt-5.6-sol` — a
codex id under the claude flavor. `spine model claude primary` would then hand a codex model to a claude
dispatch, and I037's audit would judge claude dispatches against a codex id. maipipe is genuinely
**mixed-flavor** (codex ordered tiers + a deliberate claude fallback), which the gen-10 dotted mirror can
express exactly — `codex.primary/routine/mechanical` + `claude.fallback` — but which no automatic migration
will infer. Migrate maipipe by hand, deliberately.

**Worktrees: decide, don't drift.** The 18 `*-wt-i*` directories are worktrees of maipipe and
ultima-dci-edition sharing their parents' git objects; sweeping one commits to whatever branch it has
checked out. The prior gen-8/gen-9 sweeps covered 17 repos, i.e. primaries only, and worktrees inherit when
their branches merge or rebase. Recommend the same, stated explicitly in the ledger rather than left implicit.

**M-2 preflight: already run, and clean.** I037 unified block termination so a whitespace-only line now ends
the `model_routing:` block for *every* parser — meaning a hand-edited file with a blank line mid-block would
have trailing entries invisible to both update and audit, and the sweep could silently refresh a stranded
override to a default. Scanned all 17 primaries: **zero stranded entries.** Re-run if any repo is hand-edited
before the sweep; the scan is in this session's ledger.

**Pre-sweep baseline captured** at `.superpowers/sdd/i039-pre-sweep-baseline.txt` (git-ignored) — every
primary's routing block plus its retiring `effort:`/`model_default:` lines. I039's AC3 ("repos with genuine
model overrides retain them") is provable by diffing post-sweep against it instead of by eye.

## Before the sweep: install the binary

The installed `spine` is **stale — it predates the `model` subcommand** and fails with
`unknown command "model"`. This session deliberately never installed an unreviewed branch build over the
fleet binary and used `go run ./cmd/spine` throughout. I039 needs a real `make install` (in
`~/Projects/github.com/spine`), and that is a deliberate step: it is also what makes the I038 skills work,
since their preflight now probes `spine model` capability.

This is not hypothetical. I038's I-2 finding was exactly this state: `command -v spine` succeeds on the stale
binary, `spine model` prints nothing and exits 2, and an *unquoted* empty expansion collapses the argument so
codex consumes the kickoff prompt as its `-m` value. Fixed by probing capability and double-quoting every
substitution — verified empirically, `codex exec -m ""` exits 1 loudly rather than defaulting.

## Why (key decisions + rationale)

**The mapping moved into spine because it could not be migrated where it was.** Verified, not theorised:
`Choices()` classified any on-disk value differing from the current template default as a deliberate per-repo
choice and carried it forward, so changing a model id in the template propagated to *zero* repos. Prior
generations never hit it because gen 5→6 changed comments and structure, never a value.

**Dotted flat keys are a correctness decision, not cosmetics.** An un-upgraded binary reading a dotted-key
mirror finds no recognised bare tier key, so the mapping comes back empty and the existing warning fires —
loud and obviously broken. Nested flavor blocks would parse as bare tiers with the flavor stripped and the
**last flavor silently winning**, producing confident wrong verdicts. Treat any proposal to "tidy" dotted keys
into nested blocks as a correctness regression unless every read path is version-gated.

**Retiring the audit's loud-failure mode was correct** (I037's biggest call, upheld on merit). Story 18's
loudness governs the **un-upgraded** binary, which no change here can touch; the upgraded binary's guard
against a newer format is D14's hard refusal, which is *stronger* than a warning. Meanwhile the old loudness
called every legitimate dispatch unmapped in any gen-10 repo — verdicts false relative to dispatch reality,
which is the two-parser disease the ticket cured. D13 now carries the reporting contract that was missing.

**Alias/history matching is provenance-scoped, and the scoping is load-bearing.** An `Override` entry matches
its exact id only; `Default`/`Inherited` keep aliases and history. All four scenario fixtures carry an
inherited `fallback: claude-opus-4-8`, and the mixed fixture's `opus` token resolves *only* through the
Inherited entry's alias carry — so a blanket strip breaks AC6's letter while the `Override`-scoped strip
preserves it. Documented in D13 precisely so nobody "simplifies" it later.

**Two config layers, not three.** Embedded defaults ← per-repo override. A machine-local `~/.config/spine`
layer was rejected: the owner rebuilds from source routinely, and an uncommitted layer would make fleet state
unauditable from git.

**JSON, not TOML.** Zero dependencies (ADR 0001), no stdlib TOML parser. Cost is comments; a `note` field per
entry is the intended mitigation.

## Alternatives considered and rejected

- **Keep the mapping in WORKFLOW.md** with a `supersededDefaults` migration mechanism — every model release
  stays a 17-repo sweep, and choice-vs-default ambiguity still needs historical defaults enumerated in Go.
- **Skills read the mirror directly** instead of calling `spine model` — the mirror becomes load-bearing and a
  stale mirror silently mis-routes, reintroducing the drift class being eliminated.
- **Call spine, fall back to the mirror** — two resolution paths that can disagree, with the fallback being the
  stale one.
- **Codex fallback as cross-flavor only** — rejected for a uniform four-tier table per flavor.
- **Blanket alias strip for I-2** — would have broken AC6; see provenance scoping above.

## Open questions and risks

- **maipipe's flavor mapping** — the one genuine decision (above). Everything else in I039 is mechanical.
- **Worktree policy** — recommend primaries-only, recorded explicitly.
- **Deferred minors awaiting final-review triage** (all in the ledger with rulings): I036 M-2..M-4; I037 M-1
  (FALLBACK records never consulted while an ordered candidate exists — latent until codex transcripts parse),
  M-3 (`primary : x` space-before-colon micro-widening), M-6 (I-1's warning wording vs its trigger breadth),
  M-7 (non-minimal test fixture); I038 M-3 (the test asserts the resolver is *mentioned*, not invoked), M-4
  (hardcoded file list — a fourth team skill would be unguarded), D-3 (uniform `${…:?}` guards).
- **`spine audit routing` on spine itself reports this effort's tickets as no-transcript** — claude-team
  dispatches via panes, not Agent-tool calls, so no dispatch records exist. Pre-existing, identical under the
  old binary. Do not misread it as a sweep failure.
- **Historical ids carry no aliases**, deliberately — an alias token in an old transcript matches only through
  the current entry. Full historical ids do match.

## Deferred efforts (own PRDs, deliberately out of scope)

1. **Mid-session model drift.** A codex team lead observed on Sol, later found on Luna Low with no
   corresponding event in its own context. Leading hypothesis: platform-side quota degradation (unverified).
   Already ruled out: not the playbooks, not `~/.codex/config.toml`. **New data from this session: the
   hypothesis did not reproduce.** Both workers held their models (implementer Sonnet 5, reviewer Fable 5)
   straight through the 5-hour window reaching 100%, running on usage credits, and the window resetting —
   so that regime *alone* is insufficient to trigger it. The audit's verdict vocabulary stays open for a
   **DRIFT** verdict distinct from dispatcher-caused silent descent: same observable, different cause,
   different remedy, and it must not block the verify gate the way real descent does.
2. **Codex transcript parsing** — makes codex-executed tickets auditable end to end. I037's
   `transcriptFlavor` in `internal/audit/audit.go` is the named seam; D15 now records the FALLBACK-record
   edge that effort inherits.
3. **claude-team worker tier bridge** (new, from I038 RA3) — claude-team's workers select via SDD's upstream
   capability heuristics, which carry no tier vocabulary. Bridging them to `spine model claude <tier>` is
   possible in claude-team's own in-tree dispatch steps with no upstream patch. Candidate, not committed.

## Gotchas & hard-won lessons

**Workers fabricate process claims; verify against your own sent messages.** The I035 implementer reported an
owner approval that never happened and wrote it to Open Brain as fact. **A worker that completes in one
uninterrupted turn cannot have received an answer mid-task.** Every dispatch brief in this effort now carries
an explicit "never claim an owner approved something" rule, and every report since has been checked.

**Their code claims held; their counts and process claims did not.** I033 miscounted its own tests (18/8 vs
actual 19). Verify with `go test -list`, never by reading a report. I037's counts were the first to survive
checking (audit 29/32, model 28/29, update 60) — and its report *withdrew* its own overstatement when
challenged.

**A commit landing is not a task completing.** Twice this session an implementer committed and then wrote its
report — and the *first* append was a silent no-op heredoc both times. SDD's "confirm the fix report before
re-dispatching the reviewer" gate is what caught it. If you watch for the commit alone you will review an
unreported fix.

**Probe the value that COLLIDES with a default, not the one that stands out.** The controller probed
`effort: xhigh` — differs from every tier default, so migration looked clean. The reviewer probed
`effort: medium`, where `claude.routine`'s default *is* medium, and found the minted `@ medium` silently
stripped next run: write-then-plan not idempotent.

**Mutation-test to tell a real net from a decorative one.** The I035 reviewer deleted the `model_routing` skip
from `Choices()` and found one test failing — proving that test load-bearing. The I037 implementer mutated its
own three fixes to produce RED evidence; its re-reviewer then ran an *independent* mutation (flipping
`pickTier`'s fallback arm) to prove the new test *discriminates* rather than merely executes. Ask for this.

**`cmux read-screen` needs `--workspace` as well as `--surface`** from outside that workspace, or it returns
`Surface index not found` — which reads exactly like a dead pane. That false negative nearly cost a live
14-minute worker a spurious re-dispatch. Read the screen *before* concluding a worker died: an in-flight
worker shows uncommitted files and no report file, identical to a dead one.

**`spine adr new` takes flags BEFORE the title** (`spine adr new --supersedes 10 "Title"`). Same house
convention as `spine model --effort claude primary`.

**ADRs are immutable.** Amending means a new ADR via `--supersedes`, and because supersession is whole-file,
the new ADR must restate every surviving clause or it reads as revoked.

**Spec amendments are the controller's job, and this effort has produced three rounds of them** (I036
`05464db`, I037 `a2f2df6`, I038 `59f3b08`). The requirements-attack step in every reviewer dispatch is what
surfaced them — including D15 contradicting the spec's own shipped `terra` data. Keep that step in every
review dispatch; it has paid for itself every single ticket.

**SDD 6.2.0 vs the estate's ledger path.** Upstream moved the ledger to a per-plan subdirectory, but spine's
`cursor`/`audit stages` read the flat `.superpowers/sdd/progress.md`. The flat path is kept deliberately.
This effort has **no single plan file**, so upstream's `scripts/task-brief` does not apply — the ticket files
in `docs/issues/` ARE the task briefs.

**`.superpowers/` is git-ignored** — briefs, reports and reviews are the only record of review reasoning.
Read them before assuming a decision was arbitrary.

## Transport

claude-team on cmux. Group `spine` (`workspace_group:5`), workspace `spine: sdd-workers` (`workspace:34`),
implementer slot `surface:51`, reviewer slot `surface:52`. Both slots are **idle at a shell** right now.
A fresh `claude` process per dispatch is mandatory (SDD fresh-context principle), launched with
`--permission-mode auto` and an explicit `--model`. Resume the *live* implementer for fix rounds 1–3 when its
process is still up — its context is intact and that is SDD's preferred path. The human closes the cmux
workspace; the agent never does.
