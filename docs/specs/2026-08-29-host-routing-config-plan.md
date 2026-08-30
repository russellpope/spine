# Host routing configuration (I072) implementation plan

> **For agentic workers:** Use a fresh implementation worker for each task or
> execute serially with a TDD loop. Steps use checkbox (`- [ ]`) syntax for
> tracking. Do not begin I073 or I077 from this plan.

**Goal:** Add a local, secret-free host capability and tier-pin config that
returns a dispatchable final model@effort pair without making repository
mirrors or updates host-dependent.

**Architecture:** `internal/hostconfig` owns parsing, default-path lookup, and
closed-schema validation. `internal/model` retains `Resolve` for estate plus
repository preference and adds a host-aware result carrying requested and
final pairs. The CLI, doctor, and audit consume that shared result. Update and
template rendering stay deliberately outside the host-aware call path.

**Tech stack:** Go standard library only. JSON uses `encoding/json`; default
path uses `os.UserConfigDir`; executable presence uses an injected
`exec.LookPath` boundary and never runs a process.

**Spec:** `docs/specs/2026-08-29-host-routing-config-design.md`

## Global constraints

- This plan is for I072 at routine tier. Artifacts name tiers, not fixed model
  IDs, except schema fixtures that prove concrete model@effort behavior.
- Do not reserve a doctor ID now. I108 takes D14 despite its stale D11 ticket
  text, and I050 also adds a doctor check. Immediately before the I072 doctor
  task, inspect the merged current source and allocate the first unclaimed
  D-number after those changes. Expect D16 only if I108 and I050 both land
  first; otherwise record the actual allocation in the implementation report,
  ticket, tests, and changelog.
- Read the local config only from `os.UserConfigDir()/spine/routing-host.json`.
  Tests inject a file path. Do not add a normal path flag or environment
  override.
- The schema is closed. Do not add endpoint, token, auth, credentials,
  `modelOverrides`, arbitrary `args`, or `env` fields. Never execute a
  configured command, read environment values, make a network call, or print
  config content.
- Preserve the public I072 CLI spelling `spine model <flavor> <tier>` and its
  current JSON fields. I073 owns the public flavor-to-harness migration.
- `models/defaults.json`, `model.MirrorRows`, template render, init, adopt,
  and update remain host-blind. Do not modify aliases, historical defaults,
  `transcriptFlavor`, or `--alternate` behavior.
- I074 owns heterogeneous confirmation. I077 owns evidence interpretation.
  `observed_ids` and `evidence_refs` must remain exact opaque data in I072.
- Stage explicit paths only. Do not stage `.cache/`,
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`, concurrent code,
  or scratch files.

## File map

| File | Responsibility |
| --- | --- |
| `internal/hostconfig/hostconfig.go` | Schema-v1 types; `DefaultPath`, `Load`, and validation for a local host file. |
| `internal/hostconfig/hostconfig_test.go` | Parser, security boundary, route, observed-ID, and injected executable/path tests. |
| `internal/model/model.go` | Existing `Resolve` stays preference-only; new host-aware `ResolveForHost` and resolution-trail types return requested and final pairs. |
| `internal/model/model_test.go` | Precedence, compatibility, no-substitution, pi, and alternate negative controls. |
| `cmd/spine/main.go` | `cmdModel` prints final pair and additive JSON trail; `cmdAuditRouting` passes host validation through audit. |
| `cmd/spine/main_test.go` | CLI success, compatibility, error, stdout, and doctor exit tests. |
| `internal/doctor/doctor.go` | `hostRoutingCheck` adds the integration-allocated I072 finding without writes. |
| `internal/doctor/doctor_test.go` | Absent, error, and active unpinned-reachability cases for that allocated finding. |
| `internal/audit/audit.go` | Audit preflight validates host routing before transcripts or ticket verdicts. |
| `internal/audit/resolve_test.go` | Host-config preflight and no-heterogeneous-confirmation regression tests. |
| `docs/specs/2026-08-29-host-routing-config-design.md` | Binding PRD for implementation and final spec review. |
| `docs/issues/I072-host-config-schema-and-precedence.md` | Closure, commit IDs, review evidence, and implementation resolution. |
| `CHANGELOG.md` | Consumer-visible `spine model` and doctor behavior after the feature ships. |

## Interfaces locked by this plan

Implement these names and keep their responsibilities narrow:

```go
// internal/hostconfig
var ErrNotConfigured error

func DefaultPath() (string, error)
func Load(path string, lookPath func(string) (string, error)) (Config, error)
func Validate(c Config, lookPath func(string) (string, error)) error

// internal/model
type Resolution struct {
    Entry     Entry          // final pair, preserves existing Entry output fields
    Requested Entry          // estate + repository preference before host constraint
    Host      HostResolution // unconfigured, reachable, or pinned trail
}

func ResolveForHost(repoDir, configPath, flavor, tier string,
    lookPath func(string) (string, error)) (Resolution, error)
```

`ResolveForHost` uses `hostconfig.DefaultPath` only when its `configPath`
argument is empty. Production callers pass the empty value; tests pass a
fixture path. `ErrNotConfigured` is the only absent-file result and causes the
legacy result, not an error. All present files are fully validated before a
final pair is returned.

The exact exported field layout may gain JSON tags or small nested types, but
it must represent: final `Entry`; requested ID, effort, and provenance; host
ID/status/config path; and a pin's model, effort, and opaque evidence refs.
The resolver must not expose endpoint or config-file contents.

### Task 1: host config parser and validation

**Files:**

- Create: `internal/hostconfig/hostconfig.go`
- Create: `internal/hostconfig/hostconfig_test.go`

**Consumes:** `os.UserConfigDir`, `encoding/json`, `os/exec.LookPath` through
an injected function.

**Produces:** schema-v1 `Config`, `DefaultPath`, `Load`, `Validate`, and
`ErrNotConfigured` for model, doctor, and audit callers.

- [ ] **Step 1: Write failing parser tests.** Cover the platform-path seam by
  injecting the user config directory function and expecting
  `<config-dir>/spine/routing-host.json`; an absent file returns exactly
  `ErrNotConfigured`; a valid Claude config has route
  `gpt-5.6-sol @ high`; a pin key `claude.primary` validates; and `Load` does
  not create a file.

- [ ] **Step 2: Write failing schema-boundary tests.** Reject malformed JSON,
  `schema_version: 2`, unknown root/harness/route/pin members, duplicate JSON
  object keys, control characters, empty IDs, invalid dotted pin keys,
  unknown tiers, duplicate effort strings, duplicate observed IDs across the
  config, unavailable pin harnesses, absent pin model, and a pin effort not in
  its route. Include table cases for `token`, `base_url`, `auth_header`,
  `credentials`, `modelOverrides`, `args`, and `env`; every case must fail.

- [ ] **Step 3: Run the focused tests red.**

  Run: `go test ./internal/hostconfig -run 'Test(DefaultPath|Load|Validate)' -count=1`

  Expected: fail because the package and symbols do not exist.

- [ ] **Step 4: Implement the minimum closed-schema parser.** Decode JSON in
  a way that detects duplicate object keys and rejects unknown members. Use
  `os.UserConfigDir` for the production default, injected lookup for tests,
  and `exec.LookPath` only to test executable presence. Do not execute the
  result. Make a present but unreadable file, invalid schema, unavailable
  harness, or absent executable an error with its config path and safe field
  name only.

- [ ] **Step 5: Verify green and prove the security boundary.**

  Run: `go test ./internal/hostconfig -count=1`

  Expected: PASS. Add a test lookup that records calls and proves validation
  performs only an executable lookup, not a shell execution or network action.

- [ ] **Step 6: Commit the parser unit.**

  Run: `git add internal/hostconfig/hostconfig.go internal/hostconfig/hostconfig_test.go && git commit -m 'feat(I072): validate host routing config'`

### Task 2: host-aware resolution without changing mirrors

**Files:**

- Modify: `internal/model/model.go` (`Resolve` retained; add
  `Resolution`, `HostResolution`, and `ResolveForHost`)
- Modify: `internal/model/model_test.go`
- Modify: `internal/update/modelrouting_test.go`
- Modify: `internal/tmpl/tmpl.go` only if a compiler-visible seam is needed;
  do not change `Render` behavior

**Consumes:** `hostconfig.Load`; existing `Resolve`, `MirrorRows`,
`model.Tiers`, and repository `WORKFLOW.md` parsing.

**Produces:** final host-aware resolution with a complete requested/final
trail, while `Resolve` and all mirror callers remain preference-only.

- [ ] **Step 1: Write failing resolver tests.** Prove all of these cases:

  - embedded default followed by repository override followed by a reachable
    pin returns the pinned final pair and preserves the requested overridden
    pair and provenance;
  - valid host with no pin returns a reachable requested pair unchanged;
  - absent file returns byte-equivalent existing `Resolve` ID, effort,
    aliases, alternate, and provenance;
  - no pin plus an unreachable requested pair errors with no substitute;
  - a pin missing its exact `model@effort` route errors;
  - pi is not silently host-filtered, and `--alternate` still uses the old
    cell-only behavior.

- [ ] **Step 2: Add host-blind negative controls.** Use a valid host fixture
  with a divergent Claude pin, then prove `MirrorRows()`,
  `update.applyModelRouting`, and rendered `WORKFLOW.md` still contain only
  the embedded/repository result. The host fixture must not create a refresh
  or an override item.

- [ ] **Step 3: Run focused tests red.**

  Run: `go test ./internal/model ./internal/update -run 'Test.*(Host|Mirror|ModelRouting)' -count=1`

  Expected: FAIL because host-aware resolution and its trail do not yet exist.

- [ ] **Step 4: Implement `ResolveForHost`.** First call existing `Resolve`
  to get the requested entry. Map `ErrNotConfigured` to an `unconfigured`
  trail. For a present config, require the selected flavor-named harness and
  executable, apply an exact tier pin when present, otherwise require the
  requested model and effort route. Keep `Resolve`, `MirrorRows`, and
  `applyModelRouting` free of hostconfig imports.

- [ ] **Step 5: Verify green.**

  Run: `go test ./internal/model ./internal/update -count=1`

  Expected: PASS, including existing historical-default and update tests.

- [ ] **Step 6: Commit the resolver unit.**

  Run: `git add internal/model/model.go internal/model/model_test.go internal/update/modelrouting_test.go && git commit -m 'feat(I072): resolve host routing constraints'`

### Task 3: model command trail and failure contract

**Files:**

- Modify: `cmd/spine/main.go` (`cmdModel`)
- Modify: `cmd/spine/main_test.go`

**Consumes:** `model.ResolveForHost` and existing `--json`, `--effort`, and
`--alternate` flag behavior.

**Produces:** text output for the final pair and additive machine-readable
requested/host/pin trail.

- [ ] **Step 1: Write failing command tests.** With a fixture path injected
  below the command seam, assert: text output is the pinned final ID;
  `--effort` is the pin effort; JSON retains current `flavor`, `tier`, `id`,
  `effort`, `aliases`, `alternate`, and `provenance`, then adds requested,
  host, and pin fields; and no-file text/old fields stay unchanged.

- [ ] **Step 2: Write failing error and compatibility tests.** For malformed
  config, unavailable harness, missing executable, unreachable preference,
  and unreachable pin, each of normal, `--effort`, and `--json` exits 2,
  writes no stdout, and has one safe stderr diagnostic. Keep the existing
  flags-before-positionals and `--alternate` tests green unchanged.

- [ ] **Step 3: Run the focused tests red.**

  Run: `go test ./cmd/spine -run 'TestModel.*(Host|Config|JSON|Alternate|Flag)' -count=1`

  Expected: FAIL because `cmdModel` still calls preference-only `Resolve`.

- [ ] **Step 4: Implement the thin CLI adapter.** Replace the one resolution
  call with `ResolveForHost`; keep flag parsing in `cmdModel`, select final
  entry for text and effort output, and marshal additive JSON. Return 2 before
  emitting any success output on host errors. Never read config contents into
  command output.

- [ ] **Step 5: Verify green.**

  Run: `go test ./cmd/spine -run 'TestModel' -count=1`

  Expected: PASS.

- [ ] **Step 6: Commit the CLI unit.**

  Run: `git add cmd/spine/main.go cmd/spine/main_test.go && git commit -m 'feat(I072): expose host routing resolution'`

### Task 4: doctor checks with an integration-allocated finding ID

**Files:**

- Modify: `internal/doctor/doctor.go` (`Run`, new `hostRoutingCheck`)
- Modify: `internal/doctor/doctor_test.go`
- Modify: `cmd/spine/main_test.go`

**Consumes:** shared hostconfig validation and preference resolution. Doctor
must pass the repository directory only for the active requested-pair check.

**Produces:** an I072 host-routing health finding without changing doctor’s
existing exit contract. Its ID is allocated from source at this task's
integration point, not pre-reserved in this plan.

- [ ] **Step 1: Allocate the doctor finding ID at integration.** After I108
  and I050 are merged, run `rg -o 'D[0-9]+' internal/doctor cmd/spine | sort
  -u` and inspect the latest commit log. Record the first unclaimed number in
  the task evidence before editing code. Use D16 only when both new checks
  precede I072; otherwise use the measured next free number. Never use D11 or
  reserve D14.

- [ ] **Step 2: Write failing doctor tests.** Assert no allocated finding when
  the file is absent. Assert one allocated-ID `error` with config path for malformed config,
  duplicate semantic route, unsupported schema, forbidden security member,
  unavailable pinned harness, missing executable, absent pinned model, and
  unsupported pin effort. Assert a valid no-pin config with the repository's
  requested pair unreachable produces one allocated-ID `warn` that names the repo,
  flavor, tier, and pair. Assert a valid pin without evidence refs is silent.

- [ ] **Step 3: Write the command-level result test.** `spine doctor` must
  print the allocated ID, path, and severity and exit 1 for either error or
  warn under the existing doctor convention. An absent config must not add an
  unrelated host-routing finding.

- [ ] **Step 4: Run focused tests red.**

  Run: `go test ./internal/doctor ./cmd/spine -run 'Test.*(HostRouting|Doctor.*Host)' -count=1`

  Expected: FAIL because no host-routing checker exists.

- [ ] **Step 5: Implement `hostRoutingCheck`.** Add it to `doctor.Run` after
  core repository checks. Reuse the same loader/validation path as model
  resolution. Return the allocated-ID error for invalid host state; inspect
  only the repository's active requested pair for the allocated-ID warn case.
  Do not inspect every unused harness and do not print JSON content.

- [ ] **Step 6: Verify green.**

  Run: `go test ./internal/doctor ./cmd/spine -run 'Test.*(HostRouting|Doctor)' -count=1`

  Expected: PASS, with I108's and I050's doctor findings unchanged.

- [ ] **Step 7: Commit the doctor unit.**

  Run: `git add internal/doctor/doctor.go internal/doctor/doctor_test.go cmd/spine/main_test.go && git commit -m 'feat(I072): diagnose host routing health'`

### Task 5: audit preflight, without I074 verdicts

**Files:**

- Modify: `internal/audit/audit.go` (`Options`, `Run`, and the
  flavor-tier resolver call path)
- Modify: `internal/audit/resolve_test.go`
- Modify: `cmd/spine/main.go` (`cmdAuditRouting` only to preserve exit/error
  handling if required)
- Modify: `cmd/spine/main_test.go`

**Consumes:** host-aware resolution and existing audit report/verdict types.

**Produces:** a configuration error before transcript traversal, never a
ticket evidence verdict for bad local routing state.

- [ ] **Step 1: Write failing audit tests.** Give audit a malformed or
  unreachable pinned host config and assert `audit.Run` returns a
  configuration error before transcript discovery, with zero ticket verdicts.
  At the CLI boundary assert `spine audit routing` exits 2. Give it a valid
  divergent Claude host pin and raw `observed_ids`; assert current
  source-derived audit behavior is retained and does not report a newly
  confirmed match or a new silent-descent verdict.

- [ ] **Step 2: Run focused tests red.**

  Run: `go test ./internal/audit ./cmd/spine -run 'Test.*(Host.*Audit|Audit.*Host|Routing.*Config)' -count=1`

  Expected: FAIL because audit currently resolves preference-only entries.

- [ ] **Step 3: Implement preflight only.** Validate the default or injected
  host path before transcript discovery. Thread final host-aware resolution
  where audit needs the final pair, but do not modify `transcriptFlavor`,
  aliases, transcript parsing, verdict names, or effort confirmation. Convert
  a host configuration failure to the existing command's exit-2 error route.

- [ ] **Step 4: Verify green.**

  Run: `go test ./internal/audit ./cmd/spine -run 'Test.*(Audit|Routing)' -count=1`

  Expected: PASS, including current I111 source/model derivation tests.

- [ ] **Step 5: Commit the audit unit.**

  Run: `git add internal/audit/audit.go internal/audit/resolve_test.go cmd/spine/main.go cmd/spine/main_test.go && git commit -m 'feat(I072): preflight host routing in audit'`

### Task 6: integration, review, verification, and closure

**Files:**

- Modify: `docs/issues/I072-host-config-schema-and-precedence.md`
- Modify: `CHANGELOG.md`
- Verify: `docs/specs/2026-08-29-host-routing-config-design.md`
- Verify: `docs/specs/2026-08-29-host-routing-config-plan.md`

- [ ] **Step 1: Run the full local suite and static checks.**

  Run: `gofmt -l internal/hostconfig internal/model internal/doctor internal/audit cmd/spine`

  Expected: no output.

  Run: `go test ./... -count=1`

  Expected: PASS.

  Run: `go vet ./...`

  Expected: PASS.

  Run: `git diff --check`

  Expected: exit 0.

- [ ] **Step 2: Run functional command simulations with a disposable config
  location injected by the test seam.** Prove: unconfigured resolution matches
  the prior ID/effort; a configured pin returns its final pair and JSON trail;
  an unreachable unpinned preference exits 2 with no stdout; doctor reports
  its integration-allocated host-routing finding; audit refuses a malformed pin before a verdict. Record commands and
  output in the implementation evidence.

- [ ] **Step 3: Run the repository gates.**

  Run: `make verify`

  Expected: PASS.

  Run: `spine doctor --dir .`

  Expected: record expected pre-existing findings separately from I072's
  integration-allocated doctor finding.

  Run: `spine audit routing --dir .`

  Expected: no new I072 configuration error in the unconfigured developer
  environment.

- [ ] **Step 4: Perform a fresh spec review.** A fresh primary-tier reviewer
  reads the finished diff and this PRD, attacks every requirement first, then
  verifies the ten acceptance criteria. The reviewer must specifically check
  the evidence-backed doctor-ID allocation, no implicit substitution, exact observed-ID behavior,
  secret-free output, host-blind update/mirror paths, no public rename, and
  no I074/I077 behavior. Resolve findings and rerun affected tests before
  approval.

- [ ] **Step 5: Perform independent verification.** A fresh primary-tier
  verifier reruns focused and full tests, static checks, functional probes,
  doctor, `spine audit routing`, and `spine audit stages`; it verifies staged
  paths and commit scope. Run `spine audit routing` with `--transcripts` when
  controller transcripts live outside this repository. Record its evidence.

- [ ] **Step 6: Close ticket and update docs.** Mark I072 fixed; write only
  actual commit IDs; add a `Resolution` with the allocated doctor ID, compatibility, security,
  I073/I074/I077 boundaries, review, and verification evidence. Add a concise
  `CHANGELOG.md` entry for final host routing and its doctor behavior. Do not claim
  an I074 confirmation or I077 evidence advisory.

- [ ] **Step 7: Commit docs and closure with explicit paths.**

  Run: `git add docs/issues/I072-host-config-schema-and-precedence.md CHANGELOG.md docs/specs/2026-08-29-host-routing-config-design.md docs/specs/2026-08-29-host-routing-config-plan.md && git commit -m 'docs(I072): close host routing configuration'`

- [ ] **Step 8: Ship only after the final exact SHA is verified.** Run the
  required `maipipe run full --wait` at that SHA, recheck `git status --short`,
  then follow the batch handoff's push and installed-binary instructions only
  when the owner has authorized the release step.

## Plan self-review checklist

- [x] Every PRD acceptance criterion maps to Tasks 1 through 6.
- [x] Every production seam named by the grill has a focused test-first task.
- [x] The doctor ID is intentionally allocated from integration state after
  I108 and I050, with D16 only an evidence-dependent expectation.
- [x] I073 public rename, I074 confirmation, and I077 evidence interpretation
  are fenced out of code tasks.
- [x] No task changes `models/defaults.json`, aliases, mirrors, update behavior,
  or normal config-path selection.
- [x] Placeholder scan found no TBD, TODO, or unspecified test step.
