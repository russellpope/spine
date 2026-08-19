---
id: "0015"
title: "gate-pack checks live in the spine binary, versioned per pack, delivered as maipipe pipelines"
status: Accepted
date: 2026-08-18
supersedes: "0013"
---

# 0015: gate-pack checks live in the spine binary, versioned per pack, delivered as maipipe pipelines

## Context

The weak-local-model harness (deepthought map I023; PRD
`deepthought/docs/specs/2026-08-17-spine-local-harness-conventions-design.md`,
grilled into spine 2026-08-18) needs an enforcement floor: a battery of
deterministic checks that maipipe runs against a repo, with clean per-finding
attribution and per-repo opt-out. The PRD text said "spine scaffolds check
scripts + a `maipipe.toml` snippet".

Two facts made that shape awkward. Several checks (dead-code call-graph,
test-enum-vs-spec diff, mutation battery, results-contract JSON emission)
are not shell-sized; scaffolded scripts would be Go or Python sources
regenerated wholesale in every repo — drift surface, doctor surface, and a
second copy of the pack per repo. And spine's template mechanism (ADR 0004)
ties every scaffolded file to one integer generation, so a check tweak would
force a fleet-wide `spine update`.

ADR 0013 (2026-08-06) had placed the mutation battery's runner in the
`/model-eval` skill as Python and ruled out spine enforcement code "without a
threshold". Since then the battery acquired a consumer that needs
machine-readable killed-row output inside the results contract: the
remediation hitlist's do-not-regress block (I033). The owner's stated
direction (2026-08-18) is to lean into extending spine's commands and building
CLIs for determinism rather than shipping scripts.

## Decision

1. **Checks are subcommands of the spine binary**: `spine gate <pack> <check>`
   (Go pack first: `tskip`, `deferred-cleanup-errcheck`, `dead-code-callgraph`,
   `test-enum-vs-spec`, `n-plus-one`, `binary-hygiene`, `gitignore-control`,
   `fixture-manifest`, plus `mutate`). No check scripts are scaffolded into
   repos.
2. **Packs are versioned per pack** (`go@1`), independent of the template
   generation. A finding is attributable as `<pack>@<v>/<check>`; the results
   contract's `code` field carries that string.
3. **Delivery is a pair of maipipe pipelines** (`gate-go` for the check
   classes, `mutation-go` for the battery), one stage per check class, emitted
   into a spine-managed region of `maipipe.toml` (ADR 0016) and composed into
   the repo's own lanes via maipipe's `pipeline = "…"` reference. Opt-out and
   per-check config live in preserved WORKFLOW.md keys (`gate_pack`,
   `gate_pack_disabled`, `gate_pack_config`); the rendered region omits
   disabled classes.
4. **Every check class ships with a positive control in spine's own test
   suite** (known-good input passes, seeded violation fails). Per-repo, doctor
   checks nothing beyond region integrity.
5. **The mutation battery is ported to Go under the pack** as `spine gate go
   mutate`, run in maipipe's audit lane (advisory, never blocking routine
   pushes; per-repo promotion allowed). This reverses ADR 0013 items 2 and 4.
   ADR 0013 items 1 and 3 stand: the checklist stays in `docs/`, and the eval
   record's Audit/Rescore body remains the battery's home in eval runs.

## Consequences

- maipipe stages depend on `spine` being on the daemon's PATH (already true
  for the cursor hook). Recorded for maipipe as a cross-product ticket.
- Pack version bumps are spine releases, not template generations; a repo
  pins `gate_pack: go@1` and moves deliberately.
- Attribution strings are stable across repos and hosts, which is what the
  remediation round record and telemetry key on.
- The `/model-eval` skill's Python runner remains until the port lands
  (its own ticket); the skill then calls the binary.
- Reversing this (back to scaffolded scripts, or a standalone checks repo)
  requires a superseding ADR.
