# Model Routing — design (2026-07-09)

> Status: approved via /grill-with-docs interview (10 decisions, D1–D10),
> Russell 2026-07-09. Vocabulary in `CONTEXT.md`. Companion plan:
> `2026-07-09-model-routing-plan.md` (to be written by /to-tickets).

## Problem Statement

The estate declares model routing (`model_routing` in every scaffolded
WORKFLOW.md) but nothing reads it at the moment that matters — dispatch time.
In practice a session on the primary model does all the dev work itself:
the objectstudio build ran 205/205 assistant messages on the primary model
with two subagent dispatches total. The declared policy and the actual
behavior are unconnected, so the owner discovers routing outcomes by
surprise, after the spend. Related symptoms: "ultracode" and
"subagent-driven" are conflated in past records; the template promises an
"auto" refusal fallback that no mechanism implements; and plans carry no
per-task model intent, so there is nothing to review up front and nothing
to audit afterward.

## Solution

Make routing a first-class, estate-owned contract with three connected
layers, per the glossary in CONTEXT.md:

1. **Declared intent** — every ticket carries `execution_mode`, a model
   `tier` (primary / routine / mechanical / fallback — tiers, never model
   ids), an optional `effort` override, and named risk triggers. /to-tickets
   assigns them; plan review checks them.
2. **Dispatch discipline** — the scaffolded WORKFLOW.md (always in context)
   carries the full dispatch contract: tier→id mapping, tier-default
   efforts, escalate-freely-with-reason / never-silent-descent, reviewer
   floor + risk triggers, proactive + reactive fallback routing with push
   notification, and plan-gated ultracode opt-in.
3. **Verification** — `spine audit routing` deterministically diffs declared
   annotations against the models actually used (from transcript records),
   required at the verify stage: reasoned escalations advisory, silent
   descent blocking.

Quality ceiling first (the primary model remains the default thinker);
down-routing exists to stop waste on provably mechanical work; the audit
makes actual-vs-declared visible every build.

## User Stories

1. As the estate owner, I want every ticket to declare its execution mode (inline / subagent-driven / ultracode), so that how work will run is decided and reviewable before any work starts.
2. As the estate owner, I want inline execution to be a rare, justified exception, so that build work defaults to subagent or swarm shapes with independent review.
3. As the estate owner, I want tickets to carry a model tier per task, so that the mechanical-vs-judgment call is made at plan time, where the plan's completeness makes it answerable.
4. As the estate owner, I want plans to speak tier names rather than model ids, so that I can remap the estate to new model families, local models, or another provider by editing one mapping instead of every plan ever written.
5. As an orchestrator session, I want the tier→model mapping in repo context (WORKFLOW.md), so that I can translate tiers to dispatch parameters without guessing.
6. As an orchestrator session, I want the right to escalate a task above its annotated tier with a recorded reason, so that a task that grows teeth mid-build is not hobbled on principle.
7. As the estate owner, I want silent descent (dispatching below annotation without a recorded reason) to be a blocking gate failure, so that the quality ceiling cannot erode invisibly.
8. As a plan author (/to-tickets), I want the ticket template to have structural fields for mode, tier, effort, and risk triggers, so that assignment cannot be skipped by prose drift.
9. As the estate owner, I want a `mechanical` tier that is definitionally narrow (verbatim plan-transcription and single-file mechanical fixes only), so that the cheapest model is used exactly where the plan already contains the code and nowhere else.
10. As the estate owner, I want reviewer tier never below implementer tier, so that a cheaper reviewer cannot rubber-stamp a more capable implementer's judgment.
11. As a plan author, I want named risk triggers (cross-task integration, concurrency/subtle state, security surfaces, plan-flagged ambiguity) to force a primary-tier review, so that review capability keys to task risk rather than implementer cost.
12. As the estate owner, I want the final whole-branch review and acceptance simulation to always run on the primary tier, so that the pass with the proven catch record is never down-routed.
13. As a reviewer at any tier, I want the re-run-the-claims procedure to remain mandatory, so that tier choices never substitute for evidence discipline.
14. As the estate owner, I want security-framed requests routed to the fallback tier from the first dispatch, so that known classifier-tripping work never burns a primary refusal first.
15. As an orchestrator session, I want a defined reactive path when the primary tier refuses (re-dispatch on fallback with quality framing, record the event), so that refusals degrade gracefully instead of stalling the build.
16. As the estate owner, I want a push notification when a reactive fallback fires, so that I learn about refusal events when they happen, not when I read the ledger.
17. As the estate owner, I want effort routing to follow the same rails as model routing (tier-implied defaults, per-ticket override, escalate-only), so that there is one rule-set to remember for both knobs.
18. As the estate owner, I want ultracode opt-in to be plan-gated through ticket approval, so that swarm orchestration is authorized by me in advance rather than improvised mid-build.
19. As an orchestrator session, I want mid-build ultracode escalation to be recommend-only, so that I can propose a swarm when work outgrows its ticket but the owner keeps the gate.
20. As the estate owner, I want the vocabulary (ultracode vs subagent-driven vs inline) pinned in the glossary, so that records and plans stop conflating different execution machines.
21. As the estate owner, I want `spine audit routing` to produce a per-task table of declared tier vs actual model with verdicts, so that every build ends with routing ground truth instead of recollection.
22. As the estate owner, I want the audit to be a deterministic CLI, so that verification costs zero tokens and cannot itself hallucinate.
23. As an orchestrator session, I want the audit required at the verify stage, so that routing verification is a gate, not a suggestion.
24. As the estate owner, I want the audit to degrade gracefully when transcripts are missing or their format shifts, so that parser rot warns rather than failing builds spuriously.
25. As a fleet repo, I want the routing contract to arrive via the normal template generation bump (`spine update`), so that adoption is the standard dry-run-diff-then-write flow.
26. As the estate owner, I want the rollout dogfooded on deepthought and objectstudio and exercised by one real build before the fleet sweep, so that a wording bug in the dispatch contract is caught before it propagates.
27. As a future maintainer, I want the routing contract to live only in estate-owned surfaces (spine templates, local skills, spine CLI), so that plugin updates can never clobber it.
28. As the estate owner, I want upstream filing (per-task model/mode fields in the plan-writing skill's schema) to remain optional and gated on my word, so that the design never depends on an external maintainer accepting a patch.
29. As a future maintainer, I want an ADR recording tiers-not-ids and the estate-owned-contract placement, so that the "why" survives the sessions that decided it.
30. As the estate owner, I want past-record conflations superseded in persistent memory when this ships, so that future sessions recall the pinned vocabulary rather than the blurred one.

## Implementation Decisions

- **Scaffold WORKFLOW template (gen 6)** rewrites the routing block as the
  full dispatch contract: four tiers with the tier→model-id mapping
  (primary=claude-fable-5, routine=claude-sonnet-5,
  mechanical=claude-haiku-4-5, fallback=claude-opus-4-8 at gen 6; ids are
  per-repo remappable); tier-default efforts (primary=high, routine=medium,
  mechanical=low; xhigh reserved for final verification and
  security-critical passes); the escalation rule (freely upward with a
  recorded reason; silent descent is a gate failure); reviewer floor + the
  four named risk triggers; fallback semantics (proactive for
  security-framed work, reactive orchestrator-mediated on refusal with
  quality framing, ledger-record + push notification — the word "auto" is
  removed); execution-mode defaults (subagent-driven/ultracode default,
  inline as justified exception); ultracode opt-in rule (plan-gated via
  ticket approval; recommend-only mid-build). The existing standalone
  `security_routing` key folds into the fallback semantics.
- **Issue/ticket template (gen 6)** gains optional frontmatter fields:
  `execution-mode`, `tier`, `effort` (override only), `risk-triggers`,
  `review-tier`. Optional so plain bug-ledger issues stay lean; /to-tickets
  fills them for build tickets.
- **`spine audit routing` subcommand** (two-level command tree per the
  existing ADR): a new audit package whose boundary is a pure function from
  (repo, transcript records) to a routing report — per task: declared tier,
  actual model(s), verdict ∈ {match, escalated-with-reason,
  silent-descent, unmapped-dispatch, no-transcript}. CLI layer is a thin
  printer. Exit non-zero only on silent-descent; missing/unparseable
  transcripts produce warnings (parser rot must not fail builds).
- **Transcript source**: the harness's per-project session records (JSONL)
  are the ground truth for actual models; this is an undocumented internal
  format, hence the graceful-degradation requirement.
- **Escalation records**: escalation reasons live in the build ledger
  (execution progress file) in a greppable one-line form the audit can
  consume.
- **Local skill /to-tickets** gains the assignment step: for each ticket it
  emits, assign execution mode, tier, effort override if any, and risk
  triggers, using the glossary vocabulary; recommend ultracode where the
  work's shape demands it (the owner's ticket approval is the opt-in).
- **No plugin-cache modifications.** The upstream subagent-driven skill
  already mandates explicit model choice at dispatch; the estate contract
  supplies the vocabulary and values it lacks. Optional upstream filing for
  first-class plan-schema fields, only on the owner's word.
- **Generation bump 5→6** with the standard migration so `spine update`
  carries existing gen-5 repos forward; ADR authored for tiers-not-ids +
  estate-owned placement.
- **Rollout order**: spine self-update → deepthought + objectstudio adopt →
  one real build exercises the machinery end-to-end (praxis I001 is the
  natural candidate) → fleet sweep.

## Testing Decisions

- Good tests here assert external behavior at two seams only: (1) what the
  templates render / what `spine update` rewrites, and (2) what the audit
  function reports for a given fixture repo + fixture transcript. No
  assertions on internals or intermediate representations.
- **Template/scaffold seam** (existing): scaffold tests assert the gen-6
  WORKFLOW contract content (tiers present, "auto" absent, effort defaults,
  escalation rule) and the ticket template's new fields; a gen5→6 migration
  test follows the established genNtoM pattern (prior art: the existing
  gen1→2 … gen4→5 tests).
- **Audit seam** (the one new seam): package tests drive the pure audit
  function with fixture repos and fixture transcript JSONL covering: clean
  match; escalation with reason (advisory); silent descent (blocking);
  dispatch of an unmapped model (warn); missing transcript dir and
  malformed JSONL (warn, never fail); tickets with no annotations (skipped,
  reported as unannotated). Prior art: adopt/doctor packages test against
  fixture repos in testdata.
- **CLI wiring** is covered by the same thin-dispatch convention as existing
  subcommands (no dedicated CLI test beyond dispatch reachability).
- **Skill-behavior layer** (/to-tickets assignment, dispatch discipline,
  fallback+notify) is not unit-testable prose; it is verified by the D10
  dogfood build, where `spine audit routing` is itself the runtime check
  that the discipline held.

## Out of Scope

- Auto-detection of refusals by the harness ("auto" fallback) — the
  orchestrator is the mechanism; no hook/daemon watches stop reasons.
- Patching or wrapping the upstream superpowers skills; any upstream
  contribution is a separate, owner-gated follow-up.
- Routing enforcement for inline work — inline means the session model
  executes by definition; the control on inline is that tickets must
  justify choosing it.
- Cost accounting/budget tracking (the audit reports models used, not spend).
- Multi-provider dispatch mechanics (OpenAI, local models) — the tier
  vocabulary deliberately supports the mapping, but only Anthropic ids ship
  in gen 6.
- Retrofitting annotations onto historical plans/tickets.
- The praxis I001 build itself (it is the exercise, not part of this spec).

## Further Notes

- Vocabulary is pinned in `CONTEXT.md` (execution modes, tiers, escalation /
  silent descent, fallback routing, effort routing, routing audit); the
  spec deliberately reuses those terms and no others.
- Evidence base for the design: the objectstudio transcript audit
  (205/205 primary-model messages, 2 dispatches), the vfb build history
  (same-tier task reviews + primary final review caught gate escapes four
  builds running), and the confirmed finding that the primary model's
  security classifier trips on framing rather than file contents.
- The push-notification mechanism for fallback events should reuse the
  owner's existing notification channel configuration rather than invent a
  new one; exact mechanism is a plan-time detail.
- When this ships, supersede the persistent-memory entries that conflate
  ultracode with subagent-driven (per user story 30).
