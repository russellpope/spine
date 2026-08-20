---
id: "0016"
title: "spine owns a machine-managed region inside maipipe.toml"
status: Accepted
date: 2026-08-18
---

# 0016: spine owns a machine-managed region inside maipipe.toml

## Context

Gate packs (ADR 0015) are delivered as maipipe pipelines. maipipe reads
exactly one file, `maipipe.toml` at the repo root, with no include/import
mechanism and `deny_unknown_fields` on every table (verified 2026-08-18,
`maipipe/src/pipeline.rs`); its only composition is a stage whose
`pipeline = "<name>"` inlines another pipeline defined in the same file. So
a scaffolded "snippet" cannot be a sibling file — the pack must be present
inside the repo's own `maipipe.toml`, which is a user-owned file that repos
edit freely.

Until now spine's ownership split (ADR 0002 — machine-owned regions
delimited by `<!-- spine:begin vN -->` / `<!-- spine:end -->`, choice-vs-
default preservation) has applied only to files spine itself scaffolds
(CLAUDE.md, AGENTS.md, WORKFLOW.md). Alternatives considered: a reference
file the operator pastes by hand (drifts silently, defeats `spine update`);
asking maipipe for an include mechanism first (cross-product blocking change
for one consumer).

## Decision

spine maintains a **machine-managed region inside `maipipe.toml`**,
delimited by TOML comments (`# spine:begin gate-pack go@1` … `# spine:end`),
containing the pack's pipeline tables (`[pipelines.gate-go]`,
`[pipelines.mutation-go]` and their stages). The region is a **pure
projection of WORKFLOW.md**: `spine update` re-renders it from the
`gate_pack*` keys on every run, no value inside it is ever a user choice,
and ADR 0002's choice-vs-default preservation rule does not apply to it.
Edits inside the region are discarded on refresh; the `spine update` plan
diff shows what drops and is the review surface for region changes.
Everything outside the region is untouched. *(Amended 2026-08-20, I095 —
this sentence previously read "regenerated under ADR 0002's rules … edits
inside the region are unrecognized and reported, never silently kept",
which promised a preservation the implementation never had: a divergent
`env` value is refreshed by the next `spine update --write` with no
`--force`, and the code cited ADR 0002 for doing so. Reason for choosing
projection over preservation: a hand-edit inside the region and a
legitimate `gate_pack_config` change in WORKFLOW.md produce byte-identical
plan states, so "preserve divergent values" is undecidable without a record
of the last render, and every knob the region exposes already has a
WORKFLOW.md key. The shape matcher in `unrecognizedRegionLines` stays as
ADR 0002's generic unrecognized-edit guard — a line no configuration of the
pack could render, such as a rewritten `run`, skips the file until
`--force` — but that is a safety stop before the drop, not preservation.
See `docs/research/2026-08-19-gate-pack-region-ownership-analysis.md`.)*
The repo composes the pack by adding `pipeline = "gate-go"` (full lane) and
`pipeline = "mutation-go"` (audit lane) stages in its own pipelines — that
edit is the repo's, outside the region. Opt-out and per-check configuration
never happen by editing the region; they are WORKFLOW.md keys that change
what the region renders.

## Consequences

- ADR 0002's ownership model now crosses a product boundary; maipipe must
  tolerate a comment-delimited region it did not write. This is recorded on
  maipipe's side as a cross-product ticket, together with the `spine`-on-
  PATH dependency and the pipeline names.
- If `maipipe.toml` is absent, `spine update` creates it containing
  maipipe's required top-level `schema = 0` plus the region and a note; the
  repo adds its lanes. *(Amended 2026-08-19, I091 — the first dogfood
  `maipipe validate` rejected a region-only file and the plural `stages`
  array; the render is now `[[pipelines.<name>.stage]]`.)*
- Region integrity (markers present, canonical content) becomes a doctor
  finding, mirroring D3 for markdown files.
- Reviewing a region change means reading the `spine update` plan diff
  before `--write`: it is the only place the previous render and the next
  one are shown side by side, and it is the same whether the delta came
  from a WORKFLOW.md key or a hand-edit — there is no record that could
  tell the two apart, by design. *(Amended 2026-08-20, I095.)*
- Should maipipe later add an include mechanism, moving the region to a
  sibling file is a new ADR superseding this one.
