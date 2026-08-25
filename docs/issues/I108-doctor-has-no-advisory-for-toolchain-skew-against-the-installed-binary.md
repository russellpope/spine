---
id: I108
title: "spine doctor has no advisory for toolchain skew between the installed binary and the Go on PATH"
severity: low
status: open
affects: [I107]
blocked-by: [I107]
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

I107 makes the two `go/types`-loading gate classes fail *legibly* when the
installed spine binary predates the Go toolchain that populated the build cache:
the operator gets a message naming both toolchains and the remedy instead of a
`gcimporter` stack trace. But they only get it **when a gate stage fires**, part
way through a lane, on a commit that has nothing to do with the problem.

`spine doctor` is where a repo owner goes to ask "is this environment sane
before I run anything". It has no check for this condition. Adding one would
move the notice from mid-lane to on-demand.

## Why this is not part of I107

Detection in the gate keys on the real condition — the importer actually failed
to decode — which cannot false-positive. A doctor check cannot do that: nothing
has been loaded yet, so the only available signal is comparing the binary's
build toolchain (`runtime.Version()`) against the `go` on PATH (`go env
GOVERSION`). That is a **proxy**. The panic is keyed to the export-data format
version, which does not change on every Go release, so a 1.26.7-built binary
reading a 1.26.8-populated cache is fine and a version-string comparison would
flag it anyway.

ADR 0021 rejects that proxy for the gate, where a false positive fails a lane.
It is *arguably* tolerable in doctor, where an advisory cannot fail anything —
but that is a real design question with a real chance the answer is "don't ship
it", and it does not belong bolted onto I107's diff.

## Open questions

1. Is a check that warns on benign patch-level skew worth having at all, or does
   crying wolf make `spine doctor` less trusted than having no check?
2. If yes: severity `warn` (an operator can ignore it) or `advise`? Doctor
   already exits 1 on two long-standing D4 notes, so adding routine noise has a
   cost.
3. Can the comparison be narrowed to something less false-positive-prone than
   string inequality — major/minor only, ignoring the patch component? That
   would flag 1.26.x → 1.27.0 and stay quiet on 1.26.7 → 1.26.8, which is much
   closer to the real condition without loading anything.
4. Doctor findings carry a `path`. This condition has no file. What goes there?
5. Precedent check: ADR 0018 already put a machine-state precondition
   (maipipe-on-PATH) inside doctor's remit, so a non-repo check is not novel.

## Notes

- Next free doctor check id is **D11** (`internal/doctor/doctor.go` uses D1–D10).
- `runtime.Version()` appears nowhere in the tree today; I107 introduces its
  first use.
- Blocked by I107 only in the weak sense that I107 settles the vocabulary and
  ships the first toolchain probe; nothing here needs I107's code.

## Related

- I107 — the gate-side fix, and the source of this split.
- ADR 0021 — rejects the version-comparison proxy for the gate and records why
  the same proxy might survive as an advisory.
- ADR 0018 — precedent for a machine-state precondition inside doctor.
