---
id: I120
title: "adr new: --supersedes parses zero-padded ids as octal and flips the wrong ADR"
severity: med
status: open
affects: []
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

ADR ids are conventionally zero-padded (`0001`, `0011`); `spine adr new` takes
`--supersedes` as `fs.Int` (cmd/spine/main.go:277), and Go's `flag.Int` parses
with base 0, so a leading zero switches the radix to octal:

- `spine adr new --supersedes 0011 "..."` → supersedes **9**, not 11.

Observed live in maikanban on 2026-08-28: the command scaffolded ADR 0012 with
`supersedes: "0009"` and flipped ADR **0009**'s status to `Superseded by 0012`
while the intended target 0011 stayed `Accepted`. The command exits 0 and prints
only the new file path, so the wrong-ADR status mutation is silent; it was
caught by reading the scaffold back, and all three files were repaired by hand.

Any id `0010`–`0017`, `0020`–`0027`, … misparses to a different existing ADR
(silent wrong flip); ids containing `8`/`9` after the zeros (`0018`, `0089`)
fail parse and exit 2, which at least fails loudly.

## Fix

Accept the id as a string and parse it base-10 (strip padding or
`strconv.Atoi` after a digits-only check), or keep `flag.Int` but reject any
zero-padded value with an error naming the rule. Also print which ADR was
status-flipped on success so a wrong target is visible immediately. Tests:
`--supersedes 0011` flips ADR 0011 and records `supersedes: "0011"`;
a padded id never touches any other ADR (negative control on 0009).

## Related

- I119 — same command-surface family (argument handling hardening across
  subcommands); this is a value-parsing sibling of that ordering work.
