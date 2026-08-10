---
id: I062
title: "Handoff ordering: same-date newest resolution is lexicographic on filename"
severity: low
status: fixed
affects: [cli, handoff, audit]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
---

## Problem

Handoff dates are day-granular, so `handoff.Latest` breaks same-date ties
lexicographically on path. Same-day successor efforts therefore fight over
"newest": observed 2026-08-06, when the cursor-writes build handoff could not
out-sort the predecessor effort's `…-mutation-battery-shipped.md` (m > c) and
had to be renamed `…-sole-writer-codex.md` to become the audit's recognized
cursor carrier; the build crew then had to pick their shipped handoff's name
to out-sort that in turn. Filename choice silently decides which snapshot the
newest-handoff backstop judges — a gate input decided by alphabet.

Second live occurrence: I031 filed the same defect after the 2026-07-16
derivation-polish rename. This ticket supersedes I031 (now wontfix, pointer
here) and absorbs its candidates.

## What to build

A same-date tiebreak that tracks creation recency instead of filename, so the
most recently created handoff wins "newest" regardless of topic name. Candidate
orderings (implementer picks one, records the choice in the ticket resolution):
file modification time as the secondary key, an explicit monotonic ordinal
in handoff frontmatter written by `spine handoff new`, or I031's candidate:
prefer the doc whose cursor block matches the live effort over pure filename
order when selecting "newest" for the backstop (with the discoverability
fallback of naming the tiebreak in the stale-effort finding). Whatever is
chosen must
be deterministic for the audit gate (the complete-snapshot check reads
`Latest`), keep cross-machine behavior sane for committed files (git does not
preserve mtimes on fresh clones — a frontmatter ordinal survives cloning, an
mtime tiebreak must at minimum degrade to the current lexicographic order,
never to nondeterminism), and leave different-date ordering untouched.

## Acceptance criteria

- [x] Two same-date handoffs where the lexicographically-earlier name is the
      newer creation: `spine handoff latest` and the `audit stages`
      newest-handoff check both pick the newer one
- [x] Different-date ordering unchanged; existing handoff fixtures pass
      untouched
- [x] Fresh-clone determinism covered by test or by documented degradation
      (per the chosen mechanism)
- [x] `go test ./...` green

## Blocked by

- None — can start immediately.

## Resolution

- Mechanism: persisted `handoff_ordinal` frontmatter. `spine handoff new`
  assigns the next repository-wide positive ordinal; `handoff.List` orders by
  date first, then this ordinal descending, then filename descending.
- Allocation: `handoff new` atomically creates an exclusive reservation file
  for the candidate ordinal under
  `docs/handoffs/.spine-handoff-ordinal-reservations/`, rechecks that the
  ordinal was not committed while its initial scan was stale, and holds the
  reservation through the existing exclusive handoff-file write. Separate CLI
  processes therefore retry rather than share an ordinal; normal failures and
  successes release the reservation, preserving the never-overwrite contract.
- Crash behavior: a crash can leave a reservation marker behind. Future
  creators include such markers when finding the maximum and permanently skip
  that ordinal; this intentionally favors a harmless sequence gap over reuse.
  Independently created branches cannot share a reservation directory, so a
  merge can still produce equal ordinals and uses the documented filename
  fallback deterministically.
- Rationale: the ordinal records creation order in committed content, so the
  same-date answer is stable across fresh clones and unaffected by checkout
  mtimes. A repository-wide counter also avoids resetting the sequence every
  day while keeping the filename date as the primary ordering key.
- Compatibility: handoffs without a valid positive ordinal, including all
  historical handoffs, receive ordinal zero and retain the previous
  deterministic filename tiebreak. Duplicate ordinals (for example from
  independently created branches) also fall back to filename deterministically.
- Acceptance coverage: command-level text/JSON latest plus `audit stages`
  selection and blocking inverse, doctor D9 warn agreement, same-date creation
  order, numeric/malformed/equal ordinal fallback, separate-process concurrent
  uniqueness, different-date precedence, legacy fallback, and fresh-clone
  mtime independence are covered by the Go tests.
