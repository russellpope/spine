---
id: I037
title: Routing audit consumes the resolver — flavor-scoped, aliased, version-gated
severity: high
status: fixed
affects: [audit, models]
blocked-by: [I033, I036]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## What to build

Design D13–D15. The routing audit stops parsing WORKFLOW.md with its own independent parser and calls the shared resolver instead, collapsing **three** parsers into one so that what gets dispatched and what gets verified cannot disagree (corrected from "two" 2026-07-24 per I037 review RA2 — `internal/model.readOverride`, `internal/update.ExtractKeys` and `internal/audit.readMapping`, the third found during I035 review).

Tier resolution becomes flavor-scoped, and alias matching moves from substring containment to explicit aliases declared per entry — substring matching will collide as model names multiply across flavors with unrelated naming schemes.

The audit gains the `template_version` gate the update path already has: a WORKFLOW.md stamped newer than the binary compiles is refused. Its absence today is why an un-upgraded binary can emit confident verdicts from a misparse.

Flavor of a dispatch is derived from the transcript source. While codex transcript parsing is out of scope, this resolves to claude for every audited dispatch; the derivation point must be named explicitly in the code so the deferred codex-audit effort has a defined seam rather than a redesign. Where an id is declared under more than one flavor, the transcript-derived flavor decides.

What this ticket must **not** change: what the audit blocks on. Reasoned escalations stay advisory, silent descent stays blocking.

## Acceptance criteria

- [ ] Audit resolves tiers through the shared resolver; no second WORKFLOW.md parser remains in the audit path
- [ ] A dotted-key mirror resolves correctly; a generation newer than the binary compiles is refused with a clear message
- [ ] Explicit aliases resolve where substring matching was previously relied on
- [ ] A dispatch whose model maps to no entry reports unmapped rather than guessing a tier
- [ ] The flavor-derivation point is explicit and documented as the codex-audit seam
- [ ] Existing scenario fixtures (clean, degraded, mixed, vacuous) pass unchanged — block/advisory behaviour is identical
- [ ] `go test ./...` green
