---
id: 0006
title: stdlib dispatch holds for the two-level v2 command tree
status: Accepted
date: 2026-07-02
---

# 0006: stdlib dispatch holds for the two-level v2 command tree

## Context

ADR 0001 kept spine on stdlib-only map-based dispatch and flagged one reconsider-trigger: v2
growing nested command trees. v2 grew the command surface with `adopt`, `handoff`, and `eval`,
and two of those — `handoff` and `eval` — are two-level trees (`adopt` stays flat and
flag-driven, per ADR 0008). That nesting fired the trigger, so dispatch needed a real look,
not a rubber stamp: does the resulting tree actually need cobra, or does stdlib dispatch
still hold it?

## Decision

It holds. The v2 tree is exactly two levels deep — `spine handoff new|list|latest` and
`spine eval new|add-run|list` — the same shape v1 already had with `adr new|list`. Sub-actions
stay single-token (`add-run`, not `eval run new`), so each verb is still one map lookup plus one
`flag.NewFlagSet`, no nested subcommand routing, no persistent-flag inheritance. This ADR does
not supersede 0001 — it reaffirms it. Cobra's reconsider-trigger moves forward to "three levels
of nesting or persistent flags shared across a subtree," neither of which v2 introduces.

## Consequences

Dispatch code stays flat and auditable; no `--supersedes` flag on this record because 0001's
decision is unchanged, only its trigger condition is updated. The next command-tree feature
gets evaluated against the new bar (three levels / persistent flags) rather than re-litigating
stdlib-vs-cobra from scratch each time.
