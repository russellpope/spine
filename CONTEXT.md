# CONTEXT — unified workflow (spine estate)

Glossary only. Implementation decisions live in docs/adr/; specs in docs/specs/.

## Execution modes

How work runs. Orthogonal to model tiers (what runs it). Every ticket/stage
declares exactly one.

- **inline** — the session itself executes the work. The rare, justified
  exception, not a default: tightly-coupled sequential chains over shared
  files, pre-specified verbatim diffs, or live-system/secret/interactive
  steps. The session model does the work, so no model routing applies.
- **subagent-driven** — sequential conveyor per superpowers
  subagent-driven-development: one fresh implementer + reviewer per task,
  gate between tasks. The default for planned build work.
- **ultracode** — multi-agent Workflow orchestration: parallel fan-out,
  judge panels, adversarial verify inside a single step. For work whose
  shape demands it (unknown-size discovery, cross-cutting audits,
  grounding sweeps, N-perspective verification). NOT a synonym for
  subagent-driven; historical notes conflate the two.

## Model routing

- **flavor** — which agent runtime executes work: **claude** or **codex**. The
  second axis of the model table, orthogonal to tier. Artifacts never name a
  flavor; the dispatcher supplies it, because the same ticket may be executed
  by either. Distinct from **functional harness** (cli/rest/framebuffer),
  which is about how a project is tested, not what runs the agent. Term
  adopted from the deepthought glossary 2026-07-24.
- **model tier** — a semantic role name, deliberately provider-agnostic:
  - **primary** — the default thinker: design, judgment, orchestration,
    final review.
  - **routine** — mechanical-but-multi-step subagent roles: implementers
    working from prose, doc edits, build fixers, task-scoped reviews.
  - **mechanical** — definitionally narrow: verbatim plan-transcription
    implementers and single-file mechanical fixes ONLY (the plan text
    already contains the code).
  - **fallback** — where primary-refused or pre-flagged dual-use/security
    work runs.
  Artifacts (plans, tickets) reference tiers, never model ids (decided
  2026-07-09, unchanged). The mapping itself is keyed by **(flavor, tier)**
  and resolves in spine; each repo's WORKFLOW.md carries a **mirror** of it
  so the estate can still remap per repo (revised 2026-07-24, ADR 0011).
- **model table** — the estate's `(flavor, tier)` → model id + optional
  effort mapping. Spine ships its defaults and remembers every default it has
  ever shipped, which is what makes an inherited value distinguishable from a
  deliberate override.
- **mirror** — the rendered copy of the resolved model table in a repo's
  WORKFLOW.md, marked spine-managed. Read by humans, not by dispatch:
  authoritative only where a value has been edited to differ from a shipped
  default. A value matching a shipped default is **inherited** and refreshed
  automatically; any other value is an **override** and is preserved.
- **reviewer floor** (decided 2026-07-09) — a task's reviewer is never a
  lower tier than its implementer; plan-time risk triggers (cross-task
  integration, concurrency/subtle state, security surfaces, plan-flagged
  ambiguity) force a primary-tier review; the final whole-branch review +
  acceptance simulation always runs primary. Review procedure (re-run the
  claims, demand raw transcripts) is mandatory at every tier. Inline
  tickets carry `review-tier: n/a` — no per-task review cycle exists;
  verify-stage gates still apply (refined 2026-07-10, I004 review).
- **routing purpose** (decided 2026-07-09) — quality ceiling first: the
  primary model is the default thinker; down-routing exists to stop waste on
  provably mechanical work, not to chase spend. Auditability is the
  enforcement layer: actual model per task is verified against declared
  routing from transcripts, every build.
- **ultracode opt-in** — the harness requires explicit user opt-in for
  Workflow orchestration. Plan-gated: tickets marked ultracode by
  /to-tickets, approved by the user, carry the opt-in; mid-build escalation
  is recommend-only (user says the word).
- **escalation** (decided 2026-07-09) — a dispatch-time tier or effort
  increase above the ticket's annotation, always with a recorded reason.
  Permitted freely. The inverse — **silent descent**, dispatching below the
  annotation without a recorded reason — is a gate failure. The ESCALATION
  record grammar covers reasoned deviation in either direction — a downward
  record keeps descent advisory (refined 2026-07-10, final review).
- **fallback routing** (decided 2026-07-09) — two paths: proactive
  (security-FRAMED work is pre-flagged at intake/plan time and routed to
  fallback from the first dispatch — the classifier keys on framing, not
  file contents) and reactive (a primary refusal triggers orchestrator-
  mediated re-dispatch on fallback with quality framing, ledger-recorded,
  push-notified). Never described as "auto" — the orchestrator is the
  mechanism.
- **effort routing** (decided 2026-07-09) — effort follows the tier's
  default (primary=high, routine=medium, mechanical=low, fallback=high)
  (amended 2026-07-10, final review); per-ticket overrides follow the
  escalation rule. A model-table entry may carry its own effort, overriding
  the tier default for that (flavor, tier) — the tier's default effort does
  not always produce the expected behavior from a given model. Resolution
  always yields a determinate effort; an entry that omits one inherits the
  tier default rather than deferring to a runtime's own setting. "xhigh
  reserved for final verification and security-critical passes" is guidance,
  not a gate (revised 2026-07-24).
- **routing audit** (decided 2026-07-09) — deterministic post-build diff of
  declared tier annotations vs actual models in the transcript, per task
  (`spine audit routing`). Required at the verify stage: reasoned
  escalations advisory, silent descent blocking.
