# Approved-untested acceptance records (I050) implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` to implement this plan task by
> task. Every dispatch must name I050, use the ticket's `primary` tier, and
> carry the explicit routed model and effort required by `WORKFLOW.md`. Steps
> use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one exact, ticket-local `APPROVED-UNTESTED` record grammar,
validate it once for doctor and cursor-scoped stage audit, and ship the
convention through template generation 12 without changing repositories that
do not use it.

**Architecture:** A new `internal/acceptance` package owns ticket discovery,
candidate detection, canonical parsing, dated approval-reference validation,
and counts. `internal/doctor` maps invalid candidates to the source-allocated
D15 warning. `internal/stages.Report` carries the same package's cursor-scoped
summary, while `cmdAuditStages` prints warnings and an optional count line
without changing `Report.Blocking()`.

**Tech stack:** Go standard library, embedded Markdown templates, existing
scaffold/update machinery, and existing CLI test harness.

**Spec:** `docs/specs/2026-08-29-approved-untested-acceptance-design.md`

## Global constraints

- I050 is `subagent-driven`, `tier: primary`, `review-tier: primary`. Every
  worker and reviewer dispatch includes the literal ticket token `I050` and an
  explicit routed model and effort. Artifacts name tiers, never model IDs.
- Strict red-green TDD applies to every production change. Run the named test
  command before implementation and record the expected failure. A test that
  starts green does not prove the change and must be corrected before work
  continues.
- The correction round normalizes relative roots once, imposes no new physical
  line limit, surfaces every read error, aggregates every safely applicable
  grammar/reference failure in deterministic order, and recognizes every
  column-0 bare/space/tab ATX H1/H2 boundary while retaining H3 in section.
- The canonical grammar is copied exactly from the PRD. Do not add aliases,
  multiline forms, `WAIVED`, fragment resolution, follow-up-ticket parsing, or
  authentication.
- D15 is fixed for I050. D14 is already assigned to I108 in current source.
  Re-run the allocator before implementation; if D15 has since been claimed,
  stop and amend this PRD and plan through review instead of silently choosing
  another ID.
- `internal/acceptance` is the only parser. Doctor and stages may map its
  results but may not parse marker text or approval paths themselves.
- `stages.Report.Blocking()` must remain byte-for-byte unchanged. Acceptance
  records never become stage evidence or blockers.
- A zero-candidate, zero-applicable-scan-error repository keeps doctor and
  `audit stages` output and exits unchanged. Existing tickets are never
  rewritten.
- Generation 12 changes are additive. Add no `supersededLines` entry unless an
  emitted predecessor line is actually removed or reworded. Any such removal
  requires a reviewed PRD amendment before implementation continues.
- Preserve the `knowledge` profile's no-issue-ledger manifest.
- Use explicit paths for every `git add`. Never stage `.cache/`, the known
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md` stray, concurrent
  code, or another worker's changes.
- Do not tick the handoff stage. Use `spine` as the only cursor writer, and
  account for ADR 0014's expected audit block until a fresh handoff exists.

## File and interface map

### New shared package

- Create `internal/acceptance/acceptance.go` with these exported types and
  signatures:

  ```go
  type Record struct {
      Path      string
      Line      int
      Criterion string
      Date      string
      Approver  string
      Reference string
      Reason    string
  }

  type Problem struct {
      Path   string
      Line   int
      Failed []string
  }

  func (p Problem) Message() string

  type ScanError struct {
      Path string
      Err  error
  }

  type Summary struct {
      Records    []Record
      Problems   []Problem
      ScanErrors []ScanError
  }

  func (s Summary) ValidCount() int
  func (s Summary) InvalidCount() int
  func (s Summary) CandidateCount() int
  func ScanTicket(repoRoot, ticketPath string) Summary
  func ScanAllTickets(repoRoot string) Summary
  func ScanTicketIDs(repoRoot string, ids []string) Summary
  ```

  `ScanError` carries the slash-form ticket path and underlying read cause.
  `ScanTicket` accepts an absolute or repository-relative ticket path but
  reports `Record.Path` and `Problem.Path` as slash-form paths relative to
  `repoRoot`. `ScanAllTickets` scans eligible `docs/issues/I*.md` files at all
  statuses. `ScanTicketIDs` discovers the same eligible files, reads their
  leading frontmatter `id`, and includes only exact IDs in the supplied set.
  It skips discovery, open, and read failures before an ID is established. It
  surfaces a scan failure only after the established ID matches a supplied ID.
  All three call one unexported line parser and one unexported approval-path
  validator. Their line reader adds no fixed maximum; read failures populate
  `Summary.ScanErrors` rather than returning a clean empty result.

- Create `internal/acceptance/acceptance_test.go` for the grammar, scope,
  path-security, discovery, and count matrix.

### Doctor adapter

- Create `internal/doctor/acceptance.go` with
  `func acceptanceCheck(dir string) []Finding`. It maps every
  `acceptance.Problem` and explicit scan error to exactly one
  `Finding{"D15", "warn", ...}` without counting scan errors as invalid
  candidates.
- Create `internal/doctor/acceptance_test.go` for package-level D15 behavior.
- Modify `internal/doctor/doctor.go`: change the package comment to D1-D15 and
  append `acceptanceCheck(dir)` in `Run` after `ticketCheck(dir)`. Keep D14's
  toolchain call intact.
- Modify `cmd/spine/main_test.go` for text, JSON, and exit-code coverage. No
  production change to `cmdDoctor` is expected.

### Stage-audit adapter

- Modify `internal/stages/stages.go`: import `internal/acceptance`; add
  `Acceptance acceptance.Summary` to `Report`; after the existing
  `resolveTicketIDs` succeeds in `FromResult`, populate it with
  `acceptance.ScanTicketIDs(dir, ids)`. Do not scan when the expression is
  empty or unresolvable. Do not change `Report.Blocking()`.
- Modify `internal/stages/stages_test.go` for scoped summaries and blocking
  isolation.
- Modify `cmd/spine/main.go` in `cmdAuditStages`: print one `warning:` per
  `rep.Acceptance.Problems` and scan error; print the exact summary only when
  `CandidateCount() > 0`, after stage rows and before the handoff line.
- Modify `cmd/spine/main_test.go` for exact stdout, stderr, and exits.

### Generation 12

- Modify `templates/VERSION` from 11 to 12.
- Modify `templates/current/WORKFLOW.md.tmpl`,
  `templates/current/issue.tmpl.md`, and
  `templates/current/issues-README.md` with the additive content specified in
  the PRD.
- Create `internal/update/gen11to12_test.go` and fixture files under
  `internal/update/testdata/spine-gen11/` for `WORKFLOW.md`, `CLAUDE.md`,
  `AGENTS.md`, `docs/issues/README.md`, and `docs/issues/_template.md`.
- Modify current-generation assertions in `internal/tmpl/tmpl_test.go`,
  `internal/scaffold/scaffold_test.go`, `internal/adopt/adopt_test.go`,
  `internal/update/update_test.go`, `internal/update/hbmview_test.go`,
  `internal/update/gen5to6_test.go`, `internal/update/gen7to8_test.go`,
  `internal/update/gen8to9_test.go`, `internal/update/gen9to10_test.go`,
  `internal/update/gen10to11_test.go`, `internal/update/gatepack_test.go`, and
  current-generation cases in `cmd/spine/main_test.go`. Historical fixture
  inputs stay stamped at their captured generation.
- Regenerate repository mirrors `WORKFLOW.md`, `docs/issues/_template.md`,
  `docs/issues/README.md`, `AGENTS.md`, and `CLAUDE.md` with the updated binary
  path. Inspect the diff before staging.

### Closure

- Modify `CHANGELOG.md` for the consumer-visible grammar, D15, audit summary,
  and generation 12.
- Modify `docs/issues/I050-approved-untested-acceptance-lane.md` only after
  review and verification: set `status: fixed`, write actual implementation
  commit SHAs in `commits`, and add `## Resolution` with test and gate evidence.

## Task 1: shared record parser and ticket scanners

**Files:**

- Create: `internal/acceptance/acceptance.go`
- Create: `internal/acceptance/acceptance_test.go`

**Produces:** The exact interfaces in the file map. `Problem.Message()` returns
the format produced by
`fmt.Sprintf("line %d: invalid APPROVED-UNTESTED record: %s", p.Line,
strings.Join(p.Failed, "; "))`, with a deterministic requirement order.

- [ ] **Step 1: Write the grammar and attribution tests.** Add these named
  tests before the package implementation:

  ```go
  func TestScanTicketAcceptsCanonicalRecord(t *testing.T)
  func TestScanTicketAggregatesMissingFieldsPerCandidate(t *testing.T)
  func TestScanTicketRejectsCheckboxAndSectionDamage(t *testing.T)
  func TestScanTicketIgnoresNonCandidates(t *testing.T)
  func TestScanTicketCountsMultipleRecordsWithLineAttribution(t *testing.T)
  ```

  The canonical test creates
  `docs/handoffs/2026-08-29-i050-approval.md`, writes the exact PRD example
  under the exact heading, and asserts all seven record fields plus valid=1,
  invalid=0, candidate=1. Table cases independently remove criterion, date,
  approver, ref, fragment, and reason; alter the date to `2026-02-30`; alter
  every delimiter; use `[x]`, indentation, malformed checkbox spaces, and the
  wrong heading. Each case asserts one problem and one invalid candidate, not
  one problem per missing field.

- [ ] **Step 2: Write path-security tests.** Add
  `TestScanTicketRejectsUnsafeApprovalReferences` with table cases for an
  absolute path, `../`, `docs/../`, backslash, non-`docs/`, `.txt`, undated
  basename, missing fragment, missing file, directory target, broken symlink,
  and a symlink resolving outside the repository. Add green controls for a
  dated file in a nested `docs/` directory and an internal symlink whose
  resolved regular target stays under the repo root. Assert the fragment is
  not looked up by using a deliberately nonexistent fragment on the valid
  file.

- [ ] **Step 3: Write discovery and scoping tests.** Add
  `TestScanAllTicketsIncludesClosedTicketsAndSkipsLedgerDocs` and
  `TestScanTicketIDsUsesExactFrontmatterIDs`. Cover open and fixed tickets,
  multiple candidates, README, `_template.md`, a directory ending in `.md`, a
  non-ticket Markdown file, an unselected ticket, and a filename whose prefix
  resembles but does not equal the selected frontmatter ID.

- [ ] **Step 4: Run the package tests red.** Run:

  ```bash
  go test ./internal/acceptance -count=1
  ```

  Expected: FAIL to compile because `internal/acceptance` and the declared
  types/functions do not exist. Record the command and failure.

- [ ] **Step 5: Implement the minimum parser.** Use line-by-line scanning with
  1-based counters. Candidate detection is broader than canonical parsing.
  Track the exact acceptance heading until the next column-0 H1/H2. Parse the
  canonical form once, validate real dates with `time.Parse("2006-01-02",
  value)`, validate slash paths with `path.Clean`, and use absolute/root-rel
  plus symlink-resolved containment before accepting a regular artifact.
  Sort discovered ticket paths so output is deterministic.

- [ ] **Step 6: Run red cases green.** Run:

  ```bash
  go test ./internal/acceptance -count=1
  go test ./internal/acceptance -run 'TestScanTicketRejectsUnsafeApprovalReferences' -count=20
  ```

  Expected: PASS. The repeated security table must remain deterministic.

- [ ] **Step 7: Run a parser negative control.** Temporarily allow checked
  `[x]` candidates through the canonical parser, rerun
  `TestScanTicketRejectsCheckboxAndSectionDamage`, and record the expected
  FAIL naming the checked-state case. Restore the strict parser and rerun the
  package green.

- [ ] **Step 8: Commit the shared unit with explicit paths.** Run:

  ```bash
  git add internal/acceptance/acceptance.go internal/acceptance/acceptance_test.go
  git commit -m 'feat(I050): validate approved-untested records'
  ```

## Task 2: doctor D15 mapping and CLI behavior

**Files:**

- Create: `internal/doctor/acceptance.go`
- Create: `internal/doctor/acceptance_test.go`
- Modify: `internal/doctor/doctor.go` (`Run`, package comment)
- Modify: `cmd/spine/main_test.go`

**Consumes:** `acceptance.ScanAllTickets`, `Problem.Message()`.

**Produces:** `acceptanceCheck(dir string) []Finding`, one D15 warn per invalid
candidate and no finding for valid or absent candidates.

- [ ] **Step 1: Reconfirm D15 before writing tests.** Run:

  ```bash
  rg -o '"D[0-9]+"' internal/doctor cmd/spine | tr -d '"' | sort -Vu
  git log -8 --oneline -- internal/doctor
  ```

  Expected: D1 through D14 are claimed and D15 is absent. If not, stop for a
  PRD amendment and fresh approval.

- [ ] **Step 2: Write failing package-level doctor tests.** Add:

  ```go
  func TestD15SilentOnValidAndAbsentRecords(t *testing.T)
  func TestD15WarnsOnceForReasonlessRecord(t *testing.T)
  func TestD15WarnsForMissingArtifactAndClosedTicket(t *testing.T)
  func TestD15FindingCarriesPathLineAndAggregatedFailures(t *testing.T)
  ```

  Scaffold clean `library-cli` repos, neutralize unrelated D14 variability by
  filtering only D15, and assert ID, `warn`, slash-form ticket path, 1-based
  line in the message, and one finding per candidate. The reason-less test is
  I050's required negative control.

- [ ] **Step 3: Write failing CLI tests.** In `cmd/spine/main_test.go`, add
  `TestDoctorD15TextAndExitContract` and `TestDoctorD15JSONShape`. A
  reason-less candidate must yield exit 1, text containing `D15 warn`, the
  ticket path and line, and JSON containing only the existing finding keys
  with `id="D15"`. A valid-record fixture must contain no D15 in either mode.

- [ ] **Step 4: Run focused tests red.** Run:

  ```bash
  go test ./internal/doctor ./cmd/spine -run 'Test.*D15' -count=1
  ```

  Expected: FAIL because `acceptanceCheck` is absent and `doctor.Run` emits no
  D15.

- [ ] **Step 5: Implement the thin doctor adapter.** Map shared problems only.
  Append it after `ticketCheck(dir)` and before `toolchainCheck()`. Update the
  package comment to D1-D15. Do not change `cmdDoctor`, `Finding`, JSON, or the
  warn/error exit loop.

- [ ] **Step 6: Verify focused and compatibility tests green.** Run:

  ```bash
  go test ./internal/doctor ./cmd/spine -run 'Test.*(D15|DoctorCleanAndJSON|DoctorInfoOnlyExitsZero)' -count=1
  ```

  Expected: PASS, including the pre-I050 doctor exit controls.

- [ ] **Step 7: Run a doctor negative control.** Temporarily map problems to
  `info`; rerun `TestDoctorD15TextAndExitContract` and observe FAIL because the
  command exits 0. Restore `warn` and rerun green.

- [ ] **Step 8: Commit the doctor unit.** Run:

  ```bash
  git add internal/doctor/acceptance.go internal/doctor/acceptance_test.go internal/doctor/doctor.go cmd/spine/main_test.go
  git commit -m 'feat(I050): report invalid acceptance records in doctor'
  ```

## Task 3: cursor-scoped stage audit summary

**Files:**

- Modify: `internal/stages/stages.go` (`Report`, `FromResult`)
- Modify: `internal/stages/stages_test.go`
- Modify: `cmd/spine/main.go` (`cmdAuditStages` only)
- Modify: `cmd/spine/main_test.go`

**Consumes:** `acceptance.ScanTicketIDs`, `Summary` count methods, and the IDs
already returned by `resolveTicketIDs`.

**Produces:** `Report.Acceptance acceptance.Summary`, advisory stderr warnings,
and the exact optional stdout count line. `Report.Blocking()` remains unchanged.

- [ ] **Step 1: Write failing stages tests.** Add:

  ```go
  func TestAcceptanceSummaryUsesResolvedCursorTicketsOnly(t *testing.T)
  func TestAcceptanceSummarySkipsUnresolvableTickets(t *testing.T)
  func TestAcceptanceProblemsNeverAffectBlocking(t *testing.T)
  ```

  Copy the existing clean fixture into a temp directory. Add one valid marker
  to I001, one invalid marker to I002, and one marker to an unscoped I003.
  Assert valid=1, invalid=1, candidate=2 for the I001-I002 cursor. For an
  unresolvable cursor value, assert the existing Notes warning and a zero
  summary. For an otherwise clean report containing only an invalid marker,
  assert `Blocking()==false`.

- [ ] **Step 2: Write failing CLI output tests.** Add:

  ```go
  func TestAuditStagesPrintsValidAcceptanceSummary(t *testing.T)
  func TestAuditStagesInvalidAcceptanceWarnsWithoutBlocking(t *testing.T)
  func TestAuditStagesOmitsAcceptanceLineWhenNoCandidates(t *testing.T)
  func TestAuditStagesExistingBlockStillBlocksWithAcceptance(t *testing.T)
  ```

  Pin the exact line
  `acceptance: approved-untested=1 invalid=0\n` after stage rows and before
  `handoff:`. Pin one `warning:` line for the invalid case, exit 0 on the clean
  fixture, and exit 1 only when an existing stage or handoff condition blocks.
  Capture the current zero-marker clean-fixture stdout/stderr before the
  production edit and assert byte equality after it.

- [ ] **Step 3: Run focused tests red.** Run:

  ```bash
  go test ./internal/stages ./cmd/spine -run 'Test.*Acceptance' -count=1
  ```

  Expected: FAIL because `Report.Acceptance` and the audit printer do not
  exist.

- [ ] **Step 4: Implement summary wiring.** In `FromResult`, call the existing
  `resolveTicketIDs(dir, res.Cursor.Tickets)` a second time for acceptance
  scope after `deriveStages`; populate the summary only when it returns
  `ok=true`. Do not fall back to `ScanAllTickets`. In
  `cmdAuditStages`, print problems with the same deterministic
  path/line/message data as D15, then print the summary only for a nonzero
  candidate count at the required position.

- [ ] **Step 5: Verify focused and existing stage tests green.** Run:

  ```bash
  go test ./internal/stages ./cmd/spine -run 'Test.*(Acceptance|AuditStages|DoctorAdvisesOnNonCanonicalCursor)' -count=1
  ```

  Expected: PASS. Existing malformed cursor, noncanonical cursor, handoff, and
  zero-ledger exits must remain unchanged.

- [ ] **Step 6: Run a blocking negative control.** Temporarily add
  `rep.Acceptance.InvalidCount() > 0` to `Report.Blocking()`. Rerun
  `TestAcceptanceProblemsNeverAffectBlocking` and
  `TestAuditStagesInvalidAcceptanceWarnsWithoutBlocking`; observe both FAIL.
  Restore `Blocking()` byte-for-byte and rerun green.

- [ ] **Step 7: Commit the audit unit.** Run:

  ```bash
  git add internal/stages/stages.go internal/stages/stages_test.go cmd/spine/main.go cmd/spine/main_test.go
  git commit -m 'feat(I050): audit scoped acceptance exceptions'
  ```

## Task 4: generation-12 templates and safe migration

**Files:**

- Create: `internal/update/gen11to12_test.go`
- Create: `internal/update/testdata/spine-gen11/WORKFLOW.md`
- Create: `internal/update/testdata/spine-gen11/CLAUDE.md`
- Create: `internal/update/testdata/spine-gen11/AGENTS.md`
- Create: `internal/update/testdata/spine-gen11/docs/issues/README.md`
- Create: `internal/update/testdata/spine-gen11/docs/issues/_template.md`
- Modify: `templates/VERSION`
- Modify: `templates/current/WORKFLOW.md.tmpl`
- Modify: `templates/current/issue.tmpl.md`
- Modify: `templates/current/issues-README.md`
- Modify: all current-generation assertion files listed in the file map
- Modify by generated update: `WORKFLOW.md`, `docs/issues/_template.md`,
  `docs/issues/README.md`, `AGENTS.md`, `CLAUDE.md`

**Produces:** Compiled generation 12, additive convention wording, safe
generation-11 migration, and no issue ledger for knowledge profiles.

- [ ] **Step 1: Capture the pristine generation-11 fixture.** Copy only the
  five current generated source files named above into
  `internal/update/testdata/spine-gen11/` before changing templates. Verify
  their stamps are 11. Do not copy concurrent working-tree edits.

- [ ] **Step 2: Write failing template and scaffold tests.** Update
  `internal/tmpl/tmpl_test.go` to expect `tmpl.Version()==12` and exact
  WORKFLOW grammar fragments. In `internal/scaffold/scaffold_test.go`, assert
  a non-knowledge scaffold has generation 12, the acceptance heading and
  README pointer, while a knowledge scaffold has the WORKFLOW convention and
  still lacks `docs/issues/`.

- [ ] **Step 3: Write failing migration tests.** In
  `internal/update/gen11to12_test.go`, add:

  ```go
  func TestGen11To12PristineUpdatesCleanly(t *testing.T)
  func TestGen11To12WritesConventionAndIsIdempotent(t *testing.T)
  func TestGen11To12PreservesLocalEditRefusals(t *testing.T)
  func TestGen12ChangesAreAdditive(t *testing.T)
  ```

  The pristine test requires zero `Unrecognized` lines for all five managed
  files. The write test requires generation 12, exact WORKFLOW grammar,
  issue-template heading, README pointer, v12 AGENTS/CLAUDE stamps, and an
  entirely `UpToDate` second pass. The local-edit table changes one content
  line in each touched managed file and expects `SkippedUnrecognized` without
  force. The additive test permits stamps and declared added lines only and
  rejects any undeclared removed prose.

- [ ] **Step 4: Run template and migration tests red.** Run:

  ```bash
  go test ./internal/tmpl ./internal/scaffold ./internal/update -run 'Test.*(Version|Acceptance|Knowledge|Gen11To12|Gen12)' -count=1
  ```

  Expected: FAIL because the compiled version remains 11 and current templates
  lack the convention.

- [ ] **Step 5: Make the additive template edits and bump VERSION.** Add the
  exact grammar only to WORKFLOW. Add the empty acceptance section and pointer
  to the issue template. Add only the semantic pointer to the issue README.
  Do not place a checklist-shaped live sentinel in a template or README.

- [ ] **Step 6: Update every current-generation assertion.** Use:

  ```bash
  rg -n 'template_version: 11|begin v11|generation 11|want 11|!= 11|future generation|template_version: 12' internal cmd templates --glob '*.go' --glob 'VERSION'
  ```

  Change assertions about the compiled current version to 12. Keep captured
  historical input fixtures at 11. Change the future-generation refusal to 13.
  Review each hit rather than running a blind replacement.

- [ ] **Step 7: Run focused generation tests green.** Run:

  ```bash
  go test ./internal/tmpl ./internal/scaffold ./internal/adopt ./internal/update ./cmd/spine -run 'Test.*(Version|Init|Adopt|Update|Gen|GatePack|Knowledge)' -count=1
  ```

  Expected: PASS, including historical migrations and the new 11-to-12 lock.

- [ ] **Step 8: Regenerate repository mirrors through spine.** Run:

  ```bash
  go run ./cmd/spine update --dir . --write
  git diff -- WORKFLOW.md docs/issues/_template.md docs/issues/README.md AGENTS.md CLAUDE.md
  ```

  Expected: WORKFLOW, issue template, and issue README receive only the
  declared additive convention plus version stamps; AGENTS and CLAUDE receive
  stamp-only changes. No other file is written. If update reports an
  unrecognized concurrent edit, stop and coordinate instead of forcing.

- [ ] **Step 9: Verify dry-run idempotence and no superseded-line debt.** Run:

  ```bash
  go run ./cmd/spine update --dir .
  rg -n 'APPROVED-UNTESTED|Acceptance exceptions|Acceptance criteria' templates/current WORKFLOW.md docs/issues
  git diff --check
  ```

  Expected: every update report is up to date; the grammar appears only on
  intended documentation surfaces; no removed emitted line exists; diff check
  exits 0.

- [ ] **Step 10: Commit generation 12 with explicit paths.** Stage only the
  files named in this task after inspecting `git status --short`. Run:

  ```bash
  git add templates/VERSION templates/current/WORKFLOW.md.tmpl templates/current/issue.tmpl.md templates/current/issues-README.md internal/update/gen11to12_test.go internal/update/testdata/spine-gen11/WORKFLOW.md internal/update/testdata/spine-gen11/CLAUDE.md internal/update/testdata/spine-gen11/AGENTS.md internal/update/testdata/spine-gen11/docs/issues/README.md internal/update/testdata/spine-gen11/docs/issues/_template.md internal/tmpl/tmpl_test.go internal/scaffold/scaffold_test.go internal/adopt/adopt_test.go internal/update/update_test.go internal/update/hbmview_test.go internal/update/gen5to6_test.go internal/update/gen7to8_test.go internal/update/gen8to9_test.go internal/update/gen9to10_test.go internal/update/gen10to11_test.go internal/update/gatepack_test.go cmd/spine/main_test.go WORKFLOW.md docs/issues/_template.md docs/issues/README.md AGENTS.md CLAUDE.md
  git commit -m 'feat(I050): ship acceptance records in generation 12'
  ```

  The recorded staged-path list is part of the task evidence. Do not use
  `git add .` or `git add -A`.

## Task 5: integration checks and consumer documentation

**Files:**

- Modify: `CHANGELOG.md`
- Verify: every production, test, template, fixture, and mirror path from
  Tasks 1 through 4

- [ ] **Step 1: Add the CHANGELOG entry before final review.** State the exact
  marker purpose, D15 warning behavior, cursor-scoped nonblocking audit count,
  and template generation 12. Do not describe the record as authenticated or
  claim ordinary unchecked criteria now gate.

- [ ] **Step 2: Run formatting and the complete local suite.** Run:

  ```bash
  gofmt -l internal/acceptance/acceptance.go internal/acceptance/acceptance_test.go internal/doctor/acceptance.go internal/doctor/acceptance_test.go internal/doctor/doctor.go internal/stages/stages.go internal/stages/stages_test.go cmd/spine/main.go cmd/spine/main_test.go internal/update/gen11to12_test.go internal/tmpl/tmpl_test.go internal/scaffold/scaffold_test.go
  go test ./... -count=1
  go vet ./...
  git diff --check
  ```

  Expected: `gofmt -l` and `git diff --check` print nothing; tests and vet
  exit 0.

- [ ] **Step 3: Run command-level functional probes in disposable repos.** Use
  temp repositories created by `spine init`, then prove and record:

  - no marker: doctor and audit output match the saved pre-I050 controls;
  - canonical marker plus a real approval artifact: no D15, audit prints
    `approved-untested=1 invalid=0`, exit 0 absent other blockers;
  - reason-less marker: exactly one D15 warn and doctor exit 1;
  - the same invalid scoped marker: one audit stderr warning, invalid=1, and
    exit 0 absent other blockers;
  - an unscoped marker contributes no audit count;
  - a missing artifact and outside-root symlink each fail safely;
  - `doctor --json` retains the existing finding schema.

- [ ] **Step 4: Run repository gates before review.** Run:

  ```bash
  go test ./internal/acceptance ./internal/doctor ./internal/stages ./cmd/spine -count=1
  go test ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  go run ./cmd/spine update --dir .
  ./bin/spine doctor --dir .
  ./bin/spine audit routing --dir .
  ./bin/spine audit stages --dir .
  git diff --check
  ```

  Expected: the focused/full Go, vet, build, update, and diff checks pass.
  Record and separate pre-existing or concurrent doctor findings. Routing has
  no I050 silent descent. Audit stages may show ADR 0014's expected
  stale-handoff block; it must not show an I050 block or scan tickets outside
  the active cursor. The final exact-SHA verifier runs
  `maipipe run full --wait` after all commits that belong to that SHA.

- [ ] **Step 5: Commit the CHANGELOG independently.** Run:

  ```bash
  git add CHANGELOG.md
  git commit -m 'docs(I050): document approved-untested acceptance records'
  ```

  Stage no concurrent CHANGELOG hunk. If the file has overlapping edits, stop
  and coordinate or use a patch-based partial stage that contains only I050.

## Correction round after failed fresh review

The first fresh primary review failed. This correction round owns every
finding in `.superpowers/sdd/I050-review-worker1-report.md`; none is narrowed
to the review's smallest reproducer.

- [ ] **Step 1: Reopen and amend before product edits.** In one docs-only
  commit, set I050 to `status: open`, leave all three ticket criteria
  unchecked, clarify that references split once at the first `#` and preserve
  all later fragment bytes, replace the false “15 acceptance criteria” review
  instruction, and replace every nonexistent `make verify` step with the
  focused/full Go, vet, build, update, doctor/audit, diff, and final maipipe
  sequence. The blind report is the red evidence for the premature fixed
  state; direct assertions against the old docs must fail before the edit.

- [ ] **Step 2: Relative-root slice.** Add failing package regressions for
  both `ScanAllTickets` and `ScanTicketIDs` from a named relative repository
  root. Add compiled-binary CLI regressions that run `doctor --dir .`,
  `doctor --dir <named-relative>`, and `audit stages --dir .` from their
  parent/current directories and pin D15, acceptance summary, stderr, and
  exit behavior. Observe red because discovery passes an already root-prefixed
  relative ticket path back through root joining. Normalize the repository
  root once with `filepath.Abs`, pass internally consistent absolute paths,
  and rerun focused green.

- [ ] **Step 3: Arbitrary-line and read-error slice.** Add one failing test
  for a candidate line longer than 64 KiB and another for a long noncandidate
  followed by both valid and invalid candidates. Add a failing injected-reader
  test proving a non-EOF read error is surfaced explicitly, plus doctor and
  stage-adapter assertions that the error is visible and never becomes a
  candidate count or stage blocker. Observe the default-Scanner/ignored-error
  red, then use a `bufio.Reader` loop with no new fixed maximum and explicit
  error propagation. Rerun focused green.

- [ ] **Step 4: Deterministic aggregation slice.** Add failing tables that
  combine independent grammar failures and reference failures. Pin the exact
  ordered `Problem.Failed` sequence and exact D15/audit message. The reference
  `outside/approval.txt#x` must report every applicable prefix, suffix, dated
  basename, and existence failure. Parse recoverable fields independently;
  evaluate every safe lexical check and every filesystem check whose
  prerequisites are valid; skip only unsafe dependent checks. One candidate
  still produces exactly one problem. Rerun the table repeatedly to prove
  deterministic green.

- [ ] **Step 5: ATX boundary slice.** Add a failing table for all column-0
  bare, space-delimited, and tab-delimited H1/H2 boundaries (`#`, `##`,
  `# Title`, `## Title`, `#\tTitle`, `##\tTitle`) and H3 controls in the same
  forms that remain inside the acceptance section. Implement one boundary
  helper accepting one or two `#` bytes followed only by end of line, ASCII
  space, or tab. Rerun focused green.

- [ ] **Step 6: Preserve passing behavior and commit coherent units.** Rerun
  the existing canonical grammar, path containment/symlink, D15 text/JSON,
  scoped nonblocking audit, zero-marker byte-compatibility, and generation-12
  migration suites cited as passing by the blind report. Commit explicit
  paths in coherent correction units. Append every correction SHA to I050,
  but keep `status: open`, keep affected criteria unchecked, and do not add a
  closure resolution in this dispatch.

- [ ] **Step 7: Correction verification before handoff to fresh gates.** Run:

  ```bash
  go test ./internal/acceptance -run 'Test.*(Relative|Long|Read|Aggregate|Heading|Fragment)' -count=1
  go test ./internal/doctor ./internal/stages ./cmd/spine -run 'Test.*(D15|Acceptance|Relative)' -count=1
  go test ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine update --dir .
  ./bin/spine doctor --dir .
  ./bin/spine audit routing --dir .
  ./bin/spine audit stages --dir .
  git diff --check
  maipipe run full --wait
  ```

  Also run compiled-CLI relative-root and hostile-reference fixtures directly,
  recording raw stdout, stderr, and exit codes. Known unrelated repository
  advisories or stale-handoff state are recorded separately and never
  misreported as I050 success. A different fresh primary reviewer and then a
  different independent primary verifier own the later closure gates.

## Second correction round after failed fresh re-review

The second fresh primary review failed at exact SHA `3a0a46a`. This correction
round treats the PRD contradictions and all three findings in
`.superpowers/sdd/I050-rereview-worker2-report.md` as binding. It does not close
I050 or advance review, verification, lane, or handoff state.

- [ ] **Step 1: Amend the contract before code.** Narrow every no-marker
  byte-compatibility promise to zero candidates and zero applicable scan
  errors. Define identity-scoped discovery to skip failures before a
  frontmatter ID can be established, leave those failures to estate-wide
  doctor, and surface only later failures for an established wanted ID. Define
  the marker as one unique exact ` -- APPROVED-UNTESTED ` structural marker;
  sentinel bytes in criterion text remain allowed and multiple structural
  markers are invalid. Commit only the design, plan, and open I050 ticket.

- [ ] **Step 2: Whitespace-token TDD slice.** Before production edits, add
  `TestScanTicketRejectsWhitespaceInApproverAndReferenceToken` with ordered
  table cases for ASCII space, tab, and Unicode whitespace in the approver,
  reference base, and reference fragment. Pin the exact aggregated failure
  order, with the approver failure before the full-reference-token failure.
  Add `TestD15WarnsForWhitespaceAcceptanceTokens`,
  `TestAuditStagesWarnsForWhitespaceAcceptanceTokensAndCountsInvalid`, and
  `TestCompiledCLIRejectsWhitespaceAcceptanceTokens` to cover package doctor,
  doctor text and JSON, nonblocking audit warnings and counts, and the compiled
  binary. Observe red because the current parser checks only emptiness. Reject
  every code point accepted by Go's Unicode whitespace classification in both
  extracted tokens, then rerun focused green.

- [ ] **Step 3: Structural-marker TDD slice.** Before production edits, add
  `TestScanTicketSelectsUniqueStructuralMarker` with a valid criterion that
  contains bare `APPROVED-UNTESTED` bytes and
  `TestScanTicketRejectsAmbiguousStructuralMarkers` with two exact structural
  markers. Add `TestDoctorAndAuditUseStructuralAcceptanceMarker` and
  `TestCompiledCLIUsesStructuralAcceptanceMarker` for the adapters and built
  binary. Observe the sentinel-in-criterion case red against first-substring
  parsing. Select only the unique exact structural marker, retain broad damaged
  checklist candidate detection, reject ambiguity with one deterministic
  problem, and rerun focused green.

- [ ] **Step 4: Identity-scoping TDD slice.** Before production edits, add
  `TestScanTicketIDsSkipsPreIDFailuresAndSurfacesWantedPostIDFailures`,
  `TestAcceptanceSummaryExcludesUnreadableUnknownIDTickets`, and
  `TestCompiledCLIAcceptanceIdentityScoping`. The compiled fixture contains a
  broken unscoped ticket, a readable invalid unscoped ticket, and a wanted
  ticket that fails after its matching ID is read. Audit must exclude both
  unscoped controls while retaining the wanted post-ID nonblocking warning;
  `ScanAllTickets`, doctor text, doctor JSON, and doctor exit 1 must still
  surface the estate-wide broken-unscoped failure. Observe red because current
  scoped discovery appends pre-ID errors. Implement the fail-closed identity
  policy without changing candidate counts or `Report.Blocking()`, then rerun
  focused green.

- [ ] **Step 5: Per-iteration cleanup TDD slice.** Record
  `spine gate --dir . go@1 deferred-cleanup-errcheck` red at
  `internal/acceptance/acceptance.go` inside the ticket loop. Add
  `TestScanTicketIDsClosesEachDiscoveredTicketPerIteration` using the existing
  scanner seam or the smallest test-only file hook that can prove close timing
  without asserting source text. Replace the loop-scoped `defer f.Close()`
  with per-iteration cleanup that preserves open, read, and close errors in the
  applicable estate-wide or identity-scoped error policy. Rerun the focused
  package test and the deferred-cleanup gate green.

- [ ] **Step 6: Rerun every retained regression.** Run all original and
  first-correction parser, reference-containment, arbitrary-line, read-error,
  aggregation, H1/H2 boundary, D15 text/JSON, cursor-scoping, blocking,
  zero-candidate/zero-error byte-compatibility, generation-12 migration,
  relative-root, hostile-symlink, and compiled-CLI controls. Run the whitespace,
  sentinel, and identity-scoping tables repeatedly to prove deterministic
  output.

- [ ] **Step 7: Verify and report without advancing gates.** Run focused and
  full Go tests, `go vet ./...`, `go build -o bin/spine ./cmd/spine`, `gofmt
  -l`, `git diff --check`, the compiled relative/hostile/whitespace/sentinel/
  scoping matrix, update dry-run, doctor, routing audit, and stages audit. Check
  the concurrent I032 gate-layout correction before interpreting maipipe, and
  rerun the go@1 `dead-code-callgraph` and `deferred-cleanup-errcheck` gate
  classes after that correction lands. Commit coherent code/test/docs units
  with explicit paths, append their SHAs to the open I050 ticket while leaving
  every criterion unchecked, and write
  `.superpowers/sdd/I050-correction2-worker3-report.md` with red/green evidence,
  commits, tests, and remaining fresh-review and independent-verifier gates.

## Task 6: fresh spec review, verification, ticket closure, and final commits

**Files:**

- Review: `docs/specs/2026-08-29-approved-untested-acceptance-design.md`
- Review: `docs/specs/2026-08-29-approved-untested-acceptance-plan.md`
- Modify after approval: `docs/issues/I050-approved-untested-acceptance-lane.md`
- Modify only for review corrections: files already owned by Tasks 1 through 5

- [ ] **Step 1: Run the mandatory fresh spec review.** Dispatch a fresh
  primary-tier reviewer with I050 in the prompt and an explicit routed model
  and effort. The reviewer first attacks this PRD for contradictions, then
  compares every finished diff hunk with every binding PRD requirement and
  all ticket acceptance criteria. It must
  explicitly inspect D15 allocation, candidate breadth, exact heading scope,
  aggregated one-line failures, symlink containment, provenance versus
  authentication wording, doctor/audit exit asymmetry, scoped ID resolution,
  zero-marker byte compatibility, generation-12 additive migration, knowledge
  profile behavior, and `supersededLines` debt. Surface proposed resolutions;
  never silently reinterpret the PRD.

- [ ] **Step 2: Resolve every review finding test-first.** For a code defect,
  add or tighten a failing regression test, record red, apply the minimum fix,
  and rerun focused plus full tests. For a real PRD contradiction, amend the
  PRD and plan, obtain approval, then update code. Commit review fixes with
  explicit paths and an I050 commit message.

- [ ] **Step 3: Run independent verification.** Dispatch a different fresh
  primary-tier verifier with I050 in the prompt and an explicit routed model
  and effort. The verifier reruns parser, doctor, audit, migration, full suite,
  focused and full Go tests, vet, `go build -o bin/spine ./cmd/spine`, diff
  check, compiled-CLI relative-root and hostile-reference probes, update dry
  run, `spine doctor`, `spine audit routing`, and `spine audit stages`, then
  runs `maipipe run full --wait`; checks raw exit codes and output; verifies
  all prior negative controls were observed red; and checks the
  staged/committed path set excludes concurrent files.

- [ ] **Step 4: Close I050 only after both gates approve.** Set `status: fixed`,
  add `commits: [...]` with the actual implementation, template, CHANGELOG, and
  review-fix SHAs, and append `## Resolution`. The resolution names D15,
  grammar, provenance boundary, audit nonblocking behavior, generation 12,
  tests, fresh spec review, and independent verification. Do not claim the
  final closure commit inside its own `commits` list.

- [ ] **Step 5: Commit ticket closure with an explicit path.** Run:

  ```bash
  git add docs/issues/I050-approved-untested-acceptance-lane.md
  git commit -m 'docs(I050): close approved-untested acceptance lane'
  ```

- [ ] **Step 6: Reverify the exact closure SHA.** Run:

  ```bash
  go test ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine update --dir .
  ./bin/spine doctor --dir .
  ./bin/spine audit routing --dir .
  ./bin/spine audit stages --dir .
  git diff --check
  git status --short
  maipipe run full --wait
  ```

  Expected: local and maipipe gates pass at the exact final SHA. The status
  output contains only known concurrent or ignored work, never unstaged I050
  changes.

- [ ] **Step 7: Create the fresh handoff and rerun spine audits.** Use
  `spine handoff new` so the handoff embeds the current cursor, never copy the
  cursor block by hand, and never tick `handoff`. Commit the generated handoff
  with an explicit I050 docs commit, then rerun `spine audit routing` and
  `spine audit stages`. After those audits pass, rerun
  `maipipe run full --wait` because the handoff commit changed HEAD. The audits
  and lane must all be recorded at the same final handoff SHA.

- [ ] **Step 8: Ship and deploy only with owner authority.** Follow the batch
  handoff's push procedure only after the exact-SHA lane is green. Then run
  `make install` and refresh the `~/.local/bin/spine` copy as directed by the
  owner. Record the shipped and installed SHAs in the final team report.

## Plan self-review

- [x] Every PRD goal, non-goal, grammar rule, scope rule, security check,
  command behavior, compatibility promise, migration rule, requirements-attack
  resolution, and acceptance criterion maps to a named task and test.
- [x] D15 is pinned from current source; no stale D14 reference remains as the
  I050 allocation.
- [x] Doctor and stages consume one shared parser, and the plan contains a
  mutation control proving invalid acceptance cannot enter `Blocking()`.
- [x] Zero-marker compatibility is tested at package, CLI, migration, and
  functional levels.
- [x] Template generation 12 covers fresh scaffolds, generation-11 update,
  local-edit refusal, idempotence, knowledge-profile behavior, current-version
  assertions, future-generation refusal, and additive-line accounting.
- [x] Every production task has an explicit red command, expected failure,
  minimum implementation step, green command, negative control, and commit.
- [x] Final work includes CHANGELOG, fresh spec review, separate independent
  verification, ticket closure, explicit commits, exact-SHA lane, handoff, and
  post-handoff audits.
- [x] Placeholder scan found no unfinished marker, unnamed test, vague error
  handling step, or unresolved interface.
- [x] Scope excludes production implementation from this PRD-authoring
  dispatch and excludes unrelated tickets, fleet rollout, authentication,
  fragment lookup, a new ADR, and existing-ticket rewrites.
