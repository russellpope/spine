# I123 Update Gate-Configuration Advisory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `spine update` give a deterministic pre-write advisory for enabled gate classes missing required configuration, without changing render, write, or exit behavior.

**Architecture:** Keep requiredness in the update gate-pack metadata, derive sorted advisory values from the pinned pack settings during the existing plan pass, and expose a narrow pre-write emission seam so CLI text is emitted before atomic writes. The existing `FileReport`, maipipe candidate preflight, and whole-plan refusal remain the source of update state and safety.

**Tech Stack:** Go standard library, current `internal/update` test fixtures, CLI test harness, maipipe.

**Spec:** `docs/specs/2026-08-30-i123-update-gate-config-advisory-design.md`

## Global Constraints

- Ticket: `I123`; execution mode is subagent-driven and every dispatch carries `I123` plus an explicit routed model tier.
- Preserve ADR 0015's explicit-disable-only rendering and I096/I104's candidate preflight / no-partial-write behavior.
- The advisory line, class order, stdout channel, and exit behavior are exactly those in the design; observe every negative control red before implementation.
- Stage only explicit paths. Do not modify concurrent work or the known research stray.
- Every implementation commit cites I123. The final maipipe lane must pass at the exact final SHA.

---

## File map

| File | Responsibility |
|---|---|
| `internal/update/gatepack.go` | Declare required gate-input metadata and derive deterministic advisory records from pinned settings. |
| `internal/update/update.go` | Carry advisories through the plan and invoke the pre-write seam only after candidate preflight and before writes. |
| `internal/update/gatepack_test.go` | Unit-test class/key selection, exclusions, ordering, and single-key/disable removal. |
| `internal/update/update_test.go` | Prove the pre-write seam runs before the atomic writer and refusal remains no-write. |
| `cmd/spine/main.go` | Format advisory lines on stdout in dry-run and write modes without altering exits. |
| `cmd/spine/main_test.go` | Lock exact CLI output, exits, dry-run stability, and configured compatibility. |

### Task 1: define and test required-configuration advisory planning

**Files:**

- Modify: `internal/update/gatepack.go`
- Modify: `internal/update/gatepack_test.go`

**Interfaces:**

- Produces: a value advisory containing `Class` and `Key`, derived from a shipped pin's frozen classes and `gatePackSettings`.
- Consumes: `gateSettings`, `packClassesFor`, `gate_pack_disabled`, and `gate_pack_config` exactly once per plan.

- [ ] **Step 1: Write failing table-driven unit cases.** Cover all-empty `go@1` (the four exact class/key pairs in bytewise class order), each individually configured key, each individually disabled class, empty `tskip_allow`, each config-free class, unknown pin, and a duplicate class/key source negative control. Assert no rendered region bytes or stage list change merely from advisory calculation.

- [ ] **Step 2: Run the focused tests red.**

  Run: `go test ./internal/update -run 'Test.*(Gate.*Config.*Advis|Advis.*Gate.*Config)' -count=1`

  Expected: FAIL because no requiredness metadata/advisory derivation exists.

- [ ] **Step 3: Implement minimal declarative derivation.** Keep optional environment keys separate from required keys; filter disabled classes, sort records by class, and never manufacture a disable or alter `renderGateRegion`.

- [ ] **Step 4: Run focused tests green and regression coverage.**

  Run: `go test ./internal/update -run 'Test.*(Gate.*Config.*Advis|GatePack|Render)' -count=1`

  Expected: PASS, including the empty-`tskip_allow` and config-free exclusions.

- [ ] **Step 5: Commit the planning unit.**

  Run: `git add internal/update/gatepack.go internal/update/gatepack_test.go && git commit -m 'feat(I123): advise on missing required gate config'`

### Task 2: emit the plan result before writes and bind CLI behavior

**Files:**

- Modify: `internal/update/update.go`
- Modify: `internal/update/update_test.go`
- Modify: `cmd/spine/main.go`
- Modify: `cmd/spine/main_test.go`

**Interfaces:**

- Consumes: sorted I123 advisory records and the existing candidate preflight verdicts.
- Produces: the exact stdout advisory line and an injection-tested pre-write callback/seam; no new exit state.

- [ ] **Step 1: Write failing update and CLI tests.** Use a writer recorder or injected atomic-write boundary to prove the callback sees all four advice lines before any write; make maipipe preflight refuse and prove advice still appears with zero changed bytes. At the CLI seam assert the exact text, stdout-not-stderr, deterministic order, dry-run exit 1 when a diff exists, advisory-only exit 0, successful write exit 0, and byte-identical configured behavior with no advisory.

- [ ] **Step 2: Run the focused tests red.**

  Run: `go test ./internal/update ./cmd/spine -run 'Test.*(PreWrite.*Advis|Update.*Gate.*Advis|Gate.*Config.*Advis)' -count=1`

  Expected: FAIL because update cannot emit a plan advisory before `WriteFileAtomic`.

- [ ] **Step 3: Implement the narrow plan-reporting seam.** Collect advisories in the same plan pass, run current maipipe candidate preflight first, invoke the seam before the refusal/write loop, and let the CLI use it only for `--write`; dry-run prints the same formatter before report diffs. Do not change `Pending`, `SkippedPreflight`, refusal text, or outstanding-count logic.

- [ ] **Step 4: Run focused tests green.**

  Run: `go test ./internal/update ./cmd/spine -run 'Test.*(PreWrite.*Advis|Update.*Gate.*Advis|Gate.*Config.*Advis|DryRun.*Refusal)' -count=1`

  Expected: PASS, with no duplicate line in write mode.

- [ ] **Step 5: Commit the emission unit.**

  Run: `git add internal/update/update.go internal/update/update_test.go cmd/spine/main.go cmd/spine/main_test.go && git commit -m 'feat(I123): emit gate config advice before update writes'`

### Task 3: requirements attack, independent verification, and closure

**Files:**

- Modify: `docs/issues/I123-update-advises-on-unconfigured-gate-classes.md`
- Modify: `docs/specs/2026-08-30-i123-update-gate-config-advisory-design.md`
- Modify: `docs/specs/2026-08-30-i123-update-gate-config-advisory-plan.md`

- [ ] **Step 1: Run the requirements attack before review.** A fresh routine-or-higher reviewer with I123 in the prompt attacks: optional-versus-required `tskip`, disabled precedence, pin freezing, byte ordering, stdout/stderr, advice-before-write, refusal-before-write, one-key/one-disable isolation, no implicit disable, and unchanged configured plans. Record proposed resolutions; do not silently weaken a requirement.

- [ ] **Step 2: Run full, race, static, and CLI-functional evidence.**

  Run:

  ```bash
  go test ./... -count=1
  go test -race ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine update --dir <fresh-configured-fixture>
  ./bin/spine update --dir <fresh-empty-required-config-fixture> --write
  git diff --check
  ```

  Expected: full/race/static checks pass; functional probes show the exact advisory only in the missing-required-config fixture, no advisory for config-free/empty-tskip fixtures, and no write before advice.

- [ ] **Step 3: Perform mandatory fresh spec review.** A fresh verifier reads the final diff against the design, reruns the attack, all focused/full/race/static/CLI probes, and explicitly confirms preflight refusal leaves every planned file byte-identical.

- [ ] **Step 4: Commit closure evidence after the required gates.** The round-2 owner ruling already authorizes implementation and closure. After the specified review and independent verification pass, set I123 to fixed, add exact commit/evidence references, and commit explicit issue/spec paths with `docs(I123): record gate config advisory verification`. Stop only for a genuine spec contradiction or out-of-scope expansion; do not add a redundant owner-acceptance round trip.

- [ ] **Step 5: Verify the final exact SHA.** At the final commit SHA run `spine audit routing --dir .`, `spine audit stages --dir .`, and `maipipe run full --wait`; record raw exit/output and rerun the maipipe lane after any final docs commit.

## Plan self-review

- [ ] Every required class/key, class order, remediation text, timing, output channel, exit outcome, and exclusion maps to a test/task.
- [ ] No task changes the renderer to disable or omit a class implicitly.
- [ ] All commits use explicit paths and preserve unrelated work.
