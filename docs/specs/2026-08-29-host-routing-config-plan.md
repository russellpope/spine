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
final pairs. Plain model output may consume that host-aware result. Controlled
`model validate` validates its strict I051 repository route, then permits only
an absent host pin or a pin whose model ID is byte-identical to that active
route. Doctor checks declared available harnesses and diagnoses divergent pins
deterministically. Audit performs structural config preflight only. Update and
template rendering stay outside every host-aware call path.

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
  Its platform lookup may read standard configuration environment state; JSON
  values never expand environment input, execute, or make a network request.
  Tests use private argument-based directory-provider helpers or absolute
  fixture paths. Never mutate a package global in a test. Do not add a normal
  path flag or environment override.
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
- The host file is a trusted owner/fleet-managed authority. I072 proves route
  feasibility only. It does not ratify pins or require evidence refs.
- Audit validates declared config and pins before Claude transcript discovery,
  but retains its preference-only mappings and every verdict/output byte. It
  does not test unpinned reachability or host-to-observed conformance.
- Until I074 adds host conformance to audit, a safe divergent pin is valid for
  plain host-aware model inspection but is not a controlled launch route.
  `spine model [--dir REPO] validate ...` refuses it with exit 2 and no stdout.
  A pin whose model ID is byte-identical to the repository active ID may
  validate. No `--expect` value bypasses this gate.
- Do not change templates or template generation.
- Stage explicit paths only. Do not stage `.cache/`,
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`, concurrent code,
  or scratch files.

## File map

| File | Responsibility |
| --- | --- |
| `internal/hostconfig/hostconfig.go` | Schema-v1 types; default-path helper, `Load`, and validation for a trusted local host file. |
| `internal/hostconfig/hostconfig_test.go` | Parser, nested closed-schema, security boundary, route, observed-ID, and injected executable/path tests. |
| `internal/model/model.go` | Existing `Resolve` stays preference-only; host-aware plain resolution returns requested and final pairs, while I051 controlled validation refuses divergent pins until I074. |
| `internal/model/model_test.go` | Precedence, I051 compatibility, one-result consumption, no-substitution, pi, and alternate negative controls. |
| `cmd/spine/main.go` | `cmdModel` and `cmdModelValidate` print or consume the final pair; audit CLI keeps its existing output path. |
| `cmd/spine/main_test.go` | CLI success, compatibility, final-ID expectation, error, stdout, and doctor exit tests. |
| `internal/doctor/doctor.go` | `hostRoutingCheck` adds the integration-allocated I072 finding without writes. |
| `internal/doctor/doctor_test.go` | Absent, config-error, and every-available-harness/tier reachability tests. |
| `internal/audit/audit.go` | Structural host-config preflight before transcript discovery, without host-aware audit mappings. |
| `internal/audit/resolve_test.go` | Claude-only preflight and byte-compatible verdict/output regression tests. |
| `docs/specs/2026-08-29-host-routing-config-design.md` | Binding PRD for implementation and final spec review. |
| `docs/issues/I072-host-config-schema-and-precedence.md` | Closure, commit IDs, review evidence, and implementation resolution. |
| `CHANGELOG.md` | Consumer-visible `spine model` and doctor behavior after the feature ships. |

## Interfaces locked by this plan

Implement these names and keep their responsibilities narrow:

```go
// internal/hostconfig
var ErrNotConfigured error

func DefaultPath() (string, error)
func Load(path string, flavors []string,
    lookPath func(string) (string, error)) (Config, error)

// internal/model
type Resolution struct {
    Entry     Entry          // final pair, preserves existing Entry output fields
    Requested Entry          // estate + repository preference before host constraint
    Host      HostResolution // unconfigured, reachable, or pinned trail
}

func ResolveForHost(repoDir, configPath, flavor, tier string,
    lookPath func(string) (string, error)) (Resolution, error)

func ValidateLaunchForHost(req LaunchRequest, configPath string,
    lookPath func(string) (string, error)) (Resolution, error)
```

`hostconfig.Load` is the sole exported host-config validation boundary. It
accepts the current flavor vocabulary because closed-schema validation must
reject a harness or pin for a flavor the embedded model table does not expose.
For every present file, it detects duplicate JSON members; decodes the closed
schema; validates semantic routes, pins, and safe strings; then resolves each
available harness executable through `lookPath`. Its private decode, semantic,
and executable-validation helpers are not exported. This keeps callers from
obtaining a parsed config that has bypassed any part of the validation boundary
and avoids a model-package dependency cycle. `ErrNotConfigured` remains the
only absent-file result.

`hostconfig.DefaultPath` delegates to an unexported pure helper that accepts a
directory-provider function. Production passes `os.UserConfigDir`; tests pass
a closure. `ResolveForHost` uses `DefaultPath` only when `configPath` is empty;
production callers pass empty and tests pass an absolute fixture path. Doctor
and audit use unexported `runWithHostPath` helpers that take the same explicit
path. No test replaces global state, so parallel model, doctor, and audit tests
remain race-safe. An absent file causes the legacy result, not an error. All
present files pass the complete `Load` boundary before a final pair is returned.

`ValidateLaunchForHost` is the only host-aware path for controlled
`spine model [--dir REPO] validate`. It reads I051's strict repository
snapshot once, reads the host file once, applies the I051 safe-ID and deny
policy to both the requested and pinned IDs, and returns one `Resolution` only
when the pin's model ID is absent or byte-identical to the repository active
ID. A divergent pin returns a configuration error before `cmdModelValidate`
prints stdout. Plain `ResolveForHost` may still return that divergent final
pair for inspection. Existing `ValidateLaunch` stays the no-host I051
compatibility path. I074 may later relax only this identity gate after it
supplies an auditable host-active vocabulary.

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

**Produces:** schema-v1 `Config`, `DefaultPath`, `Load(path, flavors, lookup)`,
and `ErrNotConfigured` for model, doctor, and audit callers. `Load` owns the
complete exported validation boundary; no exported `Validate` is part of I072.

- [ ] **Step 1: Write failing parser tests.** Cover the private platform-path
  helper by injecting the user config directory function and expecting
  `<config-dir>/spine/routing-host.json`; an absent file returns exactly
  `ErrNotConfigured`; a valid Claude config has route
  `gpt-5.6-sol @ high`; a pin key `claude.primary` validates; and `Load` does
  not create a file.

- [ ] **Step 2: Write failing schema-boundary tests.** Reject malformed JSON,
  `schema_version: 2`, unknown root/harness/route/pin members at every nested
  object, duplicate JSON object keys, control characters, empty IDs, invalid
  or unknown-flavor dotted pin keys,
  unknown tiers, duplicate effort strings, duplicate observed IDs across the
  config, unavailable pin harnesses, absent pin model, and a pin effort not in
  its route. Prove equal routes in distinct harnesses are allowed: there is no
  duplicate-semantic-route error. Include table cases for `token`, `base_url`, `auth_header`,
  `credentials`, `modelOverrides`, `args`, and `env`; every case must fail.

- [ ] **Step 3: Run the focused tests red.**

  Run: `go test ./internal/hostconfig -run 'Test(DefaultPath|Load|Validate)' -count=1`

  Expected: fail because the package and symbols do not exist.

- [ ] **Step 4: Implement the minimum closed-schema parser.** Decode JSON in
  a way that detects duplicate object keys and rejects unknown members at every
  object depth. Use `os.UserConfigDir` only through the private
  directory-provider helper, injected lookup for tests, and `exec.LookPath`
  only to test executable presence. Do not execute the result. Make a present
  but unreadable file, invalid schema, unavailable pinned harness, invalid
  declared executable, or absent pin route an error with its config path and
  safe field name only.

- [ ] **Step 5: Verify green and prove the security boundary.**

  Run: `go test ./internal/hostconfig -count=1`

  Expected: PASS. Add a test lookup that records calls and proves validation
  performs only an executable lookup, not a shell execution or network action.
  Keep the attack table explicit for every prohibited field: `token`,
  `base_url`, `auth_header`, `credentials`, `modelOverrides`, `args`, and
  `env` each appears in an otherwise valid fixture and each call to `Load`
  fails with the fixture path in its diagnostic.

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

- [ ] **Step 2: Write failing I051 compatibility tests.** Start from a strict
  I051 repository snapshot, then apply host fixtures. Assert plain
  `ResolveForHost` returns a safe divergent final pair for inspection even
  though its ID is not a repository override, while `ValidateLaunchForHost`
  refuses that same pin as not yet auditable. Assert neither requested nor
  final `--expect` semantics can bypass the refusal. Assert a pin whose model
  ID is byte-identical to the repository active ID validates, a forbidden pin
  fails the I051 positive-ID/deny policy, and the no-host result keeps I051's
  current output and exit behavior byte-for-byte.

- [ ] **Step 3: Add host-blind negative controls.** Use a valid host fixture
  with a divergent Claude pin, then prove `MirrorRows()`,
  `update.applyModelRouting`, and rendered `WORKFLOW.md` still contain only
  the embedded/repository result. The host fixture must not create a refresh
  or an override item.

- [ ] **Step 4: Run focused tests red.**

  Run: `go test ./internal/model ./internal/update -run 'Test.*(Host|Mirror|ModelRouting)' -count=1`

  Expected: FAIL because host-aware resolution and its trail do not yet exist.

- [ ] **Step 5: Implement `ResolveForHost` and the I051 adapter.** First call
  existing `Resolve` to obtain the ordinary requested entry. For validation,
  first use I051's strict one-snapshot repository reader and policy, then
  apply the host constraint to that validated requested entry in the same
  process. Validate a pin under the same I051 ID policy. Return it to plain
  host-aware resolution, but make controlled validation reject it as not yet
  auditable when its model ID differs from the repository active ID. Permit an
  identical pin and compare `--expect` only after that gate. Map
  `ErrNotConfigured` to an `unconfigured` trail and retain exact I051 no-host
  behavior. For a present config, require the selected flavor-named harness
  and executable, apply an exact tier pin when present, otherwise require the
  requested model and effort route. Keep `Resolve`, `MirrorRows`, and
  `applyModelRouting` free of hostconfig imports.

- [ ] **Step 6: Verify green.**

  Run: `go test ./internal/model ./internal/update -count=1`

  Expected: PASS, including existing historical-default and update tests.

- [ ] **Step 7: Commit the resolver unit.**

  Run: `git add internal/model/model.go internal/model/model_test.go internal/update/modelrouting_test.go && git commit -m 'feat(I072): resolve host routing constraints'`

### Task 3: model command trail and failure contract

**Files:**

- Modify: `cmd/spine/main.go` (`cmdModel`)
- Modify: `cmd/spine/main_test.go`

**Consumes:** `model.ResolveForHost`, the I051-compatible final launch result,
and existing `--json`, `--effort`, and `--alternate` flag behavior.

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
  Capture legacy `--alternate --json` bytes before the host fixture is
  introduced, then assert byte identity both when the host file is absent and
  when a valid present fixture names a different or missing selected route. In
  both cases, the JSON must contain neither `requested` nor `host` nor `pin`.
  A malformed present fixture still exits 2 with no stdout for alternate text,
  effort, and JSON modes.

- [ ] **Step 3: Add controlled validation command tests.** Through
  `spine model [--dir REPO] validate`, assert a safe divergent pin exits 2,
  prints no stdout, names the not-yet-auditable host divergence, and never
  reaches the fake launcher. Assert both requested and final `--expect` values
  still refuse. Assert a byte-identical pin validates, a forbidden pin returns
  the I051 refusal with no stdout, and an absent file preserves I051 bytes and
  exits. Do not permit fallback to plain `spine model`.

- [ ] **Step 4: Run the focused tests red.**

  Run: `go test ./cmd/spine -run 'TestModel.*(Host|Config|JSON|Alternate|Flag)' -count=1`

  Expected: FAIL because `cmdModel` still calls preference-only `Resolve`.

- [ ] **Step 5: Implement the thin CLI adapters.** Replace the ordinary
  resolution call with `ResolveForHost`; adapt `cmdModelValidate` to the
  I051-compatible final result. Keep flag parsing in `cmdModel`, select final
  entry for text and effort output, and marshal additive JSON. Return before
  emitting any success output on host errors. Never read config contents into
  command output.

- [ ] **Step 6: Verify green.**

  Run: `go test ./cmd/spine -run 'TestModel' -count=1`

  Expected: PASS.

- [ ] **Step 7: Commit the CLI unit.**

  Run: `git add cmd/spine/main.go cmd/spine/main_test.go && git commit -m 'feat(I072): expose host routing resolution'`

### Task 4: doctor checks with an integration-allocated finding ID

**Files:**

- Modify: `internal/doctor/doctor.go` (`Run`, new `hostRoutingCheck`)
- Modify: `internal/doctor/doctor_test.go`
- Modify: `cmd/spine/main_test.go`

**Consumes:** shared hostconfig validation and preference-only resolution.
Doctor receives an explicit fixture path through its private helper in tests.

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
  the file is absent. Assert allocated-ID errors with config path for malformed
  config, nested unknown members, unsupported schema, forbidden security
  member, unavailable pinned harness, invalid declared executable, absent
  pinned model, and unsupported pin effort. Assert a Claude-only config checks
  all four Claude tiers and produces one warning per unreachable unpinned pair.
  Assert a two-available-harness config checks every tier of both available
  harnesses in lexical harness order, then `model.Tiers` order. Assert an unavailable declared
  harness produces no preference warnings, a divergent valid pin produces a
  not-yet-auditable warning for its matching tier, a byte-identical pin
  suppresses only its matching preference warning, and a valid identical pin
  without evidence refs is silent. Use two independently-created explicit
  fixture paths in `t.Parallel` subtests, each with a distinct lookup closure,
  and assert their findings retain their own paths. Run that test under
  `-race` to prove the path seam has no global-state race.

- [ ] **Step 3: Write the command-level result test.** `spine doctor` must
  print the allocated ID, path, and severity and exit 1 for either error or
  warn under the existing doctor convention. An absent config must not add an
  unrelated host-routing finding.

- [ ] **Step 4: Run focused tests red.**

  Run: `go test ./internal/doctor ./cmd/spine -run 'Test.*(HostRouting|Doctor.*Host)' -count=1`

  Expected: FAIL because no host-routing checker exists.

- [ ] **Step 5: Implement `hostRoutingCheck`.** Add it to `doctor.Run` after
  core repository checks. Reuse the same loader/validation path as model
  resolution. Return allocated-ID errors for invalid host state. For every
  available declared harness and every model tier, resolve the repository
  preference without host substitution and return one allocated-ID warning for
  each unpinned unreachable pair. Ignore absent and unavailable harnesses; do
  not print JSON content.

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

**Consumes:** hostconfig structural validation and existing preference-only
audit mappings and report/verdict types.

**Produces:** a configuration error before transcript traversal, never a
ticket evidence verdict for bad local routing state.

- [ ] **Step 1: Write failing audit tests.** Give audit a malformed config,
  nested unknown member, invalid declared executable, or unreachable pin and
  assert `audit.Run` returns a configuration error before Claude transcript
  discovery, with zero ticket verdicts. At the CLI boundary assert `spine audit
  routing` exits 2. Give it a valid Claude-only config with an unreachable
  unpinned repository preference and assert audit proceeds. Give it a valid
  divergent Claude pin and raw `observed_ids`; assert current preference-only
  mappings, verdicts, report text, and exit result are byte-for-byte unchanged.
  Use an explicit audit fixture path and a transcript-read recorder to prove
  preflight ordering without mutable global test state.

- [ ] **Step 2: Run focused tests red.**

  Run: `go test ./internal/audit ./cmd/spine -run 'Test.*(Host.*Audit|Audit.*Host|Routing.*Config)' -count=1`

  Expected: FAIL because audit currently has no host-config preflight.

- [ ] **Step 3: Implement preflight only.** Validate the default or injected
  path before Claude transcript discovery. Enforce the closed config schema,
  declared executables, and exact pins on available declared harnesses. Do not
  call `ResolveForHost`, change `resolveFlavorTiers`, modify
  `transcriptFlavor`, aliases, transcript parsing, verdict names, output, or
  effort confirmation. Convert a host configuration failure to the existing
  command's exit-2 error route. Unpinned reachability and all declared-to-
  observed correlation remain I074 work.

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
  the prior ID/effort and I051 validation bytes; plain model output exposes a
  safe divergent pin and its JSON trail; controlled validation refuses that
  pin for both requested and final `--expect`; an identical pin validates; a
  forbidden pin refuses; doctor reports all reachable and unreachable
  available-harness tiers with one warning per unpinned failure and diagnoses
  divergent pins;
  audit refuses a malformed pin before a Claude transcript read but proceeds
  for a valid unreachable unpinned preference without changing verdict bytes.
  Record commands and output in the implementation evidence.

- [ ] **Step 3: Run the repository gates.** Run the actual repository
  sequence, since this Makefile has no `verify` target:

  ```bash
  go test ./internal/hostconfig ./internal/model ./internal/doctor ./internal/audit ./cmd/spine -count=1
  go test -race ./internal/hostconfig ./internal/model ./internal/doctor ./internal/audit ./cmd/spine -count=1
  go test ./... -count=1
  go vet ./...
  go build -o bin/spine ./cmd/spine
  ./bin/spine doctor --dir .
  ./bin/spine audit routing --dir .
  ./bin/spine audit stages --dir .
  gofmt -l internal/hostconfig internal/model internal/doctor internal/audit cmd/spine
  git diff --check
  maipipe run full --wait
  ```

  Expected: Go tests, race tests, vet, build, routing and stage audits, format,
  diff, and the final exact-SHA maipipe lane pass. Record expected pre-existing
  doctor findings separately from I072's integration-allocated finding.

- [ ] **Step 4: Perform a fresh spec review.** A fresh primary-tier reviewer
  reads the finished diff and this PRD, attacks every requirement first, then
  verifies the twelve acceptance criteria. The reviewer must specifically check
  the evidence-backed doctor-ID allocation, every-available-harness/tier
  matrix, I051 final-pair/expect behavior, no implicit substitution, nested
  schema closure, injected-path race safety, exact observed-ID behavior,
  secret-free output, host-blind update/mirror paths, no template change, no
  public rename, controlled refusal of divergent pins before I074, identical
  pin validation, and no I074/I077 behavior. Resolve findings and rerun
  affected tests before approval.

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
- [x] Every production seam named by the grill has a focused test-first task,
  including private race-safe directory and explicit-path seams.
- [x] The doctor ID is intentionally allocated from integration state after
  I108 and I050, with D16 only an evidence-dependent expectation.
- [x] I073 public rename, I074 reachability/conformance verdicts, and I077
  evidence interpretation are fenced out of code tasks.
- [x] No task changes `models/defaults.json`, aliases, mirrors, update behavior,
  or normal config-path selection.
- [x] Placeholder scan found no placeholder markers or unspecified test steps.
