---
id: I099
title: "Gate-pack contract drift: D10 reports outside its specified scope, and `mutation-go` is rendered but composed nowhere"
severity: low
status: fixed
affects: [I085, I086, I089]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-19 from `docs/research/2026-08-19-gate-pack-region-ownership-analysis.md`.
Two written-contract gaps in the gate-pack area, neither blocking.

**1. D10's implemented scope exceeds its spec.** Spec story 18 and the design's
Doctor line (`docs/specs/2026-08-18-local-harness-conventions-design.md:124-125`,
`:259-260`) scope D10 to "region integrity only (markers, canonical content) …
nothing else per repo". The implementation (`internal/doctor/doctor.go:178-195`)
also emits:

- a **staleness warn** whenever the plan is Pending for any reason — including a
  legitimate WORKFLOW.md config change the owner has not applied yet
  (`:192-194`); and
- an **error** whose source is WORKFLOW.md, not the region: `gate_pack: <x> is
  not a pack this spine binary ships` (`:185-187`, via `planMaipipe:146-153`) —
  a WORKFLOW.md finding wearing a maipipe.toml path.

This matters beyond tidiness: the staleness warn is the *only* signal a
hand-edited region produces today, and a reader of the spec would conclude no
such signal exists.

**2. `mutation-go` is defined and composed nowhere.** ADR 0015 item 5 and ADR
0016 both require the repo to compose `mutation-go` into an audit lane. spine's
own `maipipe.toml:34-39` renders the pipeline; the file has no audit lane, so
the advisory battery is reachable only via `maipipe run mutation-go`. The
scaffolding is maipipe-side (`maipipe docs/specs/2026-08-19-execution-floor-phase-1-design.md:81`,
ticket I204 — open, blocked by I202+I203). So the requirement is satisfied by
neither side today, in the one repo that has adopted the pack, and neither ADR
records the gap.

## Fix

1. Amend the spec to name the three D10 cases actually implemented, **or** move
   the unknown-pack error to the WORKFLOW.md-scoped check where it belongs
   (D2/D4). Prefer the move: the finding's remedy is a WORKFLOW.md edit. Keep
   the staleness warn but make its message distinguish "WORKFLOW.md changed,
   region not yet refreshed" from "region content is not what this pack renders"
   — the two are indistinguishable today (see I095).
2. Amend ADR 0016's Consequences to record that the audit-lane composition is
   maipipe-side (I204) and that until it lands `mutation-go` is reachable only
   through `maipipe run mutation-go`. Then either add the audit lane to spine's
   own `maipipe.toml` by hand now, or accept the gap in writing.

## Acceptance criteria

- [x] D10's emitted findings match a written scope, whichever way it is settled
- [x] The staleness message distinguishes the two causes, or I095 records why it
      cannot yet
- [x] ADR 0016 Consequences names the I204 dependency
- [x] spine's own `maipipe.toml` either composes `mutation-go` in an audit lane
      or the omission is recorded with its reason

## Evidence

**Fix item 1 — settled 2026-08-20 in `243fda7`.**

The unknown-pack error moved to **D4**, as an `error` on `WORKFLOW.md`. D2 is
staleness (`template_version` behind the binary), which this is not; D4 is
unrecognized content in a machine-owned file, which is exactly the state
`planMaipipe` already assigns it (`SkippedUnrecognized`). It reached D10 only
because update files the report against `maipipe.toml` — the file it declined to
render — so the path now points at the file the owner must edit.

**Why the staleness message does not distinguish the two causes.** Per the
controller ruling, the I095 owner question is settled as reading **(A), pure
projection**: the gate-pack region is a rendering of WORKFLOW.md, no value inside
it is ever a user choice, edits inside it are discarded on refresh, and there is
deliberately **no record of what spine last rendered** — no fingerprint, no
sidecar. Distinguishing "WORKFLOW.md changed, region not yet refreshed" from
"region content is not what this pack renders" requires comparing the region
against what spine last emitted, which is precisely the record (A) declines to
keep. The distinction is therefore unavailable by design, not unimplemented, and
no fingerprint was built to make it possible. The warn is kept; its message was
reworded from "gate-pack region is stale for the pinned pack" to "gate-pack
region differs from what the pinned pack renders" so it names the remedy rather
than asserting one of the two causes. The reason is recorded in the code at the
branch itself (`internal/doctor/doctor.go`, `gatePackCheck`) and in the spec's
Doctor line.

**Written scope.** `docs/specs/2026-08-18-local-harness-conventions-design.md`
story 18 and the Doctor line now enumerate the four findings D10 emits: damaged
markers (error), non-canonical region lines (warn), `gate_pack` set with no
region (warn), and a region differing from the rendering (warn).

*Filing note:* the Problem section above says the staleness warn is "the only
signal a hand-edited region produces today". That overstates, and the Problem
section is left as filed. A hand-edit that introduces lines the pack does not
recognize produces the non-canonical warn; only an edit made entirely out of
recognized pack lines falls through to the staleness warn, and that is the
indistinguishable case reading (A) declines to resolve.

*Requirements note:* the Fix item above says "the three D10 cases actually
implemented". Counted from the switch arms, D10 emits four findings (the
unrecognized arm splits into a marker error and a non-canonical warn). The spec
now names four; the "three" in the Fix text is a miscount, not a scope change.

Tests: one fixture repo per emitted D10 case in `internal/doctor/doctor_test.go`,
plus `TestUnknownGatePackIsD4OnWorkflowNotD10` as the negative control. Full
`make test` and `go vet ./...` green at `243fda7`.

**Fix item 2 — settled 2026-08-20.**

ADR 0016's Consequences gained a bullet recording that the composing edit the
Decision asks the repo for is scaffolded on neither side: the full lane is
composed by hand, the audit lane is not, and the scaffolding that would write
it is maipipe's ticket I204 (open, blocked by I202+I203). Until I204 lands,
`mutation-go` is reachable only as `maipipe run mutation-go`. The bullet is an
inline dated amendment in the same shape as the I091 one already in that
section, and records a gap rather than changing the Decision — the ADR
convention's immutability rule (`docs/adr/README.md`) reserves new superseding
ADRs for reversing or amending a *decision*, which this is not.

**The omission in spine's own `maipipe.toml` is recorded, not filled.** A
comment in the owner-managed part of the file (below the `full` lane, outside
the spine-managed region) names the absence, its reason, and the workaround.
Writing the lane by hand was considered and declined: I204 is open and will
scaffold audit lanes, so a hand-written one here is work I204 would then have
to reconcile — and the gap this ticket exists to document would be papered
over in the one repo where it is observable.

**The durable record of the gap is this ticket and `maipipe.toml`**, not the
ADR bullet: ADR 0016 is expected to be superseded (I095, region projection),
and a reader following a status pointer to its successor would otherwise lose
the record. `maipipe.toml`'s note therefore names the gap, its owner (maipipe
I204) and the workaround without requiring a hop to an ADR whose number is in
flight, and the ADR bullet is worded against the pack's composition
requirement rather than against 0016's decision, so it still reads correctly
once the document is superseded.

Tests: `TestD10RegionMissing` gained the blast-radius assertion its siblings
carry (no D4 from the removed file, no other finding).

```
$ go vet ./...
$ make test
go test ./...
ok  	github.com/russellpope/spine/cmd/spine	(cached)
...
ok  	github.com/russellpope/spine/internal/doctor	0.502s
ok  	github.com/russellpope/spine/internal/update	(cached)
ok  	github.com/russellpope/spine/templates	(cached)
```

Negative control: flipping the new `len(fs) != 1` to `!= 0` fails the test
with the single expected D10 in the dump, so the assertion is load-bearing.

## Notes

Item 2 bears on I097: opt-out's out-of-region reference scan must consider the
`mutation-go` composition maipipe is about to start writing.
