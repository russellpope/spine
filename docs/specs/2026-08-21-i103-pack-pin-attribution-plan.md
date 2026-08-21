# I103 pack pin attribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the pack pin reach every rendered stage on its run line and
have `spine gate` honour it, so a `go@1` repo's findings are coded
`go@1/<check>` from any spine binary; surface the region rewrite as changed
stages in the plan.

**Architecture:** The gate package resolves a pin to its attribution id and
frozen class list and codes findings from the resolved pin. The renderer
writes the pin into each stage's run line. The region reader accepts both
run-line forms (pinned form only for the repo's own pin). The plan's stage
delta gains a changed list.

**Tech Stack:** Go standard library, spine CLI, maipipe executable.

**Spec:** `docs/specs/2026-08-21-i103-pack-pin-attribution-design.md`
**Decision:** `docs/adr/0019-the-pack-pin-rides-the-stage-run-line-not-an-env-var.md`

## Global Constraints

- I103 is routine-tier, subagent-driven; all commits cite I103.
- Zero third-party dependencies (ADR 0001). `spine gate` never reads
  `WORKFLOW.md`.
- Tests drive `update.Run`, the gate package's public surface, and the CLI
  run helpers with real temp repos and real bytes; no source-text assertions.
- Stage explicit paths only. `spine cursor` is the only cursor writer.
- Commit `maipipe.toml` (spine's own region is rewritten) with the code that
  rewrites it, before running any maipipe lane.

### Task 1: gate — pin resolution and attribution

**Files:**
- Modify: `internal/gate/gate.go`, `internal/gate/gate_test.go`
- Modify: `cmd/spine/main.go` (gate argument parsing, usage text),
  `cmd/spine/main_test.go`

**Interfaces:**
- Produces: a public resolve of `<pack>[@<v>]` → (attribution id, class
  list, shipped bool); the finding-code helper keyed by the resolved pin.
- Consumes: the existing per-version class table.

- [ ] **Step 1: Failing gate-package tests.** Add a stub version 2 to the
  class table for the test's lifetime. Assert: resolving `go@1` on that
  binary codes `go@1/<check>`; a class present only in version 2 is refused
  under a `go@1` pin; `go@9` is not shipped; bare `go` resolves to the
  binary's own pack.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Failing CLI tests.** `spine gate go@1 tskip` on a fixture
  → finding code `go@1/tskip`; `spine gate go@9 tskip` → exit 2, stderr names
  the pin, no results document; `spine gate go tskip` unchanged.
- [ ] **Step 4: Verify red.**
- [ ] **Step 5: Implement the minimum.** Parse the versioned pack argument,
  resolve it, refuse as misconfiguration, thread the resolved pin to the
  code helper. Update `gateUsage` to show `<pack>[@<v>]`.
- [ ] **Step 6: Verify green**, then gofmt.
- [ ] **Step 7: Negative control.** Temporarily drop the pin's class-list
  check; the out-of-pin test must fail. Restore.

### Task 2: update — pinned render and reader

**Files:**
- Modify: `internal/update/gatepack.go`, `internal/update/gatepin_test.go`
  (and/or `gatepack_test.go`)
- Modify: `maipipe.toml` (spine's own region, via `spine update --write`)

**Interfaces:**
- Consumes: the `packClassesFor` seam; the repo's `gate_pack` value.
- Produces: run lines `spine gate <pin> <check>` for every stage including
  mutate; reader recognising bare form (any shipped class) and pinned form
  (repo's pin only).

- [ ] **Step 1: Failing render tests.** Through `update.Run` on a temp repo
  pinned at `go@1`: every rendered stage, including mutate, carries
  `spine gate go@1 <check>`.
- [ ] **Step 2: Failing reader tests.** A region in the bare form is reported
  stale, not unrecognized; a region line `spine gate go@2 tskip` in a `go@1`
  repo is unrecognized region content; the pinned form with the repo's own
  pin is recognised.
- [ ] **Step 3: Verify red.**
- [ ] **Step 4: Implement the minimum.** Render the pin; extend the reader
  to accept both forms per the spec.
- [ ] **Step 5: Verify green**, gofmt.
- [ ] **Step 6: Negative controls.** Drop the pinned-form recognition → the
  migration test must report unrecognized. Drop the pin-equality check → the
  `go@2`-in-`go@1` test must fail. Restore both.
- [ ] **Step 7: Rewrite spine's own region.** `spine update --write`; confirm
  the plan shows only run-line changes; commit `maipipe.toml` with the code.

### Task 3: update — changed-stages notice

**Files:**
- Modify: `internal/update/gatepack.go` (stage delta), `internal/update/update.go`
  (report), `cmd/spine/main.go` (plan output), tests alongside.

- [ ] **Step 1: Failing test.** An existing bare-form region vs the pinned
  render: the report lists every stage as changed, none added/removed; the
  plan output prints "this render changes N stage(s) not added or removed:
  …" and names re-approval. A render that only adds a stage lists nothing as
  changed.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum** — a third list next to
  added/removed, computed only where a region already exists.
- [ ] **Step 4: Verify green**, gofmt.
- [ ] **Step 5: Negative control.** Drop the changed delta; the notice test
  must fail. Restore.

### Task 4: records

**Files:**
- Modify: `docs/specs/2026-08-18-local-harness-conventions-design.md`
  (story 23), `docs/issues/I098-*.md` (dated note under Resolution),
  `docs/issues/I103-pack-attribution-is-not-pinned.md` (Resolution,
  `status: fixed`).

- [ ] **Step 1:** Story 23: the pin is a frozen class list **and** the
  attribution string; name ADR 0019.
- [ ] **Step 2:** I098 Resolution: dated note that I103 closed the other
  half. I103: Resolution with what landed, the negative controls run, and
  the fleet re-approval note.

### Task 5: verification

- [ ] gofmt on every modified Go file; `go vet ./...`.
- [ ] `SPINE_REQUIRE_MAIPIPE=1 make test` — all packages green.
- [ ] `spine update` dry-run at HEAD shows no pending changes for spine's
  own region.
- [ ] `maipipe run full --wait` at the final commit — passes with the pinned
  run lines executing under real maipipe.
- [ ] Paste the commands and the relevant output in the team report.
