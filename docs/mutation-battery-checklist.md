# Behavioural mutation battery — checklist

**Status:** convention, adopted as a **reporting gate** (grilled 2026-08-06 — see
`docs/research/2026-08-05-behavioural-mutation-battery.md`, *Grill outcomes*)
**Consumer:** the `/model-eval` skill, for evals with `functional_harness: cli`

This document is the normative checklist for the behavioural mutation battery. It
does not live in `templates/` — see ADR 0013.

## Superseded by `spine gate go mutate`

The runner is now the spine binary: `spine gate --dir <tree> go mutate` reads the
per-tree spec (`docs/mutation-spec.json` by default, or the path in
`SPINE_GATE_MUTATE_SPEC`), copies the tracked tree, applies each probe to the copy,
and emits one results-contract row per probe (`code = go@1/mutate`) plus both kill
rates — running the unmutated negative control first. It ships as the `mutation-go`
pipeline (profile `audit`) in the gate pack's `maipipe.toml` region. See ADR 0015,
which supersedes ADR 0013 items 2 and 4; the Python runner bundled with the
`/model-eval` skill is superseded by the binary.

This document stays normative for the ten classes, the record format and the
reporting rules below (ADR 0013 items 1 and 3 stand) — the binary executes them,
it does not redefine them.

## What this is

The battery is an **agent-assisted instrument**, not an automated gate with a pass
threshold. For each mutation class, a site is authored per tree (candidates from
`sites.sh`, an agent pass writes the exact literal spec), then a mutation runner
applies the mutation, rebuilds, and re-runs the tree's suite. The suite going red is
a **KILL** (detected); staying green is a **SURVIVED** (blind spot).

## The ten runnable classes

Provenance marks are part of the convention and must be carried verbatim wherever
this checklist is reproduced or referenced:

- **[report-only]** — the class runs and is recorded, but is **excluded from the
  scored denominator**. These classes are near-untestable in the prescribed harness
  style (see Finding 3 of the research doc); a future tree that kills one is signal,
  but its survival is not counted against a tree.
- **[CANDIDATE]** — reasoned extrapolation with **no probe data yet**. These classes
  graduate to fully-scored once a wired tree actually runs them.

1. **Invocation** — replace a unit's body with a plausible constant. Is it called at all?
2. **Wiring** `[CANDIDATE]` — delete a flag registration / binding. Is the wiring tested, or re-created by the test?
3. **Flag honoured** — ignore a flag's *value* while leaving it registered.
4. **Column presence** — delete a column from header and rows.
5. **Column order** — swap two adjacent columns of the same type.
6. **Ordering** — reverse a documented sort.
7. **Units / labels** — change a unit without changing its label.
8. **Security default** `[report-only]` — flip TLS verification / auth to the unsafe value.
9. **Lifecycle** `[report-only]` — delete cleanup (logout, teardown, `defer`, context cancel).
10. **Error-path behaviour** `[CANDIDATE]` — turn a required per-row *degrade* into an abort, and vice versa.

### Former class 11 — not a battery entry

Fixture strength ("is a wrong answer *reachable* by the fixture?") is not
mechanically checkable by the runner. It is a **reviewer instruction** applied at
the review/verify stage, not a battery class: a reviewer checking a mutation-battery
record should separately ask whether the fixtures in use could even expose a wrong
answer for the classes above. It does not appear in the verdict matrix and does not
factor into the scalar.

## The record format

A battery result must carry:

- The **full per-class verdict matrix** — KILL / survive / report-only / no-site —
  for all ten classes. This is the mechanical ground truth and must not be
  summarized away.
- A required one-line **distinct-cause summary** for survivors, authored by the
  agent pass (e.g. "6 survivors, 1 cause: `cmd/` has zero tests"). Class counts
  overstate granularity when survivors share one structural cause — the summary
  exists to prevent that overstatement, not to decorate the record.
- The scalar, if one is quoted, is **killed / valid-scorable-probes**. Report-only
  rows (classes 8/9) and build-breakers are excluded from both numerator and
  denominator.

## Reporting rules

- **No pass threshold.** No submission fails on kill rate; low rates drive
  remediation by judgment, not by gate. This is a deliberate decision — see the
  research doc's *Grill outcomes*, item 2.
- **Presence is required by the `/model-eval` skill's process, not by ordinary
  tooling.** I077 adds a narrow exception for a run explicitly cited as host-pin
  evidence: its front matter must state `battery_version: 1`, a matching
  `battery_verdict: pass` or `fail`, and the exact ten-key comma-separated
  `battery_results` matrix. Doctor reports a cited missing, malformed, or failing
  record as advisory D17 evidence only. It does not impose a threshold, block a
  pin, or change ordinary eval handling.
- **A mutation that breaks the build is not a kill.** It is an invalid probe,
  excluded from the denominator, and must be disclosed in the record (a
  `BUILD-ERR` row), not silently dropped.
- **Gaming is dissolved by construction:** there is no threshold to game, and
  sites are authored fresh per tree rather than reused.
