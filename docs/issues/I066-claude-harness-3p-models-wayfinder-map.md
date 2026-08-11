---
id: I066
title: Claude Code as a harness for 3rd-party models — wayfinder map
severity: med
status: open
affects: [model, audit, cli, workflow, fleet]
blocked-by: []
labels: [wayfinder:map]
---

## Destination

Spine routes tiers to any model reachable from the current host through an
available harness. A per-host capability config declares which harnesses exist
here and which models each can reach (gateways, local endpoints); tiers resolve
through owner-ratified equivalence pins ("routine on this host = gpt-5.6-sol @
high via claude"); every dispatch declares its (harness, model, effort) going in
and the audit confirms it in the work product coming out; effort is a
first-class dispatch parameter, not a tier-default constant. Audit routing and
yield feedback treat every (harness, model) combo first-class. This batch is
claude-harness-first: Claude Code drives GPT/GLM/open-weight models via
gateways; the codex harness is untouched.

## Notes

- Origin: `docs/handoffs/2026-08-10-claude-harness-3p-models-wayfinder.md`;
  charted with the owner 2026-08-10.
- Driving use cases (owner, 2026-08-10): local open-weight models in the pool;
  cost/capability arbitrage; and the work-laptop reality that GPT models now sit
  *behind* Claude Code and a custom gateway — one harness, many families, with
  availability varying per machine.
- Seed material: `docs/research/2026-08-05-routing-yield-feasibility.md`
  (committed with this map) and [I052](I052-routing-yield-feedback-charter.md)
  (resolved by it; forward build folded in as [I076](I076-routing-yield-review-record-and-yield-verb.md)).
- Invocation reality: driving other models through Claude Code changes the
  invocation itself (e.g. `claude-auto` + arguments vs bare `claude`), so
  cmux/herdr claude-team dispatch skills are in blast radius ([I071](I071-claude-auto-invocation-contract.md)).
- Build path after the map: standard gate chain per effort (grill → PRD →
  tickets → dispatch per WORKFLOW.md routing).

## Decisions so far

- [Harness vs flavor axis](I067-harness-vs-flavor-axis.md) — flavor is renamed
  **harness**; harness (execution vehicle) and model (id + family) split;
  claude-harness-first scope for this batch.
- [Host-scoped availability + tier equivalence pins](I068-host-scoped-availability-and-tier-pins.md)
  — availability is a per-host constraint living in spine config; tiers pin
  per host to owner-ratified comparables; same ticket resolving to different
  models on different hosts is accepted behavior.
- [Attribution stance](I069-attribution-declare-then-confirm.md) —
  declare-then-confirm: the dispatch names (harness, model, effort) going in;
  audit confirms against the subagent's transcript/work product.
- [Proxied transcript ids](I070-proxied-model-ids-in-claude-transcripts.md) —
  the empirical LM Studio/Claude Code path persisted the selected
  `google/gemma-4-12b` identifier verbatim in `message.model` on two assistant
  events. The raw field is future-confirmation-capable only after I072 supplies
  the host pin/mapping and I074 supplies heterogeneous verdict/correlation;
  current routing audit cannot judge this unannotated controller-only session.
  No `models/defaults.json` alias is justified; a differing production gateway
  alias needs that later host-scoped mapping.
- Routing-yield: I052's charter is answered by the committed feasibility note;
  recommendation is build-differently (review-time record, not retrospective
  file mining) — the forward build is mapped as [I076](I076-routing-yield-review-record-and-yield-verb.md),
  keyed to carry the actual model id heterogeneous pools require.
- [Claude Code invocation boundary](I071-claude-auto-invocation-contract.md)
  — stock Claude Code selects per-dispatch model/effort with explicit flags;
  gateway transport is `ANTHROPIC_BASE_URL`, not model selection; raw
  OpenAI-compatible endpoints need an adapter; every `claude-auto` wrapper
  contract remains owner-verified before cmux/herdr changes land.

## Not yet specified

- [I072](I072-host-config-schema-and-precedence.md) — host config schema and
  the estate-default → repo-override → host-constraint precedence story.
- [I073](I073-flavor-to-harness-rename-migration.md) — the rename migration.
- [I074](I074-audit-heterogeneous-verdicts.md) — audit verdicts for
  heterogeneous dispatches.
- [I075](I075-effort-first-class-dispatch-parameter.md) — effort in the
  dispatch/record grammar.
- [I077](I077-eval-informed-equivalence-pins.md) — eval evidence feeding pin
  ratification.

## Out of scope

- Symmetric design — the codex harness proxying non-OpenAI families. Claude-
  harness-first was ratified; revisit only if a concrete need appears.
- Retiring the codex harness — **permanently out**, not deferred (owner,
  2026-08-10): codex will always be here, and its workflow is expected to be
  unchanged by this batch. The claude-harness path is an additive option on
  hosts that need it (e.g. the work laptop), never a replacement track.
- Building or operating gateways/proxies themselves — spine consumes what a
  host declares reachable; it does not provide the plumbing.
