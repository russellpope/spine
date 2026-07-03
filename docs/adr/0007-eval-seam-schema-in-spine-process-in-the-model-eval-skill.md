---
id: 0007
title: eval seam: schema in spine, process in the model-eval skill
status: Accepted
date: 2026-07-02
---

# 0007: eval seam: schema in spine, process in the model-eval skill

## Context

v2 needed `docs/evals/` to support the model-eval workflow (wire an eval, audit it, score it,
compare runs, remediate, rescore) without spine's Go code becoming the place where eval logic
lives. Baking stage names or scoring rules into the binary would mean every loop-process tweak
is a code change and a release, and it would put spine in the business of judging eval content
it has no way to validate meaningfully.

## Decision

spine owns the structure only: `eval new`/`add-run`/`list` scaffold versioned templates
(`eval.md`, run records, the evals `README`) and doctor's D7 check validates shape — files
present, front matter parseable — never content. Stage and score fields in run records are
opaque strings; no Go code branches on their values. The `/model-eval` skill owns the process —
it drives wire -> audit -> score -> compare -> remediate -> rescore and is the only thing that
writes stage/score values into the records spine scaffolds.

## Consequences

The eval loop can change — new stages, different scoring conventions — by bumping the run
template, not by touching `internal/eval` or `internal/doctor`. spine stays a dumb, reliable
scaffolder and validator; the skill stays free to iterate on methodology without a spine release.
The seam is enforced by the "opaque strings" global constraint, so any future PR that adds a
`switch` on stage/score content is a regression against this ADR.
