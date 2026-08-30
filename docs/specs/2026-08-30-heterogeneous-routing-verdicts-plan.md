# Heterogeneous routing verdicts (I074) implementation plan

> **For agentic workers:** Start only after I075 is merged and reviewed. Use
> fresh routine-tier implementation and review workers. Every dispatch names
> `I074`, the primary repository, and a complete recorded declaration. Pass
> effort only through a verified transport field.

**Goal:** Judge explicit heterogeneous declarations as confirmed, mismatch, or
unconfirmable through exact host-local correlation while preserving legacy
routing audit behavior.

**Architecture:** `internal/audit` adds a per-event declaration judgment that
uses I075 state, final host-aware model resolution, and I072 observed-ID
routes. Model and effort observations remain separate exact facts before the
ticket aggregator selects the strongest result.

**Tech stack:** Go standard library, `internal/audit`, `internal/model`,
`internal/hostconfig`, existing Claude/Codex JSONL readers.

**Spec:** `docs/specs/2026-08-30-heterogeneous-routing-verdicts-design.md`

## Global constraints

- I075 and verified I072 are hard implementation predecessors. Do not
  reimplement I075 parsing, raw effort validation, exact-pair authorization,
  or I072 host validation in I074.
- I072's verified host configuration is the only observed-ID authority. Use
  the final host target, never the repository preference, for a pin.
- Require complete `(source, session, dispatch)` identity and linked worker
  event evidence. Coarse Codex root linkage may not confirm a declaration.
- No transcript field currently proves effort. Default I074 output remains
  observed-effort `-` until a documented extractor and fixtures exist.
- Keep public flavor spelling, old model-only verdicts, table prefixes,
  aliases/history, model-tier records, and host-blind behavior compatible.
  I073 owns public renaming after verified I072.
- A malformed host configuration remains preflight exit 2. It must never
  degrade into an `unconfirmable` ticket verdict.
- Stage only task files. Do not stage reports, host files, or unrelated work.

## File map

| File | Responsibility |
| --- | --- |
| `internal/audit/audit.go` | Event evidence, exact correlation, new verdicts, severity, aggregation, and report types. |
| `internal/audit/codex.go` | Preserve complete event identities while exposing supported observed fields. |
| `internal/audit/teamspawn.go` | Preserve Claude event identity and add an extractor only with documented evidence. |
| `internal/audit/i072_host_test.go` | Host-pinned, observed-ID, preflight, and no-host regressions. |
| `internal/audit/*_test.go` | Matrix, correlation, legacy compatibility, and blocking tests. |
| `internal/model/model.go` | Narrow final-host-target seam only if audit cannot consume an existing safe resolution API. |
| `cmd/spine/main.go` | Additive audit output and exit adaptation. |
| `cmd/spine/main_test.go` | Leading-column, JSON, and blocking compatibility tests. |
| `WORKFLOW.md` and template sources | Publish exact accepted grammar and verdict wording after code is proven. |

## Interfaces locked by this plan

```go
type DeclarationEvidence struct {
    Identity        evidenceIdentity
    Harness         string
    Model           string
    ExpectedModel   string
    ExpectedEffort  string
    DeclaredEffort  string
    ObservedModel   string
    ObservedEffort  string
}

type DeclarationVerdict string

const (
    DeclarationConfirmed          DeclarationVerdict = "confirmed"
    DeclarationObservedMismatch   DeclarationVerdict = "declared-observed-mismatch"
    DeclarationEffortMismatch     DeclarationVerdict = "declared-effort-mismatch"
    DeclarationUnconfirmable      DeclarationVerdict = "unconfirmable"
)
```

The exact Go names may change to fit current package style. The fields and
semantics may not: `ExpectedModel` and `ExpectedEffort` are the final
host-selected route, raw values remain distinct, absence is explicit, and a
correlated observation is never synthesized from a declaration. A private
host lookup returns exactly one route for a raw observed ID or no route; it
never searches aliases, history, or another host.

## Task 1: establish exact host-route and identity correlation

**Files:** `internal/audit/audit.go`, `internal/audit/i072_host_test.go`,
`internal/audit/codex_test.go`, `internal/audit/i090_test.go`

- [ ] Write failing tests for a host pin whose expected final pair differs
  from the repository preference, a raw observed ID exact-mapped to that
  pin, a raw ID exact-mapped to another route, an unmapped raw ID, and no
  host file.
- [ ] Write failing identity attacks: source/session match with another
  dispatch, root-only Codex linkage, another ticket's worker, partial identity,
  and two events in one session. None may confirm the target.
- [ ] Build a private exact observed-ID index from the validated host config.
  Resolve expected target through the final host route. Require the same
  harness and complete identity before consuming a worker model field.
- [ ] Run `go test ./internal/audit -run 'Test.*(Host|Observed|Identity|Pin)' -count=1`
  and then `go test ./internal/audit -count=1`.
- [ ] Commit this isolated unit with explicit paths.

## Task 2: add the declaration verdict matrix

**Files:** `internal/audit/audit.go`, `internal/audit/audit_test.go`,
`internal/audit/resolve_test.go`

- [ ] Write table-driven failing tests for every row of the design matrix.
  Include exact target effort, an I075-authorized retry, an unauthorized
  declaration, model-confirmed/no-observed-effort, model mismatch, and model
  unconfirmable cases.
- [ ] Keep every current parser fixture without a separately approved,
  documented observed-effort extractor unconfirmable. Do not create a
  synthetic observed-effort fixture. A later extractor contract must supply
  durable field-semantics and identity-correlation evidence before it can add
  the confirmed and observed-effort-mismatch rows to executable coverage.
- [ ] Add the four declaration verdicts, their severity entries, strongest
  event aggregation, and `Report.Blocking()` handling. Preserve existing
  verdict ordering and legacy token judgment.
- [ ] Run `go test ./internal/audit -run 'Test.*(Declaration|Verdict|Blocking|Effort)' -count=1`
  and then `go test ./internal/audit -count=1`.
- [ ] Commit this isolated unit with explicit paths.

## Task 3: publish additive, explainable output

**Files:** `cmd/spine/main.go`, `cmd/spine/main_test.go`, `internal/audit/audit.go`

- [ ] Add failing text and report-structure tests proving legacy rows retain
  their current leading columns and fields. Add a declared heterogeneous
  fixture whose trailing details expose expected pair, declared triple,
  per-field states, and complete correlation identity.
- [ ] Implement additive text columns and structured in-memory event details.
  Print `-` for absent observation. Do not label it a default or a confirmed
  effective effort. Do not add an audit JSON flag.
- [ ] Add end-to-end tests that a mismatch exits blocking, unconfirmable exits
  nonblocking, and a legacy silent descent remains blocking.
- [ ] Run `go test ./cmd/spine ./internal/audit -count=1`.
- [ ] Commit this isolated unit with explicit paths.

## Task 4: document and verify the accepted behavior

**Files:** `WORKFLOW.md`, `templates/current/WORKFLOW.md.tmpl`,
`docs/issues/README.md`, `templates/current/issues-README.md`, `CONTEXT.md`,
`docs/issues/I074-audit-heterogeneous-verdicts.md`

- [ ] Document exact host-local model confirmation, observed-effort limits,
  matrix verdicts, and block/nonblock behavior. Retain exact I075 effort
  authorization grammar without adding an order claim.
- [ ] Record I073 as the owner of public flavor-to-harness migration. Do not
  introduce a renamed CLI spelling or fleet migration.
- [ ] Run focused audit and CLI tests, `go test ./...`, and
  `spine audit routing` with the required transcript scope.
- [ ] Perform the required requirements-attack spec review against I074 and
  I075. A fresh verifier reviews the final diff before ship.
