---
id: I096
title: "`spine update` can write a maipipe.toml the daemon cannot load — parse (and `maipipe validate`) the spliced result before writing"
severity: high
status: fixed
affects: [I085, I091]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-19 from `docs/research/2026-08-19-gate-pack-region-ownership-analysis.md`.

spine renders the gate-pack region as a string
(`internal/update/gatepack.go:106-134`) and splices it between marker lines
(`:194-195`), then writes the whole file atomically (`update.go:182`). It never
parses the result. Two reproduced consequences, both leaving **every** lane in
the repo unloadable:

1. **Moved stage → duplicate stage name.** Move a
   `[[pipelines.gate-go.stage]]` block three lines past `# spine:end` (a
   plausible hand edit, and a plausible merge resolution). `spine update`
   reports nothing about maipipe.toml; `spine update --write` — **no `--force`,
   no warning** — re-renders the stage back inside the region, and the file now
   declares `tskip` twice. `maipipe validate` → `pipeline "gate-go" stage
   "tskip": duplicate stage name`, exit 1.
2. **Pre-existing `[pipelines.gate-go]` outside the region → duplicate key.**
   spine appends the region blind, TOML parse fails, exit 1.

This is the I091 class recurring: spine's positive controls assert spine's own
string shape rather than maipipe's grammar, so spine's tests structurally cannot
catch it.

## Fix

Before `WriteFileAtomic` in the maipipe.toml path:

1. Parse the spliced result as TOML. On failure, refuse the write and report the
   parse error with the file path — never write.
2. When a `maipipe` binary is resolvable, additionally run `maipipe validate
   <path>` against the candidate content (a temp file is fine) and refuse on a
   non-zero exit, quoting maipipe's message verbatim. Confirmed feasible:
   `maipipe validate` accepts a path argument, returns OK on spine's real file
   and exit 1 on both variants above. When the binary is absent, the TOML parse
   still applies and the report says validation was skipped.
3. The refusal must name the likely cause when it is knowable — a duplicate
   stage name inside the region after a splice almost always means a copy of
   that stage now sits outside the markers.

This is a refusal, not a repair: spine must not move the user's stage back.

## Acceptance criteria

- [x] Positive control: spine's own `maipipe.toml` renders and validates clean;
      `spine update --write` is unchanged in the normal path
- [x] Negative control 1: fixture with a `gate-go` stage moved past
      `# spine:end` → `spine update --write` refuses, file unchanged on disk,
      message names the duplicate stage
- [x] Negative control 2: fixture with a pre-existing `[pipelines.gate-go]`
      table outside the region → refuses, file unchanged
- [x] Removing the check from either control reproduces the unloadable file
      (proves the guard is load-bearing)
- [x] With no `maipipe` on PATH, the TOML-parse refusal still fires and the
      report states that `maipipe validate` was skipped

## Resolution (2026-08-20)

`internal/update/maipipecheck.go`: `checkMaipipeContent(path, content)` runs
before any file is written by `update.Run` — a refusal leaves the whole tree
untouched, not just maipipe.toml. It parses the candidate as TOML
(`checkStructure`, hand-rolled: ADR 0001 keeps spine on zero third-party
dependencies) and, when `exec.LookPath("maipipe")` resolves, runs `maipipe
validate` against a temp copy, quoting maipipe's message verbatim. A
`duplicate stage name` in that message carries the hint that a copy of the
stage now sits outside the markers. With no binary on PATH the refusal says
`maipipe validate skipped: no maipipe binary on PATH`. Refusal, not repair:
nothing outside the region is moved or rewritten.

**Escalation beyond the ticket's wording (fix round 1, controller ruling).**
The ticket says refuse "before `WriteFileAtomic` in the maipipe.toml path";
the refusal is in fact all-or-nothing — no file is written, WORKFLOW.md
included. `spine update` presents one plan and applies it as a whole, and a
partial application would leave a rendered region stale against a WORKFLOW.md
that already moved. The refusal message says so explicitly ("no files were
written…"), because an error naming only maipipe.toml would read as if
maipipe.toml were the only file skipped. `--force` does not bypass it.

**AC5 wording (fix round 1, ruled).** The brief says "the report says
validation was skipped" while AC5 says the *refusal* fires and the report
states it. AC5 is normative and is what is implemented: the note rides in
the refusal string. A standing "validate skipped" line on the successful
path is a follow-up nicety, not a gap here — noted so the next reader does
not re-derive the question.

**Behaviour change for existing repos.** "The normal path is unchanged" is
not literally true for every repo: `maipipe validate` rejects a maipipe.toml
with no top-level `schema` key, so a repo carrying that pre-existing defect
can no longer run `spine update --write` until it adds the key. The file was
already unloadable by maipipe before spine touched it, and the refusal names
the exact problem. Two existing user-lane fixtures in `gatepack_test.go`
were carrying that defect and now carry `schema = 0` the way a real repo
does.

**Fix round 2.** Two regressions the round-1 diff introduced: the timeout
branch was unreachable (`CommandContext` kills the child, so `Wait` returns
an `*exec.ExitError` and a timeout read as maipipe's verdict on the file —
the deadline is now checked first), and quoted and bare keys were keyed
distinctly, so `[pipelines."a"]` + `[pipelines.a]` — one table in TOML, and
a duplicate-key error from maipipe — was accepted. Key segments are now
unquoted (basic-string escapes decoded, literal strings verbatim) before
being used as identity and re-quoted only for messages; canonical segments
join on a byte no key can contain, so `"a.b"` (one segment) stays distinct
from `a.b` (two).

**Parser (fix round 1, Important 1).** `checkStructure`'s scanner replaces each
consumed string with a placeholder instead of dropping it: dropping made
`[pipelines."e2e.smoke"]` and `"my key" = 1` — both legal TOML — look
malformed, which would have hard-blocked writes in any repo using a quoted
key. A standard table under an array-of-tables entry is qualified by that
entry, so `[[a]] [a.b] [[a]] [a.b]` is not read as a duplicate.

**Gap in this ticket, closed in the final-review fix wave (2026-08-20).**
The ticket says refuse "before `WriteFileAtomic`", and the implementation
did exactly that — so the check ran only under `--write`. A reader could see
a clean plan with a diff and exit 1, run `--write`, and be refused: the plan
diff is the review surface (ADR 0017), yet the one thing that would stop the
write was invisible on it. Recorded here as a gap in the ticket's wording
rather than a slip in the code. The pre-flight now runs in the plan pass on
every run; the verdict rides on `FileReport.Refusal` and the plan prints it
before that file's diff, and `--write` returns the same verdict. Also closing
the AC5 follow-up noted above: on a *passing* run with no maipipe on PATH the
plan now says the pre-flight was structural only, so silence no longer reads
as a full check (`FileReport.StructuralOnly`).

**Naming (final review, Important 3).** `parseTOML` is renamed
`checkStructure`, and its doc comment, `TestCheckStructure`'s comment and the
no-binary note now describe what it is — a duplicate-table/duplicate-key and
bracket-balance scan, not a TOML parser. Text it accepts that TOML rejects
(`a b = 1`, `a = = 1`, `"\101"`) is recorded, with the rest of the residual
list and the question of whether the scanner should exist at all, in **I104**.
The scanner itself was deliberately not changed again.

## Evidence

- `go vet ./...` — clean.
- `make test` (`go test ./...`) — all packages ok, including
  `internal/update` with the five new tests.
- Negative control 1 (`TestMovedStageRefusesWrite_requiresMaipipeOnPATH`):
  refusal names `duplicate stage name`, `tskip` and `# spine:end`; file
  byte-identical on disk after the refused run.
- Negative control 2 (`TestPreExistingGatePipelineRefusesWrite`): refusal
  names `duplicate table [pipelines.gate-go]`; file unchanged. Fires with no
  maipipe binary needed.
- Load-bearing (`TestCheckIsLoadBearingFor…`): the candidate written without
  the check gives `line 12: duplicate table [pipelines.gate-go]` (fixture 2,
  and `maipipe validate` → `TOML parse error … duplicate key`) and
  `pipeline "gate-go" stage "tskip": duplicate stage name` (fixture 1).
- No-binary path (`TestNoMaipipeOnPATHStillRefusesAndSaysValidateSkipped`):
  `PATH` emptied, parse refusal still fires and states validate was skipped.
- Positive control (`TestPositiveControlRealRepoFileAndNormalWrite`): spine's
  own repo-root maipipe.toml passes parse + `maipipe validate`; the normal
  `Run(Write: true)` path still renders and writes the region.
- No `t.Skip` anywhere: the maipipe-dependent tests carry the condition in
  their name and log which half of the check ran; `SPINE_REQUIRE_MAIPIPE=1`
  turns a missing binary into a failure so CI can assert they really ran.
- Round-2 negative controls: reverting the deadline ordering makes
  `TestValidateTimeoutIsNotAVerdict` report `rejected the result: signal:
  killed`; reverting the unquoting makes `TestCheckStructure` accept all seven
  quoted/bare duplicate pairs.
- Quoted-key negative control (fix round 1): reverting the scanner to drop
  strings fails `TestCheckStructure` (`empty key`, `duplicate table
  [pipelines.]`) and `TestScanKeepsQuotedSegments`.

## Notes

Sequenced before I097: region removal must pass this gate before it is written.
`maipipe` reserves the pipeline names `gate-go` and `mutation-go`
(maipipe I206 documents this); nothing enforces it on either side today, and
this check is what makes a collision visible to spine.

## Follow-up (2026-08-20, I104)

ADR 0018 removed this ticket's spine-side structural half and its scanner
tests. For a configured gate pack, `maipipe` on `PATH` is now the precondition
for touching `maipipe.toml`; a missing binary makes the plan skip that one file
while allowing unrelated pending files to apply. `maipipe validate` remains
the sole candidate pre-write authority when present.
