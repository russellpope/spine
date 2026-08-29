# Ergonomics + portability batch (I116 + I117 + I118) Implementation Plan

> **For agentic workers:** solo inline execution in the owning session — TDD
> per task, stage gates as normal. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Make the `spine model` flag-order failure name its rule (I116),
split the implement-tick zero-evidence message so ledger-wording misses stop
masquerading as tickets-value typos (I117), and give spine a documented
`go install` path plus build provenance in `spine version` (I118).

**Architecture:** Three serial tasks, one session (execution-mode inline,
tier primary, review-tier n/a). I116 and I118 live in `cmd/spine/main.go`
(+ README for I118); I117 in `internal/stages`. Branch off main, ff-merge at
ship. Never claude-sonnet-5.

**Tech Stack:** Go standard library only (ADR 0001);
`runtime/debug.ReadBuildInfo` for I118.

**Spec:** `docs/specs/2026-08-28-ergonomics-portability-batch-design.md`

## Global Constraints

- All commits cite their ticket id; batch commits so one
  `maipipe run full --wait` covers each HEAD move.
- Every negative control **observed red** (command + output recorded).
- `spine cursor` is the only cursor writer; never write the literal cursor
  marker in prose; never tick the handoff stage.
- Stage explicit paths only. Read exit codes unpiped under fish.
- The leading-flag `spine model` form and the first line of `spine version`
  output are behavior contracts — unchanged.

### Task 1: I116 — flag-order error names the rule

**Files:**
- Modify: `cmd/spine/main.go` (`cmdModel` + new helper)
- Modify: `cmd/spine` tests

**Interfaces:**
- Produces: a helper that, given `fs.Args()` after a successful parse,
  returns the first flag-like token (leading `-`) among the positionals, or
  "". `cmdModel` calls it before arity validation: a hit prints
  `model: flags must precede positionals (saw %q after %q)` (offending
  token, preceding positional) + the usage line, exit 2.

- [ ] **Step 1: Failing tests**: `model X primary --effort` ⇒ exit 2,
  stderr names the rule and `--effort`; `model X --json` (arity 2) ⇒ same
  shape; leading-flag form still resolves (green control); unknown flavor
  without any flag-like token still yields `model.Resolve`'s error, not the
  ordering message (negative control on detection).
- [ ] **Step 2: Verify red.** Record command + output.
- [ ] **Step 3: Implement the minimum.**
- [ ] **Step 4: Verify green**, `gofmt -l`, `go vet ./...`.
- [ ] **Step 5: Negative control.** Revert the wiring hunk (keep tests);
  observe the new tests red; restore.

### Task 2: I117 — implement-evidence message split

**Files:**
- Modify: `internal/stages/stages.go` (implement-evidence collector,
  `judgeSet` or its implement caller)
- Modify: `internal/stages/stages_test.go`,
  `implement_evidence_internal_test.go` as fits

**Interfaces:**
- Produces: the collector also reports, per anchored id, whether any ledger
  line starts with that id. In the `existing == 0` ticked-missing branch:
  any anchored line ⇒ detail names the done/complete/completed whole-word
  requirement (typo hint suppressed); zero anchored lines ⇒ typo hint
  verbatim as today. Other branches and prd/issues judging untouched.

- [ ] **Step 1: Failing tests**: ledger `I0NN: … declared`, stage ticked ⇒
  wording message, no typo hint; no line for the id ⇒ typo hint verbatim
  (negative control that the split is load-bearing); mixed case (one id
  anchored sans done-word, one absent) ⇒ wording message AND the
  missing-ids list still names both.
- [ ] **Step 2: Verify red.** Record command + output.
- [ ] **Step 3: Implement the minimum.**
- [ ] **Step 4: Verify green**; existing evidence tests stay green.
- [ ] **Step 5: Negative control.** Revert the branch-split hunk; new tests
  red; restore.

### Task 3: I118 — install one-liner + version provenance

**Files:**
- Modify: `cmd/spine/main.go` (`version` case → small `cmdVersion` with a
  testable formatter), `README.md` (Install subsection)
- Modify: `cmd/spine` tests

**Interfaces:**
- Produces: line 1 unchanged (`spine template generation N`); line 2
  `build: <module-version> <rev-12> <vcs-time> [dirty]` from
  `debug.ReadBuildInfo`, omitting absent fields; `build: (no build info)`
  on a failed read. README documents
  `go install github.com/russellpope/spine/cmd/spine@latest`, the
  self-contained binary, and `spine version` as the drift check.

- [ ] **Step 1: Failing tests**: `version` exits 0, line 1 matches today's
  format, a `build:` line follows; formatter unit test covers the
  nil-BuildInfo fallback and the dirty flag.
- [ ] **Step 2: Verify red.** Record command + output.
- [ ] **Step 3: Implement + README edit.**
- [ ] **Step 4: Verify green**; `spine version` run live from `make build`
  output shows a real revision (record it).
- [ ] **Step 5: Negative control.** Revert the formatter hunk; new tests
  red; restore.

### Task 4: functional test, review, verify, ship, docs

- [ ] `go test ./...`, `gofmt -l`, `go vet ./...` — exit 0, record output.
- [ ] Functional pass on the installed binary (`make install` to `~/bin`):
  the I116 trailing-flag invocation prints the rule; a scratch repo with a
  wording-miss ledger shows the I117 message; `spine version` prints
  provenance.
- [ ] Spec-review of the finished diff against the PRD (mandatory gate),
  including the requirements-attack step — attack the spec itself for
  internal contradictions first; surface with proposed resolutions, never
  silently resolve.
- [ ] ff-merge to main; `maipipe run full --wait` green at the merge SHA.
- [ ] `make install`; record sha256 prefix; `spine version` on the
  installed binary as the drift baseline for other devices.
- [ ] Ledger close per ticket: status fixed, `commits:` written — each
  resolution line carries a done-word (dogfooding I117's rule).
- [ ] CHANGELOG entry; retire the flag-order and done-word gotchas from
  living handoff prose (they become code behavior).
