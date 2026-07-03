---
title: spine v3 shipped
created: 2026-07-03
---

# Handoff — spine v3 shipped (2026-07-03)

## Context

spine v3 "ledger sweep + gen-3 template batch" designed, planned, built, reviewed, and
live-accepted in one session (full superpowers spine: brainstorm → spec → plan →
subagent-driven execution → final whole-branch review with acceptance simulation →
C6 live acceptance). Spec `docs/specs/2026-07-03-spine-v3-design.md` (approved, one
post-review dated amendment), plan `-plan.md`, execution ledger
`.superpowers/sdd/progress.md` (gitignored).

## State (verify before relying)

- spine main @ a51d1c6 (FF from build/v3, 10 commits, branch deleted). **PUSHED** —
  first-ever push; origin (`git@github.com:russellpope/spine.git`) main now exists and
  tracks.
- `~/bin/spine` = generation 3. New in v3: `fsutil.WriteFileExclusive` (temp+link
  create-only) at all four create paths — including `adr new`, which previously had NO
  collision guard and silently overwrote on an ID race; three swallowed-error fixes
  (eval checkDoc read-vs-missing, handoff.List title reads, update evals-dir Stat);
  fleet `age_days` = local calendar-day diff (today = 0d, verified live in the fleet
  scan); `handoff list`/`eval list` headers + handoff path column; `update` prints
  `preserved (hand-authored):` for ADR-0009 files; gen-3 `adr.tmpl.md` YAML-quotes
  id/title/supersedes (strconv.Quote in New, Unquote for display; octal-id quirk dead).
- Fleet: every gen-2 repo shows ONE pending stamp-only update (`template_version` 2→3 +
  `spine:begin` v2→v3). Rollout = `spine update --write` per repo as each is touched.
  Verified stamp-only live on praxis (preservation notice renders) and ccq dry-runs;
  neither repo written.

## Next steps (v4 ledger — full list in .superpowers/sdd/progress.md "v4 LEDGER SEEDS")

- handoff.tmpl.md / eval.tmpl.md still emit unquoted `title:` — same strict-YAML defect
  class the v3 ADR fix closed; NOT covered by v3's Non-goals (final-review req-attack
  finding). Template edit = gen-4 bump.
- TestGen2To3IsStampOnly hardening: assert both files seen (vacuous-pass shape), scope
  comment (lock only covers WORKFLOW.md/CLAUDE.md, not other emitted templates).
- Checked-in regression tests: backslash-in-title roundtrip; pre-gen-3 unquoted-title
  verbatim passthrough (both verified live in review, untested in suite).
- `handoff latest --fleet --dir X` binds `--dir` as fleet's value → confusing
  `open --dir` error (pre-existing papercut).
- WriteFileExclusive: os.Link fails EPERM/ENOTSUP on no-hardlink filesystems (known
  constraint; fleet is APFS); success-path os.Remove error ignored (stray temp).
- handoff-list path column misaligns on >28-char topics (cosmetic).

## Gotchas

- `eval add-run` takes `--eval`/`--name` FLAGS — unlike `eval new`/`adr new`/
  `handoff new` (flags then positional title). Bit the controller during C6 smoke.
- Third consecutive build where the final-review acceptance simulation out-caught task
  gates — this time at SPEC level (C3.1 promised doctor exit-2, contradicting the
  ratified always-nil doctor.Run; shipped behavior = D7 error finding → exit 1; spec
  amended 2ff7e4c with a dated note). Keep the sim, and keep requirements-attack in
  every reviewer dispatch.
- Gen-N template batches: snapshot the gen-(N-1) fixture with gen-(N-1) code and COMMIT
  IT before any template edit; template edit + VERSION bump = one atomic commit.
- eval.List is now fail-loud (first checkDoc error aborts the listing, partial results
  lost) — plan-mandated; doctor still exits 1 via the D7 error finding. Revisit only if
  multi-damage-tree doctor UX hurts.
