---
id: I120
title: "adr new: --supersedes parses zero-padded ids as octal and flips the wrong ADR"
severity: med
status: fixed
commits: [865ddc2]
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

Accept the id as a string and parse it base-10 (`strconv.Atoi` after a
digits-only check). Also print which ADR was status-flipped on success so a
wrong target is visible immediately. Tests: `--supersedes 0011` flips ADR
0011 and records `supersedes: "0011"`; a padded id never touches any other
ADR (negative control on 0009).

Rejected alternative (verify-stage requirements-attack, 2026-08-28: this
originally stood as an "or" arm, contradicting the test prescription above —
which requires `0011` to *work*): keeping `flag.Int` and rejecting any
zero-padded value. Zero-padded ids are the documented convention
(docs/adr/README.md), so rejecting them would break every idiomatic
invocation.

## Resolution

Fixed 2026-08-28. `--supersedes` is now a string flag parsed by `parseADRID`
in cmd/spine/main.go: digits-only check then base-10 `strconv.Atoi`, so
`0011` means 11 (and `0018`-style ids now work instead of failing parse).
Invalid values exit 2 naming the rule: non-digits (`0x11` previously parsed
as hex 17), `0`/`0000` (previously a silent no-op — the flag.Int default),
an explicitly empty value (detected via `fs.Visit`, previously
indistinguishable from unset and silently ignored — same failure class as
the filed bug), and out-of-range values (without leaking `strconv` text).
A successful supersede prints `superseded: NNNN` as a second stdout line.
`adr.New`'s signature is unchanged. Tests in cmd/spine/adr_supersedes_test.go
pin all of it, including the negative control on 0009; the pre-fix run
reproduced the filed symptom exactly (0009 flipped, `supersedes target 0017
not found` for `0x11`). The stale flags-after-title example in
docs/adr/README.md (broken since I119) was corrected in the same change.
Verified by a fresh-context subagent (requirements-attack first, independent
binary repro, negative-control revert): pass.

## Related

- I119 — same command-surface family (argument handling hardening across
  subcommands); this is a value-parsing sibling of that ordering work.
