# Flavor-to-harness migration (I073) implementation plan

> **For agentic workers:** use `superpowers:subagent-driven-development` or
> `superpowers:executing-plans` task by task. Do not write outside this
> repository or install a binary before the owner grants the separate boundary
> in the design.

**Goal:** rename the model table's first axis to harness, ship a compatible
generation-14 primary binary, then sweep the named primary estate without
changing route values or I072 host-routing behavior.

**Architecture:** canonical harness terminology runs through defaults, model
resolution, CLI, audit, templates, and update. Generation 14 retains the old
JSON key and legacy defaults input only as a narrow compatibility layer. The
rendered model mirror stays dotted, value-stable, and host-blind.

**Tech stack:** Go standard library, embedded JSON/templates, and existing
`spine update`, `spine doctor`, `spine model`, and `spine audit routing`.

**Spec:** `docs/specs/2026-08-30-flavor-to-harness-migration-design.md`

## Global constraints

- Do not start Task 1 until I072's independent verifier records PASS for the
  exact final SHA. Stop, rather than infer success, if that evidence is absent.
- This is routine-tier implementation. Final whole-branch review and fresh
  verification are primary-tier under `WORKFLOW.md`.
- Set `templates/VERSION` to 14. Do not revise host config v1 or the four
  harness values, model IDs, efforts, aliases, history, alternates, or mirrors.
- Preserve I072's `Load`, `--alternate`, D16, host-blind renderer/update, and
  preference-only audit behavior.
- In gen 14 output both JSON `harness` and deprecated equal `flavor`; accept
  exactly one defaults `harnesses` or legacy `flavors` key. Do not remove either.
- Do not stage worker reports, external files, caches, scratch data, or
  unrelated dirty paths with implementation commits.

## File map

| Files | Responsibility |
| --- | --- |
| `models/defaults.json`, `models/embed.go` | Canonical defaults keys and compatibility input coverage. |
| `internal/model/model.go`, `model_test.go` | Resolver API, strict launch policy, host trail, defaults decoding, mirror rows. |
| `internal/hostconfig/*`, `cmd/spine/*` | Unchanged host schema behavior, canonical CLI wording, JSON compatibility. |
| `internal/audit/*` | Harness names without changing source/transcript verdict behavior. |
| `templates/*`, `internal/tmpl/*`, `internal/scaffold/*` | Generation-14 generated wording. |
| `internal/update/*` and fixtures | Safe migration from every supported prior generation. |
| `README.md`, `CONTEXT.md`, `CHANGELOG.md`, I073, root generated files | Live documentation and root repository migration. |

### Task 1: Verify I072 and capture a no-write baseline

**Files:** read `docs/issues/I072-host-config-schema-and-precedence.md`, its
independent verifier report, `docs/specs/2026-08-29-host-routing-config-design.md`,
and the I073 design.

**Produces:** the exact prerequisite SHA and recorded proof that I073 leaves
host config v1 untouched.

- [ ] Confirm the verifier report says PASS and names the exact I072 SHA. If
  absent or mismatched, stop without editing.
- [ ] Run baseline fixture commands for model JSON, alternate JSON, model
  validate, and routing audit. Record stdout/stderr/exit status without copying
  host config content or secrets.
- [ ] Make no baseline-only commit.

### Task 2: Rename defaults and resolver vocabulary

**Files:** modify `models/defaults.json`, `models/embed.go`,
`internal/model/model.go`; test `internal/model/model_test.go`.

**Consumes:** I072's existing `ResolveForHost` behavior.

**Produces:** `Harnesses`, `Entry.Harness`, `LaunchRequest.Harness`,
harness-named helpers, and canonical defaults keys.

- [ ] Add table tests for canonical `harnesses`, legacy `flavors`, and
  both/neither failure. Assert all current resolved route values are identical.
- [ ] Run `go test ./internal/model -run 'Test.*(Harness|Flavor|Defaults|Resolve)' -count=1`.
  Expected: red because canonical fields and compatibility validation do not exist.
- [ ] Rename first-axis APIs, fields, helpers, comments, diagnostics, and
  `tierDefaultEffortByFlavor`. Add a compatibility envelope that accepts only
  one key, then validate one in-memory harness map. Change checked-in data to
  canonical keys.
- [ ] Preserve strict active resolution, I051 policy, legacy bare-tier reads,
  history/alias behavior, and mirror rendering byte values.
- [ ] Run `gofmt -w internal/model/model.go internal/model/model_test.go` and
  `go test ./internal/model -count=1`. Expected: green.
- [ ] Commit only these paths:

```bash
git add models/defaults.json models/embed.go internal/model/model.go internal/model/model_test.go
git commit -m "refactor(I073): name model table axis harness"
```

### Task 3: Migrate host terms and the CLI boundary

**Files:** modify/test `internal/hostconfig/*`, `cmd/spine/main.go`,
`cmd/spine/main_test.go`, `cmd/spine/strictargs_test.go`, and
`cmd/spine/i072_host_test.go`.

**Produces:** canonical `<harness>` usage/errors and additive JSON fields.

- [ ] Add failing tests that assert `harness == flavor == "claude"` in normal,
  no-host, pinned-host, and alternate JSON. Assert old fields keep their
  values, and usage/errors say harness.
- [ ] Reuse I072 fixtures for malformed config, missing/unavailable harness,
  unreachable route, and valid alternate. Assert their exit/stdout behavior is
  unchanged except allowed successful JSON naming fields.
- [ ] Run `go test ./cmd/spine ./internal/hostconfig -run 'Test.*(Model|Host|Alternate|Harness)' -count=1`.
  Expected: red before implementation.
- [ ] Rename host-config parameter/message terms only. Do not alter JSON-v1
  `harnesses`, add a path flag, apply host rules to alternates, or alter
  positional parsing. Add `harness` while retaining deprecated `flavor` in the
  model JSON formatter.
- [ ] Run `gofmt -w internal/hostconfig cmd/spine` then
  `go test ./cmd/spine ./internal/hostconfig -count=1`. Expected: green.
- [ ] Commit `internal/hostconfig` and `cmd/spine` as
  `feat(I073): expose harness naming in model CLI`.

### Task 4: Rename audit internals, retain transcript judgment

**Files:** modify `internal/audit/audit.go`, `codex.go`, `teamspawn.go`; test
`audit_test.go`, `resolve_test.go`, `i047_test.go`, `i111_test.go`,
`i072_host_test.go`, and `codex_test.go`.

**Produces:** harness-named audit code with source still separate.

- [ ] Add regressions that preserve old Claude/Codex verdicts and actual IDs.
  Cover I111 mixed Claude/openweights model-derived selection, D28
  source-qualification, and source tiebreak on deliberate collision.
- [ ] Run `go test ./internal/audit -run 'Test.*(I111|I072|Codex|Flavor|Harness|Routing)' -count=1`.
  Expected: red before canonical names/assertions exist.
- [ ] Rename `evidenceToken.flavor`, `deriveFlavor`, `transcriptFlavor`, maps,
  comments, and user-visible first-axis wording. Do not rename source, parse
  a new transcript field, change D28, or use host-aware mapping in audit.
- [ ] Run `gofmt -w internal/audit` and `go test ./internal/audit -count=1`.
  Expected: stable verdict tokens, source behavior, and raw model strings.
- [ ] Commit `internal/audit` as
  `refactor(I073): use harness terminology in routing audit`.

### Task 5: Ship generation 14 and migration guards

**Files:** modify `templates/VERSION`, current templates, `internal/tmpl/*`,
`internal/scaffold/*`, `internal/update/{modelrouting,keys,update,diff}.go`,
and applicable generation fixtures/tests.

**Produces:** generation-14 templates and safe updates from old generations.

- [ ] Add failing fixtures at generations 10, 11, 12, and 13. Assert dry run
  recognition, write to 14, canonical routing-harness wording, unchanged
  `functional_harness`, identical mirror values, no host content, and a
  byte-stable second update.
- [ ] Add tests that a dotted override survives and an unrelated hand edit is
  reported and not written without `--force`.
- [ ] Run `go test ./internal/tmpl ./internal/scaffold ./internal/update -run 'Test.*(Gen|Update|Mirror|Harness)' -count=1`.
  Expected: red while generation remains 13.
- [ ] Set version 14. Update template prose and generation-diff recognition.
  Rename update/parser internals but keep dotted syntax and historic generated
  line recognition. Do not add host-dependent rendering.
- [ ] Run `gofmt -w internal/tmpl internal/scaffold internal/update`, then
  `go test ./internal/tmpl ./internal/scaffold ./internal/update -count=1` and
  `go test ./... -count=1`. Expected: green.
- [ ] Commit templates and implementation as
  `feat(I073): ship generation 14 harness terminology`.

### Task 6: Update live documentation and root generated files

**Files:** modify `README.md`, `CONTEXT.md`, `CHANGELOG.md`, I073, root
`WORKFLOW.md`, `AGENTS.md`, `CLAUDE.md`, and only generated paths reported by
the root update.

- [ ] Search active docs for `flavor`. Classify each hit as gen-14 compatibility
  or historical evidence. Keep I112's openweights warning.
- [ ] Run `go run ./cmd/spine update --dir .`, review every path, and stop if
  unrelated work appears.
- [ ] Run `go run ./cmd/spine update --dir . --write`, then doctor, model JSON,
  and model validate at root. Expected: generation 14, equal JSON fields, no
  new doctor finding, and unchanged validation route.
- [ ] Commit only reviewed active docs/generated paths as
  `docs(I073): document harness migration compatibility`.

### Task 7: Whole-branch verification and release gate

**Files:** read final diff, this plan, and the design.

- [ ] Run `gofmt -w models internal cmd templates`, `git diff --check`,
  `go test ./... -count=1`, `go vet ./...`, and `go build ./cmd/spine`.
  Expected: all zero.
- [ ] Build an isolated candidate binary. On temporary fixtures prove text,
  effort, JSON compatibility, alternate, validate, malformed-host refusal, and
  no-host behavior. Do not install it over the owner binary.
- [ ] Run scoped `spine audit routing --dir .` and `spine audit stages --dir .`.
  Record standing findings separately from regressions.
- [ ] Obtain fresh primary requirements-attack review against every PRD
  criterion. It must check delayed removal, I072 preservation, source/harness
  separation, generation 14, and no premature fleet write.
- [ ] Obtain independent primary verification with exact SHA and raw command
  transcripts. Resolve all blockers before requesting fleet authorization.
- [ ] Commit ticket closure evidence only after both gates pass. Do not sweep
  the fleet in this task.

### Task 8: Owner-authorized fleet sweep

**Files:** per named primary, only generated paths from its reviewed update;
create migration-only commit and fleet ledger entry.

- [ ] Obtain an explicit owner authorization naming the candidate binary/commit
  and roster sequence. It is separate from the implementation authorization.
- [ ] For each primary in the design order, record status, branch, HEAD,
  generation, and excluded worktrees. Stop if state differs from preflight.
- [ ] Review `spine update --dir <repo>` before every write. An unrecognized
  report needs an owner decision; never use `--force`.
- [ ] Write one repository, commit only reviewed generated paths, then run
  doctor, all declared harness/tier model text/effort/JSON/validate checks, and
  a scoped routing audit where transcripts exist. Record outputs and exit codes.
- [ ] On the first unexpected result stop. Revert only that migration commit,
  rerun its verification with the prior binary, and correct spine before retry.
- [ ] Before considering removal, prove 20 primary results, explicit worktree
  exclusions, all at gen 14, and JSON compatibility consumer checks. This does
  not authorize the later removal effort.
