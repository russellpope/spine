---
id: I010
title: Stage-skipping controls — wayfinder map (gen 8)
severity: med
status: fixed
affects: [template, doctor, audit, cli, fleet]
blocked-by: []
labels: [wayfinder:map]
---

## Destination

Every spine repo mechanically resists stage-skipping: the stage-cursor + handoff rule is spine-owned (template gen 8), a shared derivation engine judges the ledger cursor against on-disk artifacts — `spine audit stages` blocks on mismatch, doctor advises — session starts surface the cursor automatically via a global hook + `spine cursor`, /handoff embeds the cursor verbatim, and the whole fleet (17 repos) is on gen 8.

## Notes

- Origin: 2026-07-15 workflow-adherence audit; the ultima-dci-edition /to-tickets skip (session 913451b3) and its repo-local ledger-cursor fix (ultima WORKFLOW.md "Stage cursor (consistency rule)").
- **Execution override (owner, 2026-07-15):** this map carries through to delivery — decisions were grilled in the charting session; the build runs the full gates the same night (PRD via to-spec → tickets via to-tickets → overnight subagent-driven development with routing records + `spine audit routing`), morning /spec-review + verify with the owner.
- Skills: /grilling for any reopened decision; deepthought WORKFLOW.md model routing applies (primary claude-fable-5).

## Decisions so far

- [Blocking home for the stage check](I011-blocking-home-stage-check.md) — new `spine audit stages` blocks; doctor gains an advisory check on the same engine.
- [Effort anchor for stage derivation](I012-effort-anchor-stage-derivation.md) — a structured, machine-parseable cursor block in `.superpowers/sdd/progress.md` anchors the effort; bidirectional mismatch blocks; no ledger ⇒ warn-only.
- [Session-start cursor surfacing](I013-session-start-cursor-surfacing.md) — one global SessionStart hook + new read-only `spine cursor` subcommand; no per-repo wiring.
- [Handoff cursor hardening](I014-handoff-cursor-hardening.md) — /handoff runs `spine cursor` and embeds output verbatim; doctor advises + `audit stages` blocks when the newest handoff lacks a cursor block.
- [Fleet sweep scope](I015-fleet-sweep-scope.md) — all 17 repos to gen 8; ultima last, via the sanctioned reconciliation.
- [Overnight execution gates](I016-overnight-execution-gates.md) — full gates tonight: to-spec + to-tickets this session, SDD overnight, spec-review + verify in the morning.
- [Ultima WORKFLOW.md gen7→8 reconciliation](I017-ultima-gen8-reconciliation.md) — gen 8 lists ultima's hand-written section lines in `supersededLines` so plain `spine update --write` upgrades it cleanly; `--force` with reviewed diff is the fallback.

## Resolution

Destination reached and SHIPPED 2026-07-16 (main @ d1aed62, pushed): stage cursor spine-owned in gen 8, `spine audit stages` blocking + doctor advisory live, SessionStart hook + `spine cursor` live, /handoff hardened, fleet swept 17/17 (ultima clean). Residual tracked outside the map: [I028](I028-story11-closure-fleet-residue.md) (objectstudio/maipipe WORKFLOW reconciliation + 4 uncommitted repos) keeps the "whole fleet" clause honest; polish follow-ups I024-I027.

## Not yet specified

(empty — the way is clear; remaining work proceeds as build tickets per the PRD)

## Out of scope

- Per-repo `.claude/settings.json` hooks for collaborators who don't share the global settings — global hook chosen ([I013](I013-session-start-cursor-surfacing.md)); revisit only if spine repos gain outside collaborators.
- Stale-`template_version` on-touch nudge for dormant repos — mooted by the all-17 sweep ([I015](I015-fleet-sweep-scope.md)).
- Enforcing /spec-review and audit-routing *frequency* fleet-wide (a separate adherence gap from the 2026-07-15 audit) — beyond this destination; a fresh effort if pursued.
