# openweights flavor + model-id flavor derivation (spine) — design

**Status:** ready for upstream implementation
**Target repo:** `spine` (`git@github.com:russellpope/spine.git`)
**Written from:** `spine` @ `7795d2f` (2026-08-25). Re-verify before starting; see "Positional references" below.
**Downstream consumer:** deepthought `/openweights-team` — see `2026-08-25-openweights-team-design.md`. That work is inert until this ships.

> **This document is written for an implementer with no access to the conversation that produced it.** Every claim below was verified against the repository at the stated commit. Where a claim could not be verified, it is marked UNVERIFIED.

## Problem Statement

An engineer wants to run an agent team on open-weights models (Kimi K3, DeepSeek V4 Pro, GLM 5.2) served through the Cascade gateway, with the same workflow guarantees a Claude or codex team gets: tier-annotated tickets, tier-resolved dispatch, and a `spine audit routing` verdict that blocks silent descent.

Two things in spine prevent that today.

**First, there is no flavor for these models.** `spine model` resolves `(flavor, tier)` pairs from an embedded table whose flavor set is `claude`, `codex`, `pi`. Asking for anything else returns `unknown flavor "openweights" (known: claude, codex, pi)`. A repo's `WORKFLOW.md` cannot introduce one: resolution looks the flavor up in the embedded defaults *before* any per-repo override is read, so an unknown flavor errors out before `WORKFLOW.md` is consulted. This matters because the consuming team skill treats a failed `spine model <flavor> primary` as fatal by design — there is no offline fallback.

**Second, and more subtly, the audit cannot tell an open-weights dispatch from a Claude one.** Design D15 states that flavor is derived from the *transcript source*. That holds for codex, which has its own session layout and tags its records `codex` directly. But open-weights sessions run the ordinary `claude` CLI (via a wrapper that passes `--model` through), so their transcripts land in the same `~/.claude/projects` layout as real Claude sessions. `transcriptFlavor` returns the constant `"claude"` for that whole source. Every open-weights dispatch would therefore be judged against the `claude` tier table, where its model id does not appear — producing a wrong verdict on exactly the runs (weak open models) where the routing gate matters most.

## Solution

Two changes, deliberately unequal in risk and separable into two tickets.

**Change 1 — declare the flavor (data only).** Add an `openweights` flavor to the embedded defaults with all four tiers, and a per-flavor effort block pinning every tier to `high`. No Go changes. The table is data-driven by design; its own source comment states that a new flavor becomes known by adding it to the defaults file with no code change.

**Change 2 — derive flavor from the observed model id (behavioural).** Extend D15 so that a record's flavor comes from the model id observed in the transcript, with the transcript source retained as the tiebreaker for ids declared under more than one flavor. This is an *extension* of D15, not a reversal: D15's own final clause already says "where a model id is declared under more than one flavor, the transcript-derived flavor decides" — it reasons about model ids and uses source only to break ties. Today the tie is trivial because one source yields one flavor; this change makes the unambiguous case authoritative and leaves the tiebreaker intact.

Crucially, flavor and transcript source stop being the same axis. That is the whole point, and it is also the main hazard (see Implementation Decisions).

## User Stories

1. As a workflow owner, I want `spine model openweights primary` to resolve, so that a team skill's fatal capability check passes and dispatch can proceed.
2. As a workflow owner, I want `openweights` to carry all four tiers, so that the table validates and every tier annotation a ticket can express has a target.
3. As a workflow owner, I want every `openweights` tier to resolve at effort `high`, so that weak models get maximum reasoning budget regardless of tier.
4. As a workflow owner, I want `openweights.fallback` to resolve to the same model as `openweights.primary`, so that a refusal re-run never silently leaves open weights and contaminates a measurement.
5. As a workflow owner, I want the `openweights` model ids to stay disjoint from every other flavor's ids, so that model-id-derived flavor is unambiguous.
6. As a fleet operator, I want adding this flavor to leave `claude`, `codex` and `pi` resolution byte-identical, so that no existing repo's dispatch or audit changes behaviour.
7. As a fleet operator, I want the embedded table to keep validating at load, so that an incomplete flavor is a build-time failure rather than a runtime surprise.
8. As an auditor, I want a dispatch on an open-weights model id to be judged against the `openweights` tier table, so that a correct dispatch is not reported as silent descent.
9. As an auditor, I want a dispatch on a Claude model id to keep being judged against the `claude` table, so that this change does not weaken existing verdicts.
10. As an auditor, I want a single transcript containing both Claude and open-weights dispatches to have each token judged within its own flavor, so that mixed runs are judged correctly rather than by majority.
11. As an auditor, I want the D28 repo-qualification gate to keep applying to open-weights records, so that re-tagging them does not let them claim tickets they should not.
12. As an auditor, I want ids declared under more than one flavor to keep falling back to transcript-source derivation, so that D15's tiebreaker still resolves genuine collisions.
13. As an auditor, I want an unrecognised model id to be treated exactly as it is today, so that this change introduces no new failure mode for unknown models.
14. As a maintainer, I want the flavor-derivation point to remain a single named seam, so that a future fourth source is a data change rather than a redesign.
15. As a maintainer, I want the decision recorded as an ADR that references D15, so that a future reader of the derivation code finds the rationale without knowing why it changed.
16. As a downstream skill author, I want `spine model --dir <repo> openweights <tier>` and its `-effort` form to work, so that a dispatch can resolve id and effort independently.
17. As a downstream skill author, I want a repo's `WORKFLOW.md` to be able to override `openweights` rows like any other flavor, so that a repo can pin different open models without a spine release.
18. As a release manager, I want the two changes in separate tickets, so that the low-risk data change can ship even if the audit change needs another review round.

## Implementation Decisions

### Change 1 — the flavor row

Add to the embedded defaults file (the JSON the model package embeds):

| tier | id | effort |
|---|---|---|
| primary | `FW-Kimi-K3` | high |
| routine | `DeepSeek-V4-Pro` | high |
| mechanical | `FW-GLM-5.2` | high |
| fallback | `FW-Kimi-K3` | high |

Entries follow the existing per-tier shape (`id`, `aliases`, `history`). Aliases should be the human names the picker uses (`kimi-k3`, `deepseek-v4-pro`, `glm-5.2`) — chosen so they cannot collide with another flavor's ids or aliases, which is what keeps derivation unambiguous.

**The effort block is required, not cosmetic.** The global tier-default effort map gives `routine → medium` and `mechanical → low`. "High everywhere" must therefore be stated as a per-flavor tier-default-effort entry for `openweights`. The `pi` flavor already establishes this pattern, so no schema change is needed.

An effort-vocabulary entry is **not** expected to be required: only `pi` carries one, because it uses `xhigh`; `high` is already in the default vocabulary. Confirm against the validator during implementation — if the validator requires every flavor to declare a vocabulary, add one listing at minimum `high`.

`fallback` intentionally equals `primary`. This is a deliberate product decision, not an oversight: the flavor exists to measure open-weights models, and escalating to Claude on refusal would both defeat that and make the id sets overlap, breaking Change 2's unambiguity. The `pi` flavor sets the same precedent (all four tiers on one model). The known consequence is that a recorded `FALLBACK` is a tier no-op for this flavor; that is accepted.

### Change 2 — flavor derivation

**What exists already (do not rebuild it).** The architecture already supports per-record flavor. The evidence-token type carries a flavor field; the tier mappings are keyed by flavor so that token judging resolves each token within its own flavor; and the codex effort proved the seam by tagging its records `codex` at read time rather than deriving them from the shared helper. The only thing that changes is *where a claude-layout record's tag comes from*.

**The change.** Today the audit stamps one flavor for an entire transcript source via `transcriptFlavor`, which ignores its argument and returns the constant `"claude"`. Replace that single stamp with per-record derivation:

- If the record's observed model id is declared under exactly one flavor in the resolved table, that flavor is the record's flavor.
- If the id is declared under more than one flavor, fall back to the transcript-source derivation (D15's existing tiebreaker).
- If the id is declared under no flavor, preserve today's behaviour exactly — do not invent a new failure mode.

A third tier mapping for `openweights` must be resolved alongside the existing claude and codex mappings, following the same shape.

### The hazard — D28 must not silently stop applying

**This is the single most important paragraph in this document.**

The audit contains a match predicate gated on a record's flavor being `claude`, which enforces D28 (ticket I047): a claude dispatch claims a ticket only if it *also* references the audited repo or its session shows cwd evidence inside it. Codex records are exempt because they are hard-scoped to the repo upstream, so re-gating them would be redundant.

Open-weights records come from the claude-layout source and have identical cwd and description semantics — they need that same gate. But the condition is written in terms of *flavor*, and the moment these records are tagged `openweights`, **they fall out of the D28 check and begin claiming tickets they should not**.

The fix is small: the condition must test "this record came from the claude-layout transcript source", not "this record's flavor is claude". The two were the same thing before this change and are not afterwards.

**This failure passes every existing test.** No current test has an open-weights record, so nothing goes red; the damage appears only as wrong verdicts in the field. A regression test asserting D28 still applies to open-weights records is mandatory (see Testing Decisions).

**Generalisation:** any other site that conflates source and flavor needs the same treatment. Before implementing, grep for every comparison against a flavor literal and classify each as genuinely flavor-dependent or actually source-dependent. Treat that classification as part of the deliverable, not incidental cleanup — record the list and the verdict for each in the ticket.

### ADR

Record the derivation change as an ADR in spine's `docs/adr/`. **Use 0022 or higher — 0021 is taken** (gate panic as contractual misconfiguration exit). The ADR must state that it extends D15 rather than replacing it, name the tiebreaker as retained, and state that source and flavor are no longer the same axis.

### Positional references

This document deliberately contains **no line numbers**. Between the grill that produced it and the day it was written, upstream changed the audit file by roughly 145 lines and moved every location originally cited, including rewriting the expression inside the D28 predicate. Anchor on function names, D-numbers, and behaviour; re-derive positions from the working tree.

## Testing Decisions

A good test here asserts an externally observable verdict — what `spine model` prints, or what verdict the audit reaches for a given transcript and ticket set — never the internal shape of the derivation. Prior art for every case below already exists in the audit and model test suites.

**Change 1** is largely covered on arrival: an existing model test iterates every flavor and tier and asserts each resolves to its default, so a new flavor is picked up automatically. Add explicitly:

1. Each `openweights` tier resolves to the id in the table above, and its effort resolves to `high` — with `routine` and `mechanical` called out, since those are the two the global defaults would otherwise give `medium` and `low`.
2. A repo-level `WORKFLOW.md` override of an `openweights` row is honoured, matching the existing override test for other flavors.
3. Resolution for `claude`, `codex` and `pi` is unchanged — the regression guard for story 6.

**Change 2** has direct prior art in the audit suite: existing tests assert that one flavor's id is invisible within another flavor, that a token resolves within its own flavor, and that a transcript mixing claude and codex evidence is judged per flavor. Model the new tests on those.

4. A dispatch on an open-weights id, in a claude-layout transcript, is judged against the `openweights` table — the core of the change.
5. A dispatch on a Claude id in the same transcript is still judged against the `claude` table.
6. A single transcript containing both is judged per-token, not per-source. This is the direct analogue of the existing mixed claude/codex test and is the highest-value case.
7. **D28 still applies to open-weights records.** Construct a record whose model id is open-weights and whose text and cwd do *not* qualify for the audited repo, and assert it does **not** claim the ticket. This is the regression test for the hazard above; without it the hazard ships silently.
8. An id declared under two flavors still resolves via the transcript-source tiebreaker.
9. An unrecognised id behaves exactly as before.

Run the full suite, not the targeted tests: the change touches a file with substantial recent churn.

## Out of Scope

- **Any deepthought change.** The team skill, its parameterisation, and the new `/openweights-team` skill live in the downstream spec and must not be attempted here.
- **Teaching spine about the `claude-auto` wrapper.** spine resolves ids and judges transcripts; it never launches anything. The launcher is entirely a downstream concern.
- **A `WORKFLOW.md` mirror migration.** Repos pick up `openweights` rows through the normal `spine update` regeneration; no bulk fleet sweep is part of this work.
- **Changing what effort *means* for open-weights models.** Whether the gateway honours a reasoning-effort setting for these models is unverified and outside spine — spine only resolves and reports the value.
- **Adding further flavors.** The derivation change should generalise, but only `openweights` is being added.
- **Retiring `transcriptFlavor`.** It remains the tiebreaker and the seam for future sources; this work narrows its authority, it does not delete it.

## Further Notes

**Ship order.** Change 1 is safe to ship alone and unblocks the downstream skill's capability check. Change 2 is the one that needs care. They are separate tickets deliberately.

**Blast radius.** `spine` is a fleet-wide binary and `spine audit routing` is a blocking verify gate for every repo that uses it. Change 1 cannot realistically hurt anyone — it is additive data. Change 2 edits the code that decides whether work is allowed to ship, and its failure mode is not a crash but a wrong verdict. That asymmetry is the reason for the split and for the mandatory D28 regression test.

**Why the ids are disjoint.** The product decision to keep `openweights.fallback` on Kimi rather than escalating to a Claude model is what makes model-id-derived flavor unambiguous. If a future change points any `openweights` tier at a `claude-*` id, Change 2's core assumption breaks and the tiebreaker path becomes load-bearing. Note this in the ADR.

**Cosmetic gateway warning (informational).** Dispatching these ids through the Claude CLI emits `[claude-code:unrecognized_model]` on stderr while returning correct output — verified live for all three models. It is not a spine concern and requires no handling here; it is recorded so an implementer seeing it in a transcript fixture does not treat it as a defect.
