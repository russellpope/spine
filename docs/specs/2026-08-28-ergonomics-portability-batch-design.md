---
title: "Ergonomics + portability batch: flag-order error, implement-evidence message split, install/version provenance"
tickets: I116,I117,I118
created: 2026-08-28
status: draft
---

# Ergonomics + portability batch (I116 + I117 + I118) — design

## Problem Statement

Three small, disjoint gaps — two message-quality traps that have been
re-learned from handoff prose across sessions instead of being fixed at the
point of failure, and one portability gap raised by the owner at the grill:

1. **`spine model` flag-order trap** (I116). Stdlib flag parsing stops at
   the first positional, so `spine model openweights primary --effort`
   prints bare usage and exits 2 with no hint that ordering is the problem.
   It reads as the flavor being broken — exactly how it presented during the
   openweights rollout (2026-08-25 handoff gotcha).

2. **Implement-tick zero-evidence misdirection** (I117). Implement evidence
   requires a ledger line starting with the ticket id AND carrying
   `done`/`complete`/`completed` as a whole word (I019). When every anchored
   ticket misses, the detail appends the I032 typo hint ("check it for a
   typo") — but when the ids ARE present in the ledger and only the done-word
   is absent, the hint sends the operator to audit the tickets value while
   the actual fix is one word in a ledger line. Hit live 2026-08-25 (I110).

3. **No portable install story** (I118, filed from this grill). The repo is
   public and the binary self-contained (templates go:embed'd, no hardcoded
   paths), so `go install github.com/russellpope/spine/cmd/spine@latest`
   already works on any macOS device with Go — but the README documents only
   clone-and-`make build`, and `spine version` prints only the template
   generation, so cross-device build drift is invisible short of sha256-ing
   binaries (which is what the handoffs currently do).

## Solution

One effort, `ergonomics-portability-batch`, three tickets, solo inline in
this session (no team dispatch — the batch does not amortize one), executed
serially with TDD and the full stage gates. The cursor's
`tickets: I116,I117,I118` value is a live exercise of the comma-list grammar
the previous batch shipped.

- **I116, option B** (grill Q1): keep strict stdlib ordering; when parsing
  leaves a flag-like token among the positionals, the error names the rule
  ("flags must precede positionals") next to the offending argument. A small
  helper in `cmd/spine/main.go`, wired into `cmdModel` only (grill Q2);
  wiring other subcommands is a follow-up ticket if wanted.
- **I117**: at derivation time, distinguish "no ledger line starts with the
  id" from "a line starts with the id but carries no done-word". The wording
  message REPLACES the typo hint whenever any anchored line exists (grill
  Q3); in the mixed case the wording message wins and the existing
  missing-ids list still names the fully-absent ids (grill Q4). The typo
  hint survives unchanged for the no-line-at-all case.
- **I118**: README gains the `go install …@latest` one-liner beside the
  existing development build; `spine version` gains a second line of build
  provenance from `runtime/debug.ReadBuildInfo` (module version, vcs
  revision, vcs time, dirty flag — no ldflags), with a graceful fallback
  when build info is absent. Scope is the spine binary only (grill Q6);
  release machinery is out (grill Q7).

## User Stories

1. As an operator typing `spine model openweights primary --effort`, I want
   the error to name the ordering rule and the offending token, so that the
   failure stops reading as a broken flavor.
2. As an operator using the documented leading-flag form, I want it to
   behave exactly as today, so that the fix is purely additive.
3. As an operator whose ledger line says "I110: … declared", I want the
   ticked-missing detail to name the done/complete whole-word requirement,
   so that I fix one word instead of auditing the tickets value.
4. As an operator with a genuinely typo'd tickets value, I want the typo
   hint exactly as before, so that the narrowing loses nothing.
5. As an operator in the mixed case (some ids anchored without a done-word,
   others absent), I want the wording message plus the named missing ids, so
   that neither cause is hidden.
6. As the owner installing spine on another Mac, I want a documented
   one-liner requiring only Go, so that a second device does not need the
   clone-and-make ritual.
7. As the owner comparing two devices, I want `spine version` to print build
   provenance, so that drift is answered by one command instead of sha256.
8. As a CI or non-VCS build consumer, I want `spine version` to degrade
   gracefully when build info is missing, so that the command never errors.
9. As the human reviewer, I want the final review to include the
   requirements-attack step, so that spec contradictions surface with
   proposed resolutions instead of being silently resolved.

## Implementation Decisions

- **I116 detection**: after `fs.Parse` succeeds, any remaining positional
  beyond the first beginning with `-` triggers the ordering error — not
  just when the positional count is wrong. (`spine model openweights
  --json` leaves NArg == 2 but `--json` is never a valid tier; without
  this the trap survives in a second shape.) The first positional is
  exempt: a flag-like token there is only reachable via an explicit `--`
  terminator — a deliberate positional, not an ordering mistake — and
  falls through to `model.Resolve`'s own error. Message shape:
  `model: flags must precede positionals (saw "--effort" after "primary")`
  — the offending token and the *immediately preceding* positional, which
  pinpoints where ordering broke — followed by the existing usage line;
  exit 2 unchanged. [Amended at spec-review: example previously named the
  first positional (C1); the first-position exemption was uncodified (C3).]
- **I116 helper stays in `cmd/spine/main.go`** and is wired into `cmdModel`
  only. Other subcommands keep today's behavior; generalizing is deliberate
  follow-up scope, not silent creep.
- **I117 plumbing**: the implement-evidence collector already reads the
  ledger; it additionally records, per anchored id, whether any line starts
  with the id (regardless of done-word). `judgeSet` (or its caller) uses
  that only in the `existing == 0` branch: any anchored line ⇒ the wording
  message (naming the done/complete/completed whole-word requirement)
  replaces the typo hint; zero anchored lines ⇒ typo hint verbatim as
  today. prd/issues judging is untouched.
- **I118 version output**: first line unchanged
  (`spine template generation N` — scripts may parse it). Second line
  `build: <module-version> <rev-12> <vcs-time> [dirty]`, omitting fields
  ReadBuildInfo does not provide; `build: (no build info)` when the read
  fails or yields no usable fields — never a bare `build:` with an empty
  payload. No ldflags, no goreleaser, no VERSION bump. [Amended at
  spec-review: fallback previously scoped to a failed read only (C2).]
- **I118 README**: an Install subsection next to Development:
  `go install github.com/russellpope/spine/cmd/spine@latest`, note that the
  binary is self-contained (embedded templates), and that `spine version`
  identifies the build. The macOS-focus caveat already present stands.
- **Routing** (solo inline): all three tickets `execution-mode: inline`,
  `tier: primary` (this session, fable-5), `review-tier: n/a` per the
  ledger convention — no per-task review cycle; the verify-stage gates and
  the mandatory spec-review of the finished diff still apply. Never
  claude-sonnet-5.
- **Branch mechanics**: work on a short-lived branch off main, ff-merge at
  ship, batch commits so one `maipipe run full --wait` covers each HEAD
  move. Stage explicit paths only.

## Testing Decisions

External behavior only: assert on exit codes, stderr/stdout text, and
evidence-report notes.

- **I116 (cmd seam)**: trailing-flag invocation exits 2 and stderr names
  the ordering rule and the offending token; flag-like token with correct
  arity (`--json` as tier) likewise; the leading-flag form still resolves
  identically (green control); a genuinely unknown flavor still reports via
  `model.Resolve`'s own error, not the ordering message (negative control
  that detection keys on `-`, not on failure).
- **I117 (stages seam)**: ledger line `I0NN: … declared` + stage ticked ⇒
  wording message, not the typo hint; ledger with no line for the id ⇒ typo
  hint verbatim (negative control that the split is load-bearing); mixed
  case ⇒ wording message and the missing-ids list still names the absent
  id; any-evidence-present case unchanged.
- **I118 (cmd seam)**: `spine version` first line matches today's format;
  a `build:` line is present; the command exits 0. (Test binaries carry
  build info, so the fallback arm is covered by unit-testing the formatter
  on a nil BuildInfo.)
- Every negative control observed red — command + output recorded; a
  prescribed control is a hypothesis until run.

## Out of Scope

- Permuting flags (I116 option A) — rejected at the grill.
- Wiring the I116 helper into other subcommands — follow-up ticket if the
  trap bites elsewhere.
- Release binaries, Homebrew tap, goreleaser — beyond "mildly portable".
- Syncing the workflow surround (skills, maipipe, dotfiles) to other
  devices — chezmoi territory, separate conversation.
- Linux/Windows testing — README caveat stands.
- I111/I112 (openweights) — parked; I072/I102/I105 — passed over for this
  batch.

## Further Notes

- Effort mechanics: `spine cursor start --force --effort
  ergonomics-portability-batch --tickets I116,I117,I118 --prd <this file>`
  runs BEFORE any `spine handoff new`. Never tick the handoff stage. Read
  exit codes unpiped under fish.
- The comma-list tickets value makes this effort itself the second live
  proof of I114's grammar.
- I117 retires the 2026-08-25 handoff gotcha ("ledger implement evidence
  needs a done/complete whole word") as a live trap; the rule remains, only
  the misdirection on miss goes.
