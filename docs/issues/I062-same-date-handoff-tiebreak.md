---
id: I062
title: "Handoff ordering: same-date newest resolution is lexicographic on filename"
severity: low
status: open
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

## What to build

A same-date tiebreak that tracks creation recency instead of filename, so the
most recently created handoff wins "newest" regardless of topic name. Candidate
orderings (implementer picks one, records the choice in the ticket resolution):
file modification time as the secondary key, or an explicit monotonic ordinal
in handoff frontmatter written by `spine handoff new`. Whatever is chosen must
be deterministic for the audit gate (the complete-snapshot check reads
`Latest`), keep cross-machine behavior sane for committed files (git does not
preserve mtimes on fresh clones — a frontmatter ordinal survives cloning, an
mtime tiebreak must at minimum degrade to the current lexicographic order,
never to nondeterminism), and leave different-date ordering untouched.

## Acceptance criteria

- [ ] Two same-date handoffs where the lexicographically-earlier name is the
      newer creation: `spine handoff latest` and the `audit stages`
      newest-handoff check both pick the newer one
- [ ] Different-date ordering unchanged; existing handoff fixtures pass
      untouched
- [ ] Fresh-clone determinism covered by test or by documented degradation
      (per the chosen mechanism)
- [ ] `go test ./...` green

## Blocked by

- None — can start immediately.
