---
title: "Flag-ordering generalization: no spine subcommand silently discards input"
tickets: I119
created: 2026-08-28
status: draft
---

# Flag-ordering generalization (I119) — design

## Problem Statement

The I116 fix taught exactly one subcommand (`spine model`) to name the
flags-must-precede-positionals rule. Everywhere else the trap survives, and
in worse shapes than an unhelpful error — observed live this session:

```
$ spine cursor show --dir /Users/ldh/Projects/github.com/spine
no spine cursor found in .
$ echo $status
0
```

Three silent failures stack in that one invocation: `show` is not a cursor
subcommand (the dispatch switch has no case for it, so it falls through to
the bare printer), the trailing `--dir` lands in `fs.Args()` which nothing
reads (stdlib parsing stops at the first positional), and the answer comes
back exit 0 about the *wrong directory* — a clean exit and wrong data, the
worst shape for hook and script consumers.

The code sweep (this grill's fact-finding) found the class repo-wide: ~25
hand-rolled `flag.NewFlagSet` + `fs.Parse` sites in `cmd/spine/main.go`,
almost all of which silently ignore post-positional flags AND stray
positionals (`spine doctor foo` discards `foo` without comment). `gate`
inverts the house rule outright: it reads its two positionals from raw args
*before* any flag parsing, so `spine gate --dir X pack check` mis-reads the
pack as `--dir`. Only `cursor` among the dispatchers swallows unknown
sub-subcommands; `adr`/`handoff`/`eval`/`audit`/`checkpoint` already error.

## Solution

One effort, `flag-ordering-generalization`, one ticket (I119), solo inline
in this session with TDD and the full stage gates. All grill rounds
ratified on the recommended answers (Q1–Q11).

- **Strict ordering everywhere, I116 error shape** (Q1): a flag-like token
  among the positionals errors
  `<cmd>: flags must precede positionals (saw %q after %q)` + usage,
  exit 2. No permutation, no lenient re-parse. [Amended at spec-review,
  C1: "everywhere" means every *parsing* subcommand — the ratified
  carve-outs stand beside it: `--force` stays position-free on
  `cursor start`/`cursor tick` (takeForce, documented), and
  `version`/`help` stay lenient (Q11). The README's one-line rule is the
  operator summary; this paragraph is the precise reading.]
- **Every subcommand, via a shared parse helper** (Q2, Q6): a `parseArgs`
  helper in `cmd/spine/main.go` owns Parse + ordering guard + arity check +
  usage printing; every `cmd*` converts to it. Per-site wiring was rejected
  as 25 chances to drift and a guarantee the class recurs with the next new
  subcommand.
- **Exact arity enforced** (Q8): a stray positional beyond `wantN` errors
  `<cmd>: unexpected argument %q` + usage, exit 2. Same defect class —
  input silently discarded.
- **`cursor` gets the dispatcher treatment** (Q7): an unknown cursor
  sub-subcommand errors exit 2 with a usage line naming
  `start|tick|here|set`, mirroring the other dispatchers. The documented
  exit-0-always contract narrows to *flag-only* invocations (hooks never
  pass positionals); the `cmdCursor` doc comment is amended in the same
  commit. The no-cursor-found exit 0 for flag-only invocations stands (Q3
  — documented contract, not a bug).
- **`gate` joins the house grammar** (Q9): cmdGate parses flags first via
  the shared helper (wantN 2), taking pack/check from the parsed
  positionals. No estate caller passes flags to gate at all
  (`maipipe.toml`, ADRs 0015/0019, README all use the bare
  `spine gate go@1 <check>` form), so nothing breaks; the previously-working
  trailing-flag form now errors with the rule named instead of mis-reading.
- **`handoff latest` keeps its bespoke diagnostic** (Q10): the
  value-swallowed-a-flag check is a different shape. [Amended at
  spec-review, C2: the Solution's "aligns with the I116 wording" and this
  section's "reworded only as needed" conflicted; "as needed" governs.
  Alignment was judged at implementation: the existing message already
  names the flag and the mistake in the house style, so no rewording was
  needed and none was made.]
- **`version`/`help` stay lenient** (Q11): no silent wrong answer is
  possible there, and erroring on `spine version --help` would be hostile.
- **Guidance sweep** (owner requirement at ratification): after the fix,
  sweep the surfaces that teach the workaround — repo docs
  (README/WORKFLOW examples), `~/.claude` skills that invoke spine
  (model-eval and any grep hits), auto-memory entries, and living handoff
  prose. The flags-first rule remains true and documented; what retires is
  the "silently ignores" warning and any example that would now error.

## User Stories

1. As an operator typing `spine cursor show --dir X`, I want an error
   naming the unknown subcommand, so that I never read the wrong repo's
   cursor with a clean exit.
2. As an operator putting a flag after positionals on ANY spine subcommand,
   I want the I116-shaped error naming the rule and the offending token, so
   that the failure never reads as broken data or a broken subcommand.
3. As an operator typing a stray positional (`spine doctor foo`), I want an
   `unexpected argument` error, so that discarded input is impossible.
4. As a hook consumer invoking `spine cursor` with flags only, I want
   exit 0 always, exactly as documented today, so that no hook breaks.
5. As a maipipe lane running `spine gate go@1 <check>`, I want identical
   behavior, so that every green lane stays green.
6. As an operator using any currently-documented flags-first invocation, I
   want it to behave identically, so that the change is purely additive on
   the happy path.
7. As the author of the next spine subcommand, I want the parse helper to
   be the obvious single entry point, so that the guard cannot be forgotten.
8. As a future session reading skills/memory/docs, I want the
   "spine silently ignores trailing flags" warnings gone, so that retired
   gotchas stop being re-taught.
9. As the human reviewer, I want the spec-review to include the
   requirements-attack step, so that spec contradictions surface with
   proposed resolutions instead of being silently resolved.

## Implementation Decisions

- **`parseArgs` helper** (name indicative), in `cmd/spine/main.go`:
  signature on the order of
  `parseArgs(fs *flag.FlagSet, args []string, name, usage string, wantN int, stderr io.Writer) ([]string, bool)`.
  It runs `fs.Parse` (stdlib parse errors keep their behavior, exit 2),
  then the ordering guard (`flagAmongPositionals` on `fs.Args()` — the
  existing helper with its first-position `--` exemption intact, which the
  single-positional free-text commands `adr new` / `handoff new` /
  `eval new` depend on), then arity: `wantN >= 0` enforces exact NArg
  (`unexpected argument` on overrun; usage on underrun, matching today's
  messages where they are bespoke-but-equivalent); `wantN == -1` skips
  arity for any command with genuinely variable positionals (none known
  today). Returns the positionals and ok.
- **Guard placement**: always post-`fs.Parse`, on `fs.Args()`, never raw
  args — `cursor start`/`cursor tick` legitimately accept `--force` after
  the stage positional because `takeForce` strips it pre-parse; the helper
  is called with the post-`takeForce` args, so that documented form stays
  green.
- **Conversion is total**: every `cmd*` FlagSet site in main.go calls the
  helper — including `cmdModel`, whose guard moves into it with byte-for-
  byte identical output (its I116 tests must pass unmodified; they are the
  behavior contract).
- **cursor dispatch**: in the sub-subcommand switch, a first arg that is
  neither a known subcommand nor flag-like errors
  `unknown cursor subcommand %q` + usage, exit 2 — same shape as
  `adr`/`handoff`/`checkpoint`. Flag-only invocations reach the bare
  printer exactly as today (exit 0 always, including no-cursor-found). Doc
  comment at the top of `cmdCursor` amended to state the narrowed contract.
- **gate rework**: flags parsed first (shared helper, wantN 2), pack and
  check taken from the returned positionals. Exit-code contract unchanged
  (0 pass, 1 findings, 2 misconfiguration — usage errors are exit 2,
  consistent). ADR run-line grammar `spine gate <pack>@<v> <check>`
  untouched.
- **handoff latest**: bespoke consumed-value check stays; message reworded
  only as needed to share vocabulary with the ordering error.
- **No new files, stdlib only** (ADR 0001). All changes in
  `cmd/spine/main.go` + tests, plus the docs/skills sweep.
- **Guidance sweep, concretely**: grep repo docs and `~/.claude` (skills,
  CLAUDE.md, auto-memory) for spine invocations and flag-ordering
  guidance; fix any example that would now error; retire "silently
  ignores" warnings (the rule itself stays documented — it is now enforced
  with a helpful error, which is the point). Out-of-repo edits are listed
  in the handoff, not committed here. [Amended at spec-review, C3: the
  sweep's scope is the broad reading — every *living* repo doc whose
  example would now error (the review caught
  docs/mutation-battery-checklist.md, fixed). Historical records — shipped
  specs/plans, handoffs, closed issues — are archives and are not
  rewritten.]
- **Routing** (solo inline): I119 `execution-mode: inline`,
  `tier: primary` (this session, fable-5), `review-tier: n/a`; the
  verify-stage gates and mandatory spec-review still apply. Never
  claude-sonnet-5.
- **Branch mechanics**: short-lived branch off main, ff-merge at ship,
  batch commits so one `maipipe run full --wait` covers each HEAD move.
  Stage explicit paths only.

## Testing Decisions

External behavior only: exit codes and stderr/stdout text via the existing
`runCmd` harness (`main_test.go`), which calls `run` directly.

- **Table-driven ordering sweep**: one table enumerating a trailing-flag
  invocation per converted subcommand ⇒ exit 2, stderr names the rule and
  the offending token. The table doubles as the conversion checklist — a
  subcommand missing from it is a review finding.
- **Arity sweep**: representative stray-positional invocations
  (`doctor foo`, `update junk`, `cursor start` extras) ⇒ exit 2,
  `unexpected argument`.
- **cursor**: `cursor show` ⇒ exit 2 naming the unknown subcommand and the
  real ones; `cursor --dir <empty>` flag-only ⇒ exit 0, no-cursor message
  (green control on the narrowed contract); `cursor tick <stage> --force`
  in a scratch repo ⇒ still works (takeForce interplay).
- **gate**: `gate --dir X go@1 <check>` resolves the same as
  `gate go@1 <check> ` run in X (flags-first now valid);
  `gate go@1 <check> --dir X` ⇒ exit 2 naming the rule;
  bare `gate go@1 binary-hygiene` at repo root stays green (maipipe form —
  green control). [Amended at spec-review: the repo-root control is
  satisfied live by the maipipe full lane (which runs all six bare gate
  stages against the real tree at each verified commit), not by a unit
  test — a unit test's CWD is not the repo root, so the lane is the
  honest observation point.]
- **First-position exemption**: `adr new -- "-Title"` (and the model `--`
  form) still reaches the command's own logic, not the ordering error.
- **Behavior contracts pinned**: the three I116 model tests pass
  unmodified; `spine version` / `help` accept anything as today.
- **Negative control**: revert the guard wiring inside the helper (keep
  tests) ⇒ the sweep goes red; restore. Every negative control observed
  red — command + output recorded; a prescribed control is a hypothesis
  until run.
- Full lane: `go test ./...`, `gofmt -l`, `go vet ./...`, then
  `maipipe run full --wait` at each HEAD move.

## Out of Scope

- Permuting flags to any position (rejected at the I116 grill, reaffirmed
  Q1).
- Changing `cursor`'s flag-only exit-0 contract or the no-cursor-found
  message (Q3 — documented hook contract).
- Strict-ifying `version`/`help` (Q11).
- Folding `handoff latest`'s bespoke diagnostic into the general guard
  (Q10 — revisit only if spec-review finds the shapes genuinely overlap).
- A cobra/pflag-style framework or any non-stdlib dependency (ADR 0001).
- Committing out-of-repo guidance edits (skills, memory) inside this repo —
  they ride the handoff.

## Further Notes

- Effort mechanics: `spine cursor start --force --effort
  flag-ordering-generalization --tickets I119 --prd <this file>` runs
  BEFORE any `spine handoff new`. Never tick the handoff stage. Read exit
  codes unpiped under fish.
- This effort retires the last living "flags before positionals" handoff
  gotcha: after it, every spine subcommand either works or names the rule.
  I116's Resolution names this exact generalization as the follow-up.
