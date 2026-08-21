---
id: "0018"
title: "maipipe on PATH is a precondition for touching maipipe.toml when gate_pack is set"
status: Accepted
date: 2026-08-20
---

# 0018: maipipe on PATH is a precondition for touching maipipe.toml when gate_pack is set

## Context

I096 added a spine-side duplicate/balance scanner before writing a rendered
gate-pack region, then optionally asked `maipipe validate` to judge the same
candidate. The scanner was not a TOML parser, duplicated the consuming
authority, and accumulated both over- and under-refusal cases. ADR 0001 keeps
spine stdlib-only, so adding a TOML dependency is not this decision.

The managed region is a pure projection of `WORKFLOW.md` (ADR 0017), and its
plan diff is the review surface. Gate packs themselves are maipipe pipelines
(ADR 0015). I104 therefore considered whether a machine that cannot run
maipipe should touch its input file at all.

## Decision

When `gate_pack` is non-empty, a resolvable `maipipe` binary is a precondition
for planning or writing `maipipe.toml`. spine has no structural TOML scanner.
With no binary on `PATH`, the plan names `maipipe.toml` as skipped, says that
the maipipe pre-flight did not run, leaves that file byte-for-byte unchanged,
and applies other pending files with exit 0.

When maipipe is available, `maipipe validate` is the sole candidate pre-write
check and runs during planning as well as before application. A failed
validation remains a refusal and leaves every pending file untouched. This
decision cites I104; ADR 0001 is unchanged.

## Consequences

- Users enabling a gate pack must install maipipe before `spine update` can
  create or refresh its region. A workflow-only refresh can still apply its
  other files and clearly reports the skipped maipipe file.
- The hand-written scanner and its scanner-specific test surface are removed;
  maipipe remains the grammar authority for the file it consumes.
- I097's later opt-out/removal path is also subject to this precondition when
  it would touch `maipipe.toml`.
