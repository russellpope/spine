---
id: I119
title: "spine subcommands silently ignore trailing flags and stray positionals; cursor swallows unknown sub-subcommands"
severity: medium
status: open
batch:
commits: []
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
