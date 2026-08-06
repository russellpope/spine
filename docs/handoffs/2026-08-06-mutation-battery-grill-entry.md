# Handoff — mutation-battery, entering grill (2026-08-06)

A transcript audit of the estate produced one finding with a reproducible experiment
behind it: **a behavioural mutation battery discriminates test strength where `go test`
pass/fail does not.** The experiment is done, independently reproduced, and already
survived one adversarial review that corrected four overclaims. Nothing is committed.
No PRD exists. This effort enters at **grill**.

Incoming session: you are the **primary tier** (`spine model claude primary` →
`claude-fable-5`). The audit that produced this ran on `claude-opus-5`, which is the
**fallback** tier — design and judgment work was done off-tier. That is the first thing
the grill should discount for: the framing below is fallback-tier work and should be
attacked, not inherited.

## Stage cursor

<!-- spine:cursor -->
effort: mutation-battery
prd:
tickets:
stages: grill[<] prd[ ] issues[ ] implement[ ] functional-test[ ] review[ ] verify[ ] ship[ ] deploy[ ] docs[ ] handoff[ ]
<!-- /spine:cursor -->

`prd:` and `tickets:` are empty on purpose. No PRD, no tickets. `spine audit stages`
is clean apart from this handoff being the newest one (which this file resolves).

## The experiment

Eight behaviour-changing mutations per tree; apply → build must still succeed → run
`go test ./...`. Suite red = KILLED (detected), green = SURVIVED (blind spot). Build
breaks are excluded as invalid probes.

| Tree | State | Kill rate |
|---|---|:---:|
| claude-code-opus-4.7 | as-submitted (30/30) | **2/8 = 25%** |
| gpt-5.5 | post-remediation r1 (26→29) | **2/8 = 25%** |
| laguna-s-2.1 (local, 118B) | post-remediation r4 (18→22) | **5/8 = 62%** |

Killed everywhere: classifier→constant, sort reversed. Survived everywhere: column
order, TLS verification disabled, session logout deleted.

Reproduce: `python3 scripts/mutate.py <tree> scripts/<model>.json`. All three rates were
independently reproduced by a fable-5 reviewer on fresh copies, 2026-08-05.

## Corrected claims — settled, do NOT re-litigate

An adversarial fable-5 review already caught these. They are fixed in the research doc.

1. Frontier survivors are **undetectable-change classes, not shipped defects.** The Opus
   and GPT-5.5 binaries do not have TLS disabled or logout removed.
2. **p = 0.315** (Fisher exact, 2/8 vs 5/8). The design cannot distinguish 25% from 62%.
   Treat every rate as a point estimate motivating a bigger experiment.
3. Opus's six survivors **collapse to one cause** — all in `cmd/root.go`, zero tests
   there. Not six findings. GPT-5.5 (flat package, co-located tests) is what makes the
   class taxonomy meaningful.
4. Checklist in spine `templates/` is **not** zero-code — ADR 0004 compiles templates
   into the binary behind an integer generation, plus a 17-repo fleet refresh.
5. ADR 0007 (`stage`/`score` opaque) — **verified accurate.** A kill-rate field can ride
   `spine eval` with no spine code change.
6. B7 (TLS) / B8 (logout) may be **near-untestable** in the eval's prescribed
   `simulator.Test` style. They supply the "universally blind" finding; that finding may
   not survive scrutiny.

Reviewer error, checked and rejected: it claimed the `fileContent` count was 177.
Re-queried — single-session **160**, corpus total **180**. The 25.8% write-failure rate
it confirmed exactly.

## The decision the grill exists to make

**Mutation sites are hand-authored per tree. Unsolved.** This is the hinge: it decides
whether the battery is a manual instrument (useful, occasional, ~5 min/tree) or
infrastructure that can run unattended at 24×7 scale. Every other open question is
downstream. Do not let the grill spend itself on the checklist's contents while this
sits unanswered.

Secondary, genuinely open:
- Adopting this as a gate means **every tier** fails it initially, including frontier.
  That is real added work everywhere. Worth it? (The prior session's "do no harm"
  conclusion defined harm as false-positives-only and declared victory — too narrow.)
- Per-class kill rate vs a floor on total. A floor invites padding the cheap classes.
- Should the gate score *classes* or *distinct causes*? Scoring classes rewards a tree
  that spreads one weakness across many files (see correction 3).

## Packaging position (fallback-tier opinion — attack it)

Avoid a super-binary; keep spine's namespace uncrowded and the artifact redistributable.

| Piece | Form | Home |
|---|---|---|
| The 11-class checklist | Markdown | `docs/` — **not** `templates/` (ADR 0004) |
| The runner | ~60-line Python, JSON in / result out | standalone, own tiny repo |
| Kill-rate record | one field | rides `spine eval` opaque score, no code change |

Unresolved: "one **required** field" — required by *what*? `spine doctor` D7 validates
eval-record shape only. Nothing enforces field presence today.

## Artifacts

- `docs/research/2026-08-05-behavioural-mutation-battery.md` — findings, 11-class
  checklist, do-not-regress block template, packaging, open questions. Marked DRAFT.
- `scripts/mutate.py` + `scripts/{opus,gpt55,laguna}.json`
- `.superpowers/sdd/progress.md` — this effort's ledger (gitignored, local only)
- Overnight batch queued: see `scripts/overnight/README.md`

[Paths relocated 2026-08-06 by I056 — see docs/research/2026-08-06-mutation-battery-repro/overnight-README.md]

**Nothing is committed.** `docs/research/` and `scripts/` are untracked; no tracked file
in spine has been modified. The eval corpus in `local-model-evaluation` was never
modified — all mutation work ran on copies.

## Provenance of the wider audit (not carried into this effort)

The transcript audit surfaced more than the battery. Unticketed, in the research doc's
lineage only: opencode local-model corpus (66 sessions, 5,142 tool calls) showing a
25.8% `write`-tool failure rate concentrated in one model emitting `fileContent` instead
of `content` 160 times without adapting; heredoc bypass of the write tool under tool
friction; and the finding that auditor-prescribed fixes produced fakes where
self-prompted fixes produced real ones (agentworld 16→19 prescribed vs 19→23
self-prompted; ornith-35b 16→25 fully self-prompted). Those are worth their own effort;
they are **not** in scope here.
