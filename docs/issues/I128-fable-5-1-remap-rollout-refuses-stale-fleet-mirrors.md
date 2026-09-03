---
id: I128
title: "Fable 5.1 primary remap (68aa28f) refuses dispatch on every unrefreshed fleet mirror as retired-model, with no rollout signal, a stuck override path, and host-config coupling"
severity: med
status: open
affects: [I051, I072, I063]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: [cross-task-integration]
review-tier:
---

## Problem

Findings from a primary-tier code review of commit 68aa28f (the
`claude.primary` remap to `claude-fable-5-1`, shipped 2026-09-02 without a
ticket, spec pair, or spec-review record — the only `models/defaults.json`
commit in history without one). Each was reproduced with the HEAD binary.

1. **Rollout window with no signal.** Moving `claude-fable-5` into the
   primary row's history makes `spine model validate claude primary` exit 1
   `retired-model` on every fleet repo whose WORKFLOW.md still mirrors
   `claude-fable-5` (20 of ~28 checkouts under `~/Projects/github.com` at
   review time; only maipipe and maikanban were refreshed). The claude-team
   skill runs that validate as its dispatch preflight and, on non-zero exit,
   tells the operator to rebuild spine — the actual remedy is
   `spine update --dir R --write`. `spine doctor` on such a repo reports only
   D2 "behind template generation" and never names the retired id.
2. **Stuck override.** A mirror that pins the retired id at a non-default
   effort (`claude.primary: claude-fable-5 @ xhigh`, which the gen-9 to 10
   migration itself minted) is refused as retired-model; the refusal's
   remedy (`spine update --write`) preserves the override verbatim and
   changes nothing, so the repo can never validate without a hand edit the
   message does not describe.
3. **Host-config coupling.** `applyHostConfig` matches the resolved id
   byte-exactly with no history awareness. A `routing-host.json` that still
   lists only `claude-fable-5` makes every refreshed repo's primary tier
   unreachable (exit 2, doctor D16), and the retired-model remedy moves
   stale repos into that state. Nothing in the commit, handoff, or remap
   precedent mentions updating host configs; the I072 doctor fixture still
   ships with only the old id.
4. **Test precision.** Several render locks assert
   `strings.Contains(x, "claude-fable-5")`, a substring of the new id, so
   they pass vacuously after the remap. The gen-13 to 14 pristine lock checks
   "row changed => itemized" but not the converse, and the
   `modelRefreshMirrorRows` text exemption has no negative control; the
   ten per-generation lock skip blocks should share one helper that
   sanctions a mirror-row diff only when that lock's own `ModelRefreshes`
   itemizes it (as gen13to14 already does). `modelDefaultDivergence`
   compares a retired `model_default:` against the resolved primary without
   the row's history, so a gen-0 repo hand-set to the old id is now skipped
   as a divergence instead of retiring quietly (low; one-time `--force`).

## Fix

Split by urgency:

1. Rollout (now): refresh the remaining fleet mirrors (`spine update --dir R
   --write` per repo, or a fleet sweep), fix the claude-team preflight
   message to name the real remedy, and give doctor a finding that names a
   retired mirrored id distinctly from generation lag.
2. Stuck override: make the retired-model remedy correct for overrides
   (either migrate a historical-id override to the current id keeping its
   effort, itemized as a refresh, or say "edit the override" in the message).
3. Host config: decide whether history-aware matching belongs in
   `applyHostConfig` or whether host configs are refreshed alongside
   mirrors; update the I072 fixture and the remap precedent docs either way.
4. Tests: replace the substring locks with `hasRow` on the exact id, add the
   converse assertion to the gen-13 to 14 lock, add a negative control for
   the text exemption, and consolidate the skip blocks.

Retroactively, the remap itself needs its PRD pair and spec-review record
(the I063 routine remap shows the shape).

## Acceptance criteria

- [ ] A fleet repo mirroring the retired id fails the dispatch preflight with a message naming `spine update --dir R --write`, and `spine doctor` names the retired mirrored id.
- [ ] A repo with `claude.primary: claude-fable-5 @ xhigh` can reach a validating state by following the printed remedy alone.
- [ ] A host config listing only the retired id either resolves the current primary or is reported by doctor with the host-config remedy, and the I072 fixture carries the current id.
- [ ] Reverting `models/defaults.json` primary to `claude-fable-5` with empty history fails the render locks it currently passes.
- [ ] `docs/specs/` carries a design/plan pair for the 68aa28f remap and the ticket records its spec-review.

<!-- Record an approved-without-test exception using the exact grammar in WORKFLOW.md's Acceptance exceptions section. -->
