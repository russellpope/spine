---
title: "spine v4 shipped: quoting class closed"
created: 2026-07-03
---

# Handoff — spine v4 shipped: quoting class closed (2026-07-03)

## Context

spine v4 designed, planned, built, reviewed, and live-accepted in one session,
same full superpowers spine as v3 (brainstorm → spec → plan → subagent-driven
→ fable final review with acceptance simulation → C7 live acceptance). Spec
`docs/specs/2026-07-03-spine-v4-design.md` (approved, two dated final-review
amendments), plan `-plan.md`, execution ledger `.superpowers/sdd/progress.md`
(gitignored). This handoff's own front matter is the release demo: the gen-4
binary quoted its colon title.

## State (verify before relying)

- spine main @ 2a55aef (FF from build/v4, 8 commits, branch deleted). NOT
  pushed — main is ahead of origin/main by 10 (v4 spec+plan+branch); push on
  Russell's word.
- `~/bin/spine` = generation 4. New in v4: handoff/eval `title:` front matter
  YAML-quoted via `{{HANDOFF_TITLE_YAML}}`/`{{EVAL_TITLE_YAML}}` +
  `strconv.Quote` (same class as v3's ADR fix; body H1s keep raw titles);
  shared `meta.UnquoteScalar` (adr refactored to it, handoff.List unquotes for
  display, legacy titles verbatim); gen-lock tests hardened (seen-map kills
  vacuous pass) + `TestGen3To4IsStampOnly` with ccq-gen3 fixture; backslash
  roundtrip + legacy-passthrough regression tests checked in;
  `handoff latest` rejects `-`-prefixed --fleet/--dir values (exit 2);
  WriteFileExclusive documents the hardlink-FS constraint and fails loud on
  success-path temp-cleanup errors; handoff-list topic column sizes to the
  widest entry.
- Fleet: gen-2 repos each still owe ONE stamp-only `spine update --write` —
  now to v4 (verified stamp-only 2→4 live on praxis + ccq dry-runs; praxis
  preservation notice renders; both repos untouched). Rollout stays lazy,
  per-repo as touched.

## Next steps (v5 ledger — full list in .superpowers/sdd/progress.md "v5 LEDGER SEEDS")

- handoff.List: hand-made `title: ""` blanks the filename-topic fallback
  (guard with `if t := meta.UnquoteScalar(...); t != ""`, or ratify as
  YAML-faithful).
- strconv.Quote vs YAML escape divergence on invalid-UTF-8 titles
  (informational; same class as the ratified gen-3 mechanism).
- `./-something` directory workaround: verified live in v4 review, untested
  in suite.

## Gotchas

- FOURTH consecutive build where the final-review acceptance simulation
  out-caught task gates — this time in the acceptance PLAN itself (Task 7's
  `&&` chain died on the dry-run's expected exit 1; caught by rehearsing every
  step against the shipped binary). Keep the sim, keep requirements-attack.
- Gen-N VERSION bumps ripple into test files that hardcode the generation
  (main_test, scaffold_test, update/hbmview tests) — v4's plan missed 4 of
  them; the implementer bumped the literals as ratified mechanical collateral
  inside the atomic commit. Future gen-bump plans: list these files.
- `update` dry-run exits 1 on pending changes — never `&&` it ahead of the
  write step in scripts.
- eval doctor checkKeys is presence-only (values opaque per ADR 0007) — spec
  text originally over-claimed "non-empty"; amended.
