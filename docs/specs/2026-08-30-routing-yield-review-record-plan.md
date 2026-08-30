# Routing yield review record (I076) implementation plan

> **For agentic workers:** execute one checkbox step at a time with TDD.
> Every task needs its own review before the next task begins.

**Goal:** add an explicit task-review record parser and the read-only `spine
yield` report without deriving outcomes from ambient artifacts.

**Architecture:** `internal/yield` parses and aggregates only selected
progress-ledger records. `cmd/spine` owns flag parsing and rendering. The
existing audit ledger reader remains behaviorally intact; yield has no
transcript, ticket, or model-resolution dependency.

**Tech stack:** Go standard library, existing `cmd/spine` test harness, and
repository-local fixtures.

**Spec:** `docs/specs/2026-08-30-routing-yield-review-record-design.md`

## Global constraints

- Do not begin Task 1 until I073 is `fixed` and its independent verifier has
  recorded PASS for I073's exact final implementation SHA, including
  `maipipe run full --wait` at that SHA. Stop and report the blocker if any
  part is absent or names a different SHA.
- I076 is routine-tier implementation. Use the ticket-id token and an explicit
  routed model for every dispatch. Each task gets a reviewer at no lower than
  the ticket review tier. The final whole-branch review and fresh verification
  are primary-tier as required by `WORKFLOW.md`.
- The new record is column-zero REVIEW, harness-only, and exact. It supports
  task records, attributable final records, and the PRD's bounded
  unattributable-final form. Do not add a `flavor` alias, a model lookup, a
  rating rule, or retrospective filename/transcript mining.
- `--fleet` scans immediate non-hidden primary child repositories only. It
  excludes children with a `.git` file, does not recurse, and writes nowhere.
- Stage only the paths named by each commit. Do not stage `.cache/`, unrelated
  source edits, the untracked research stray, local transcripts, or worker
  scratch files.
- Before ship, commit the final candidate, capture its SHA, then run
  `maipipe run full --wait` at that exact SHA. Any later commit requires a new
  full run and a new recorded SHA.

## File map

| Files | Responsibility |
| --- | --- |
| `internal/yield/yield.go` | Exact REVIEW and existing escalation/fallback parsing, identity handling, scope discovery, aggregation, and typed diagnostics. |
| `internal/yield/yield_test.go` | Parser, duplicate/conflict, thresholds, ordering, no-inference, and isolated-fleet test matrix. |
| `cmd/spine/main.go` | `yield` dispatch, flags-first command boundary, text/JSON presentation, and exit selection. |
| `cmd/spine/main_test.go` | Public CLI grammar, output, JSON, exit-code, privacy, and fleet-isolation coverage. |
| One active writing surface selected after I073 | Minimal canonical REVIEW instruction only. Historical documents remain untouched. |
| `docs/issues/I076-routing-yield-review-record-and-yield-verb.md` | Closure evidence only after all gates pass. |

### Task 1: Prove I073 and freeze the implementation contract

**Files:** read I073's ticket, design, plan, independent verification report,
and exact-SHA maipipe evidence. Create no production file if proof is missing.

**Produces:** a recorded prerequisite SHA and a task contract for the strict
REVIEW line, fleet reader, thresholds, output, and exits.

- [ ] Check that I073 says `status: fixed`, names its final implementation SHA,
  and that a fresh independent verifier says PASS for that SHA.
- [ ] Check the maipipe evidence names that same SHA, not an earlier candidate
  or a later unverified commit.
- [ ] If either check fails, stop I076 without edits. Record the exact missing
  artifact in the task report and leave I076 open.
- [ ] Write a contract table in the task notes containing the valid REVIEW line,
  malformed variants, `n` thresholds, fleet child classes, text/JSON fields,
  and exit codes from the PRD.
- [ ] Have the task reviewer confirm no `flavor` compatibility form, no rule
  derives an outcome from a filename or transcript, and final outcomes remain
  separate from task-rate denominators.

### Task 2: Build strict parsing and single-repository aggregation

**Files:**

- Create: `internal/yield/yield.go`
- Create: `internal/yield/yield_test.go`

**Produces:** typed cells keyed by `(Harness, ModelID, Tier)`, totals,
ignored identities, bounded diagnostics, and report confidence state.

- [ ] Write failing tests for this exact accepted line:

  ```text
  REVIEW I076 harness:codex model:gpt-5.6-terra tier:routine round:1 verdict:accepted scope:task
  ```

  Assert the parser returns `Harness: "codex"`, opaque
  `ModelID: "gpt-5.6-terra"`, `Tier: "routine"`, first round, and an
  accepted task result.
- [ ] Add failing cases for legacy `flavor:`, reordered fields, leading
  whitespace, quoted model, `round:0`, `round:01`, unknown harness/tier,
  missing field, and an added field. Assert a line-number-only diagnostic and
  no raw-line echo.
- [ ] Add failing final-series cases: an attributable
  `scope:final verdict:accepted`, an attributable `needs-fixes` record
  whose model is genuinely unavailable (`model:-`), and the only valid
  unattributable form:

  ```text
  REVIEW - harness:- model:- tier:- round:1 verdict:needs-fixes scope:final condition:F-001
  ```

  Reject `ticket:-` task records, an unattributable accepted record, a
  partial-unattributable form, a missing condition, and `condition:` on a
  task or attributable-final record.
- [ ] Add failing aggregation tests: exact duplicate counts once; a conflicting
  `(scope, ticket, round)` excludes that identity; a conflicting
  `(scope, condition, round)` excludes an unattributable final condition;
  first-round task accepted and needs-fixes partition `n`; round two adds
  `rework_verdicts`; final attributable accepted/needs-fixes and
  unattributable needs-fixes stay separate; model IDs remain opaque keys; and
  no defaults, ticket, transcript, or filename input is accepted by the package
  API.
- [ ] Add failing threshold tests for `n=19`, `n=20`, `n=39`, and `n=40`.
  Assert `refused/insufficient`, `low-confidence`, `low-confidence`, and
  `stated` respectively, with a one-decimal first-pass percentage only when
  the PRD permits it.
- [ ] Run `go test ./internal/yield -run 'Test(ParseReview|Aggregate|Threshold)' -count=1`.
  Expected: FAIL because the package and parser do not exist.
- [ ] Implement a column-zero ordered-token parser. Keep model IDs opaque.
  Deduplicate and exclude conflicts before sorting. Count only exact model-tier
  ESCALATION/FALLBACK lines as report-wide totals. Add final totals with no
  rate and no task-cell contribution. Do not change `internal/audit/readLedger`.
- [ ] Run `gofmt -w internal/yield` and `go test ./internal/yield -count=1`.
  Expected: PASS.
- [ ] Request task review against the PRD grammar, no-inference boundary,
  conflict handling, and all four threshold cases. Resolve findings and rerun
  the focused suite.
- [ ] Commit only these paths:

  ```bash
  git add internal/yield/yield.go internal/yield/yield_test.go
  git commit -m "feat(I076): parse recorded review yield"
  ```

### Task 3: Add fleet discovery with isolation, not recursive collection

**Files:**

- Modify: `internal/yield/yield.go`
- Modify: `internal/yield/yield_test.go`

**Produces:** lexical child statuses and aggregate cells whose identity includes
repository name.

- [ ] Write failing filesystem tests with immediate children for a primary
  repository (`.git` directory), linked worktree (`.git` file), hidden
  directory, nested repository, missing ledger, unreadable/broken ledger, and
  two repositories that share ticket `I076`.
- [ ] Assert primary children are read once; worktrees, hidden children, and
  nested entries are not read; a missing ledger contributes explicit zero
  counts; one broken child leaves other aggregates intact; and same-number
  tickets do not collide.
- [ ] Assert fleet statuses and aggregate cells sort lexically by repository
  then harness/model/tier. Assert the package does not follow symlinks or read
  the fleet parent itself as a repository.
- [ ] Run `go test ./internal/yield -run 'TestFleet' -count=1`.
  Expected: FAIL because fleet discovery is absent.
- [ ] Implement immediate-child directory discovery, a `.git` directory
  check, safe no-follow inspection, per-child error capture, and aggregation
  after repository-qualified identity checks. Keep parent errors distinct from
  child errors.
- [ ] Run `gofmt -w internal/yield` and `go test ./internal/yield -count=1`.
  Expected: PASS.
- [ ] Request task review that attempts double counting through a linked
  worktree, a parent ledger, and a malformed child. Resolve findings and rerun
  the focused suite.
- [ ] Commit only these paths:

  ```bash
  git add internal/yield/yield.go internal/yield/yield_test.go
  git commit -m "feat(I076): isolate fleet yield reads"
  ```

### Task 4: Expose the flags-first CLI and public report contract

**Files:**

- Modify: `cmd/spine/main.go`
- Modify: `cmd/spine/main_test.go`

**Produces:** `spine yield [--dir D] [--json]` and
`spine yield --fleet P [--json]` with deterministic output and exit codes.

- [ ] Add failing `runCmd` tests for default `--dir`, an explicit dir,
  `--fleet`, mutually supplied `--dir` and `--fleet`, a flag after a
  positional, an invalid root, and a missing ledger.
- [ ] Add failing text/JSON assertions for zero totals, `n=19`, `n=20`, and
  `n=40`. Assert counts always print, rates are refused below 20, confidence
  labels match the PRD, and exits are 1, 0, and 0 respectively. Add a final
  accepted, attributable needs-fixes, and unattributable needs-fixes fixture;
  assert their separate totals have no percentage and do not alter task `n`.
- [ ] Add failing tests for a malformed REVIEW identity and a broken fleet child
  that still print valid peer counts and return exit 1. Assert root and usage
  errors return 2 before a report.
- [ ] Add failing privacy tests that place distinctive text in a malformed
  ledger line, an unattributable `condition:` token, and a transcript-like
  file. Assert none appears in stdout/stderr and changing the transcript-like
  file leaves output unchanged.
- [ ] Run `go test ./cmd/spine -run 'Test.*Yield' -count=1`.
  Expected: FAIL because `yield` is not a CLI command.
- [ ] Add the usage line, dispatch case, `parseArgs` call with zero
  positionals, mutual `--dir`/`--fleet` validation, and text/JSON renderers
  over typed results. Print scope and totals before cells. Map root/usage
  errors to 2; map refused/ignored/isolated-child states to 1; otherwise return
  0.
- [ ] Run `gofmt -w cmd/spine/main.go cmd/spine/main_test.go` and
  `go test ./cmd/spine -count=1`.
  Expected: PASS.
- [ ] Request task review against output ordering, exit-code precedence,
  privacy, flags-first behavior, and fleet failure isolation. Resolve findings
  and rerun the focused suite.
- [ ] Commit only these paths:

  ```bash
  git add cmd/spine/main.go cmd/spine/main_test.go
  git commit -m "feat(I076): report recorded routing yield"
  ```

### Task 5: Add the minimum live writing instruction

**Files:** inspect `README.md`, `WORKFLOW.md`, templates, and the active
I073 generation-14 output. Modify only the active writing surface that I073's
completed migration leaves appropriate; modify its focused test if generated.

**Produces:** one canonical REVIEW example with no rewrite of archived specs,
handoffs, closed tickets, or fleet repositories.

- [ ] Search active writing guidance for progress-ledger grammar. Classify each
  hit as active, generated, or historical before editing.
- [ ] Add a focused failing test if the selected surface is template-generated.
  Assert the exact harness-only REVIEW line and no template generation bump.
- [ ] Run the smallest relevant test package. Expected: FAIL before the
  instruction is added.
- [ ] Add one task grammar example, the bounded unattributable-final example,
  and the n<20 / 20-39 / >=40 interpretation. State that reviewers write
  actual IDs and never infer records from files or transcripts. State that
  final totals have no rate. Do not add a flavor compatibility sentence.
- [ ] Run the focused test and the relevant generation/update check.
  Expected: PASS with no generated fleet write.
- [ ] Request task review that checks prose against the parser word for word
  and confirms no historical document was rewritten. Resolve findings.
- [ ] Commit only the approved task paths, naming every path explicitly after
  the active surface is chosen.

### Task 6: Review, independently verify, and close only at the final SHA

**Files:** read final diff, this plan, the PRD, I073 verification evidence, and
I076. Modify the I076 ticket only after all gates pass.

- [ ] Run `gofmt -l internal/yield cmd/spine` and fail on any output.
- [ ] Run `git diff --check`, `go test ./internal/yield ./cmd/spine -count=1`,
  `go test ./... -count=1`, `go vet ./...`, and `go build ./cmd/spine`.
- [ ] Build an isolated candidate binary and run per-repo and fleet fixtures.
  Capture text/JSON output and exits for no ledger, malformed data, n=19, n=20,
  n=40, duplicate/conflict, linked-worktree exclusion, and a failed fleet
  child. Verify no transcript or filename changes alter output.
- [ ] Obtain a task-by-task review, then a fresh primary whole-branch
  requirements-attack review. It must test every acceptance criterion, I073
  sequencing, parser ownership, no flavor alias, no unsupported escalation
  attribution, final-series separation, confidence boundaries, privacy, and
  failure isolation.
- [ ] Obtain independent fresh verification with command transcripts and the
  exact candidate SHA. Resolve every blocker before closure.
- [ ] Commit final ticket evidence with explicit paths, record the commit SHA,
  then run `maipipe run full --wait` at that exact SHA. Record the lane result.
  If anything changes after the lane, commit again and rerun the lane at the
  new SHA.
- [ ] Re-read the PRD line by line for the final spec review. Record the
  requirements attack and all evidence in I076; leave it open if any gate or
  exact-SHA lane remains absent.

## Plan self-review

- [x] Every PRD acceptance criterion maps to Tasks 1 through 6.
- [x] Tasks 2 through 5 require a failing focused test, a red run, the minimum
  implementation, a green run, and task review.
- [x] The plan preserves I073 as a verified prerequisite rather than treating
  its current documentation or merge state as completion.
- [x] The final task requires an independent verifier, a requirements attack,
  and maipipe at the exact final SHA.
