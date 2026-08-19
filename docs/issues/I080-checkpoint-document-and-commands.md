---
id: I080
title: "Checkpoint document + spine checkpoint new|latest|list"
severity: high
status: fixed
affects: [checkpoint, templates, cli]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [concurrency-subtle-state]
review-tier: primary
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md`
(Checkpoint). Glossary: CONTEXT.md "Checkpoint" section. Sole-writer and
canonical-form rules mirror the cursor (ADR 0014 lineage).

## What to build

A Pi-extension author can distil a running session into a checkpoint with
one command and a reloaded session can print it back byte-stably.
`spine checkpoint new --from <narrative.md> --touched <csv> --gate
<pass|fail|none> --effort <level> [--slug s] [--facts-only]` validates the
model region's three sections (`## Task`, `## Conclusions`, `## Next
moves`), refuses (exit 2, naming the section) when one is missing or empty,
and with `--facts-only` writes a checkpoint flagged `narrative: missing`.
It writes `NNN-<slug>.md` into the working home
`.superpowers/sdd/checkpoints/` (ordinal reserved exclusively, like
handoffs), with frontmatter (`ordinal`, `created`, `effort`, `narrative`),
a `<!-- spine:checkpoint:model -->` region holding the narrative verbatim,
and a `<!-- spine:checkpoint:facts -->` region holding the canonical facts
block (`touched`, `gate`, `sha`, `effort_recommended`, `written`) — sha,
timestamp, ordinal computed by spine, touched list caller-supplied.
`spine checkpoint latest` prints the embedded reload preamble followed by
the newest checkpoint, byte-identical across invocations (exit 1 when
none); the preamble states the model-region/facts-region trust split and
the facts-only case. `spine checkpoint list` enumerates the working home.
New sibling package following the handoff/cursor pattern; no generic
document-family abstraction.

## Acceptance criteria

- [ ] new/latest/list round-trip at the CLI seam on a tempdir repo;
      ordinals increase and never reuse.
- [ ] Missing/empty section → exit 2 naming the section; `--facts-only` →
      `narrative: missing` and an empty model region.
- [ ] Facts block is byte-deterministic (package-level canonical-form test);
      `latest` output identical across two runs.
- [ ] Preamble is an embedded template; its text names both regions and the
      trust split; the facts-only wording is present.
- [ ] `sha` matches `git rev-parse HEAD` of the repo; `touched` preserves
      caller order.
- [ ] Documented in the CLI usage; CONTEXT.md terms used verbatim.

## Blocked by

- None — can start immediately.

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commits 38b2e29,
acb2708, 473b318). New package `internal/checkpoint`: `spine checkpoint new
--from <narrative.md> --touched <csv> --gate <pass|fail|none> --effort <level>
[--slug s] [--facts-only] [--dir D]`, `latest` (embedded reload preamble
`templates/current/checkpoint-preamble.md` + newest checkpoint, byte-stable;
exit 1 when none), `list`. Working home `.superpowers/sdd/checkpoints/
NNN-<slug>.md`; ordinals reserved via the exclusive-marker primitive now
shared with handoffs (`fsutil.ReserveOrdinal`). Frontmatter `ordinal`,
`created`, `effort`, `narrative`; `<!-- spine:checkpoint:model -->` region
(narrative verbatim; sections `## Task`, `## Conclusions`, `## Next moves`
required non-empty, extra sections tolerated; region markers inside the
narrative are refused so narrative can never masquerade as fact) and
`<!-- spine:checkpoint:facts -->` canonical block (`touched`, `gate`, `sha`,
`effort_recommended`, `written`; byte-deterministic, package-level test).
Rulings: `effort` and `effort_recommended` both carry `--effort` in v1; CRLF
normalized to LF. Exported for I081: `Latest`, `Split`, `ParseFacts`,
`Canonical`, `Facts.Block`. Primary-tier review + two scoped re-reviews clean.
