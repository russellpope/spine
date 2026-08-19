---
id: I079
title: "pi harness rows + alternate cell in the model table"
severity: med
status: fixed
affects: [models, model, update, cli]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md` (Model
table). Glossary: CONTEXT.md "harness", "alternate". Respects ADR 0011.

## What to build

A dispatcher or maipipe evaluator stage can resolve the `pi` harness from
spine's model table today: `spine model pi <tier>` returns
`qwen3.8-27b-q8_0` with an explicit per-cell effort (primary xhigh, routine
medium, mechanical low, fallback xhigh); `spine model pi <tier> --alternate`
returns the owner-tuned alternate `(id, effort)` — qwen @ xhigh on every
cell; `--json` includes `alternate` when present. The pi effort vocabulary
is `low | medium | xhigh`; asking pi for `high` fails with a clear error
rather than mapping. WORKFLOW.md mirror rows render pi cells with a trailing
`alt: <id> @ <effort>` and parse back under the same inherited/override
rules as any other cell (history entries may carry an alternate). The JSON
key stays under the legacy `flavors` map — the flavor→harness rename is
I073's. claude/codex behavior is unchanged.

## Acceptance criteria

- [ ] `spine model pi primary|routine|mechanical|fallback` resolve to the
      spec'd id and effort; `--alternate` returns qwen @ xhigh; `--json`
      carries `alternate`.
- [ ] `spine model pi routine --effort` style queries with `high` requested
      for pi error out with a message naming the pi vocabulary.
- [ ] Mirror row rendering includes `alt:`; a repo with an edited alternate
      is reported as override, an unedited one as inherited; gen-10 repos
      without pi rows are unaffected by `spine update` (negative control).
- [ ] Existing claude/codex model tests pass unchanged; new tests at the CLI
      seam per the spec's testing decisions.
- [ ] No reachability/doctor check for pi models (I072 territory).

## Blocked by

- None — can start immediately.

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commits 9b02de6, 6dfc7f0).
`models/defaults.json` ships the `pi` harness (JSON key still `flavors`) with
explicit per-cell efforts and `alternate: {qwen3.8-27b-q8_0, xhigh}`; the pi
effort vocabulary is data (`low|medium|xhigh`, `high` rejected naming the
vocabulary); `spine model pi <tier> [--alternate] [--json]`; mirror rows carry
a trailing `alt: <id> @ <effort>` parsed under inherited/override rules.
Rulings: gen-10 repos gain pi rows as inherited additions on `spine update`
(D8), no gen bump; `tierDefaultEffort` gained a per-harness override
(`tierDefaultEffortByFlavor.pi`) so bare-id pi rows resolve (spec D3 vs pi
vocabulary contradiction). Task review + scoped re-review clean.
