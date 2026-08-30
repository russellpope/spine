# I124 Scoped Update Force-File Authority Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add repeatable `spine update --force-file` authority that safely regenerates exact named managed plan members while preserving global force and whole-plan preflight atomicity.

**Architecture:** Parse repeated values into update options, normalize and validate them against the complete current report set, then make the existing local-edit policy consult either one global authority mode or an exact scoped set. Scoped reports follow the unchanged pending-candidate and atomic-write path, including maipipe preflight.

**Tech Stack:** Go standard library `flag`/`filepath`, existing update and CLI test harnesses, maipipe.

**Spec:** `docs/specs/2026-08-30-i124-update-force-file-authority-design.md`

## Global Constraints

- Ticket: `I124`; execution mode is subagent-driven, ticket tier is routine, and **every review is primary-tier or above** because `plan-flagged-ambiguity` sets the review floor.
- Preserve all standalone `--force` behavior. Mixed `--force`/`--force-file` is an exit-2 fail-closed error, never a union.
- Validate all values and the complete current plan before changing any report state or writing a file; preserve I096/I104 whole-plan candidate preflight and `fsutil.WriteFileAtomic` behavior.
- Observe negative controls red, stage only explicit paths, and preserve concurrent work plus the known research stray.
- Every commit cites I124. `maipipe run full --wait` must be green at the exact final SHA after every final-HEAD move.
- The round-2 owner ruling already authorizes implementation and closure. Stop only for a genuine spec contradiction or out-of-scope expansion, not a redundant owner-acceptance round trip.

---

## File map

| File | Responsibility |
|---|---|
| `internal/update/update.go` | Add scoped authority options, normalization/current-plan validation, mode enforcement, and report-level authorization state. |
| `internal/update/update_test.go` | Lock normalization, rejection, exact targeting, marker protection, preflight atomicity, and byte stability. |
| `cmd/spine/main.go` | Declare repeatable flags-first CLI grammar and print scoped-plan / skip wording. |
| `cmd/spine/main_test.go` | Lock repeated flag behavior, errors/exits, compatibility, and compiled CLI output. |
| `internal/update/gatepack_test.go` | Exercise a real managed `maipipe.toml` scoped candidate and its existing preflight. |
| `docs/issues/I124-update-force-file-scopes-overwrite-authority.md` | Record implementation evidence and closure after gates. |

### Task 1: make scoped authority a validated update-plan input

**Files:**

- Modify: `internal/update/update.go`
- Modify: `internal/update/update_test.go`

**Interfaces:**

- Produces: `Options.ForceFiles []string` (raw CLI values) and a private normalized exact-path authority set bound to this run's `FileReport.Path` values.
- Consumes: `Options.Dir`, existing planned reports, `filepath.Clean`, and the existing forceable-content guard.

- [ ] **Step 1: Write failing table-driven core tests.** Cover empty, absolute, raw traversal, normalized duplicate, unknown/unmanaged, profile-not-owned, no-planned-maipipe, selected clean member, exact selected unrecognized member, unselected sibling byte stability, and mixed `Force` plus `ForceFiles`. Assert each rejection returns before an injected writer is called and before a report state is changed.

- [ ] **Step 2: Run the focused core tests red.**

  Run: `go test ./internal/update -run 'Test.*(ForceFile|ScopedForce|Update.*Authority)' -count=1`

  Expected: FAIL because update has no scoped value validation or exact authorization set.

- [ ] **Step 3: Implement validation and policy only.** Reject a raw `..` component and absolute/empty values, clean safe relative paths, reject normalized duplicates, and compare only to this run's managed report path set. Reject global/scoped mixed mode before plan/write. Replace the local-edit policy's sole `opts.Force` check with an exact `opts.Force || selected[path]` decision while retaining non-regenerable marker-damage skips and global legacy-preserve semantics.

- [ ] **Step 4: Run core tests green.**

  Run: `go test ./internal/update -run 'Test.*(ForceFile|ScopedForce|Update.*Authority|LegacyADR)' -count=1`

  Expected: PASS; a scoped path affects only its report and standalone global force remains unchanged.

- [ ] **Step 5: Commit the policy unit.**

  Run: `git add internal/update/update.go internal/update/update_test.go && git commit -m 'feat(I124): scope update force authority by file'`

### Task 2: wire repeatable flags and audit-facing plan wording

**Files:**

- Modify: `cmd/spine/main.go`
- Modify: `cmd/spine/main_test.go`
- Modify: `internal/update/update.go`

**Interfaces:**

- Consumes: repeatable `--force-file <path>` values in flags-first `update` grammar and report authorization state.
- Produces: exact scoped authorization stdout line, exact safer unselected skip stderr line, and unchanged standalone global-force output.

- [ ] **Step 1: Write failing CLI tests.** Assert repeated flags before `--write` authorize exactly their named reports; `./maipipe.toml` normalizes; duplicate/unknown/absolute/traversal/missing-value/mixed-mode invocations exit 2 with no output file mutation; a post-positional token follows shared flags-first errors. Assert exact dry-run and write wording, stdout/stderr channels, unrelated `WORKFLOW.md` byte stability, and existing `--force` snapshots unchanged.

- [ ] **Step 2: Run the focused CLI tests red.**

  Run: `go test ./cmd/spine -run 'Test.*(Update.*ForceFile|ForceFile.*Update|Update.*Force)' -count=1`

  Expected: FAIL because the CLI does not register a repeatable scoped flag or emit its plan wording.

- [ ] **Step 3: Implement strict CLI wiring.** Use a repeatable `flag.Value`/slice collector without custom positional parsing, update the usage string, pass raw values to `update.Options`, and format only report-level scoped authorization. Keep parser ownership and the existing standalone `--force` skip/message behavior intact.

- [ ] **Step 4: Run focused CLI coverage green.**

  Run: `go test ./cmd/spine ./internal/update -run 'Test.*(Update.*ForceFile|ForceFile.*Update|ScopedForce|Update.*Force)' -count=1`

  Expected: PASS with deterministic messages and no write on every bad input.

- [ ] **Step 5: Commit the CLI unit.**

  Run: `git add cmd/spine/main.go cmd/spine/main_test.go internal/update/update.go && git commit -m 'feat(I124): add repeatable update force-file flag'`

### Task 3: prove scoped maipipe candidate preflight and atomicity

**Files:**

- Modify: `internal/update/gatepack_test.go`
- Modify: `internal/update/update_test.go`
- Modify: `cmd/spine/main_test.go`

**Interfaces:**

- Consumes: a tampered forceable `maipipe.toml`, scoped selection, and an invalid resulting maipipe candidate.
- Produces: a plan/refusal where zero files are written, including `WORKFLOW.md` and unrelated managed files.

- [ ] **Step 1: Write failing integration tests.** Start from `gateRepo`, tamper the managed region, add an unrelated local WORKFLOW edit, select only `maipipe.toml`, and make the candidate invalid under the existing maipipe test seam. Assert dry-run has the authority note and refusal; `--write` exits 2; every before/after byte slice is identical. Add a positive selected-maipipe case that regenerates only maipipe while WORKFLOW remains byte-identical.

- [ ] **Step 2: Run focused preflight tests red.**

  Run: `go test ./internal/update ./cmd/spine -run 'Test.*(ForceFile.*Maipipe|Scoped.*Preflight|ForceFile.*Atomic)' -count=1`

  Expected: FAIL until scoped authorization enters candidate preflight before the atomic write loop.

- [ ] **Step 3: Implement the smallest integration correction.** Ensure policy state is finalized before the existing pending maipipe-preflight scan. Do not create a force-file-only writer, bypass, or second validation path.

- [ ] **Step 4: Run focused preflight tests green.**

  Run: `go test ./internal/update ./cmd/spine -run 'Test.*(ForceFile.*Maipipe|Scoped.*Preflight|ForceFile.*Atomic|DryRun.*Refusal)' -count=1`

  Expected: PASS; selected bad candidates produce current whole-plan no-write semantics.

- [ ] **Step 5: Commit the atomicity unit.**

  Run: `git add internal/update/gatepack_test.go internal/update/update_test.go cmd/spine/main_test.go && git commit -m 'test(I124): lock scoped-force preflight atomicity'`

### Task 4: primary review, requirements attack, verification, and closure

**Files:**

- Modify: `docs/issues/I124-update-force-file-scopes-overwrite-authority.md`
- Modify: `docs/specs/2026-08-30-i124-update-force-file-authority-design.md`
- Modify: `docs/specs/2026-08-30-i124-update-force-file-authority-plan.md`

- [ ] **Step 1: Run a primary-tier requirements attack before spec review.** A fresh primary-tier reviewer with `I124` in the prompt attacks flags-first parsing, normalized duplicates, absolute/traversal bypasses, current-plan membership, clean selections, unrecognized sibling stability, global compatibility, mixed-mode ambiguity, marker damage, authorization wording, candidate preflight ordering, and all-or-nothing writes. Record each proposed resolution before accepting the diff.

- [ ] **Step 2: Run full, race, static, and CLI-functional evidence.**

  Run:

  ```bash
  go test ./... -count=1
  go test -race ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine update --dir <fixture> --force-file maipipe.toml --write
  ./bin/spine update --dir <fixture> --force-file ./maipipe.toml --force-file maipipe.toml
  ./bin/spine update --dir <fixture> --force --force-file maipipe.toml
  git diff --check
  ```

  Expected: full/race/static checks pass; the positive compiled-CLI probe touches only selected managed content; duplicate and mixed negative controls exit 2 before writes; global force behavior remains compatible.

- [ ] **Step 3: Perform mandatory primary-tier fresh spec review.** The verifier compares final code/tests to the design, reruns the requirements attack, verifies raw command exits/output and before/after bytes, confirms all negative controls were observed red, and refuses closure if any authority can bypass current-plan membership, preflight, or marker protection.

- [ ] **Step 4: Commit closure evidence after required gates.** The round-2 owner ruling already authorizes implementation and closure. After the specified primary review and independent verification pass, set I124 to fixed, add actual commit IDs and primary-review evidence, and commit explicit docs paths in `docs(I124): record scoped force verification`. Stop only for a genuine contradiction or out-of-scope expansion; do not add a redundant owner-acceptance round trip.

- [ ] **Step 5: Verify the final exact SHA.** At the exact final commit SHA, run `spine audit routing --dir .`, `spine audit stages --dir .`, `maipipe run full --wait`, and `git status --short`; rerun the maipipe lane after any docs/closure commit so the reported green result is for the final SHA.

## Plan self-review

- [ ] Every acceptance criterion has both a focused test and an independent functional/verification probe.
- [ ] The sole force modes are global or exact scoped; mixed flags are fail-closed.
- [ ] Candidate preflight and write behavior are shared with existing update, not reimplemented.
- [ ] The planned primary review floor, requirements attack, full/race/static/CLI evidence, and exact-SHA maipipe requirement are explicit.
