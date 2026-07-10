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
  Artifacts (plans, tickets) reference tiers, never model ids; the tier→id
  mapping lives in each repo's WORKFLOW.md `model_routing` so the estate can
  remap (new model families, local models, other providers) without
  touching plans (decided 2026-07-09).
- **reviewer floor** (decided 2026-07-09) — a task's reviewer is never a
  lower tier than its implementer; plan-time risk triggers (cross-task
  integration, concurrency/subtle state, security surfaces, plan-flagged
  ambiguity) force a primary-tier review; the final whole-branch review +
  acceptance simulation always runs primary. Review procedure (re-run the
  claims, demand raw transcripts) is mandatory at every tier.
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
  annotation without a recorded reason — is a gate failure.
- **fallback routing** (decided 2026-07-09) — two paths: proactive
  (security-FRAMED work is pre-flagged at intake/plan time and routed to
  fallback from the first dispatch — the classifier keys on framing, not
  file contents) and reactive (a primary refusal triggers orchestrator-
  mediated re-dispatch on fallback with quality framing, ledger-recorded,
  push-notified). Never described as "auto" — the orchestrator is the
  mechanism.
- **effort routing** (decided 2026-07-09) — effort follows the tier's
  default (primary=high, routine=medium, mechanical=low; xhigh reserved for
  final verification and security-critical passes); per-ticket overrides
  follow the escalation rule.
- **routing audit** (decided 2026-07-09) — deterministic post-build diff of
  declared tier annotations vs actual models in the transcript, per task
  (`spine audit routing`). Required at the verify stage: reasoned
  escalations advisory, silent descent blocking.
