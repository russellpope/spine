---
id: I119
title: "spine subcommands silently ignore trailing flags and stray positionals; cursor swallows unknown sub-subcommands"
severity: medium
status: fixed
batch: 2026-08-28-flagorder#1
commits: [65d347c, d39398d, 0b42006]
affects: [I116]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

The I116 fix covers `spine model` only. Everywhere else, stdlib parsing
stops at the first positional and nothing inspects the remainder, so input
is silently discarded — and in the worst case the answer is wrong data with
a clean exit. Observed live (2026-08-28):

```
$ spine cursor show --dir /Users/ldh/Projects/github.com/spine
no spine cursor found in .          # exit 0
```

`show` is not a cursor subcommand (no dispatch case — falls through to the
bare printer), the trailing `--dir` lands unread in `fs.Args()`, and the
command reports on the CWD with exit 0. Worse than an error for hook and
script consumers.

The class is repo-wide: ~25 hand-rolled FlagSet sites in
`cmd/spine/main.go` ignore post-positional flags and stray positionals
(`spine doctor foo` discards `foo`); `gate` inverts the house rule by
reading its positionals before parsing (`spine gate --dir X pack check`
mis-reads the pack as `--dir`); `cursor` is the only dispatcher without an
unknown-subcommand error.

## Fix

Generalize the I116 guard via a shared parse helper owning
Parse + ordering guard + arity + usage, converted into every subcommand;
unknown cursor sub-subcommands error like the other dispatchers (flag-only
cursor invocations keep the documented exit-0 contract); gate parses flags
first. Full decisions (grill Q1–Q11, all ratified) in
docs/specs/2026-08-28-flag-ordering-generalization-design.md. Includes a
guidance sweep retiring the "silently ignores" warnings from docs, skills,
and memory prose.

## Related

- **I116** — the `spine model` guard this ticket generalizes; its
  Resolution names `spine cursor show --dir X` as the candidate.
- docs/handoffs/2026-08-28-ergonomics-portability-batch-shipped.md —
  gotcha recording the live observation.

## Resolution

Shipped 2026-08-28 (grill Q1–Q11 all ratified on recommendations, PRD
amended C1–C3 at spec-review). `parseArgs` in cmd/spine/main.go owns
Parse + ordering guard + exact arity + usage for all 24 FlagSet sites;
the 24-entry ordering-sweep table in strictargs_test.go is the
conversion checklist. Unknown cursor sub-subcommands error exit 2 naming
start|tick|here|set (flag-only invocations keep the exit-0 hook
contract, doc comment amended); `spine gate` flipped to flags-first
(`gate [--dir D] <pack>[@<v>] <check>`), no estate caller affected —
maipipe run lines are bare positionals. First-position `--` exemption
and trailing `--force` preserved (both pinned); version/help stay
lenient. TDD both arms: sweep observed RED (the filing repro reproduced
verbatim — wrong repo answered, exit 0), then green; negative controls
observed red twice (guard disabled: 24/24 fail; cursor dispatch
disabled: unknown-subcommand test fails). Living-docs sweep fixed
README and docs/mutation-battery-checklist.md; no skill, memory, or
hook taught the retired workaround. Commits `65d347c` (implementation),
`d39398d` (README rule), `0b42006` (spec-review response). maipipe full
lanes #1 @d39398d and #3 @0b42006 passed; ff-merged to main at
`0b42006`. Work done.
