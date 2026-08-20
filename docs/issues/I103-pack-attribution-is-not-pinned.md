---
id: I103
title: "`gate_pack: go@1` freezes the class list but not the attribution string: a pinned repo's findings will be coded `go@2/<check>` once go@2 ships"
severity: med
status: open
affects: [I098, I082]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Filed 2026-08-20 from the final whole-branch review of I095/I096/I098
(Important 2).

I098 made `gate_pack: go@N` a frozen class list: a repo pinned at go@1 renders
the go@1 classes and only those, from any spine binary
(`gate.PackClassesFor`, `internal/update/gatepack.go`). That fixed the class
set. It did not fix the other half of what the pin names.

`spine gate go <check>` attributes every finding through `gate.Code(check)` →
`PackID()/<check>` (`internal/gate/gate.go`), and `PackID()` is built from the
**binary's** `PackVersion` const, not from the repo's pin. The stage that runs
was rendered from the repo's frozen list; the code string it emits comes from
whatever pack version the binary happens to be.

So once go@2 ships, a repo pinned at go@1:

- renders go@1's stages (correct, I098), and
- emits findings coded `go@2/tskip` from them (wrong).

Attribution strings are what the remediation round record, the do-not-regress
blocks and telemetry key on (ADR 0015 Consequences), so a silently shifting
`code` field mislabels history rather than just looking untidy. ADR 0015 item 2
makes the version an attribution identifier *and* — per I098's controller
ruling — a frozen class list; today only the second is enforced.

**Not reachable yet.** One pack version ships, so `PackID()` and every repo's
pin are the same string. This is filed while it is free to fix, the same
reasoning I098 was filed under.

## Fix

Direction suggested by the review — the pin has to reach the stage, since the
stage is a separate process:

1. Render the pinned pack into each stage's environment (e.g.
   `SPINE_GATE_PACK = "go@1"` in the region, alongside the existing
   `SPINE_GATE_*` config vars).
2. Have `gate.Code`/`PackID` honour that value when set, and fall back to the
   binary's own `PackVersion` when it is absent (a hand-run
   `spine gate go tskip` outside a rendered stage).
3. Refuse, don't guess, when the value names a pack this binary does not ship
   — the same refusal `planMaipipe` already makes.

Note the byte cost: adding an env var to every stage rewrites the region, so
every adopting repo re-runs `maipipe gate approve-definition`. The plan's
added-stage notice (I098) does not currently cover "same stages, changed env".

## Acceptance criteria

- [ ] A stage rendered from a `go@1` pin emits findings coded `go@1/<check>`
      on a binary whose `PackVersion` is 2, with a test that ships a stub
      later pack (the `packClassesFor` seam in `internal/update` and a
      corresponding seam in `internal/gate`)
- [ ] A pack identifier the binary does not ship is refused, not approximated
- [ ] `spine gate go <check>` run by hand, with no pin in the environment,
      still attributes as the binary's own pack
- [ ] Spec story 23 and I098's Resolution updated to say the pin covers both
      the class list and the attribution string

## Notes

Depends on nothing; I098 is the reason this half is now visible. Do not fold
this into I098's Resolution as "done" — it is a separate render change with
its own definition_hash cost.
