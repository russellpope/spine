---
id: I083
title: "Gate classes: gitignore-control, fixture-manifest, test-enum-vs-spec"
severity: med
status: fixed
affects: [gate]
blocked-by: [I082]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md` (Gate
pack — check definitions). ADR 0015.

## What to build

Three config-driven check classes on the I082 skeleton, each reading its
inputs from environment variables that the `maipipe.toml` region will later
supply from `gate_pack_config`: `gitignore-control` — arm 1: every declared
build output path is ignored *at that path*; arm 2: no `package main`
source file is ignored (the hidden-entry-point control); `fixture-manifest`
— the configured manifest path exists and is non-empty (content judgment is
never done here); `test-enum-vs-spec` — the enum/const set in code vs the
values enumerated in the configured spec file, reporting each side's
extras. Each with a positive-control fixture pair; missing config → exit 2.

## Acceptance criteria

- [ ] All three pass on good fixtures and fail on seeded ones at the CLI
      seam; findings carry `code = go@1/<check>` and file:line where
      applicable.
- [ ] `gitignore-control` reports both arms distinctly (an ignored
      `main.go` is a finding even when binaries are ignored).
- [ ] Missing/unreadable configured paths → exit 2 naming the variable.

## Blocked by

- I082 (gate skeleton + emitter).

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commit af8e86c).
Three config-driven check classes on the I082 skeleton, config via env
(rendered by I085 from `gate_pack_config`): `SPINE_GATE_BUILD_OUTPUTS`
(gitignore-control arm 1 — each declared build output ignored at that path via
`git check-ignore --no-index`; arm 2 — no `package main` file in the working
tree is ignored; arms reported distinctly under `go@1/gitignore-control`),
`SPINE_GATE_FIXTURE_MANIFEST` (manifest absent/empty-after-trim = finding,
exit 1 — the manifest is the subject; env unset or unreadable = exit 2),
`SPINE_GATE_TEST_ENUM_SPEC` (typed string const enums in code vs backticked
tokens inside `<!-- spine:enum <TypeName> -->` … `<!-- /spine:enum -->`
markers in the spec; extras on each side are separate findings; only
marker-named types are compared; no marker → exit 2). Positive-control pairs
and exit-2 config cases at the CLI seam; usage lists checks from
`gate.CheckNames()`. Review clean, no fix round.
