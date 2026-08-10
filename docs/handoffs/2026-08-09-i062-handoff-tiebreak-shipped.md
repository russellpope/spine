---
title: "i062 handoff tiebreak shipped"
created: 2026-08-09
handoff_ordinal: 1
---

# Handoff — i062 handoff tiebreak shipped (2026-08-09)

## Context

I062 is shipped locally. Same-date handoffs now sort by a persisted positive
`handoff_ordinal` after filename date and before the deterministic filename
fallback. `spine handoff new` allocates the ordinal with exclusive on-disk
reservations plus a post-reservation committed-state recheck, so simultaneous
CLI processes cannot silently share a value.

## State (verify before relying)

- Implementation and the I062 Resolution are committed at `a56f274` on
  `main`; no push was performed.
- Primary blind review initially found a concurrent-allocation race and missing
  edge regressions. The correction passed primary re-review with no findings.
- Fresh-context primary verification passed with no product/spec findings.
  Focused and full Go tests, race checks, `go vet`, clean build, diff/format
  checks, CLI simulations, clone determinism, and routing audit were green.
- This handoff self-hosts the fix: its lexicographically earlier topic carries
  ordinal 1 and is selected over the prior same-date handoff by both text and
  JSON `handoff latest`.
- The only unrelated working-tree entries at ship time were the owner's
  `.DS_Store` and `docs/research/2026-08-05-routing-yield-feasibility.md`; both
  were left untouched.

## Next steps

- Owner may push `main` when desired.
- No further I062 implementation work is required. If installing the new CLI is
  desired, do so as an explicit owner/deployment action; this build did not
  replace `/Users/ldh/bin/spine`.

## Gotchas

- A crash after ordinal reservation can leave a hidden marker under
  `docs/handoffs/.spine-handoff-ordinal-reservations/`. Future creation skips
  it deliberately, producing a safe sequence gap rather than value reuse.
- Independently created branches cannot share live reservations. Equal
  ordinals after merge retain the documented deterministic filename fallback.
- The current-filesystem guarantee depends on ordinary `O_EXCL` semantics;
  verification covered the repository's local macOS filesystem.

<!-- spine:cursor -->
effort: i062-handoff-tiebreak
prd: docs/specs/2026-08-06-cursor-writes-design.md
tickets: I062
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
