# Pre-dispatch model validation (I051) implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development or superpowers:executing-plans to
> implement this plan task by task. Every dispatch names I051, the primary
> repository path, and an explicit model from the `primary` tier. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fail-closed model-ID validation command to Spine, install the
verified binary, then integrate it at all eight controlled deepthought launch
sites under separate repository authorization.

**Architecture:** `internal/model` owns the embedded validation policy, a
single-read strict launch snapshot, typed refusal reasons, and the exact
active-ID predicate shared with audit. `cmd/spine` exposes the nested
`model validate` verb without changing the existing resolver command.
Deepthought skill recipes capture the validated ID in guarded assignments and
pass only those variables to launchers.

**Tech stack:** Go standard library, embedded JSON, Go RE2 regular
expressions, POSIX shell regression tests, Markdown skill sources.

**Spec:** `docs/specs/2026-08-29-predispatch-model-validation-design.md`

## Global constraints

- I051 is primary-tier, subagent-driven work with primary-tier task review,
  final review, and verification. Every dispatch contains the token `I051`
  and the primary repo's absolute path.
- The two repository roots are
  `/Users/ldh/Projects/github.com/spine` and
  `/Users/ldh/Projects/github.com/deepthought`.
- Phase 1 may write only the Spine repository. Phase 2 is blocked until the
  Phase 1 binary is verified, shipped, installed at both expected paths, and
  the owner separately authorizes deepthought edits.
- I051 remains open between phases. No worker may mark it fixed after the
  Spine-only shipment.
- Keep `internal/model.Resolve`, `HistoricalIDs`, aliases, update provenance,
  mirrors, old model CLI output, effort output, and alternate output
  compatible.
- Validation accepts only the active requested ID. Audit keeps aliases and
  history as evidence.
- Active and retired classification is current-first across a flavor. Any ID
  current in any tier wins; otherwise any historical ID in any tier of that
  flavor is retired, including cross-cell history.
- I119's flags-first contract applies at both parser layers. Outer `model`
  flags, including `--dir`, precede the `validate` positional. Nested
  `--expect` follows `validate` and precedes flavor/tier. Never accept an outer
  `--dir` passed to the nested parser as a compatibility exception.
- No bypass flag, environment variable, allow file, ledger exception, retry
  through plain resolution, or model-argument omission is permitted.
- I075 owns dispatch effort. I051 retains resolver effort-vocabulary errors
  but does not compare the effort passed to a launcher. Alternate validation
  is excluded.
- Do not edit `templates/current/AGENTS.md.tmpl`,
  `templates/current/WORKFLOW.md.tmpl`, `templates/VERSION`, plugin caches, or
  installed skill copies.
- All launch arguments remain quoted even though the positive ID grammar
  rejects shell syntax.
- Stage explicit paths only. Never use `git add .` or `git add -A`.
- In Spine, never stage `.cache/`,
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`, concurrent
  specs, or unrelated work.
- In deepthought, never stage
  `docs/research/2026-08-24-paperclip-steal-list.md` or unrelated work.
- A failed negative control is evidence only when the mutation is restored
  and the original test returns green afterward.

## File map

### Phase 1, Spine-owned files

| File | Responsibility |
| --- | --- |
| `models/defaults.json` | Closed `modelValidation` policy, safe-ID pattern, exact tokens, and named RE2 patterns. |
| `internal/model/model.go` | Policy types/invariants, strict one-read launch snapshot, typed validation result, and shared active-ID matcher. |
| `internal/model/model_test.go` | Policy, mirror, history, override, candidate, reason, no-bypass, syntax, and TOCTOU-boundary unit tests. |
| `cmd/spine/main.go` | Nested `model validate` grammar, output, diagnostic, and exit-code adapter. |
| `cmd/spine/main_test.go` | CLI contract, flags-first behavior, compatibility, output separation, and compiled-binary fixtures. |
| `internal/audit/audit.go` | Delegate only the active-ID equality leg to `internal/model`; retain evidence compatibility. |
| `internal/audit/resolve_test.go` | Same-state validation-to-audit invariant and alias/history/shared-ID regressions. |
| `README.md` | Consumer command and fail-closed launch guidance. |
| `CONTEXT.md` | Active launch ID versus audit evidence vocabulary and I075 boundary. |
| `CHANGELOG.md` | Consumer-visible I051 behavior and rollout order. |
| `docs/issues/I051-fail-closed-predispatch-model-validation.md` | Remains open through Phase 1; final closure records both repository SHAs. |

### Phase 2, deepthought-owned files

| File | Responsibility |
| --- | --- |
| `skills/codex-team/SKILL.md` | Validated cmux worker models, validated herdr lead/worker models, new capability probe. |
| `skills/claude-team/SKILL.md` | Validated cmux/herdr role and lead models, guarded effort reads, plain-mode refusal. |
| `skills/handoff-to-codex/SKILL.md` | Validated cmux/herdr Codex lead models and new capability probe. |
| `skills/lib/test-no-hardcoded-models.sh` | Preserve the literal-ID guard and require `model validate` in preflight/launch prose. |
| `skills/lib/test-model-validation-preflight.sh` | Site-scoped dataflow, fake-launcher refusal, old-binary, and mutation tests for all eight sites. |

## Interfaces locked by this plan

Add these names in `internal/model/model.go`:

```go
type LaunchReason string

const (
    ReasonForbiddenModel   LaunchReason = "forbidden-model"
    ReasonInvalidModelID   LaunchReason = "invalid-model-id"
    ReasonRetiredModel     LaunchReason = "retired-model"
    ReasonRouteMismatch    LaunchReason = "route-mismatch"
    ReasonUnmappedDispatch LaunchReason = "unmapped-dispatch"
)

type LaunchRequest struct {
    RepoDir            string
    Flavor             string
    Tier               string
    Expected           string
    MaxTemplateVersion int
}

type LaunchRefusal struct {
    Reason LaunchReason
    Key    string
    Value  string
    Rule   string
    Detail string
}

func (e *LaunchRefusal) Error() string
func ValidateLaunch(req LaunchRequest) (Entry, error)
func ActiveIDMatches(activeID, candidate string) bool
```

`LaunchRequest.Expected == ""` means no internal expectation. The CLI tracks
whether `--expect` appeared and rejects an explicitly empty value as usage
before it builds the request. `MaxTemplateVersion` is supplied by
`tmpl.Version()` at the CLI boundary, which avoids an import cycle from
`internal/model` back into `internal/tmpl`.

Use these unexported units so tests can keep strict parsing separate from the
compatibility resolver:

```go
type launchSnapshot struct { /* one parsed WORKFLOW.md read */ }

func readLaunchSnapshot(repoDir string, maxTemplateVersion int) (launchSnapshot, error)
func parseLaunchRouting(content string, maxTemplateVersion int) (launchSnapshot, error)
func parseLaunchValue(value string) (override, error)
func validateLaunchFrom(t table, snap launchSnapshot, flavor, tier, expected string) (Entry, error)
```

The implementation may add private fields, but it must not rename these
interfaces after neighboring tasks depend on them. `ValidateLaunch` returns a
`*LaunchRefusal` only for exit-1 policy decisions. Invocation and repository
configuration failures return other errors so `cmdModelValidate` can map them
to exit 2 without parsing error strings.

In deepthought, each launch recipe is bounded by stable source markers:

```text
<!-- i051-site: <site-name> begin -->
... guarded assignment and launcher recipe ...
<!-- i051-site: <site-name> end -->
```

The eight names are `codex-team-cmux-worker`, `codex-team-herdr-lead`,
`codex-team-herdr-worker`, `claude-team-cmux-role`,
`claude-team-herdr-lead`, `claude-team-herdr-role`,
`handoff-cmux-lead`, and `handoff-herdr-lead`. The new shell test extracts
these exact blocks. Duplicate or missing markers fail.

---

## Phase 1: Spine validation command and shipment

### Task 1: embed and validate the model-ID policy

**Files:**

- Modify: `models/defaults.json`
- Modify: `internal/model/model.go` (`table`, policy types,
  `mustLoadDefaults`, `validateTable`)
- Modify: `internal/model/model_test.go`

**Consumes:** the current embedded model table, `models.FS`, and the existing
`validateTable` panic convention.

**Produces:** a compiled policy whose schema and invariants are proven before
any runtime validation call.

- [ ] **Step 1: Add failing policy-shape tests.** Add table tests that clone
  the loaded table, mutate one policy property, and assert `validateTable`
  panics for each of these cases: missing `idPattern`, invalid `idPattern`,
  unknown policy member through a strict decode fixture, empty token,
  duplicate token, empty pattern name, duplicate pattern name, invalid RE2,
  current-ID syntax failure, current-ID deny overlap, historical-ID syntax
  failure, and a shorthand alias absent from `forbiddenTokens`.

  Add pre-decode mutation fixtures for duplicate member names at the root
  `modelValidation`, `idPattern`, pattern object, flavor, tier, and entry
  levels. Every named mutation must fail before last-member-wins typed decode.

- [ ] **Step 2: Run the focused tests red.**

  Run:

  ```sh
  cd /Users/ldh/Projects/github.com/spine
  go test ./internal/model -run 'TestValidateTable.*ModelValidation' -count=1
  ```

  Expected: FAIL to compile because the validation-policy fields and checks do
  not exist.

- [ ] **Step 3: Add the exact JSON object from the PRD.** Place
  `modelValidation` beside `flavors` in `models/defaults.json`. Do not change
  a current ID, alias, historical ID, effort, or alternate.

- [ ] **Step 4: Implement the closed policy types and invariants.** Decode the
  embedded file with unknown-field rejection and one trailing-value check.
  Compile `idPattern` and every named pattern once during table load. Validate
  exact token/name uniqueness, current and historical ID syntax, current-ID
  deny non-overlap, and the complete shorthand-alias inventory. Keep compiled
  regex values in private runtime fields that JSON does not populate.

- [ ] **Step 5: Verify policy tests green.**

  Run:

  ```sh
  go test ./internal/model -run 'TestValidateTable.*ModelValidation' -count=1
  ```

  Expected: PASS.

- [ ] **Step 6: Run load-bearing negative controls.** In the test table only,
  remove `opus` from `forbiddenTokens`; expect the alias-inventory case to
  fail. Add the current ID `DeepSeek-V4-Pro` as a forbidden token; expect the
  overlap case to fail. Replace one regex with `[`; expect the invalid-RE2
  case to fail. Restore each mutation and rerun Step 5 green.

- [ ] **Step 7: Commit the policy unit with explicit paths.**

  ```sh
  git add models/defaults.json internal/model/model.go internal/model/model_test.go
  git commit -m 'feat(I051): embed model launch policy'
  ```

### Task 2: add strict one-snapshot launch validation

**Files:**

- Modify: `internal/model/model.go` (`LaunchReason`, `LaunchRequest`,
  `LaunchRefusal`, `launchSnapshot`, `readLaunchSnapshot`,
  `parseLaunchRouting`, `validateLaunchFrom`, `ValidateLaunch`,
  `ActiveIDMatches`)
- Modify: `internal/model/model_test.go`

**Consumes:** Task 1 policy, existing table data, `parseValue` grammar,
`everShipped` history, `checkEffort`, `Tiers`, and dotted-over-bare Claude
precedence.

**Produces:** one public validated resolver and one exact matcher for the CLI
and audit tasks.

- [ ] **Step 1: Write failing positive tests.** Cover embedded default with no
  repo, absent `WORKFLOW.md`, a current dotted mirror row, a current legacy
  bare Claude row, a current ID with a deliberate effort change, a safe custom
  dotted override, a safe custom legacy bare override, and the active custom
  value `automatic-model`. Assert the returned `Entry.ID`, provenance where
  relevant, and `ActiveIDMatches` byte equality.

- [ ] **Step 2: Write failing strict-input tests.** Assert ordinary errors,
  not `LaunchRefusal`, for unreadable present input; empty, duplicate,
  malformed, non-decimal, and newer `template_version`; malformed
  `model_routing:` headers; duplicate blocks; duplicate requested dotted key;
  duplicate selected bare key; missing colon on the exact requested key;
  raw-key whitespace; empty selected ID; repeated `@`; repeated or empty exact
  ` alt:` delimiters; and existing invalid Pi effort vocabulary. Assert
  comment-only headers and values behave consistently. Assert safe IDs such as
  `salt:model` and `vault:autoencoder` pass when active. Assert a malformed
  unrelated row does not block a valid requested row. Assert one valid dotted
  Claude row wins before duplicate or malformed shadowed bare rows, while the
  same bare defects fail when the bare row is selected.

- [ ] **Step 3: Write failing policy-refusal tests.** Cover a historical
  mirror ID from the requested cell, a historical ID from another cell in the
  same flavor, the same historical ID with a changed effort,
  an unsafe active custom override, every exact token, case variants matched
  by the named patterns, separator-bound `vendor-auto` values, and
  `automatic-model` as the no-substring negative control. Check exact
  `LaunchReason`, key, escaped value, and rule name.

- [ ] **Step 4: Write failing `--expect`-level tests through
  `ValidateLaunch`.** Assert exact active match passes. Assert whitespace and
  case differences do not. Assert syntax failure precedes deny matching,
  shorthand aliases are `forbidden-model`, another same-flavor active tier is
  `route-mismatch`, a historical ID not active anywhere is `retired-model`,
  and a safe unknown is `unmapped-dispatch`. Include a current ID shared by
  several tiers and prove it validates for every declared cell.

- [ ] **Step 5: Run the validator tests red.**

  ```sh
  go test ./internal/model -run 'Test(ValidateLaunch|ActiveIDMatches|ParseLaunchRouting)' -count=1
  ```

  Expected: FAIL to compile because the Task 2 symbols do not exist.

- [ ] **Step 6: Implement the strict snapshot reader.** Read
  `WORKFLOW.md` exactly once. Treat only `os.IsNotExist` as absence. Parse one
  optional decimal template generation and one optional `model_routing`
  block. Track exact requested keys and duplicate counts instead of routing
  through `RoutingKeys`, whose last-write-wins map is intentionally retained
  for compatibility consumers. `parseLaunchValue` must apply `CommentIndex`
  before enforcing one unambiguous model, effort, and alternate grammar.

- [ ] **Step 7: Implement active route classification.** Resolve the requested
  cell and the other tiers from the same snapshot. Preserve existing
  effort-vocabulary validation. Classify current IDs before history across all
  tiers of the flavor. If no current ID matches, classify any flavor-wide
  history by model ID even when effort or alternate differs. Treat every other
  selected ID as a deliberate override, then apply syntax and deny policy. Build the
  other-tier active set only from rows that parse unambiguously; malformed
  unrelated rows remain non-blocking and add no route-mismatch candidate.

- [ ] **Step 8: Implement candidate classification and typed errors.** Apply
  the PRD order: invalid syntax, deny, exact requested match, another active
  same-flavor tier, history, then unmapped. Render no terminal text here;
  populate `LaunchRefusal` fields and let the CLI own presentation.

- [ ] **Step 9: Verify focused and compatibility tests green.**

  ```sh
  go test ./internal/model -run 'Test(ValidateLaunch|ActiveIDMatches|ParseLaunchRouting|Resolve_)' -count=1
  ```

  Expected: PASS, including the unchanged `Resolve` suite.

- [ ] **Step 10: Run negative controls.** Temporarily make candidate matching
  accept `Entry.Aliases`; the alias test must fail. Temporarily remove the
  history-before-override classification; the stale mirror tests must fail.
  Temporarily call the compatibility `readOverride`; duplicate/unreadable
  strict-input tests must fail. Restore and rerun Step 9 green.

- [ ] **Step 11: Commit the validator unit.**

  ```sh
  git add internal/model/model.go internal/model/model_test.go
  git commit -m 'feat(I051): validate active launch models'
  ```

### Task 3: expose the atomic CLI without changing old model modes

**Files:**

- Modify: `cmd/spine/main.go` (`usage`, `cmdModel`, new
  `cmdModelValidate`)
- Modify: `cmd/spine/main_test.go`

**Consumes:** `model.ValidateLaunch`, `model.LaunchRefusal`,
`tmpl.Version()`, `parseArgs`, and existing flags-first diagnostics.

**Produces:** the exact Design C command, stable output separation, and
0/1/2 exit mapping.

- [ ] **Step 1: Write failing success and grammar tests.** Assert these
  positive commands through `run`: `model validate codex primary`,
  `model --dir D validate codex primary`, `model --dir=D validate
  --expect=ID codex primary`, and the split-value `--expect ID` form. Assert
  usage failure when outer `--dir` appears in the nested parser's arguments,
  for outer `--expect`, and for nested `--expect` after flavor/tier. Also cover
  missing flavor/tier, explicit empty expect, and rejected `--alternate`,
  `--effort`, `--json`, and `--force`. Success must be exact ID plus newline on
  stdout and empty stderr.

- [ ] **Step 2: Write failing refusal and configuration tests.** Table-drive
  all five stable exit-1 reasons. Assert one stderr line, `%q` escaping, named
  deny rule, empty stdout, and exit 1. Assert unreadable input, malformed row,
  unknown flavor, unknown tier, and unsupported generation exit 2 with empty
  stdout. Run a refusal with `SPINE_MODEL_VALIDATE_BYPASS=1` and prove it still
  refuses. Add newline, tab, and carriage-return `--dir` cases. Each extracts
  and quotes the `PathError` path, exposes only the underlying OS error, and
  emits exactly one physical stderr line.

- [ ] **Step 3: Pin old CLI compatibility.** Re-run existing tests and add one
  byte comparison fixture for each of plain ID, `--json`, `--effort`, and
  `--alternate`. The nested verb must not reinterpret any old flag sequence.
  Compatibility preserves the old resolver forms; it does not exempt
  after-positional `--dir` from I119.

- [ ] **Step 4: Run CLI tests red.**

  ```sh
  go test ./cmd/spine -run 'TestModel(Validate|Bare|JSON|Effort|Alternate|TrailingFlag|Unknown)' -count=1
  ```

  Expected: FAIL because current `cmdModel` treats `validate` as a flavor.

- [ ] **Step 5: Implement `cmdModelValidate`.** Make the outer `model` parser
  consume `--dir` before its positionals, then dispatch when its first
  positional is the literal `validate`. The nested validate parser owns only
  `--expect` and the flavor/tier positionals. Do not forward or reinterpret an
  after-`validate` `--dir`, and do not let old resolver flags select validate
  mode. Pass `tmpl.Version()` in `LaunchRequest`. Use `errors.As` for
  `*model.LaunchRefusal`, return 1 for it, and return 2 for every other error.
  Print only after full success.

- [ ] **Step 6: Add safe diagnostics.** Prefix exit-1 lines with
  `model validate: <reason>:` and exit-2 lines with `model validate:`. Use
  `%q` for route values and include the actual quoted `--dir` path in the
  stale-mirror refresh hint.

- [ ] **Step 7: Verify CLI tests green.**

  ```sh
  go test ./cmd/spine -run 'TestModel' -count=1
  ```

  Expected: PASS.

- [ ] **Step 8: Run compiled-binary acceptance cases.**

  ```sh
  go build -o bin/spine ./cmd/spine
  bin/spine model --dir . validate codex primary
  bin/spine model --dir . validate --expect gpt-5.6-sol codex primary
  ```

  Expected: both exit 0, print exactly `gpt-5.6-sol`, and print no stderr.
  Run an isolated fixture with `codex.primary: auto`; expect exit 1,
  `forbidden-model` on stderr, and zero stdout bytes. Run a fixture with a
  newer template generation; expect exit 2 and zero stdout bytes.

- [ ] **Step 9: Run CLI negative controls.** Temporarily print the ID before
  checking the error; the zero-stdout refusal test must fail. Temporarily map
  `LaunchRefusal` to exit 2; all reason-table tests must fail. Temporarily
  accept `--force`; the grammar test must fail. Temporarily let the nested
  parser accept outer `--dir`; the I119 compatibility test must fail. Restore
  and rerun Steps 7 and 8 green.

- [ ] **Step 10: Commit the CLI unit.**

  ```sh
  git add cmd/spine/main.go cmd/spine/main_test.go
  git commit -m 'feat(I051): add model validate command'
  ```

### Task 4: share active matching with audit and prove the invariant

**Files:**

- Modify: `internal/audit/audit.go` (`resolvedTier.matches` only for its
  active-ID leg)
- Modify: `internal/audit/resolve_test.go`
- Modify: `internal/model/model_test.go` only if the matcher needs an
  additional direct case

**Consumes:** `model.ActiveIDMatches`, existing `model.Resolve`, aliases,
`HistoricalIDs`, `deriveFlavor`, `tiersOf`, FALLBACK, and escalation logic.

**Produces:** one active-ID definition shared by validation and audit without
changing retained evidence behavior.

- [ ] **Step 1: Write failing contract tests.** For an embedded current route
  and a safe deliberate override, first call `model.ValidateLaunch`, then feed
  the returned ID into audit under the unchanged fixture and assert it is not
  `unmapped-dispatch`. Keep flavor explicit in the test so the state-scoped
  invariant is unambiguous.

  Add same-file fixtures for malformed `model_routing:` headers, raw-key
  whitespace, comments, duplicates, selected and shadowed legacy bare rows,
  and safe overrides. Run validation first and then audit the exact same file;
  no validated ID may produce `unmapped-dispatch`, and malformed selected-like
  syntax must fail closed rather than default differently between consumers.

- [ ] **Step 2: Add compatibility tests.** Assert a rejected shorthand alias
  and historical ID still receive their pre-I051 audit treatment. Pin shared
  current IDs across tiers, cross-flavor derivation, deliberate-override
  alias withholding, FALLBACK, escalation, severity, and public verdict
  strings.

- [ ] **Step 3: Run the focused audit tests.**

  ```sh
  go test ./internal/audit -run 'Test.*(Validated|Alias|History|SharedID|Flavor|Fallback|Escalation|Unmapped)' -count=1
  ```

  Expected before the production edit: new contract tests may already pass
  behaviorally, but a source assertion must fail because
  `resolvedTier.matches` still compares `token == rt.id` directly instead of
  calling `model.ActiveIDMatches`. This source assertion makes the refactor
  load-bearing.

- [ ] **Step 4: Replace only the active leg.** Change the first comparison in
  `resolvedTier.matches` to `model.ActiveIDMatches(rt.id, token)`. Leave alias
  and history loops, provenance scoping, and every verdict branch unchanged.

- [ ] **Step 5: Verify model and audit green, including race tests.**

  ```sh
  go test ./internal/model ./internal/audit -count=1
  go test -race ./internal/model ./internal/audit -count=1
  ```

  Expected: PASS.

- [ ] **Step 6: Run the negative control.** Reintroduce direct equality in
  `resolvedTier.matches`; the source assertion must fail while behavior tests
  show why behavior alone was insufficient. Restore the shared call and
  rerun Step 5 green.

- [ ] **Step 7: Commit the audit unit.**

  ```sh
  git add internal/audit/audit.go internal/audit/resolve_test.go internal/model/model_test.go
  git commit -m 'refactor(I051): share active model matching'
  ```

  Omit `internal/model/model_test.go` from `git add` when Step 2 required no
  change there. Check `git diff --cached --name-only` before committing.

### Task 5: document, review, verify, ship, and install the Spine phase

**Files:**

- Modify: `README.md`
- Modify: `CONTEXT.md`
- Modify: `CHANGELOG.md`
- Verify: `docs/specs/2026-08-29-predispatch-model-validation-design.md`
- Verify: `docs/specs/2026-08-29-predispatch-model-validation-plan.md`
- Do not close: `docs/issues/I051-fail-closed-predispatch-model-validation.md`

**Consumes:** Tasks 1 through 4 and the batch handoff's exact-SHA ship/install
requirements.

**Produces:** a reviewed, verified, shipped, and installed Spine binary. This
is a necessary but insufficient I051 outcome.

- [ ] **Step 1: Update consumer documentation.** Add the exact grammar
  `spine model [--dir D] validate [--expect MODEL_ID] <flavor> <tier>` to
  `README.md`, state that outer `--dir` precedes `validate`, and state that
  passing outer `--dir` to the nested parser is a usage failure, not a
  compatibility form. Define active launch ID versus audit evidence and the
  I075 effort boundary in `CONTEXT.md`; add an I051 entry to `CHANGELOG.md`
  that states fail-closed behavior, no bypass, unchanged old modes, and
  binary-first rollout. Do not add template prose.

- [ ] **Step 2: Commit documentation separately.**

  ```sh
  git add README.md CONTEXT.md CHANGELOG.md
  git commit -m 'docs(I051): document model launch validation'
  ```

- [ ] **Step 3: Run formatting, focused tests, full tests, vet, and build.**

  ```sh
  gofmt -l internal/model internal/audit cmd/spine
  go test ./internal/model ./internal/audit ./cmd/spine -count=1
  go test -race ./internal/model ./internal/audit ./cmd/spine -count=1
  go test ./... -count=1
  go vet ./...
  make build
  git diff --check
  ```

  Expected: `gofmt -l` and `git diff --check` print nothing; all tests and vet
  pass; `bin/spine` builds.

- [ ] **Step 4: Run the command acceptance matrix against disposable repo
  fixtures.** Record stdout, stderr, and the unpiped exit status for: embedded
  current, current mirror, safe override, current legacy bare row, historical
  dotted row, historical bare row, every deny class, unsafe syntax, another
  active tier, unknown safe candidate, malformed requested row, unreadable
  present file, and newer generation. Include the `automatic-model` and
  changed-effort controls. Include positive outer-`--dir` and nested-`--expect`
  cases plus negative after-`validate` `--dir`, outer-`--expect`, and trailing
  nested-flag cases. Expected codes and reasons must match the PRD.

- [ ] **Step 5: Run repository workflow checks.**

  ```sh
  spine doctor --dir /Users/ldh/Projects/github.com/spine
  spine audit routing --dir /Users/ldh/Projects/github.com/spine
  spine audit stages --dir /Users/ldh/Projects/github.com/spine
  ```

  Expected: no new I051 configuration error, routing failure, or stage failure.
  Record separately any known pre-existing doctor findings.

- [ ] **Step 6: Perform the mandatory fresh spec review.** Dispatch a fresh
  primary-tier reviewer with I051, both spec paths, the finished Spine diff,
  and the primary repo path. The reviewer attacks the requirements before the
  code and checks all Spine-owned portions of acceptance criteria 1 through
  10 and 15 through 17. It must inspect strict-versus-compatible mirror
  readers, classification precedence, deny invariants, no-bypass behavior,
  stdout ordering, unchanged old modes, shared audit matching, and TOCTOU
  claims. Resolve every finding in a dedicated fix commit and rerun affected
  tests.

- [ ] **Step 7: Perform independent verification.** A second fresh
  primary-tier verifier reruns Steps 3 through 5 and the negative controls
  from Tasks 1 through 4. It checks `git diff`, staged paths, commit boundaries,
  and raw outputs instead of relying on worker reports. Run
  `spine audit routing --transcripts` with the controller transcript directory
  when the controller session is outside this repo.

- [ ] **Step 8: Ship the exact verified SHA.** With a clean intended diff,
  record `git rev-parse HEAD`, then run:

  ```sh
  maipipe run full --wait
  ```

  Expected: PASS at that exact SHA. Push according to
  `docs/handoffs/2026-08-29-open-ledger-batch-codex.md` only after the gate
  records that SHA. Do not mark I051 fixed.

- [ ] **Step 9: Install before consumer edits.**

  ```sh
  make install
  cp /Users/ldh/bin/spine /Users/ldh/.local/bin/spine
  /Users/ldh/bin/spine model --dir /Users/ldh/Projects/github.com/spine validate codex primary
  /Users/ldh/.local/bin/spine model --dir /Users/ldh/Projects/github.com/spine validate codex primary
  /Users/ldh/bin/spine version
  /Users/ldh/.local/bin/spine version
  ```

  Expected: both validation commands exit 0 with the same exact ID and empty
  stderr. Both version commands identify the Phase 1 shipped build. Record the
  shipped SHA and outputs in the implementation report.

- [ ] **Step 10: Stop at the cross-repository gate.** Report that Spine is
  shipped and installed, deepthought remains untouched, and I051 remains open.
  Do not begin Task 6 without a new explicit authorization covering the
  deepthought paths.

## Phase 1 correction after failed blind review

The first fresh primary blind review failed. The correction owns every finding
in `.superpowers/sdd/I051-review-worker1-report.md` and does not narrow a
finding to its smallest reproducer.

- [ ] **Step 1: Amend the specs before product edits.** In one docs-only
  commit, resolve empty-input exit classification, flavor-wide current-first
  history, the real verification sequence, I072 command grammar, and the
  fail-closed host/audit boundary. Keep I051 open and leave deepthought,
  templates, and installed binaries untouched.

- [ ] **Step 2: Same-file parser and audit slice.** Add failing tests for
  malformed `model_routing:` headers, raw-key whitespace, comments,
  duplicates, selected and shadowed legacy bare rows, and safe overrides.
  Each fixture passes through validation and audit against the same unchanged
  file. Implement one strict active-route result that both consumers use;
  retain audit aliases and history only after that active leg.

- [ ] **Step 3: Exact alternate delimiter and dotted precedence slice.** Add
  failing active-ID tests for `salt:model` and `vault:autoencoder`, plus
  repeated and empty exact ` alt:` delimiters. Add failing positive tests in
  which a valid dotted row shadows duplicate or malformed bare rows, and keep
  failing configuration tests for the same bare defects when the bare row is
  selected. Implement only exact-delimiter parsing and dotted-first selection.

- [ ] **Step 4: Recursive duplicate-JSON slice.** Add pre-decode mutation
  tests for duplicate root `modelValidation`, `idPattern`, pattern `name`,
  flavor, tier, and entry members. Observe last-member-wins decode red, then
  reject duplicate object names recursively before typed decoding.

- [ ] **Step 5: Diagnostic path slice.** Add failing CLI tests for newline,
  tab, and carriage-return bytes in a repository path. Assert exit 2, zero
  stdout, escaped bytes, and one physical stderr line. Extract the
  `PathError.Path`, quote it, and wrap only `PathError.Err`.

- [ ] **Step 6: Classification and future-host seam.** Add failing tests that
  current IDs win anywhere in a flavor and otherwise cross-cell history is
  retired. Add a host-divergent validation-refusal seam without implementing
  I072 host configuration: divergent pins are not auditable before I074, while
  an identical pin may validate. Preserve no-host byte compatibility.

- [ ] **Step 7: Commit coherent red/green units.** For every slice, record the
  exact failing command and expected failure before the product edit, then the
  focused green command after it. Stage only the named model, audit, CLI, and
  test paths. Recheck shared-checkout status before every commit and stop on
  overlap with I050 or unrelated work.

- [ ] **Step 8: Run the actual correction verification sequence.** Run:

  ```bash
  go test ./internal/model ./internal/audit ./cmd/spine -count=1
  go test -race ./internal/model ./internal/audit ./cmd/spine -count=1
  go test ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine doctor --dir .
  ./bin/spine audit routing --dir .
  ./bin/spine audit stages --dir .
  gofmt -l internal/model internal/audit cmd/spine
  git diff --check
  maipipe run full --wait
  ```

  Also run an independent compiled CLI matrix, recursive JSON mutation builds,
  and same-file validation-to-audit fixtures with raw stdout, stderr, and exit
  codes. A different fresh primary reviewer and verifier own the later gates.

## Phase 2: separately authorized deepthought integration

### Blocking gate

Phase 2 is explicitly blocked until all of these facts are recorded:

- Phase 1 spec review approved.
- Phase 1 independent verification passed.
- The exact Spine SHA passed maipipe and was shipped.
- Both installed Spine paths passed `model validate`.
- The owner explicitly authorized edits and landing in
  `/Users/ldh/Projects/github.com/deepthought`.

If any fact is absent, stop. This is an authority blocker, not a test failure
and not permission to edit a copied install.

### Task 6: add red site-scoped and fake-launcher tests in deepthought

**Files:**

- Create: `skills/lib/test-model-validation-preflight.sh`
- Modify: `skills/lib/test-no-hardcoded-models.sh`
- Read only: the three skill files until the red run is recorded

**Consumes:** the eight site names and guarded-flow contract from the PRD.

**Produces:** tests that fail against current skill prose for the right
reasons before any skill edit.

- [ ] **Step 1: Recheck external state and authorization.**

  ```sh
  cd /Users/ldh/Projects/github.com/deepthought
  git status --short --branch
  git log -5 --oneline
  readlink /Users/ldh/.codex/skills/codex-team
  readlink /Users/ldh/.claude/skills/claude-team
  readlink /Users/ldh/.claude/skills/handoff-to-codex
  ```

  Expected: the three links resolve into this checkout. Record and preserve
  all unrelated state, including the known research stray.

- [ ] **Step 2: Implement the site extractor and structural assertions.** The
  new POSIX shell test requires exactly one begin/end block for each of the
  eight names. Inside each block it requires a guarded assignment from
  `spine model --dir "$REPO" validate`, a nonempty check, and the corresponding
  launcher model argument using only the declared captured variable. It
  rejects an after-`validate` `--dir`, a nested validator substitution in an
  outer launch command, plain `spine model` in a model assignment, literal
  current/historical IDs, missing quotes, and an unguarded launcher.

- [ ] **Step 3: Add executable fake-command cases.** The test prepends a temp
  `bin` directory containing fake `spine`, `cmux`, `herdr`, `claude`, and
  `codex` commands. The fake launchers append their argv to one log. Run each
  extracted site block under these validator modes: success, exit 1 with no
  output, exit 1 after printing an ID, exit 2 unknown verb, and success with
  an empty line. Only success with a nonempty ID may append one launcher call.
  The old-binary case must select the skill's install/rebuild refusal text.

- [ ] **Step 4: Add the mutation suite.** Copy the skill sources to a temp
  root, then mutate one marked site at a time: move `--dir D` from its outer
  parser position into the nested parser's arguments, remove `validate`, remove
  the guard, replace the launch variable with a literal, use a different
  variable, delete the nonempty check, and move validation inside the launcher
  substitution. Each mutation must make the structural or fake execution test
  fail. The test exits nonzero if any mutant survives.

- [ ] **Step 5: Strengthen the existing hardcoded-model test.** Require all
  three shared preflights to invoke `spine model --dir "$REPO" validate` and
  retain the install/rebuild hint. Keep its existing literal-ID exemptions
  visible.

- [ ] **Step 6: Run both tests red before editing skills.**

  ```sh
  sh skills/lib/test-no-hardcoded-models.sh
  sh skills/lib/test-model-validation-preflight.sh
  sh skills/lib/test-model-validation-preflight.sh --mutation-suite
  ```

  Expected: the first two fail because current skill sources use plain
  resolution and contain no site markers; the mutation suite must not report
  success against an uninstrumented source. Save the exact failures in the
  deepthought implementation report.

### Task 7: wire all controlled launches and refuse Claude plain mode

**Files:**

- Modify: `skills/codex-team/SKILL.md`
- Modify: `skills/claude-team/SKILL.md`
- Modify: `skills/handoff-to-codex/SKILL.md`

**Consumes:** Task 6 tests and the installed Spine command.

**Produces:** eight marked, guarded, executable recipes and three updated
capability preflights.

- [ ] **Step 1: Update all shared preflights.** Probe
  `spine model --dir "$REPO" validate <flavor> primary` after the repo path is
  known. On any failure, refuse with a message that says the binary may
  predate `model validate`, gives `make install`, and promises no spawn. Keep
  the per-launch checks; the preflight is capability feedback only.

- [ ] **Step 2: Update codex-team site 1.** In cmux worker workspace creation,
  validate each worker's highest cluster tier into its corresponding
  `WORKER_N_MODEL` before the single `cmux new-workspace` call. Abort before
  cmux when any assignment fails or is empty. Each generated `codex -m`
  command uses only its worker's captured variable. Preserve ESCALATION record
  ordering and the highest-tier cluster rule.

- [ ] **Step 3: Update codex-team sites 2 and 3.** The herdr Master validates
  the primary lead into `LEAD_MODEL`; the herdr Lead validates each dispatch
  tier into `WORKER_MODEL`. Both guard and nonempty-check before
  `herdr agent start`, then pass the quoted variable to `-m`. Preserve prompt
  race handling and fresh-process behavior.

- [ ] **Step 4: Update claude-team site 4.** For each cmux role launch,
  validate the selected tier into `ROLE_MODEL`. Resolve effort separately
  into `ROLE_EFFORT` through `spine model --effort`, with its own guard. Build
  the cmux command only after both assignments pass. The `--model` argument
  uses `ROLE_MODEL` and `--effort` uses `ROLE_EFFORT`.

- [ ] **Step 5: Update claude-team sites 5 and 6.** The herdr Master uses
  guarded `LEAD_MODEL` plus guarded `LEAD_EFFORT`; the herdr Lead uses guarded
  `ROLE_MODEL` plus guarded `ROLE_EFFORT`. Preserve permission mode,
  fresh-process, report, and prompt-race behavior.

- [ ] **Step 6: Refuse claude-team plain mode.** Replace the upstream plain
  SDD fallback with a hard refusal containing `plain mode`, `cannot prove
  pre-dispatch model validation`, and `no worker spawned`. Do not invoke the
  Agent tool after this branch. Keep codex-team and handoff-to-codex plain
  refusals.

- [ ] **Step 7: Update handoff-to-codex sites 7 and 8.** The cmux and herdr
  lead recipes each validate primary into `LEAD_MODEL`, guard and nonempty
  check it, then use only that variable for `codex -m`. Preserve the primary
  repo text in kickoff prompts and all workspace cleanup rules.

- [ ] **Step 8: Add the exact site markers.** Bound only the assignment and
  launch recipe for each site. Do not let one marker block contain two
  launchers or let one validation block claim another site's launch.

- [ ] **Step 9: Run focused tests green.**

  ```sh
  sh skills/lib/test-no-hardcoded-models.sh
  sh skills/lib/test-model-validation-preflight.sh
  sh skills/lib/test-model-validation-preflight.sh --mutation-suite
  ```

  Expected: PASS. The fake launcher log is empty for all validator failures
  and has exactly one correctly quoted model argument for each success case.

- [ ] **Step 10: Run live installed-binary dry preflights without launching.**
  Execute only each recipe's guarded assignment and nonempty check against the
  deepthought repo for the tiers used by fixtures. Do not call cmux, herdr,
  Claude, or Codex in this step. Expected: valid routes return nonempty IDs;
  a disposable stale-mirror fixture refuses with the named reason.

- [ ] **Step 11: Run negative controls and restore.** Remove `validate` from
  one site, move one site's `--dir` after `validate`, launch a literal from
  another, remove one guard, make fake Spine print an ID then exit 1, and
  switch Claude preflight to `plain`. Each matching test must fail and the fake
  launcher log must stay empty for the validator failure. Restore and rerun
  Step 9 green.

### Task 8: review, verify, and land the deepthought phase

**Files:**

- Verify and commit only the five deepthought paths in Tasks 6 and 7

- [ ] **Step 1: Run shell syntax and all relevant tests.**

  ```sh
  sh -n skills/lib/test-no-hardcoded-models.sh
  sh -n skills/lib/test-model-validation-preflight.sh
  sh skills/lib/test-no-hardcoded-models.sh
  sh skills/lib/test-model-validation-preflight.sh
  sh skills/lib/test-model-validation-preflight.sh --mutation-suite
  git diff --check
  ```

  Expected: syntax checks and tests pass; whitespace check prints nothing.

- [ ] **Step 2: Perform a fresh deepthought task review.** A fresh
  primary-tier reviewer receives I051, both Spine spec paths, the deepthought
  diff, and both repository roots. It checks all eight local flows, fake
  failure behavior, old-binary hints, plain refusal, effort separation,
  quoting, marker uniqueness, prompt/audit text preservation, and no copied
  install edits. Fix every finding and rerun Step 1.

- [ ] **Step 3: Perform independent deepthought verification.** A different
  fresh primary-tier verifier reruns Step 1 and every Task 7 negative control
  from a clean temp root. It checks that the installed skill symlinks still
  point to the reviewed sources and confirms no launcher call on refusal.

- [ ] **Step 4: Check the exact staged scope.**

  ```sh
  git status --short
  git add skills/codex-team/SKILL.md skills/claude-team/SKILL.md skills/handoff-to-codex/SKILL.md skills/lib/test-no-hardcoded-models.sh skills/lib/test-model-validation-preflight.sh
  git diff --cached --name-only
  ```

  Expected staged paths: exactly the five paths in the `git add` command. The
  known research stray and all unrelated work remain unstaged.

- [ ] **Step 5: Commit and land under the recorded deepthought authorization.**

  ```sh
  git commit -m 'fix(I051): validate team models before launch'
  ```

  Record the full SHA. Push or merge only through the authorization that
  opened Phase 2. If that authorization covers edits but not landing, stop
  after the local commit and report the remaining blocker. I051 stays open.

## Phase 3: cross-repository verification and I051 closure

### Task 9: verify both landed SHAs, then close I051

**Files:**

- Modify: `docs/issues/I051-fail-closed-predispatch-model-validation.md`
- Modify: `CHANGELOG.md` only if the final review corrects a shipped claim
- Verify: all Phase 1 and Phase 2 files

**Consumes:** the landed Spine implementation SHA, landed deepthought SHA,
both fresh review reports, and raw verification evidence.

**Produces:** an honestly closed I051 and a final exact-SHA Spine gate.

- [ ] **Step 1: Run the installed cross-repository smoke.** From the landed
  deepthought checkout, run both shell tests against the installed Phase 1
  binary. Run the fake Spine modes again. Confirm all eight site blocks call
  zero launchers on failure and exactly one launcher with the validated value
  on success.

- [ ] **Step 2: Re-run Spine verification at current integration state.**

  ```sh
  cd /Users/ldh/Projects/github.com/spine
  go test ./... -count=1
  go test -race ./internal/model ./internal/audit ./cmd/spine -count=1
  go vet ./...
  make build
  spine doctor --dir /Users/ldh/Projects/github.com/spine
  spine audit routing --dir /Users/ldh/Projects/github.com/spine
  spine audit stages --dir /Users/ldh/Projects/github.com/spine
  git diff --check
  ```

  Expected: tests, race tests, vet, build, routing audit, and stage audit pass;
  no I051-specific doctor error appears; whitespace check prints nothing.

- [ ] **Step 3: Perform the mandatory final cross-repository spec review.** A
  fresh primary-tier reviewer sees neither implementation report. Give it
  this PRD, the Spine diff from the pre-I051 base through the current commit,
  the deepthought integration diff, and both repository roots. It attacks all
  twenty acceptance criteria, with special attention to the authority gate,
  binary-first chronology, same-state audit invariant, plain refusal,
  mutation evidence, effort/alternate exclusions, and TOCTOU wording. Resolve
  every finding in the owning repository and rerun affected checks.

- [ ] **Step 4: Perform independent final verification.** A different fresh
  primary-tier verifier reruns Tasks 8 Step 1 and Task 9 Steps 1 and 2, checks
  both full SHAs, reviews raw red/green negative-control logs, and confirms
  the installed binary predates no live symlinked skill requirement. Use the
  correct external transcript directory with `spine audit routing
  --transcripts` when required.

- [ ] **Step 5: Close the issue only now.** In
  `docs/issues/I051-fail-closed-predispatch-model-validation.md`, set
  `status: fixed`, add the actual Spine implementation commit list, and write
  a `Resolution` that names the full deepthought integration SHA, both review
  reports, verification commands, binary-first installation evidence, no
  bypass, model-ID-only scope, plain-mode boundary, arbitrary direct-spawn
  boundary, and TOCTOU limitation. Do not put the deepthought SHA in Spine's
  `commits:` field if that field is defined as same-repository commits; name it
  in the Resolution instead.

- [ ] **Step 6: Commit the closure with exact paths.**

  ```sh
  git add docs/issues/I051-fail-closed-predispatch-model-validation.md
  git diff --cached --name-only
  git commit -m 'docs(I051): close pre-dispatch model validation'
  ```

  Expected staged path: only the I051 issue, unless Step 3 required an
  evidence-backed `CHANGELOG.md` correction. If so, stage that file explicitly
  and state why in the closure report.

- [ ] **Step 7: Gate and ship the final Spine SHA.**

  ```sh
  maipipe run full --wait
  git status --short
  ```

  Expected: PASS at the exact closure SHA. The status output contains only
  known unrelated/untracked work. Push using the batch handoff procedure.

- [ ] **Step 8: Refresh both installed binaries for final provenance.**

  ```sh
  make install
  cp /Users/ldh/bin/spine /Users/ldh/.local/bin/spine
  /Users/ldh/bin/spine version
  /Users/ldh/.local/bin/spine version
  ```

  Expected: both installs report the final Spine closure revision and both
  still pass the valid-route smoke.

## Plan self-review checklist

- [x] The plan separates Spine-owned implementation from a blocked,
  separately authorized deepthought phase.
- [x] I051 cannot close after the binary-only phase; Task 9 requires both
  landed SHAs and fresh cross-repository verification.
- [x] Every PRD acceptance criterion maps to a named task and command.
- [x] The command grammar, five reasons, 0/1/2 exits, strict mirror states,
  I119 outer/nested flags-first compatibility, positive and negative parser
  cases, safe-ID policy, no bypass, audit invariant, effort/alternate
  exclusions, and TOCTOU limit have explicit tests.
- [x] All eight controlled launch sites have a stable name, local captured
  variable, source marker, mutation control, and fake-launch refusal case.
- [x] Binary shipment and both installations precede every deepthought edit.
- [x] The plan preserves unrelated Spine and deepthought files and uses only
  explicit staging commands.
- [x] Placeholder scan found no unfinished marker, generic error-handling
  step, or unnamed test command.
- [x] Interface names and reason constants are consistent across Tasks 1
  through 9.
