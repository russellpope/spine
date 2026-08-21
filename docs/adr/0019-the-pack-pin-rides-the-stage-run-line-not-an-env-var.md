---
id: "0019"
title: "the pack pin rides the stage run line not an env var"
status: Accepted
date: 2026-08-21
---

# 0019: the pack pin rides the stage run line not an env var

## Context

ADR 0015 makes `<pack>@<v>` both an attribution identifier and, per I098, a
frozen class list. I098 enforced the second half: a repo pinned at `go@1`
renders go@1's classes from any spine binary. The first half was still the
binary's: every rendered stage ran `spine gate go <check>`, and the finding
code came from the binary's own pack version. Once a second pack ships, a
go@1 repo would emit `go@2/<check>` from go@1 stages (I103).

The pin therefore has to reach the stage, which is a separate process. Two
carriers were considered: an environment variable (`SPINE_GATE_PACK`,
matching the existing `SPINE_GATE_*` config convention the ticket suggested)
or the run line itself (`spine gate go@1 <check>`).

## Decision

The **pack pin** rides the stage's run line. The rendered region's stages run
`spine gate <pack>@<v> <check>`; `spine gate` treats a versioned pack
argument as the authoritative pin for that run: findings are coded
`<pack>@<v>/<check>`, a check outside that pack's frozen class list is
refused, and a pack the binary does not ship is refused — both as
misconfiguration (exit 2), never approximated. A bare pack argument
(`spine gate go <check>`, the hand-run form) keeps attributing as the
binary's own pack; `spine gate` never reads `WORKFLOW.md`.

No env var carries the pin. The `SPINE_GATE_*` namespace stays what it is:
per-check configuration.

## Consequences

- The plan diff is the whole story: a pin change shows as the run line that
  changed, with no second place to look. The region needs no maipipe feature
  beyond what it already uses.
- The one-time cost is fleet-wide: every adopting repo's region is rewritten
  (new `definition_hash`, `maipipe gate approve-definition` again) with no
  stage added or removed. The plan names this as changed stages so the cost
  is never silent (I103 extends I098's notice).
- The region reader recognises both the bare and the pinned run-line forms
  as spine's own content, so an un-migrated region is stale, not
  unrecognized. A pinned line naming a pin other than the repo's is not a
  projection of `WORKFLOW.md` (ADR 0017) and is unrecognized region content.
- Changing the carrier later means a second fleet-wide rewrite; that is why
  this is recorded.
