---
title: spine v3 design — ledger sweep + gen-3 template batch
created: 2026-07-03
status: draft — awaiting Russell's review
---

# spine v3 — ledger sweep + gen-3 template batch

v3 clears the deferred-work ledger accumulated across the v1 and v2 builds
(`.superpowers/sdd/progress.md`) plus the v2 handoff's v3 list. No new command
surface. One deliberate design deferral, recorded below.

Scope was set without live input (Russell AFK at both scope forks); every call
below is individually veto-able at spec review.

## Goals

1. Close the Stat-then-Write TOCTOU window in all create-new paths.
2. Fix the fleet `age_days` UTC off-by-one.
3. Stop swallowing read/Stat errors in three known spots.
4. Ship the three text-output cosmetics.
5. Land the gen-3 template batch: strict-YAML-safe `adr.tmpl.md` scalars.

## Non-goals (explicit)

- **adr-README template evolution vs ADR-0009 preservation** — deferred. Gen 3
  does not touch `templates/current/adr-README.md`, so the preserve-heuristic
  question stays dormant. Reopen when a future generation actually evolves that
  template.
- New commands, profiles, or fleet operations.
- The ratified by-design items stay by-design: write-batch non-transactional,
  planSimple repo-wide generation inference, unicode slug collapse, doctor.Run
  always-nil error signature.

## Components

### C1 — fsutil.WriteFileExclusive (TOCTOU close)

New primitive in `internal/fsutil`: write content to a temp file in the target
directory, then `os.Link(tmp, path)` — atomic create-if-absent with full
content, `EEXIST` if the target appeared after planning, no partial file on
crash — then remove the temp. Mirrors `WriteFileAtomic`'s shape.

Call sites converted (the current three-way Stat guards collapse into the
exclusive write; user-facing "already exists" error text preserved):

- `handoff.New` (handoff.go:58–74)
- `eval.New` — eval.md write (eval.go:86); the directory-level existence check
  at eval.go:57 remains as a fast-path courtesy check, no longer load-bearing
- `eval.New` — README create-if-absent (eval.go:69): convert, treat `EEXIST`
  as success (two racers writing identical content is benign; the primitive
  makes it correct rather than coincidental)
- `eval.AddRun` (eval.go:138–153)
- `adr.New` (adr.go:139) if its collision guard matches the same pattern
  (plan-time check); the supersede flip at adr.go:146 is an intentional
  overwrite and stays `WriteFileAtomic`

Error semantics: `EEXIST` maps to the existing "already exists" user errors;
all other link/temp errors propagate with path context.

### C2 — fleet age_days off-by-one

Mechanism (verified at `cmd/spine/main.go:358`): `ageDays` divides
`time.Since(d)` by 24h, where `d` is the filename date parsed as **UTC
midnight**. West of UTC, a handoff dated today crosses the 24-hour mark during
the local evening and reports "1d". Fix: calendar-day difference between the
filename's `YYYY-MM-DD` and today's local date (the filename date is authored
in local time by `handoff new`, main.go:52 `time.Now().Format`). Inject the
clock (package-level `now func() time.Time` in cmd/spine) so tests pin
boundary cases: same-day = 0, local-evening-west-of-UTC = 0, yesterday = 1.

### C3 — swallowed-error fixes

1. `eval` doctor `checkDoc`: non-ENOENT read errors currently report as
   "missing eval.md". Distinguish: missing stays a D7 finding; read errors
   return as errors (doctor's exit-2 path).
2. `eval.List`: per-file read errors are silently skipped. Surface them —
   List returns the error (fail loud; a corrupt ledger should not silently
   thin the scoreboard). JSON contract unchanged on the success path.
3. `update.go:87`: evals-dir Stat swallows non-ENOENT errors (EACCES, ELOOP
   would silently skip evals-README management). Propagate them.

### C4 — text-output cosmetics

Verified: neither list prints a header today (`handoff list` = bare
`date  topic` rows, main.go:287; `eval list` = headerless fixed-width rows,
main.go:489–495). v3 defines one shared text style — a header row in the same
fixed-width alignment as the data rows — and applies it to both:

1. `eval list` (text): header row over the existing
   `name / run / stage / score` columns. `--json` untouched.
2. `handoff list` (text): header row, plus the path column (present in
   `--json`, absent in text).
3. `update` (text): preserved files (ADR 0009) currently print as up-to-date,
   silently. Print `preserved (hand-authored)` for them. JSON already carries
   `Preserved`; text catches up.

### C5 — gen-3 template batch

The only template edit in v3, and the reason the generation bumps:

- `templates/current/adr.tmpl.md`: quote the `id` and `title` front-matter
  scalars. Fixes strict-YAML invalidity for titles with colons (v2 ADR 0007
  bit this) and the `id: 000N`-parses-as-octal quirk for N<8. spine's own
  `meta.Parse` tolerates both quoted and unquoted forms, so `adr list` and
  doctor are indifferent; existing fleet ADR records are user-owned, never
  regenerated, and stay valid as-is.
- `templates/VERSION`: 2 → 3, same commit (never edit `templates/current`
  content within a generation — the edit and the bump are one change).
  Restore the trailing newline (v2 T1 minor); `tmpl.Version` already trims.
- Rename `TestVersionIsOne` (asserts 2 today, misleading since v2) to a
  generation-agnostic name asserting the current constant.
- New `TestGen2To3IsStampOnly` real-file fixture mirroring
  `internal/update/gen1to2_test.go`: a gen-2 repo updated to gen 3 diffs ONLY
  in the WORKFLOW.md generation stamp. Valid because `adr.tmpl.md` is
  embedded-only (read by `adr new` at generation time; absent from update's
  `simpleFiles` manifest) — verified against `internal/update/update.go:61-70`
  and `internal/adr/adr.go:123`.

Fleet impact: every stamped repo shows one pending stamp-only update after the
binary is reinstalled. No content diffs. Rollout is `spine update --write` per
repo, whenever each repo is next touched (same convention as gen 1→2).

### C6 — acceptance (live, inline with Russell)

- Full test regression (11 packages) on the branch and on merged main.
- Dogfood: spine repo self-update to gen 3; doctor clean.
- Two fleet dry-runs (one v2-adopted repo, one gen-2 updated repo, e.g.
  praxis + ccq): pending = WORKFLOW.md stamp only.
- Scratch-repo smoke: `adr new` (front-matter strict-YAML-parses AND
  `meta.Parse` reads it), `handoff new`, `eval new` + `add-run`.
- `~/bin/spine` reinstalled = gen 3.

## Testing strategy

Per-component unit tests named in each section; C1 additionally gets a
concurrent-create race test (two goroutines, exactly one wins, loser gets the
"already exists" error) and crash-safety by asserting no temp-file residue on
the EEXIST path. C5's fixture test is the regression lock for the whole
template batch.

## Build order

C1 (foundation primitive) → C3 (error-path fixes touch the same files) →
C2 + C4 (independent, parallel-safe) → C5 last (the gen bump lands once all
behavior is final) → C6 acceptance.

## Requirements-attack notes (spec self-check)

- "Never edit templates within a generation" vs "edit adr.tmpl.md": resolved
  by making the edit and the VERSION bump one atomic change (C5); the
  invariant constrains sequencing, not the edit itself.
- `TestGen1To2IsStampOnly` pins 1→2 only; it does not forbid 2→3 content
  changes to embedded-only templates. C5's new fixture pins the actual gen-3
  claim (emitted content unchanged).
- C3.2 (List fails loud) could conflict with fleet-scan resilience
  (`handoff latest --fleet` deliberately skips per-child errors, T8). Ruling:
  eval.List is single-repo (no fleet mode) — fail-loud is correct there and
  does not touch the fleet skip-branch contract.
