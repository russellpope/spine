---
id: I066
title: Claude Code as a harness for 3rd-party models — wayfinder map
severity: med
status: fixed
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
- Batch coordination: I121, I122, I123, and I124 joined the tracked
  open-ledger batch. They are separately scoped audit/update work; their batch
  membership records coordination only, not implementation.

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
  gateway transport is `ANTHROPIC_BASE_URL`, not model selection; record any
  `modelOverrides` provider-facing effective ID and its redacted provenance;
  raw OpenAI-compatible endpoints need an adapter; every `claude-auto` wrapper
  contract remains owner-verified before cmux/herdr changes land.
- [Host config schema and precedence](I072-host-config-schema-and-precedence.md)
  — the committed [I072 design](../specs/2026-08-29-host-routing-config-design.md)
  specifies a local, schema-versioned JSON capability/pin file at
  `os.UserConfigDir()/spine/routing-host.json`. Embedded defaults then the repo
  mirror remain preferences; a validated host constraint or explicit pin picks
  the final exact `(model, effort)` pair. Mirrors, templates, update, and fleet
  state stay host-blind. This design is approved for I072 implementation, not
  evidence that the implementation or dependent rename/eval work has landed.
- [Effort dispatch declarations](I075-effort-first-class-dispatch-parameter.md)
  — the owner-ratified I075 PRD makes raw target-harness `(harness, model,
  effort)` declarations first-class. Omitted effort resolves to the selected
  target; a supplied raw token is validated for that harness; the existing
  exact `ESCALATION <ticket> effort <from>-><to> reason: ...` line authorizes
  only that declared pair. Audit records declared effort separately from
  unconfirmed effective effort; I074 owns later observed-effort verdicts.
- [Flavor → harness migration](I073-flavor-to-harness-rename-migration.md)
  — the owner-ratified I073 PRD and migration plan define the compatibility
  window and fleet sequence. Implementation remains blocked until I072's
  independent verifier records PASS for its exact final SHA.
- [Heterogeneous routing verdicts](I074-audit-heterogeneous-verdicts.md)
  — the accepted I074 contract defines declared-versus-observed correlation,
  blocking mismatches, and nonblocking unconfirmable evidence for explicit
  heterogeneous dispatches. It depends on I072 and I075; no implementation is
  implied by this map decision.
- [Eval-informed equivalence pins](I077-eval-informed-equivalence-pins.md)
  — the round-2 owner policy selects both mechanisms, warn-only: at
  ratification, a pin records a repo-local `docs/evals/` reference for its
  exact pinned model, and doctor warns if that evidence is missing, stale, or
  failing. Those warnings never de-ratify, replace, block, or gate a pin. The
  I077 PRD waits for verified I072.

## Not yet specified

- [I076](I076-routing-yield-review-record-and-yield-verb.md) — its PRD may
  land now, but implementation follows verified I073 so its grammar uses the
  migrated harness term rather than freezing the old flavor term.
- [I077](I077-eval-informed-equivalence-pins.md) — the owner policy is bound,
  but its PRD waits for verified I072; that PRD must define the exact
  repo-local reference grammar, freshness rule, and doctor read boundary.

## Out of scope

- Symmetric design — the codex harness proxying non-OpenAI families. Claude-
  harness-first was ratified; revisit only if a concrete need appears.
- Retiring the codex harness — **permanently out**, not deferred (owner,
  2026-08-10): codex will always be here, and its workflow is expected to be
  unchanged by this batch. The claude-harness path is an additive option on
  hosts that need it (e.g. the work laptop), never a replacement track.
- Building or operating gateways/proxies themselves — spine consumes what a
  host declares reachable; it does not provide the plumbing.

## Resolution

Fixed 2026-08-30 as a wayfinder-map deliverable. The map's purpose under the
ledger convention is to record destination, decisions, frontier, and
out-of-scope boundaries; its child tickets carry the downstream implementation
work. I072-I075 now have their approved/accepted designs recorded above, and
I076-I077 have explicit PRD prerequisites. Closing this map therefore records
completed wayfinding only; it does not assert that any dependent implementation
or fleet migration has shipped.
