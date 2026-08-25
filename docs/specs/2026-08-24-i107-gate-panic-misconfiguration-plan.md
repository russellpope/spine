# I107 gate panic becomes a misconfiguration exit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert a panic inside a gate check into `gate.Run`'s documented
misconfiguration exit, with an actionable message when the cause is a stale
spine binary and a verbatim panic-and-stack when it is anything else — without
changing how a module that genuinely fails to type-check is reported.

**Architecture:** `loadModule` gains a recover that classifies the recovered
value and returns an error (D38–D42). `gate.Run` gains a blanket recover of the
same error-returning shape around every check invocation (D43). Both feed the
existing `runErr != nil` branch, which already prints to stderr, returns 2, and
never reaches `emit`.

**Tech Stack:** Go standard library only (ADR 0001).

**Spec:** `docs/specs/2026-08-24-i107-gate-panic-misconfiguration-design.md`
**Decision:** `docs/adr/0021-a-gate-panic-becomes-a-contractual-misconfiguration-exit-toolchain-version-comparison-is-rejected-as-a-proxy.md`

## Global Constraints

- I107 is routine-tier, subagent-driven, reviewed at primary tier; all commits
  cite I107.
- Zero third-party dependencies.
- **No recover may produce exit 0, exit 1, findings, or silence.** Every
  recovered panic prints and exits 2 (D44).
- No version comparison anywhere in the gate. Detection keys on the importer
  actually failing (ADR 0021).
- The existing `--dir %s does not type-check` path is untouched — same message,
  same exit code, for a module that genuinely fails to compile.
- Every task's negative control must be *observed failing*, with the command and
  output recorded. A guard asserted to be load-bearing without a red run does
  not count.
- Stage explicit paths only. `spine cursor` is the only cursor writer.

### Task 1: recover and classify at the loader seam

**Files:**
- Modify: `internal/gate/load.go`
- Create: `internal/gate/load_panic_test.go`

**Interfaces:**
- Produces: `loadModule` returns an error for any panic beneath it, in two
  classes (D39): export-data mismatch (D40) and internal error (D41).
- Consumes: an unexported seam permitting a test to inject a panicking
  importer. Keep it minimal — a package-level variable or an unexported
  parameter, not a new exported API.

- [ ] **Step 1: Failing tests.** Injected panic carrying `export data version 4
  is greater than maximum supported version 2` → error naming that text,
  `runtime.Version()`, and `make install`, and **not** containing `does not
  type-check` (D42). Injected panic with unrelated text → error carrying the
  value and a stack, and **not** suggesting a rebuild. Both assert
  `loadModule` returns rather than panicking.
- [ ] **Step 2: Verify red.** Record the command and its output.
- [ ] **Step 3: Implement the minimum.** Recover covering the whole
  `loadModule` body including the `goList` call; classify on the `export data
  version` substring only (D39); build both messages.
- [ ] **Step 4: Verify green**, then `gofmt`.
- [ ] **Step 5: Negative control.** Delete the recover; the mismatch test must
  panic and fail the run. Restore and re-verify green.

### Task 2: name the toolchain on PATH, degrading cleanly

**Files:**
- Modify: `internal/gate/load.go`
- Modify: `internal/gate/load_panic_test.go`

**Interfaces:**
- Produces: the mismatch message's PATH clause, sourced from `go env GOVERSION`
  (D40).
- Consumes: nothing new — `go` is provably on PATH because `goList` already ran
  and succeeded before any type-checking could begin.

- [ ] **Step 1: Failing tests.** With the probe succeeding, the message names
  both toolchains. With the probe failing, the message omits the PATH clause and
  still names the panic text, `runtime.Version()`, and the remedy — and does not
  report the probe's own failure as the error.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum.**
- [ ] **Step 4: Verify green**, then `gofmt`.
- [ ] **Step 5: Negative control.** Force the probe to error unconditionally;
  the degrade test must pass and the both-toolchains test must fail. Restore.

### Task 3: blanket recover at the Run boundary

**Files:**
- Modify: `internal/gate/gate.go`
- Create: `internal/gate/run_panic_test.go`

**Interfaces:**
- Produces: a panic in any check class, plain or rich, becomes `runErr` and
  takes the existing stderr-and-return-2 branch (D43).
- Consumes: D41's internal-error message shape. `Run` does not classify.

- [ ] **Step 1: Failing tests.** A check class registered for the test that
  panics, driven through `Run` for both the `plain` and the rich `reportChecks`
  paths: exit 2, a message on stderr carrying the panic value and a stack, no
  crash.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum.** Wrap both `fn(abs, cfg)` and
  `rfn(abs, cfg)`.
- [ ] **Step 4: Verify green**, then `gofmt`.
- [ ] **Step 5: Negative control.** Delete the wrap; both tests must crash the
  test binary. Restore.

### Task 4: prove the results file is not written

**Files:**
- Modify: `internal/gate/run_panic_test.go`

**Interfaces:**
- Consumes: `ResultsEnvVar` / `$SPINE_GATE_RESULTS` and the `emit` path at
  `internal/gate/results.go:53`.
- Produces: an explicit assertion of a guarantee that currently holds only as a
  consequence of statement order in `Run`.

- [ ] **Step 1: Failing test.** Set `$SPINE_GATE_RESULTS` to a path inside a
  temp dir, trigger a panic through `Run`, assert exit 2 **and** that no file
  exists at that path.
- [ ] **Step 2: Verify red** against a build where Task 3 is reverted (the crash
  is the red).
- [ ] **Step 3:** No implementation — Task 3 already satisfies it. This task
  exists so a later reordering of `emit` above the error check is caught.
- [ ] **Step 4: Verify green**, then `gofmt`.
- [ ] **Step 5: Negative control.** Temporarily move the `emit` call above the
  `runErr` check; this test must fail. Restore.

### Task 5: constraint control — genuine type-check failures unchanged

**Files:**
- Modify: `internal/gate/load_panic_test.go` or the nearest existing loader test

**Interfaces:**
- Consumes: the existing `--dir %s does not type-check` error path.
- Produces: a regression guard that this work did not alter it.

- [ ] **Step 1: Failing-or-passing test.** A fixture module that genuinely does
  not compile, run through both affected classes: today's exact message
  phrasing and exit code. Capture the message from a build of `main` at
  `ff22a55` first, so the assertion is a record of prior behavior rather than
  of the new code's behavior.
- [ ] **Step 2: Verify it passes** on the branch with Tasks 1–4 applied. If it
  does not, the recover has captured a path it must not.
- [ ] **Step 3:** No implementation expected.
- [ ] **Step 4: Verify green**, then `gofmt`.
- [ ] **Step 5: Negative control.** Widen D39's classifier to match everything;
  this test must fail, proving the classifier is what keeps the two apart.
  Restore.

### Task 6: optional end-to-end confirmation of the classifier substring

**Files:**
- Create: `internal/gate/testdata/` export-data blob, if it proves cheap

**Interfaces:**
- Consumes: `go/importer` with the `gc` compiler over a synthetic blob.
- Produces: confirmation that D39's `export data version` substring matches what
  `gcimporter` actually raises, rather than what the 2026-08-24 transcript
  recorded.

- [ ] **Step 1:** Build a minimal export-data file with its version byte bumped
  past what the current toolchain supports; feed it through the importer and
  capture the real panic text.
- [ ] **Step 2:** If the substring matches, keep the test. **If the blob proves
  fiddly or brittle, delete this task and say so in the completion report** —
  the injected seam in Task 1 is the shipped test, and this is confirmation
  only. Do not spend a long tail here.
- [ ] **Step 3:** Do **not** install a second Go toolchain
  (`golang.org/dl/go1.26.7`) for this. Explicitly out of scope.

### Task 7: verification and evidence

- [ ] `gofmt -l .` clean; `go vet ./...` clean.
- [ ] `SPINE_REQUIRE_MAIPIPE=1 make test` — full lane, all packages, not just
  `internal/gate`. Paste the summary.
- [ ] `make install`, then confirm `spine gate go@1 dead-code-callgraph --dir .`
  and `spine gate go@1 deferred-cleanup-errcheck --dir .` both still exit 0 on a
  healthy tree. **Read exit codes unpiped** — `cmd >/dev/null 2>&1; echo $status`
  in fish reports the pipeline's last command, not the gate's.
- [ ] Completion report names, per task, the exact negative-control command and
  the observed failure output. A task whose negative control was not run red is
  not done.
- [ ] Append `I107: gate panic misconfiguration done` to
  `.superpowers/sdd/progress.md`.
