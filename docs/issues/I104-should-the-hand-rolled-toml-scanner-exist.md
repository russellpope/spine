---
id: I104
title: "Should spine's hand-rolled TOML scanner exist at all, or should `maipipe` on PATH be a precondition for touching maipipe.toml? (owner call)"
severity: med
status: open
affects: [I091, I096, I098]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Filed 2026-08-20 from the final whole-branch review of I095/I096/I098
(Important 3, and the architectural question none of those tickets asked
anyone to answer). **This is an owner call, not a defect report.**

I096 added `checkMaipipeContent`: before `spine update` writes a spliced
`maipipe.toml`, spine checks the candidate. Two halves — a hand-rolled
structural scan (`checkStructure` and its scanner, `internal/update/maipipecheck.go`,
~400 lines) that always runs, and `maipipe validate` when the binary is on
PATH. The scan has now taken three rounds of fixes (quoted keys dropped with
their strings, quoted-vs-bare key identity, array-of-tables qualification) and
the final review probed a further residual list (below).

The question never weighed: **should the scan exist?**

Under ADR 0001 (zero third-party dependencies), a real TOML library was off
the table without superseding an ADR, so the choice looked binary — hand-roll
it or have nothing. But there is a third option: treat `maipipe` on PATH as a
**precondition** for touching `maipipe.toml` when `gate_pack` is set. If it is
absent, refuse or warn-and-skip that one file and apply the rest of the plan.
The authority then does the whole job, and the ~400 lines and the entire
over-/under-refusal surface go away.

Arguments the review recorded, faithfully:

**For dropping the scan** — the gate pack *is* maipipe pipelines, so a machine
rendering them without maipipe installed is arguably misconfigured; every
residual defect below is in the scan, none in the maipipe path; a
duplicate-and-balance scan is not a TOML parser and cannot be made into one
cheaply (Important 3 forced the doc comments to stop calling it one).

**For keeping it** — adopters who run `spine update` on machines without
maipipe (a CI image refreshing only WORKFLOW.md would now fail or skip a file
it used to write), and the possibility of a second TOML consumer inside spine.
If a second consumer arrives, the review's own answer is to supersede ADR 0001
for a vetted parser rather than grow this one.

## Residual defect list (evidence; carried, deliberately unfixed)

Each was probed against spine's verdict and maipipe's. All are caught when
maipipe is on PATH; all are the no-binary path only. None is a shape spine's
own splice can produce, which is why the review did not treat them as Critical.

Under-refusals (spine accepts, TOML/maipipe rejects):

- key-then-table: `[a]` `b = 1` then `[a.b]`
- dotted-key-then-table: `a.b = 1` then `[a.b]`
- inline-table-then-subtable
- bare key containing a space: `a b = 1`
- Go escape TOML does not have: `"\101"` (via `strconv.Unquote`)
- `a = = 1`
- `"a"x` as a key

Over-refusals (spine refuses, TOML accepts):

- `"" = 1` (the empty quoted key is legal TOML)
- nested arrays-of-tables (final review, Minor 4)

## Decision needed

One of:

- **(A) Keep the scan** as an explicitly-named best-effort structural check
  (its current, now honestly-documented state), and either fix or knowingly
  accept the list above.
- **(B) Drop the scan**; require `maipipe` on PATH to plan or write
  `maipipe.toml` when `gate_pack` is set; refuse or warn-and-skip that file
  otherwise. Record as an ADR.
- **(C) Supersede ADR 0001** for a vetted TOML parser. The review considers
  this warranted only if a second TOML consumer appears in spine.

## Acceptance criteria

- [ ] Owner picks A, B or C, recorded as an ADR (B and C change a standing
      decision; A can be a note on I096 plus this ticket's Resolution)
- [ ] If B: the no-maipipe path is a refusal or a documented skip with the
      plan saying so, and the scan and its tests are deleted
- [ ] If A: each residual above is either fixed with a test or listed as
      knowingly accepted, in one place
- [ ] `spine update`'s plan keeps saying which half of the pre-flight ran
      (added in the final-review fix wave), whichever way this goes

## Notes

The current code and its comments no longer overstate the scan: it is
documented as a duplicate-and-balance check, and the no-binary note says the
check was structural only. That was the fix wave's whole remit here — the
scanner itself was deliberately not touched a fourth time.
