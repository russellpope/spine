---
id: I001
title: Model routing — estate-owned dispatch contract, ticket annotations, spine audit routing (gen 6)
severity: med
status: open
affects: []
blocked-by: []
labels: [ready-for-agent]
---

## Problem

Declared model routing (`model_routing` in scaffolded WORKFLOW.md) is
connected to nothing at dispatch time: sessions on the primary model do all
dev work themselves (objectstudio: 205/205 assistant messages on primary,
2 subagent dispatches), "ultracode" vs "subagent-driven" is conflated in
records, the template promises an "auto" refusal fallback no mechanism
implements, and there is no per-task declared intent to review before a
build or audit after it.

## Fix

Implement `docs/specs/2026-07-09-model-routing-design.md` (approved
2026-07-09): gen-6 templates carrying the full dispatch contract (four
provider-agnostic tiers, tier-default efforts, escalate-only rule, reviewer
floor + risk triggers, proactive/reactive fallback with notification,
plan-gated ultracode opt-in), ticket-template annotation fields filled by
/to-tickets, and a deterministic `spine audit routing` subcommand required
at the verify stage (advisory on reasoned escalations, blocking on silent
descent). Vocabulary in `CONTEXT.md`. Rollout dogfood-first: spine →
deepthought + objectstudio → one real build → fleet sweep.
