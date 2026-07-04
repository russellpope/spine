---
title: spine v4 design — YAML defect-class closure + lock hardening
created: 2026-07-03
status: approved (Russell, 2026-07-03, AskUserQuestion spec-review gate)
---

# spine v4 — YAML defect-class closure + lock hardening

v4 clears the full v3-final-review ledger (`.superpowers/sdd/progress.md`
"v4 LEDGER SEEDS"): finish the strict-YAML title-quoting defect class the v3
ADR fix opened, harden the generation-lock tests, check in the two live-only
regression tests, and land three small papercuts. No new command surface.

Scope ratified live 2026-07-03 (AskUserQuestion): full seven-seed sweep;
fsutil constraint documented, no fallback; `--fleet` rejects flag-like values.
The `WriteFileExclusive` cleanup-error fail-loud call (C5) was the controller's
pick at the design gate, flagged and approved with the design.

## Goals

1. Quote `title:` front matter in `handoff.tmpl.md` and `eval.tmpl.md`
   (gen-4 template batch) — same class as v3's ADR fix.
2. Kill the vacuous-pass shape in `TestGen2To3IsStampOnly`; add the gen-4
   mirror lock born hardened.
3. Check in the backslash-roundtrip and unquoted-title-passthrough regression
   tests (live-verified in the v3 review, absent from the suite).
4. Fix the `handoff latest --fleet --dir X` flag-value papercut.
5. Document `WriteFileExclusive`'s hardlink-FS constraint; stop ignoring the
   success-path `os.Remove` error.
6. Fix handoff-list topic-column misalignment past 28 chars.

## Non-goals (explicit)

- No `WriteFileExclusive` fallback write path for no-hardlink filesystems
  (fleet is APFS; YAGNI until a real consumer hits it — ratified).
- No `update` manifest changes: `handoff.tmpl.md`/`eval.tmpl.md` are
  embedded-only (read by `new` at generation time; absent from
  `internal/update/update.go` `simpleFiles`), so gen 3→4 is stamp-only for
  every fleet repo — verified against update.go:61-70, handoff.go:60,
  eval.go:78.
- adr-README evolution vs ADR-0009 preservation stays dormant (v3 deferral
  holds; gen 4 does not touch `adr-README.md`).
- No general flag-parsing rework; the `-`-value guard applies to
  `handoff latest` (`--fleet`, `--dir`) only, where the papercut lives.
- Ratified by-design items stay: always-nil `doctor.Run`, fail-loud
  `eval.List`, unicode slug collapse, non-transactional write batch.

## Components

### C1 — gen-4 template batch: handoff + eval title quoting

Mirror the v3 ADR pattern exactly (adr.go:133-148, adr.tmpl.md):

- `templates/current/handoff.tmpl.md`: front matter becomes
  `title: {{HANDOFF_TITLE_YAML}}`; the body H1 keeps raw `{{HANDOFF_TITLE}}`.
  `handoff.New` renders `_YAML` via `strconv.Quote(topic)`, prose raw.
- `templates/current/eval.tmpl.md`: front matter becomes
  `title: {{EVAL_TITLE_YAML}}`; the body H1 keeps raw `{{EVAL_TITLE}}`.
  `eval.New` renders likewise.
- The placeholder RENAME is deliberate (v3 precedent): if template and code
  ever skew, the output contains a literal `{{…_TITLE_YAML}}` instead of
  silently-unquoted YAML — loud failure over quiet regression.
- Read-back: `handoff.List` (handoff.go:102, `e.Title = kv["title"]`) gains
  the adr.go:78-82 unquote-if-parses step. Titles that parse as Go-quoted
  strings display unquoted; anything else (every pre-gen-4 handoff) passes
  through verbatim — the v3-ratified display contract. This covers `Latest`,
  `handoff list --json`, and the fleet JSON, all of which read through List.
- eval has NO title read-back display: `eval list` prints the directory name,
  and doctor's `checkDoc` only requires the `title` key be present and
  non-empty — a quoted title still satisfies it. No eval-side unquote path
  exists or is needed; this sentence is here so a reviewer doesn't hunt for
  one.
- The newline-injection guards in `handoff.New`/`eval.New`/`adr.New` stay
  (defense in depth; quoting alone would also neutralize injection).
- `templates/VERSION` 3 → 4 in the SAME commit as the two template edits
  (never edit `templates/current` content within a generation).

Fleet impact: each stamped repo's one pending update becomes a
`template_version 2→4` (or 3→4) stamp + marker diff — still exactly one
stamp-only `spine update --write` per repo, same lazy rollout.

### C2 — generation-lock hardening

`internal/update/gen2to3_test.go` `TestGen2To3IsStampOnly` today passes
vacuously if `Run` reports neither fixture file (empty reports → loop body
never fires). Harden:

- Track files seen; `t.Fatalf` unless both `WORKFLOW.md` and `CLAUDE.md`
  appeared in reports.
- Scope comment stating what the lock actually pins: emitted workflow files
  from the gen-2 fixture to the CURRENT generation (the test calls
  `update.Run` with no gen pin, so after this release it verifies 2→4 — that
  drift is the lock working as intended: any future gen that changes emitted
  content must consciously touch this test); embedded templates
  (adr/handoff/eval `.tmpl.md`) are out of its reach by construction.
- New `TestGen3To4IsStampOnly` mirroring the hardened shape against a new
  `testdata/ccq-gen3` fixture: the gen-3 output of the existing ccq-gen2
  fixture, generated BY GEN-3 CODE and committed BEFORE any C1 template edit
  (see Build order — this ordering is load-bearing).

### C3 — checked-in regression tests

Both behaviors were verified live in the v3 final review but have no suite
coverage; v4 checks them in and extends them to the new quoting:

1. Backslash-in-title roundtrip: `New` with a title containing `\` (e.g.
   `back\slash: "quoted"`) → front matter is strict-YAML-valid → `List`
   returns the original human title. For adr AND handoff (eval generates the
   same quoting but has no read-back display; its assertion is that the
   emitted front matter parses and `checkDoc` passes).
2. Pre-quoting unquoted-title verbatim passthrough: a hand-made file with
   `title: plain: unquoted` (or the ccq fixtures' real records) lists
   verbatim, no unquote mangling. For adr AND handoff.

### C4 — `handoff latest` flag-value guard

`cmd/spine/main.go:308`: `--fleet` is a plain `flag.String`, so
`handoff latest --fleet --dir X` binds `--dir` as fleet's value and fails
later with a confusing open error. After parse, reject values with a `-`
prefix for `--fleet` and `--dir` on this subcommand:

    handoff latest: --fleet needs a directory value (did a following flag get consumed?)

Exit 2, matching this subcommand's existing parse-error return and stderr
prefix style (main.go:309-311, 317-318). A directory
literally named `-something` remains reachable as `./-something`. No other
subcommand changes.

### C5 — fsutil polish

- Doc comment on `WriteFileExclusive` gains the constraint: requires a
  filesystem supporting hard links; `os.Link` fails EPERM/ENOTSUP otherwise
  (fleet is APFS — fine; documented, not worked around).
- Success path (fsutil.go:68): `os.Remove(name)` error is currently
  discarded. Return it. Contract note in the doc comment: if the cleanup
  error is returned, THE TARGET WAS WRITTEN — path exists with full content;
  only the temp file leaked. A re-run then truthfully reports "already
  exists". Fail-loud matches the repo's ratified direction (eval.List, v3
  swallowed-error sweep). Error-path removes stay best-effort (unchanged).
- Testability note, accepted: the success-path remove failure cannot be
  forced portably in-process (no hook between link and remove); coverage is
  by review + the doc contract, not a unit test. No injection seam — that
  would be scaffolding for one test.

### C6 — handoff-list column width

`main.go:290-292` hardcodes `%-28s` for topic. Compute the column width from
the listed entries (max topic length, floor = header word `topic`), format
with `%-*s`. Topics are `ParseName` slugs — ASCII by construction (unicode
slug collapse is ratified) — so byte width equals display width. Header and
rows share the computed width; `--json` untouched.

### C7 — acceptance (live)

- Full test regression on the branch and on merged main.
- Dogfood: spine repo self-update 3→4 — dry-run shows stamp-only, write,
  doctor 0, `spine version` → 4.
- Fleet dry-runs, READ-ONLY (praxis and/or ccq): pending = stamp-only to v4;
  repos untouched; praxis preservation notice still renders.
- Scratch-repo smoke with the installed binary: `handoff new` and `eval new`
  with colon+quotes+backslash titles → front matter strict-YAML-parses,
  `handoff list` shows the human title aligned past 28 chars, `adr new`
  unchanged, `handoff latest --fleet --dir X` yields the C4 usage error,
  doctor 0.
- `~/bin/spine` reinstalled = gen 4.

## Testing strategy

C1: unit tests asserting quoted front matter from `New` (handoff, eval) and
unquote-on-display (handoff), mirroring adr_test.go:242. C2: the two lock
tests are themselves the coverage. C3: the regression tests are the point.
C4: table test over `--fleet --dir X`, `--dir --json`, and a legitimate
`--fleet DIR`. C5: existing exclusive-write tests keep passing; remove-error
path documented-only (see C5). C6: assert that with a >28-char topic every
row's path column starts at the same offset as the header's.

## Build order

1. C2 fixture snapshot FIRST: generate `testdata/ccq-gen3` with gen-3 code
   (current main), commit it together with the hardened `TestGen2To3IsStampOnly`
   and `TestGen3To4IsStampOnly` asserting 3→"current" stamp-only (passes
   trivially while VERSION=3 reports up-to-date; bites after C1's bump).
2. C3–C6 in any order (independent; C3's handoff/eval quoting assertions land
   with C1).
3. C1 LAST: both template edits + VERSION bump + `New` quoting/unquoting code
   + its unit tests, one atomic commit — the generation lands once all other
   behavior is final.
4. C7 acceptance, then merge.

## Requirements-attack notes (spec self-check)

- Seed list vs v3 Non-goals: the v3 final review already ruled handoff/eval
  title quoting was NOT covered by v3's Non-goals (only adr-README evolution
  was deferred); no contradiction reopening it here.
- "TestGen3To4IsStampOnly passes before the bump" vs "born hardened": while
  VERSION=3 the gen-3 fixture reports up-to-date, so the both-files-seen
  assertion must count reports in ANY state (up-to-date or pending), not
  pending-only — otherwise step-1 CI is red until C1 lands. The seen-check
  counts report presence; the diff-scan applies only to Pending reports.
- C4's guard vs empty `--fleet=`: an explicitly empty value already means
  "not fleet mode" today (`*fleet != ""` gate, main.go:312); the guard only
  rejects `-`-prefixed non-empty values. No semantics change for omitted flag.
- C5 fail-loud vs caller contracts: callers branch on
  `errors.Is(err, fs.ErrExist)` only; a cleanup error takes the generic
  error path with the file actually written. Accepted and documented — the
  alternative (silent temp litter in `docs/` working trees, visible as
  untracked files) is worse than one honest non-zero exit.
- C6 dynamic width vs stable output contracts: nothing machine-parses the
  text listing (`--json` is the machine surface, T-tested); width change is
  safe.
