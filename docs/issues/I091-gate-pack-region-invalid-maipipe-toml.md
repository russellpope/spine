---
id: I091
title: "Rendered gate-pack region is invalid maipipe TOML: `[[…stages]]` should be `[[…stage]]`, created file lacks `schema = 0`"
severity: high
status: fixed
affects: [I085]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: [cross-task-integration]
review-tier: n/a
---

## Problem

Dogfood §1b/c (2026-08-19), spine self-enabled `gate_pack: go@1`, `spine
update --write` created `maipipe.toml`; then:

```
$ maipipe validate maipipe.toml
parse maipipe.toml: TOML parse error at line 10, column 21
10 | [[pipelines.gate-go.stages]]
   |                     ^^^^^^
unknown field `stages`, expected one of `description`, `profile`, `stage`
```

and with `stage` patched in:

```
missing field `schema`
```

Two defects in `internal/update/gatepack.go` (I085), both invisible to
spine's own tests because the positive controls assert spine's string shape
rather than maipipe's grammar:

1. Stage arrays are `[[pipelines.<name>.stage]]` (singular) in maipipe.
2. A `maipipe.toml` created from scratch must carry top-level `schema = 0`
   before any table header. Spec story 15 / ADR 0016 say the created file
   "contains only the region" — unsatisfiable against maipipe; amend to
   "`schema = 0` plus the region".

Any repo that enables the pack on a repo with no `maipipe.toml` gets a file
maipipe refuses to load; a repo with an existing file gets a region that
breaks its whole pipeline definition.

Verified shape (scratch, `maipipe validate` OK, 3 pipelines): `schema = 0`,
`[[pipelines.gate-go.stage]]` ×N with inline `env = {…}`, `[pipelines.full]`
stage `pipeline = "gate-go"`.

## Fix

- `renderGateRegion`/`unrecognizedRegionLines`: `.stage]]`.
- Create path: `schema = 0\n\n` + region; append-to-existing path unchanged
  (the schema line is the repo's).
- Tests: string assertions updated; new assertion that a created file begins
  with `schema = 0`.
- Spec story 15 + ADR 0016 consequence amended (dated).
- Evidence: live `maipipe validate` on spine's regenerated file.

## Resolution (2026-08-19)

`internal/update/gatepack.go`: `.stage]]` everywhere; created file =
`schema = 0\n\n` + region. Tests renamed/extended
(`TestGatePackCreatesMaipipeWithSchemaAndRegionOnly`, plural guard).
Negative control: reintroducing `.stages]]` in the renderer fails 4 update
tests. Spec story 15 / §Rendering and ADR 0016 consequence amended.
Evidence: `rm maipipe.toml && spine update --write` → `maipipe validate
maipipe.toml: OK (2 pipelines)`; with owner lanes appended → `OK (4
pipelines)`; `spine update` → `up-to-date: maipipe.toml`; doctor no D10.
Only spine carried the broken render (fleet is gen 10).
