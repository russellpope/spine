---
id: I116
title: "spine model: flags after positionals print bare usage and exit 2, reading as a broken flavor"
severity: low
status: fixed
batch: 2026-08-28-ergo#1
commits: [947a87a, b81292d]
affects: [I034]
blocked-by: []
execution-mode: inline
tier: primary
effort:
risk-triggers: []
review-tier: n/a
---

## Problem

`spine model` (stdlib flag parsing) stops at the first positional, so a flag
placed after the positionals is not a mis-set option but a hard usage error:

- `spine model --effort openweights primary` — works.
- `spine model openweights primary --effort` — prints usage and exits 2.

The failure carries no hint that ordering is the problem, so it reads as the
flavor itself being broken — exactly how it presented during the openweights
rollout (2026-08-25 handoff, gotchas). The trap has had to be re-learned from
handoff prose across sessions instead of being fixed or named at the point of
failure.

## Fix

Either accept flags in any position for this subcommand (permute before
parsing), or keep strict ordering and make the usage error name the rule
("flags must precede positionals") next to the offending argument. Decide at
the grill; the second is smaller and preserves stdlib semantics. Tests: the
trailing-flag invocation either resolves identically to the leading-flag form
(option A) or exits 2 with a message naming the ordering rule (option B); the
leading-flag form stays green either way.

## Related

- **I034** — the `spine model` command this ergonomics trap lives in.
- docs/handoffs/2026-08-25-openweights-docs-and-the-i112-axis-question.md —
  where the trap was recorded as a gotcha instead of a ticket.

## Resolution

Option B from the grill (2026-08-28 ergonomics batch): strict stdlib
ordering kept; a flag-like token among the positionals (beyond the first,
which is only reachable via an explicit `--`) now errors
`model: flags must precede positionals (saw "--effort" after "primary")`
plus the usage line, exit 2 — covering both the trailing-flag and
correct-arity (`claude --json`) shapes. Detection keys on the leading
dash, not resolution failure (negative control observed red). Helper
`flagAmongPositionals` lives in cmd/spine/main.go, wired into `cmdModel`
only per the grill's scope ruling; the same trap observed live on
`spine cursor show --dir X` (trailing `--dir` silently ignored) is the
named candidate for a generalization follow-up. Commits `947a87a`
(implementation, TDD both arms) and `b81292d` (PRD amendments C1/C3 at
spec-review). Work done.
