---
id: I110
title: "declare the openweights flavor in the embedded model table (data only)"
severity: med
status: fixed
affects: []
blocked-by: []
execution-mode: inline
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

**Inline justification (WORKFLOW.md "Execution modes"):** the spec fixes the
exact table contents, so this is a verbatim pre-specified diff against one JSON
data file plus its test — no design latitude for a worker to exercise.

## Problem

`spine model` resolves `(flavor, tier)` pairs from the embedded defaults, whose
flavor set is `claude`, `codex`, `pi`. Asking for anything else returns
`unknown flavor "openweights" (known: claude, codex, pi)`.

A repo's `WORKFLOW.md` cannot introduce one: resolution looks the flavor up in
the embedded defaults **before** any per-repo override is read, so an unknown
flavor errors out before `WORKFLOW.md` is consulted.

This blocks deepthought's `/openweights-team`, whose capability check treats a
failed `spine model openweights primary` as fatal by design — there is no
offline fallback.

## Fix

Add an `openweights` flavor to the embedded defaults file with all four tiers,
plus a per-flavor tier-default-effort block pinning every tier to `high`. No Go
changes — the table is data-driven and its own source comment states a new
flavor becomes known by adding it to the defaults with no code change.

| tier | id | effort |
|---|---|---|
| primary | `FW-Kimi-K3` | high |
| routine | `DeepSeek-V4-Pro` | high |
| mechanical | `FW-GLM-5.2` | high |
| fallback | `FW-Kimi-K3` | high |

Two things that are easy to get wrong:

- **The effort block is required, not cosmetic.** The global tier-default effort
  map gives `routine -> medium` and `mechanical -> low`. "High everywhere" must
  be stated as a per-flavor entry. `pi` already establishes the pattern, so no
  schema change is needed.
- **`fallback` intentionally equals `primary`.** The flavor exists to measure
  open-weights models; escalating to Claude on refusal would defeat that *and*
  make the id sets overlap, breaking I111's unambiguity assumption. The known
  consequence — a recorded `FALLBACK` is a tier no-op for this flavor — is
  accepted. `pi` sets the same precedent.

Aliases should be the human names the picker uses (`kimi-k3`, `deepseek-v4-pro`,
`glm-5.2`), chosen so they cannot collide with another flavor's ids or aliases.

An effort-vocabulary entry is **not** expected to be required (`high` is already
in the default vocabulary; only `pi` carries one because it uses `xhigh`).
Confirm against the validator during implementation; if it demands one per
flavor, add a vocabulary listing at minimum `high`.

## Tests

Largely covered on arrival — an existing model test iterates every flavor and
tier and asserts each resolves to its default. Add explicitly:

1. Each `openweights` tier resolves to the id above and its effort resolves to
   `high`, with `routine` and `mechanical` called out (those are the two the
   global defaults would otherwise give `medium` and `low`).
2. A repo-level `WORKFLOW.md` override of an `openweights` row is honoured,
   matching the existing override test for other flavors.
3. `claude`, `codex` and `pi` resolution is unchanged — the regression guard.

## Resolution

Fixed 2026-08-25. Data only, as designed — `models/defaults.json` gains a
`flavors.openweights` block and a `tierDefaultEffortByFlavor.openweights` block.
No Go source changed; the flavor became known purely by being in the table.

Two spec predictions confirmed against the code rather than assumed:

- **No `effortVocabulary` entry is needed.** `checkEffort` returns nil for a
  flavor absent from the vocabulary table ("a flavor absent from the vocabulary
  table accepts any effort"), and `high` is in the default vocabulary anyway.
- **The per-flavor effort block is load-bearing, not cosmetic.** Negative
  control: deleting `tierDefaultEffortByFlavor.openweights` drops
  `routine` to `medium` and `mechanical` to `low` — observed red on exactly the
  two tiers the spec singled out. Restored, green.

Verified against the live binary after `make install`:

```
openweights.primary     FW-Kimi-K3         effort=high
openweights.routine     DeepSeek-V4-Pro    effort=high
openweights.mechanical  FW-GLM-5.2         effort=high
openweights.fallback    FW-Kimi-K3         effort=high
```

`spine model openweights primary` exits 0 — the downstream skill's fatal
capability check now passes. `claude`, `codex` and `pi` resolve unchanged.

### One consequence the spec did not anticipate

`openweights.mechanical:` is longer than any prior key, and the mirror pads its
key column to the longest key, so **every pre-existing row reflows**. That is
whitespace only — story 6 holds, since resolution is byte-identical — but it
broke six tests that had the old column widths baked into string literals, and
three of those failed in *setup* ("fixture line not found to replace") rather
than in assertion. Two of them, ironically, carried comments claiming alignment
was irrelevant.

Owner decision: accept the reflow. Rather than re-hardcoding the new widths,
the fixtures now match rows by `flavor.tier` key (`replaceRow`, `hasRow`,
`pinRow`) and the gen10→11 migration guard classifies mirror changes with
`mirrorRenderDiff`, which sanctions a padding-only ± pair and a newly shipped
row but still fails a genuine value change. The next flavor cannot repeat this.

spine's own `WORKFLOW.md` was regenerated by the real `spine update --write`
path; `TestSpineOwnWorkflowModelMirrorIsByteStable` is the guard that caught it.

## Related

- Spec: `docs/specs/2026-08-25-openweights-flavor-spine-design.md` (Change 1).
- **I111** is the behavioural half and is blocked on this one. The split is
  deliberate: this change is additive data and cannot realistically hurt anyone,
  so it ships alone and unblocks the downstream capability check even if I111
  needs another review round.
- Downstream consumer: deepthought
  `docs/specs/2026-08-25-openweights-team-design.md`. That work is inert until
  this ships **and the fleet binary is rebuilt and reinstalled**.
