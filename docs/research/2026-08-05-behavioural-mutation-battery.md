# Behavioural mutation battery — findings, checklist, and packaging

**Date:** 2026-08-05 · **Kind:** experiment + proposed convention (pre-PRD) ·
**Status:** GRILLED 2026-08-06 (primary tier) — decisions ratified (see *Grill
outcomes*); n=5 table reproduced from clean copies the same day (see *Reproduction*)
**Related:** `local-model-evaluation` (12-run corpus),
`laguna-s-2.1/HITLIST-round3.md` (origin of the mutation classes),
[I050](../issues/I050-approved-untested-acceptance-lane.md) (acceptance lanes),
`docs/harness-interface.md` (`functional_harness: cli`)

## Question

The estate's verify gate accepts a green test suite as evidence. Local-model runs
repeatedly shipped green, zero-skip, `-race`-clean suites over broken binaries. Does
a behavioural mutation battery discriminate better than pass/fail — and does adding
it **harm** frontier-model workflows?

## Method

Eight behaviour-changing mutations, one per class, applied to three submissions from
the govmomi CLI eval. Each mutation: apply → `go build` (must still compile) →
`go test ./...`. Suite goes red = **KILLED** (detected). Suite stays green =
**SURVIVED** (blind spot). Mutations that break the build are excluded as invalid
probes. Runner: `mutate.py` (~60 lines, JSON spec, works on any tree with a
build+test command; now maintained in the `/model-eval` skill's `scripts/`).

All three trees were verified green at baseline before mutation. Work was done on
copies; no eval artifact was modified.

## Results

Extended to n=5 on 2026-08-06. All rates in this table are **raw 8-probe rates**
(B7/B8 included in the denominator) — the convention's *scorable* scalar
(killed / valid-scorable, report-only probes excluded) reads e.g. 2/6 = 33%
where this table says 2/8 = 25%. The runner emits both; historical numbers stay
raw for comparability.

| Model | Tree state | Kill rate | Killed |
|---|:---:|:---:|---|
| Claude Opus 4.7 (frontier ref) | as-submitted, 30/30 | **2/8 = 25%** | B1, B5 |
| GPT-5.5 (frontier) | post-remediation r1 (26→29) | **2/8 = 25%** | B1, B5 |
| ornith-1.0-397B (open-weight cloud) | post-remediation (22→28) | **2/8 = 25%** | B1, B5 |
| **Laguna S 2.1** (local 118B) | **post mutation-driven r4** (18→22) | **5/8 = 62%** | B1, B2, B4, B5, B6 |
| qwen-3.6-27b (local) | as-submitted, 16/30 | **0/8 = 0%** | — |

**Three independent trees land on exactly 25% with an identical kill pattern** (B1
invocation and B5 ordering die; everything else survives). Grill resolution
(2026-08-06): the identical *rate* is arithmetic, not anomaly — an identical kill
pattern forces an identical rate, and exactly two of the eight probes are value-class.
The finding is the *pattern's universality*: three independently authored suites share
the same blind-spot structure, which is Finding 4's diagnosis ("tests check values,
not the program"), not a new result. Strong harness-determinism is already refuted at
zero cost — Laguna reached 62% inside the same prescribed harness. Why independently
trained models converge on value-only testing is out-of-scope research residue.

**Sample-size caveat (binding on every claim below):** eight mutations, n=1 tree per
cell, probes **not independent** (see Finding 1). At n=3 the Laguna-vs-frontier contrast
was p = 0.315 (Fisher exact, 2/8 vs 5/8) — not significant. n=5 strengthens the pattern
but does not repair the design: these are point estimates motivating a larger
experiment, not measured facts. No significance claim is made.

**qwen-3.6-27b is the cleanest single data point in the corpus.** Its suite is green,
zero-skip, `-race` clean, with non-trivial coverage (config 39.4%, inventory 65.3%) —
and it detects **zero** of eight behaviour changes, including its own classifier being
replaced by a constant. Its eval record independently notes the binary fails to log in
on every subcommand. Green suite, real coverage, no detection.

Per-mutation:

| # | Class | Mutation | Opus | GPT-5.5 | Laguna |
|---|---|---|:---:|:---:|:---:|
| B1 | invocation / fabrication | classifier short-circuited to a constant | KILL | KILL | KILL |
| B2 | flag honoured | `--portgroup` silently ignored | **survive** | **survive** | KILL |
| B3 | column order | USED / AVAILABLE swapped | **survive** | **survive** | **survive** |
| B4 | column presence | LACP column deleted | **survive** | **survive** | KILL |
| B5 | ordering | VM sort reversed | KILL | KILL | KILL |
| B6 | units / labels | RAM off by 1024× | **survive** | **survive** | KILL |
| B7 | security default | TLS verification disabled | **survive** | **survive** | **survive** |
| B8 | lifecycle | session `Logout` deleted | **survive** | **survive** | **survive** |

### Reproduction (2026-08-06)

Spine's then-current `scripts/overnight/run-batch.sh` (since relocated — see
`docs/research/2026-08-06-mutation-battery-repro/overnight-README.md`; future
re-runs use the `/model-eval` skill's `run-battery.sh` with a manifest) re-ran
all five trees from fresh scratch copies:
every kill rate and kill pattern reproduced exactly — 25/25/62/25/0, with **0 no-site
and 0 build-err** rows across all 40 probes. Evidence:
`docs/research/2026-08-06-mutation-battery-repro/combined.txt` (+ per-tree logs). This
closes the "not independently reproduced" caveat on the two n=5 additions
(ornith397b, qwen36-27b); qwen's load-bearing 0/8 is confirmed.

## Findings

### 1. Opus's six "blind spots" are one structural fact counted six times

All six Opus survivors live in `cmd/root.go`; both kills live in `internal/inventory`.
Opus has tests **only** under `internal/`, and `cmd/` is an unimportable near-main
package with zero tests. The honest statement is *"no test can observe Opus's command
layer"* — not six independent class-level findings. The per-class table overstates
granularity for that tree.

GPT-5.5 partially rescues the class framing: its tree is a **flat single package with
co-located tests**, so placement cannot explain its survivors — there the same six
classes measure genuine assertion weakness. The class taxonomy is therefore meaningful,
but Opus is not evidence for it.

### 2. What remediation-driven test strength does and does not follow from

Laguna (62%) went through four rounds explicitly driven by **mutation batteries**
(kill rates 80/52/37.5% across rounds 2–3) plus the round-3 structural fix extracting
rendering into `internal/format`. GPT-5.5 also went through remediation (r1, 26→29)
and still scores 25% — but its remediation was **audit-finding-driven**, not
mutation-driven, and its added `--portgroup` test sits below the flag wiring, so B2
still survives.

The defensible claim is narrower than "process beats model tier": **mutation-driven
remediation builds test strength; ordinary finding-driven remediation does not.**
Confound remaining: Laguna's killing sites exist *because of* the prior rounds, so
this partly re-measures "did remediation add integration tests." n=1 per cell.

### 2b. What the frontier result does NOT say

The submitted Opus and GPT-5.5 binaries do **not** have TLS disabled or logout removed.
These are hypothetical mutations. The finding is that **their suites could not detect
such a change if it were made** — an undetectable-change class, not a shipped defect.
Earlier phrasing conflated the two; corrected here.

### 3. Two classes are universally blind — security defaults and lifecycle

B7 (TLS) and B8 (logout) survive on **all three** submissions regardless of tier, as
does B3 (column order). These are not local-model weaknesses.

**Weakened by review:** within the eval prompt's prescribed `simulator.Test` style,
the client-construction path (B7) and command-level teardown (B8) are near-untestable
— so universal survival may reflect an untestable seam rather than a suite gap. B3
survives it: Laguna's `format_test.go:150-159` asserts USED-before-AVAILABLE in the
*header* only, while the mutation swaps *row values* — a row-level golden line would
catch it. B3 is a genuine, testable blind spot; B7/B8 need a harness change before
they can be graded, and should not be scored until then.

Grill resolution (2026-08-06): B7/B8 stay in the battery as **report-only, unscored**
rows — they cost nothing per run, and a future tree that kills them is signal. The
harness-style question is filed against the eval repo as future work, outside this
effort. The "universally blind classes" claim is downgraded accordingly: B3 is the
only demonstrated universal blind spot that is testable in the prescribed style.

### 4. Value mutations are caught; behaviour mutations are not

B1 and B5 — the two that change a *computed value* or an *observable ordering the
tests already assert* — die everywhere. Everything touching wiring, rendering,
security posture, and lifecycle survives. This reproduces the round-3 hitlist's
one-line diagnosis exactly: *"your tests check values, not the program."*

## Do-no-harm assessment

The stated risk was that a battery tuned to local models would burden frontier
workflows with **false positives**. On that definition: zero false positives occurred
— every SURVIVED verdict is a real undetected behaviour change, spot-checked by hand
(with the B7/B8 caveat in Finding 3).

**That definition is too narrow, and the do-no-harm question is not settled.** A gate
that every frontier submission fails at 25% imposes mandatory remediation rounds on
frontier workflows — which is precisely the burden the question was asking about,
defined out of the assessment. Honest statement: the battery does not *misfire* on
frontier work, but adopting it as a gate **would** add real work at every tier. Whether
that work is worth it is a judgment for the grill, not a finding of this experiment.

Grill resolution (2026-08-06): adopted as a **reporting gate** — presence required, no
threshold. Frontier burden drops to ~10 min/tree (site authoring + run) with zero
forced remediation rounds; the qwen failure mode stays detectable everywhere. A
threshold is revisited only after cause-annotated results accumulate across evals.

Scope of "suite": the battery ran `go test ./...` only. The eval's `scripts/verify.sh`
(vcsim smoke, present in some eval trees — the Opus tree's was read; not
relocated by I056) was read and
would not catch any of the six Opus survivors either — it
asserts exit codes plus one portgroup-name extraction — so the numbers stand, but the
method is `go test`-only by construction.

Costs measured: 8 mutations × (build + full suite) ≈ 3–6 min per tree on this corpus.
Acceptable for a verify-stage gate; too slow for an inner TDD loop.

## Convention (grilled 2026-08-06 — adopted as a REPORTING gate)

The battery is an **agent-assisted instrument** for the `/model-eval` loop, and its
adoption form is a **reporting gate**: for `functional_harness: cli`, an eval record
must *carry* the battery result. There is **no pass threshold** — no submission fails
on kill rate; low rates drive remediation by judgment. (The 24×7/unattended framing
from the original audit is retired: no consumer for it exists anywhere in the estate.
Site authoring stays per-tree — `sites.sh` candidates, an agent pass writes the exact
literal spec, and `mutate.py`'s `NO-SITE`/`BUILD-ERR` rows validate it mechanically.
AST-based discovery is deferred until an unattended consumer is named.)

### The CLI behavioural mutation checklist

Ten runnable classes. Provenance marks are part of the convention: **[report-only]**
rows run but are excluded from the scored denominator (near-untestable in the
prescribed harness style, Finding 3); **[CANDIDATE]** classes are reasoned
extrapolation with no probe data yet — they graduate when a wired tree runs them.

1. **Invocation** — replace a unit's body with a plausible constant. Is it called at all?
2. **Wiring** [CANDIDATE] — delete a flag registration / binding. Is the wiring tested, or re-created by the test?
3. **Flag honoured** — ignore a flag's *value* while leaving it registered.
4. **Column presence** — delete a column from header and rows.
5. **Column order** — swap two adjacent columns of the same type.
6. **Ordering** — reverse a documented sort.
7. **Units / labels** — change a unit without changing its label.
8. **Security default** [report-only] — flip TLS verification / auth to the unsafe value.
9. **Lifecycle** [report-only] — delete cleanup (logout, teardown, `defer`, context cancel).
10. **Error-path behaviour** [CANDIDATE] — turn a required per-row *degrade* into an abort, and vice versa.

Former class 11 (fixture strength — is a wrong answer *reachable* by the fixture?) is
not mechanically checkable by this runner and moves to a **reviewer instruction** at
the review/verify stage; it is no longer a battery entry.

### What the record carries

- The **full per-class verdict matrix** (KILL / survive / report-only / no-site), the
  mechanical ground truth.
- A required one-line **distinct-cause summary** for survivors, authored by the agent
  pass (e.g. "6 survivors, 1 cause: `cmd/` has zero tests"). Finding 1 is why: class
  counts overstate granularity when survivors share a structural cause.
- The scalar, if one is quoted, is **killed / valid-scorable-probes** — report-only
  rows and build-breakers are out of the denominator.

Reporting rules: a mutation that breaks the build is **not** a kill — it is an invalid
probe, excluded from the denominator and disclosed. Presence of the battery result is
required by the `/model-eval` skill's process, not by tooling; a `spine doctor` check
is explicitly out of scope unless a threshold is ever adopted. Gaming ("write tests
that kill exactly these probes") is dissolved by construction: there is no threshold
to game, and sites are authored fresh per tree.

### The do-not-regress block

Every remediation dispatch must be prepended with a generated block listing what the
previous round **verified working**, with the file:line and the probe that proves it.
Rationale: ornith-35B introduced a latent ordering bug in round 1 and a firing
`t.Skip` in round 2; Laguna's round-2 gains were "paid for by a regression." Both
were self-inflicted during fix rounds.

```
## 0. DO NOT REGRESS  (generated from round <N-1> verified criteria)

- <file:line> — <behaviour> — proven by <test/mutation id>
- ...
Breaking one of these costs more than any fix below gains.
Report any that you must break, and why, before you break it.
```

## Packaging (grilled 2026-08-06)

Avoid a super-binary. Three separable pieces:

| Piece | Form | Home | Why |
|---|---|---|---|
| The checklist | Markdown | spine **`docs/` — NOT `templates/`** | Convention document; ADR 0004 rules out `templates/` |
| The runner | ~60-line script, JSON spec in / result out | **bundled with the `/model-eval` skill** | Colocated with its only consumer (grill Q1); a standalone repo was rejected — no named external customer |
| Per-tree specs | JSON | **`local-model-evaluation/docs/evals/2026-07-02-govmomi-vsphere-inventory-cli/mutation-specs/`** | Eval artifacts, not tooling; they belong beside the eval records they describe, not in spine |
| The battery record | matrix + cause line riding the eval record | no spine code change | `spine eval` treats `stage`/`score` as opaque strings (ADR 0007 — verified) |

An earlier draft proposed the runner get "its own tiny repo" for redistributability;
the grill rejected that — the redistribution story had no actual customer, and the
estate does not need a 19th repo for 60 lines of Python.

**Correction:** an earlier draft proposed shipping the checklist in spine `templates/`
as a "zero-code" change. That is false under **ADR 0004** — templates compile into the
binary behind a single integer generation, so adding one is a code change plus a
generation bump plus a fleet refresh across 17 repos. If the checklist must be
spine-distributed, `docs/` avoids the generation machinery; a standalone home avoids
spine entirely and is more redistributable.

Resolved by the grill: the field's presence is **required by the `/model-eval`
skill's process**, not by tooling. This is genuinely zero spine code. An earlier
draft's "one required field" conflated requiredness with enforcement — a `spine
doctor` presence check is a code change and is out of scope unless a pass threshold
is ever adopted.

## Grill outcomes (2026-08-06, primary tier)

Every open question from the draft was put to the owner; all are closed. Decisions:

1. **Identity (the hinge).** Agent-assisted instrument for the `/model-eval` loop.
   Sites stay per-tree: `sites.sh` candidates → agent-authored literal spec →
   `NO-SITE`/`BUILD-ERR` mechanical validation. AST discovery deferred — the "24×7"
   requirement was audit-invented; no doc in the estate names an unattended consumer.
2. **Adoption form.** Reporting gate: result presence required, no pass threshold.
   Nobody fails on kill rate; the do-no-harm question is answered honestly (near-zero
   frontier burden) instead of defined away.
3. **Scoring unit.** Per-class verdict matrix + required distinct-cause summary for
   survivors; scalar = killed/valid-scorable. Classes-vs-causes and floor-padding
   both dissolve: the matrix is the record, causes are annotated, no floor exists.
4. **B7/B8.** Report-only rows, kept; harness question filed against the eval repo.
5. **Identical 25%.** Closed as arithmetic (pattern forces rate); pattern universality
   is Finding 4. See Results.
6. **Unprobed classes 2/10.** Marked CANDIDATE until a wired tree runs them.
7. **Class 11.** Moved to a reviewer instruction; battery is 10 runnable classes.
8. **Gaming.** Dissolved by construction — no threshold, sites authored fresh per tree.
9. **Sample size.** Not repaired this effort; every number remains a point estimate.
   The reporting gate accumulates cause-annotated matrices across future evals — that
   corpus, not a bespoke experiment, is the larger design.
10. **Do-not-regress block.** Ships in the same PRD as a second named deliverable
    (template + dispatch-prep instruction in the `/model-eval` skill).
11. **Reproduction.** The n=5 table must be reproduced by `scripts/overnight/
    run-batch.sh` (spine's batch script at the time; since relocated by I056)
    from a clean checkout before this doc drops DRAFT; qwen's 0/8 is
    load-bearing for adoption and does not rest on one session's word. **Satisfied
    same day** — see *Reproduction* above; DRAFT dropped.
