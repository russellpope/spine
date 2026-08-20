---
id: "0017"
title: "spine owns a machine-managed region inside maipipe.toml, rendered as a pure projection of WORKFLOW.md"
status: Accepted
date: 2026-08-20
supersedes: "0016"
---

# 0017: spine owns a machine-managed region inside maipipe.toml, rendered as a pure projection of WORKFLOW.md

## Context

Gate packs (ADR 0015) are delivered as maipipe pipelines. maipipe reads
exactly one file, `maipipe.toml` at the repo root, with no include/import
mechanism and `deny_unknown_fields` on every table (verified 2026-08-18,
`maipipe/src/pipeline.rs`); its only composition is a stage whose
`pipeline = "<name>"` inlines another pipeline defined in the same file. So
a scaffolded "snippet" cannot be a sibling file — the pack must be present
inside the repo's own `maipipe.toml`, which is a user-owned file that repos
edit freely. ADR 0016 settled that spine owns a comment-delimited region
inside that file. Alternatives it weighed and rejected stand: a reference
file the operator pastes by hand (drifts silently, defeats `spine update`);
asking maipipe for an include mechanism first (cross-product blocking change
for one consumer).

ADR 0016 also said the region is "regenerated under ADR 0002's rules …
edits inside the region are unrecognized and reported, never silently
kept". That sentence, ADR 0002 ("only divergent values survive") and the
implementation disagreed three ways (I095, from
`docs/research/2026-08-19-gate-pack-region-ownership-analysis.md`): a
divergent `env` value inside the region is refreshed by the next
`spine update --write` with no `--force`, and the code cited ADR 0002 for
doing so. The owner's call between the two coherent readings — (A) pure
projection, (B) genuinely ADR-0002-governed with a last-render record — is
(A). This ADR supersedes 0016 so that one record carries the whole
decision as it now stands, I091's correction included.

## Decision

spine maintains a **machine-managed region inside `maipipe.toml`**,
delimited by TOML comments (`# spine:begin gate-pack go@1` … `# spine:end`),
containing the pack's pipeline tables (`[pipelines.gate-go]`,
`[pipelines.mutation-go]` and their stages, in maipipe's singular
`[[pipelines.<name>.stage]]` form).

**The region is a pure projection of WORKFLOW.md.** `spine update`
re-renders it from the `gate_pack`, `gate_pack_disabled` and
`gate_pack_config` keys on every run. No value inside the region is ever a
user choice; ADR 0002's choice-vs-default preservation rule does not apply
to it. Edits inside the region are discarded on refresh. The `spine update`
plan diff shows what drops and is the review surface for region changes.
Everything outside the region is untouched.

One ADR 0002 rule does still apply, and it is a stop, not preservation: a
line inside the region that *no* configuration of the pack could have
rendered — a rewritten `run`, an invented env var, a stray comment — is
reported as an unrecognized local edit and the file is skipped until
`--force` (`unrecognizedRegionLines`, by shape). A line any configuration
could have rendered — a different `env` value, a missing or reordered
stage, an older header comment — is not reported; it refreshes, and the
diff is the record.

The repo composes the pack by adding `pipeline = "gate-go"` (full lane) and
`pipeline = "mutation-go"` (audit lane) stages in its own pipelines — that
edit is the repo's, outside the region. Opt-out and per-check configuration
never happen by editing the region; they are WORKFLOW.md keys that change
what the region renders. If `maipipe.toml` is absent, `spine update`
creates it as maipipe's required top-level `schema = 0` plus the region;
the repo adds its lanes.

## Rationale

- A hand-edit inside the region and a legitimate `gate_pack_config` change
  in WORKFLOW.md produce byte-identical plan states (Pending, zero
  unrecognized lines, the same `-`/`+` env diff; reproduced live
  2026-08-19). "Preserve divergent values" is therefore undecidable without
  a record of what spine last rendered, and the only such record that
  survives clone, worktree and branch switch would be a fingerprint on the
  marker line — machinery for a file that exists in one repo.
- Every knob the region exposes already has a WORKFLOW.md key, so nothing
  is lost by refusing to treat in-region bytes as configuration.
- The shape stop is kept because dropping a line spine cannot account for
  should cost an explicit `--force`, the same as in every other
  machine-owned file; it is not a claim that the line is a choice.

## Consequences

- ADR 0002's ownership model crosses a product boundary; maipipe must
  tolerate a comment-delimited region it did not write. This is recorded on
  maipipe's side as a cross-product ticket, together with the `spine`-on-
  PATH dependency and the pipeline names.
- Reviewing a region change means reading the `spine update` plan diff
  before `--write`: it is the only place the previous render and the next
  one appear side by side, and it reads the same whether the delta came
  from a WORKFLOW.md key or a hand-edit — by design, there is no record
  that could tell the two apart.
- Region integrity (markers present, canonical content) is a doctor
  finding, mirroring D3 for markdown files.
- Composing the pack is left to each repo, and as of 2026-08-20 only half of
  that composition exists anywhere: the full-lane stage
  (`pipeline = "gate-go"`) is composed, the audit-lane stage
  (`pipeline = "mutation-go"`) is not — not in spine's own `maipipe.toml`,
  not in any other repo that has adopted the pack. The scaffolding that
  would write an audit lane belongs to maipipe (its ticket I204 — open,
  blocked by I202+I203), so until I204 lands the advisory battery of
  ADR 0013 is reachable only by naming it directly: `maipipe run
  mutation-go`. The durable records of this gap are the standalone note in
  spine's own `maipipe.toml` and ticket I099.
- Should maipipe later add an include mechanism, moving the region to a
  sibling file is a new ADR superseding this one.
