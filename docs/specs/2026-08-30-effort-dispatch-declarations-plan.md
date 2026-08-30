# Effort dispatch declarations (I075) implementation plan

> **For agentic workers:** Use a fresh routine-tier implementation worker and
> a fresh review worker. Every dispatch names `I075`, the primary repository,
> and a complete recorded declaration. Pass effort only through a verified
> transport field. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make raw effort an explicit, validated per-dispatch declaration
without claiming a provider-observed effort.

**Architecture:** `internal/model` resolves one final target and validates a
raw requested effort. `internal/audit` records exact declarations and exact
effort-record authorization while leaving current model judgment unchanged.
The CLI exposes a JSON-only target helper and appends compatible audit data.

**Tech stack:** Go standard library, existing model table and host-resolution
seams, JSONL transcript readers, Markdown workflow grammar.

**Spec:** `docs/specs/2026-08-30-effort-dispatch-declarations-design.md`

## Global constraints

- Implement I075 before I074. I074 must consume I075's declared-effort seam.
- Preserve public `flavor` spellings and old CLI/output names. I073 owns the
  public rename and begins only after verified I072.
- Use raw family-specific tokens. Do not rank, normalize, translate, or infer
  effort values.
- Preserve `Resolve`, `--effort`, `--alternate`, I051 launch validation,
  host-blind mirrors, existing model verdicts, and legacy model-only output.
- Host-aware work consumes a final target only. It does not read a host file
  directly or relax I072's divergent-pin controlled-launch restriction.
- Never invent wrapper, gateway, or Agent-tool arguments. Do not expose a
  provider-effective effort until I074 has documented an exact source.
- Stage only the files named by the implementation tasks. Do not stage
  `.superpowers/sdd/` reports or unrelated research files.

## File map

| File | Responsibility |
| --- | --- |
| `internal/model/model.go` | Final-target dispatch resolver and raw effort validation. |
| `internal/model/model_test.go` | Omission, selected-flavor vocabulary, provenance, pin, and compatibility tests. |
| `cmd/spine/main.go` | JSON-only `--dispatch-effort` grammar and compatible target output. |
| `cmd/spine/main_test.go` | CLI grammar, stdout/stderr, and legacy byte tests. |
| `internal/audit/audit.go` | Explicit harness/model/effort declaration state, strict effort ledger parsing, attribution, and report fields. |
| `internal/audit/codex.go` | Explicit Codex `reasoning_effort` extraction. |
| `internal/audit/teamspawn.go` | Attributed Claude `--effort` extraction. |
| `internal/audit/*_test.go` | Parser, retry, authorization, linked-actual, and compatibility tests. |
| `WORKFLOW.md` and generated-template sources | Exact I075 raw-token, declaration, and authorization grammar. |

## Interfaces locked by this plan

```go
type DispatchTargetRequest struct {
    RepoDir         string
    Flavor          string
    Tier            string
    RequestedEffort string
}

func ResolveDispatchTarget(req DispatchTargetRequest) (Entry, error)
```

`RequestedEffort == ""` means omitted. The function returns the selected
target effort and preserves the existing model ID, aliases, alternate, and
provenance. A host-aware caller passes an already selected final entry through
the same private effort-override helper. The implementation may choose a more
idiomatic exported name, but it must retain these semantics and not overload
`Resolve`.

```go
type effortEscalation struct {
    from   string
    to     string
    reason string
    line   int
}
```

The audit ledger indexes this by ticket and tests an exact `(expected,
declared)` pair. It is separate from `escRecord` and cannot change model-tier
judgment.

```go
type declaredDispatch struct {
    Harness string
    Model   string
    Effort  string
    Source  string
}
```

`Harness` is a raw controller-record field, not a value inferred from an
observed model or transcript layout. Each controlled transport writes the
field in its dispatch metadata before launch. A legacy record without it has
no complete I075 declaration and stays on the legacy model-only path. The
transport source may diagnose that absence but cannot fill it in.

## Task 1: resolve and validate a final dispatch target

**Files:** `internal/model/model.go`, `internal/model/model_test.go`

- [ ] Add failing tests for omitted effort inheriting the final target effort,
  a raw valid override retaining ID/provenance, whitespace-only rejection,
  an invalid Pi token, and unchanged `Resolve`/alternate behavior.
- [ ] Add a final-entry helper that validates a non-empty raw override with
  the selected flavor's existing vocabulary. Keep the token byte-exact.
- [ ] Implement the request resolver on top of the ordinary selected route.
  When I072 integration is available, call the final-entry helper after host
  selection rather than duplicating precedence.
- [ ] Run `go test ./internal/model -run 'Test.*(Dispatch|Effort|Host)' -count=1`
  and then `go test ./internal/model -count=1`.
- [ ] Commit this isolated unit with explicit paths.

## Task 2: expose a compatibility-preserving JSON target helper

**Files:** `cmd/spine/main.go`, `cmd/spine/main_test.go`

- [ ] Add failing command tests for `spine model --json --dispatch-effort low
  <flavor> <tier>`, invalid selected-flavor effort, whitespace-only effort,
  use without `--json`, flags after positionals, and no stdout on failure.
- [ ] Capture and assert byte identity for existing text, `--effort`, JSON,
  alternate, and no-host output paths.
- [ ] Wire `--dispatch-effort VALUE` only to the new dispatch resolver. Keep
  the old `--effort` flag boolean and existing positional grammar unchanged.
- [ ] Run `go test ./cmd/spine -run 'TestModel.*(Dispatch|Effort|Alternate)' -count=1`
  and then `go test ./cmd/spine -count=1`.
- [ ] Commit this isolated unit with explicit paths.

## Task 3: retain declarations and parse exact authorizations

**Files:** `internal/audit/audit.go`, `internal/audit/audit_test.go`,
`internal/audit/teamspawn.go`, `internal/audit/i090_test.go`,
`internal/audit/codex.go`, `internal/audit/codex_test.go`

- [ ] Write failing fixtures for an attributed controller record carrying
  `harness`, `model`, and an effort declaration, plus an attributed Claude
  `--effort value` and `--effort=value`, Codex `reasoning_effort`, a
  documented legacy Codex `effort` alias, and omitted values. Prove a source
  label cannot fill a missing raw harness, and linked worker model actuals do
  not become observed effort. Include two retries with distinct complete
  declarations.
- [ ] Add failing ledger tests for an exact matching record, reversed pair,
  wrong ticket, wrong expected effort, wrong declared effort, spaced arrow,
  empty endpoint, duplicate `reason:`, trailing or reordered grammar, and
  missing or empty `reason:`. Assert the parser preserves raw bytes and every
  negative case authorizes nothing.
- [ ] Add effort declaration source, expected/declared status, and observed
  `-` storage. Parse effort records into their own ledger index. Do not alter
  `judgeToken`, `severity`, or `Report.Blocking()`.
- [ ] Run `go test ./internal/audit -run 'Test.*(Effort|Escalation|TeamSpawn|Codex)' -count=1`
  and then `go test ./internal/audit -count=1`.
- [ ] Commit this isolated unit with explicit paths.

## Task 4: publish additive declared-only output and documentation

**Files:** `cmd/spine/main.go`, `cmd/spine/main_test.go`, `WORKFLOW.md`,
`templates/current/WORKFLOW.md.tmpl`, `docs/issues/README.md`,
`templates/current/issues-README.md`, `CONTEXT.md`

- [ ] Add trailing declared-effort and observed-effort output fields, using
  `-` for absent data and preserving all existing leading columns and
  unmatched formatting.
- [ ] Add command fixture tests proving old model-only prefixes and blocking
  results are unchanged, while a declaration exposes expected/declared data
  and `observed-effort=-`.
- [ ] Document exact raw-token validation and exact-pair authorization without
  suggesting an effort ordering, provider-effective observation, or I074
  public verdict.
- [ ] Run `go test ./cmd/spine ./internal/audit -count=1` and `go test ./...`.
- [ ] Commit this isolated unit with explicit paths.

## Final verification

- [ ] Compare the finished diff against the I075 design, specifically its
  no-ordering, no-fabrication, host-pin, and compatibility clauses.
- [ ] Run focused model, audit, and CLI tests, then `go test ./...`.
- [ ] Run `spine audit routing` with the required transcript scope. A fresh
  verifier must review the final diff before ship.
