# I104/I097 maipipe preflight Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove spine's TOML structural scanner in I104 while safely requiring
`maipipe validate` before `maipipe.toml` can be planned or written, then safely
remove an opted-out gate pack in I097 on a stacked branch. This worktree executes
I104 only.

**Architecture:** The existing update planning path decides whether to emit a
normal maipipe file operation, a validated operation, or a skip. The skip is a
file-level plan outcome, so unrelated operations can still apply. Validation
continues to run against the rendered candidate before its write.

**Tech Stack:** Go standard library, repository `spine` CLI, maipipe executable

**Spec:** `docs/specs/2026-08-20-i104-i097-maipipe-preflight-design.md`

## Global Constraints

- I104 is routine-tier, subagent-driven; all changes cite I104.
- Preserve zero third-party dependencies (ADR 0001).
- Do not implement, resolve, or otherwise change I097 in this I104 worktree;
  execute Task 4 only on the stacked branch after I104 review passes.
- Use `spine cursor` as the sole cursor writer and stage explicit paths only.
- Tests exercise `update.Run` with real temporary repositories and real file
  bytes; do not test source-text deletion.

### Task 1: I104 — establish planning and validation behavior

**Files:**
- Modify: `internal/update/update_test.go` and/or the existing update fixtures
- Modify: `internal/update/update.go`, `internal/update/maipipecheck.go`
- Delete: `internal/update/maipipecheck_test.go`

**Interfaces:**
- Consumes: the plan/report and `maipipeOnPath` update interfaces.
- Produces: a file-level skip for `maipipe.toml`, or a `maipipe validate`
  preflight refusal when the executable is present.

- [ ] **Step 1: Write a failing no-binary behavior test.** Create a temporary
  repository with a non-empty `gate_pack`, a sentinel `maipipe.toml`, and an
  unrelated planned file. Run `update.Run` with `PATH` unable to resolve
  maipipe; assert exit success, the plan names `maipipe.toml`, says it is
  skipped for missing maipipe and identifies the skip preflight, the unrelated
  file changes, and the sentinel bytes do not.
- [ ] **Step 2: Verify red.** Run the single Go test. Expected failure: current
  behavior either plans/writes the sentinel through its structural scanner or
  lacks the required skip/preflight report.
- [ ] **Step 3: Write a failing validate-only control.** Assert that, with a
  fake resolvable `maipipe` rejecting the rendered candidate, update refuses
  before writes and reports the maipipe validation verdict.
- [ ] **Step 4: Verify red.** Run the scoped test. Expected failure: the new
  no-binary/validate-only report contract is absent or structural scanning
  controls the result.
- [ ] **Step 5: Implement the minimum.** Keep `maipipeOnPath`; represent the
  no-binary case as a planned skip, validate candidates only through maipipe,
  and remove the structural scanner paths and scanner-only tests.
- [ ] **Step 6: Verify green.** Run the scoped update tests and the
  `internal/update` package. Expected: all pass, including existing invalid
  candidate controls adapted only where their observable contract changed.
- [ ] **Step 7: Refactor and format.** Keep the report wording at the plan
  boundary, run gofmt on modified Go files, and rerun the scoped tests.

### Task 2: I104 — durable decision and ticket record

**Files:**
- Create: `docs/adr/0018-*.md` via `spine adr new`
- Modify: `docs/issues/I104-should-the-hand-rolled-toml-scanner-exist.md`
- Modify: `docs/issues/I096-spine-update-parses-before-it-writes-maipipe-toml.md`

- [ ] **Step 1: Create the ADR.** Run `spine adr new --dir . "maipipe on PATH
  is a precondition for touching maipipe.toml when gate_pack is set"`; record
  option B, ADR 0001's unchanged stdlib policy, I104, skip behavior, and
  maipipe-only validation.
- [ ] **Step 2: Update ticket records.** Mark I104 fixed, populate
  subagent-driven/routine/routine metadata consistently, add its resolution
  and ADR reference, and add a dated I096 note that its structural half is
  gone. Do not modify I097.
- [ ] **Step 3: Verify documentation.** Check only intended I104/I096/ADR
  facts changed and the generated ADR number/title are canonical.

### Task 3: verification and review

**Files:**
- Create: `.superpowers/sdd/task-i104-worker-report.md`

- [ ] **Step 1: Run repository verification.** Run `gofmt -l .`, `go vet
  ./...`, and `SPINE_REQUIRE_MAIPIPE=1 make test` using task-specific writable
  Go caches if required. Record commands, exits, and red/green evidence.
- [ ] **Step 2: Run fresh-context spec review.** Compare the final diff to the
  binding design document. Resolve all I104 gaps; I097 files/code must remain
  absent from the diff.
- [ ] **Step 3: Audit and commit.** Run `spine audit routing`, write the
  required report with branch HEAD, stage explicit paths, and commit the I104
  branch without pushing or merging.

### Task 4: I097 — opt out only when the pack has no repo-owned consumers

**Execution boundary:** This is an approved sequential task for the stacked
I097 branch after I104 review. Do not execute it in the I104 worktree.

**Files:**
- Modify: `internal/update/gatepack.go`, `internal/update/gatepack_test.go`
- Modify: `internal/doctor/doctor.go`, `internal/doctor/doctor_test.go`
- Modify: `docs/issues/I097-gate-pack-opt-out-leaves-the-region-running.md`

**Interfaces:**
- Consumes: an existing `maipipe.toml` managed region and `gate_pack` setting.
- Produces: a refusal naming every outside `gate-go`/`mutation-go` composition,
  a marker-inclusive deletion plan where none exist, and a doctor finding for
  an unknown pack with a stale region.

- [ ] **Step 1: Write failing composition-refusal tests.** Build a controlled
  spine-layout fixture with `full/gates` composing `gate-go`, and a fixture for
  `mutation-go`. Assert that clearing `gate_pack` names each pipeline/stage,
  leaves file bytes unchanged, and reports no deletion diff.
- [ ] **Step 2: Verify red.** Run the scoped update tests. Expected failure:
  current opt-out silently leaves the region in place and reports neither the
  repo-owned composition nor a refusal.
- [ ] **Step 3: Write failing safe-deletion and no-op tests.** In a fixture with
  no external composition, assert that the plan deletes the complete marked
  region, `--write` deletes it, the rerun is up-to-date, and `maipipe validate`
  accepts the result. Include the load-bearing mutation: without the refusal,
  the spine-layout fixture must reproduce `composes unknown pipeline "gate-go"`.
- [ ] **Step 4: Verify red.** Run the scoped tests. Expected failure: removal
  is not planned and the stale region persists.
- [ ] **Step 5: Write the doctor failing test.** Configure an unknown pack
  against a file carrying an existing managed region and assert doctor reports
  the stale region rather than returning no gate-pack finding.
- [ ] **Step 6: Implement the minimum.** Scan only outside the markers for
  composition stages, refuse with all pipeline/stage names before deletion,
  otherwise render the marker-inclusive removal, and surface the unknown-pack
  stale-region doctor finding.
- [ ] **Step 7: Verify green and document.** Run update and doctor package
  tests, update I097's resolution only after review, then run full formatting,
  vet, maipipe-required tests, independent review, and routing audit.
