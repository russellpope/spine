# I077 Eval-informed equivalence-pin ratification implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let doctor warn deterministically when a host pin lacks healthy,
repo-local, exact-model eval evidence, without changing pin validity,
resolution, audit, or non-doctor command exits.

**Architecture:** Add a strict, selected-path evidence reader in
`internal/eval`; keep host configuration references opaque and optional; map
the reader's sanitized outcomes to D17 after D16. The reader sees only pin key,
pinned model, references, the audited repository, and an injected UTC date.
It never sees a host path, reads a transcript, or resolves a route.

**Tech stack:** Go standard library, existing `internal/eval`,
`internal/doctor`, host-config fixtures, embedded eval templates, Go tests,
maipipe.

**Spec:** `docs/specs/2026-08-30-eval-informed-equivalence-pins-design.md`

## Global constraints

- Ticket: `I077`, routine tier. Every dispatched worker and reviewer names
  `I077` and has an explicit routed tier. Escalate the review/verification work
  to primary because it crosses host configuration, eval parsing, doctor, and
  compatibility boundaries.
- I072 is independently verified at final product SHA `50a6a6d`; preserve its
  optional opaque `evidence_refs`, `Load` behavior, D16 behavior, redaction,
  resolution, and controlled-launch restrictions exactly.
- Do not change `internal/model`, `internal/audit`, model CLI behavior,
  `spine audit routing`, I076, transcript readers, fleet traversal, external
  fetches, or a command exit contract. D17 is a normal `warn` under the
  existing doctor exit rule only.
- Read only the audited repository's physical `docs/evals/` tree. Reject
  selected symlinks and unsafe paths without following them. Do not use mtime.
- `stage` and `score` remain opaque. Generic `spine eval list` and D7 behavior
  remain compatible with ordinary old or malformed runs.
- Use explicit `git add` paths. Preserve unrelated working-tree changes. The
  final maipipe lane must pass at the exact final SHA, repeated after any
  closure-doc commit.

---

## File map

| File | Responsibility |
| --- | --- |
| `internal/eval/pin_evidence.go` | Parse only strict `eval:` references, inspect selected repo-local files, validate dates/model/battery profile, and return sanitized outcomes. |
| `internal/eval/pin_evidence_test.go` | Table-driven grammar, symlink, front-matter, date, exact-model, battery, aggregation, and no-unrelated-scan controls. |
| `internal/eval/eval.go` | Keep generic list parsing opaque; share only safe front-matter helpers if needed, without making battery fields generally required. |
| `internal/eval/eval_test.go` | Lock generic eval-list compatibility for runs with and without I077 fields. |
| `internal/doctor/doctor.go` | Load valid host config once, retain D16, invoke the new reader, format sorted D17 results after D16, and avoid raw-value disclosure. |
| `internal/doctor/doctor_test.go` | Prove exact D17 ID/severity/path/text/order, D7 separation, and existing doctor behavior. |
| `cmd/spine/main_test.go` | Exercise built/CLI doctor JSON and text results, plus model and audit exit no-change controls. |
| `templates/current/evals-README.md` | Document the optional I077 pin-evidence profile and strict `eval:` reference grammar. |
| `templates/current/run.tmpl.md` | Scaffold blank optional battery fields with the documented exact key names. |
| `docs/mutation-battery-checklist.md` | Clarify the I077 profile, ten-key matrix, and its advisory-only meaning. |
| `internal/update/update.go` and `internal/update/update_test.go` | Record removed generated README lines in `supersededLines` if the template change otherwise makes a pristine predecessor look hand-edited. |
| `docs/issues/I077-eval-informed-equivalence-pins.md` | Add implementation, review, verification, and exact-SHA closure evidence only after all required gates pass. |

### Task 1: build the strict selected-run reader test-first

**Files:**

- Create: `internal/eval/pin_evidence.go`
- Create: `internal/eval/pin_evidence_test.go`
- Modify: `internal/eval/eval.go`
- Modify: `internal/eval/eval_test.go`

**Interfaces:**

- Produces a package-private-or-exported-for-doctor value equivalent to:

  ```go
  type PinEvidencePin struct {
      Key          string
      Model        string
      EvidenceRefs []string
  }

  type PinEvidenceFinding struct {
      PinKey string
      Kind   PinEvidenceKind
      Path   string // repository-relative, never a host path
  }

  func CheckPinEvidence(repoDir string, pins []PinEvidencePin, today time.Time) []PinEvidenceFinding
  ```

- Consumes only the loaded pin values and `repoDir`. `PinEvidenceKind` has
  separate values for no reference, bad reference, missing, malformed, stale,
  model mismatch, no battery, and failed battery. It must not carry a raw
  reference, error string, model ID, or body content.

- [ ] **Step 1: Write failing reader tests.** Create a real temporary
  `docs/evals/2026-08-30-routing-check/` with a valid `eval.md` and a valid
  run. Cover the one valid pass, no `eval:` reference, invalid date/slug/run
  grammar, absent run, missing parent `eval.md`, invalid front matter, invalid
  and future `created`, day 90 fresh, day 91 stale, quoted exact model,
  case/prefix/observed-ID/effort lookalikes, missing battery, every malformed
  matrix case, explicit fail, two references where one fails, and an ordinary
  malformed unreferenced eval that yields no reader result.

- [ ] **Step 2: Add containment and redaction tests before implementation.**
  Make `docs/evals`, the selected eval directory, `runs`, and the selected
  run each a symlink in separate cases, including a symlink to a file outside
  the temp repository. Assert `malformed`, no read outside the root, a logical
  repository-relative path only, and no raw reference/model/body text in the
  returned value. Add a file whose mtime is current but declared date is 91
  days old; it must be stale.

- [ ] **Step 3: Run focused tests red.**

  Run:

  ```bash
  go test ./internal/eval -run 'Test.*(PinEvidence|Eval)' -count=1
  ```

  Expected: FAIL because the I077 reader and result types do not exist.

- [ ] **Step 4: Implement the narrow reader.** Parse only the stated grammar;
  use `filepath.Rel` plus `Lstat` component checks before every read; validate
  the parent and run required keys; unquote only a valid Go double-quoted
  scalar; compare the model byte-for-byte; calculate date age from `today.UTC`
  calendar dates; and validate exactly the v1 matrix and verdict relation.
  Sort pin keys and candidate `eval:` strings bytewise. Do not call `List`,
  walk `docs/evals`, inspect `stage`/`score`, or follow a symlink.

- [ ] **Step 5: Run focused tests green and generic compatibility coverage.**

  Run:

  ```bash
  go test ./internal/eval -run 'Test.*(PinEvidence|NewAndAddRunAndList|ListFlagsMalformedRun)' -count=1
  ```

  Expected: PASS. `List` still reports only its existing structural problem
  for a malformed ordinary run and accepts an old run with no battery fields.

- [ ] **Step 6: Commit the reader unit.**

  Run:

  ```bash
  git add internal/eval/pin_evidence.go internal/eval/pin_evidence_test.go internal/eval/eval.go internal/eval/eval_test.go && git commit -m 'feat(I077): read pinned eval evidence'
  ```

### Task 2: attach D17 without changing host authority or doctor ordering

**Files:**

- Modify: `internal/doctor/doctor.go`
- Modify: `internal/doctor/doctor_test.go`
- Modify: `cmd/spine/main_test.go`
- Modify: `internal/hostconfig/hostconfig_test.go`

**Interfaces:**

- Consumes a successfully loaded `hostconfig.Config` once and the Task 1
  sanitized evidence outcomes.
- Produces D17 `warn` findings with the design's exact paths and messages,
  after all current D16 findings. Existing D16 invalid-host-config behavior is
  unchanged and produces no guessed D17.

- [ ] **Step 1: Write failing doctor tests.** Build a valid injected host
  config containing pins with no evidence, a malformed `eval:` string, missing
  evidence, malformed evidence, stale evidence, exact-model mismatch, no
  battery, failed battery, one pass plus one fail, and two passes. Assert the
  full `doctor.Finding` values exactly: `ID == "D17"`, `Severity == "warn"`,
  logical path, fixed message, one line, deterministic key/reference order,
  and D16 before D17. Assert a valid `owner:I068`-only legacy pin loads.

- [ ] **Step 2: Add authority and exit negative controls.** In the same test
  fixture, call `hostconfig.Load` and `model.ResolveForHost` before and after
  every evidence mutation. Their values/errors must be identical. Use the CLI
  harness to prove `spine model` and `spine audit routing` keep their prior
  exit/result behavior with a missing, stale, failed, and mismatched ref.
  Prove only doctor gains D17 under its pre-existing warn exit behavior.

- [ ] **Step 3: Run focused tests red.**

  Run:

  ```bash
  go test ./internal/doctor ./cmd/spine ./internal/hostconfig -run 'Test.*(D17|PinEvidence|I077|Host.*Evidence)' -count=1
  ```

  Expected: FAIL because doctor does not invoke the reader or format D17.

- [ ] **Step 4: Implement the single-load integration.** Refactor only as
  needed so D16 and D17 share one successful `hostconfig.Load` result. Preserve
  all current D16 content/order and return D16 alone on load failure. Map the
  typed Task 1 outcome to the exact D17 table. Append D17 after the existing
  D16 list. Do not pass a host path into `internal/eval`, expose raw errors,
  change `Finding`, change model/audit code, or add a doctor flag.

- [ ] **Step 5: Run focused tests green.**

  Run:

  ```bash
  go test ./internal/doctor ./cmd/spine ./internal/hostconfig -run 'Test.*(D14|D16|D17|PinEvidence|I077|Host.*Evidence)' -count=1
  ```

  Expected: PASS. D14 retains its proxy behavior, D16 remains unchanged, and
  every D17 condition is `warn` only.

- [ ] **Step 6: Commit the doctor unit.**

  Run:

  ```bash
  git add internal/doctor/doctor.go internal/doctor/doctor_test.go cmd/spine/main_test.go internal/hostconfig/hostconfig_test.go && git commit -m 'feat(I077): advise on pinned eval evidence'
  ```

### Task 3: publish the profile and preserve template migration behavior

**Files:**

- Modify: `templates/current/evals-README.md`
- Modify: `templates/current/run.tmpl.md`
- Modify: `docs/mutation-battery-checklist.md`
- Modify: `internal/eval/eval_test.go`
- Modify: `internal/update/update.go`
- Modify: `internal/update/update_test.go`

**Interfaces:**

- Produces a newly scaffolded run with the optional keys
  `battery_version`, `battery_verdict`, and `battery_results`, all blank until
  the eval process records them. Existing standard fields remain unchanged.
- Consumes the design's grammar verbatim; the documentation says it is an
  advisory pin-evidence profile, not a general mutation threshold.

- [ ] **Step 1: Write failing template and update tests.** Assert `eval.New`
  creates a README that describes the exact `eval:` grammar and `eval.AddRun`
  creates all three blank battery keys while preserving the six body sections.
  Stage a pristine predecessor README and prove `spine update` refreshes it
  without reporting a hand edit; stage an actual local alteration and prove it
  remains visible. If the content delta needs predecessor recognition, assert
  each removed predecessor line is present in `supersededLines`.

- [ ] **Step 2: Run focused tests red.**

  Run:

  ```bash
  go test ./internal/eval ./internal/update -run 'Test.*(PinEvidenceTemplate|EvalsREADME|RunTemplate|Superseded)' -count=1
  ```

  Expected: FAIL because the new profile is absent from generated docs and
  migration recognition.

- [ ] **Step 3: Update documentation and minimal migration recognition.**
  Keep `stage`/`score` prose opaque. State all matrix keys, verdict values,
  exact date rule, and advisory-only scope once in the generated README and
  point to the checklist for probe process. Add only actual removed template
  lines to `supersededLines`; do not bump the workflow generation merely for
  this independent eval documentation change unless the current template
  mechanism requires it.

- [ ] **Step 4: Run focused tests green.**

  Run:

  ```bash
  go test ./internal/eval ./internal/update -run 'Test.*(PinEvidenceTemplate|EvalsREADME|RunTemplate|Superseded|Update)' -count=1
  ```

  Expected: PASS, including legacy-run compatibility and the real-local-edit
  negative control.

- [ ] **Step 5: Commit the documentation unit.**

  Run:

  ```bash
  git add templates/current/evals-README.md templates/current/run.tmpl.md docs/mutation-battery-checklist.md internal/eval/eval_test.go internal/update/update.go internal/update/update_test.go && git commit -m 'docs(I077): document pinned eval evidence'
  ```

### Task 4: requirements attack, review, independent verification, and closure

**Files:**

- Modify: `docs/issues/I077-eval-informed-equivalence-pins.md`
- Modify: `docs/specs/2026-08-30-eval-informed-equivalence-pins-design.md`
- Modify: `docs/specs/2026-08-30-eval-informed-equivalence-pins-plan.md`

- [ ] **Step 1: Run task review before whole-diff review.** After each Task
  1–3 commit, a fresh reviewer checks its focused diff against the design,
  reruns that task's focused tests, and confirms its interface preserves the
  next task's contract. Task review must reject any attempt to make a host
  config invalid for evidence state, to broaden the read root, to inspect
  `stage` or `score`, or to change a model/audit path.

- [ ] **Step 2: Run a fresh requirements attack.** A primary-tier reviewer
  reads the final diff against this design and attacks: compatibility versus
  ratification SHOULD, all symlink positions, external/host-home/fleet/
  transcript read attempts, date boundary and mtime spoofing, quote handling,
  aliases/observed IDs/effort lookalikes, a healthy reference masking an
  unhealthy one, raw-value leaks, D17 path/text/order, D7 isolation, D14/D16
  preservation, and all cases where advice might de-ratify a pin. Record and
  resolve every real finding before verification.

- [ ] **Step 3: Run focused, full, race, static, and functional probes.**

  Run:

  ```bash
  go test ./internal/eval ./internal/doctor ./internal/hostconfig ./cmd/spine -count=1
  go test ./... -count=1
  go test -race ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine doctor --dir <valid-referenced-fixture>
  ./bin/spine doctor --dir <missing-stale-failed-mismatch-fixture>
  ./bin/spine model --dir <same-fixture> codex primary
  ./bin/spine audit routing --dir <same-fixture> --transcripts <empty-fixture>
  git diff --check
  ```

  Expected: the valid reference is D17-silent; each unhealthy condition emits
  exactly its one D17 warning; doctor preserves its existing warn exit rule;
  model and audit output/exits are unchanged by evidence state; full/race/vet/
  build/diff checks pass.

- [ ] **Step 4: Perform independent verification.** A different fresh
  primary-tier verifier reruns the attack, focused/full/race/static checks,
  compiled doctor probes, symlink containment checks, D7-unrelated-file
  control, and all no-advisory-de-ratifies-a-pin controls. The verifier checks
  the exact final source and records the SHA tested. Do not accept a report
  from a pre-docs or pre-correction SHA.

- [ ] **Step 5: Run exact-SHA lane and workflow checks.** At the final commit
  SHA, after any closure docs commit, run:

  ```bash
  spine doctor --dir .
  spine audit routing --dir .
  spine audit stages --dir .
  maipipe run full --wait
  ```

  Record command, SHA, result, and known unrelated warnings separately. If a
  final documentation commit changes `HEAD`, rerun the lane at that new SHA.

- [ ] **Step 6: Commit closure evidence only after all gates pass.** Keep I077
  open until Steps 1 through 5 pass. Then add the exact implementation commits,
  primary review, independent verification, functional probes, and final
  exact-SHA maipipe result to the ticket. Commit only explicit closure-doc
  paths:

  ```bash
  git add docs/issues/I077-eval-informed-equivalence-pins.md docs/specs/2026-08-30-eval-informed-equivalence-pins-design.md docs/specs/2026-08-30-eval-informed-equivalence-pins-plan.md && git commit -m 'docs(I077): record pinned eval evidence verification'
  ```

## Plan self-review

- [ ] The task set covers every D17 condition, exact grammar, path containment,
  date boundary, model comparison, matrix rule, reference aggregation,
  redaction, output ordering, and D7 separation in the design.
- [ ] TDD begins each code task with a red test. The planned implementation
  leaves `evidence_refs` optional and opaque in `hostconfig.Load`.
- [ ] Functional controls prove missing/stale/failing evidence never changes a
  model result, effort result, audit verdict, audit exit, launch behavior, or
  pin ratification.
- [ ] The final verification is independent, primary-tier, exact-SHA, and
  includes the required maipipe lane after any closure documentation change.
