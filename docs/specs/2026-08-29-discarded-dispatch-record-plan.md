# Discarded dispatch records (I078) implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an exact, identity-scoped `DISCARDED` ledger record that visibly
classifies one abandoned dispatch while preserving blocking silent descent for
every other lower-tier dispatch.

**Architecture:** Preserve an immutable dispatch identity from each reader to
each evidence token, parse identity-scoped discarded records, then consult a
record only in the otherwise-silent-descent path. A token without exact
source/session/event correlation cannot be excused. Template generation 13
publishes the same grammar to generated workflow docs.

**Tech stack:** Go standard library, existing audit fixture style, embedded
workflow templates.

**Spec:** `docs/specs/2026-08-29-discarded-dispatch-record-design.md`

## Global constraints

- I078 is a routine-tier, subagent-driven audit change. It starts after I050's
  generation-12 approved-untested release. I111 and I102 are already
  integrated; preserve their current behavior.
- Do not infer whether a diff landed. The implementation is fail-closed when
  raw evidence lacks an exact dispatch identity.
- Keep `ESCALATION`, `FALLBACK`, `pickTier`, D28 qualification, I111 flavor
  derivation, and I102 first-prompt-only pairing behavior unchanged.
- `DISCARDED` grammar is exactly:
  `DISCARDED <ticket-id> source:<claude|codex> session:<session-id> dispatch:<event-id> tier:<mechanical|routine|primary|fallback> reason: <one line>`.
- Stage explicit paths only. Never stage `.cache/`,
  `docs/research/2026-08-26-fusion-harness-borrow-hitlist.md`, concurrent
  work, or unrelated files.

## File map

| File | Responsibility |
| --- | --- |
| `internal/audit/audit.go` | `VerdictDiscardedWithReason`, severity, identity-bearing evidence, ledger parsing, diagnostics, per-token correlation, and final judgment. |
| `internal/audit/audit_test.go` | Claude fixture acceptance, negative controls, malformed/duplicate/mismatch coverage, row output, and blocking behavior. |
| `internal/audit/codex.go` | Retain `call_id` from `codexResponseItem` and thread source/root/event identity into direct Codex dispatches only. |
| `internal/audit/codex_test.go` | Direct-Codex identity coverage and root-only worker fail-closed test. |
| `internal/audit/resolve_test.go` | Fixture helpers that write session-specific discarded ledgers and Claude transcript events. |
| `cmd/spine/main_test.go` | CLI exit-code and visible `discarded-with-reason` output test. |
| `templates/VERSION` | Template generation 13. |
| `templates/current/WORKFLOW.md.tmpl` | Exact documented discarded grammar and scope. |
| `WORKFLOW.md` | This repository's generated workflow contract after `spine update --write`. |
| `internal/scaffold/scaffold_test.go` | Fresh-scaffold grammar assertion. |
| `internal/tmpl/tmpl_test.go` | Current-generation assertion updated to 13. |
| `internal/update/update.go` | Register only an exact current workflow line that I078 actually replaces; never register I050's retained approved-untested lines. |
| `internal/update/gen12to13_test.go` | New clean migration, byte-for-byte I050 wording preservation, and retained-I050-line local-edit negative control. |
| `docs/issues/I078-discarded-dispatch-record-grammar-for-audit-routing.md` | Closure, actual SHAs, and verification evidence. |
| `CHANGELOG.md` | Consumer-visible verdict and workflow-grammar note. |

## Interfaces locked by this plan

Keep these private audit shapes exact unless a focused test proves a small
adjustment is necessary:

```go
type evidenceIdentity struct {
    source   string
    session  string
    dispatch string
}

type discardedRecord struct {
    ticket   string
    identity evidenceIdentity
    tier     string
    reason   string
    line     int
}

type evidenceToken struct {
    value      string
    flavor     string
    source     string
    sourceFile string
    identity   evidenceIdentity
}
```

`readLedger` returns parsed discarded records plus diagnostics. It does not
silently broaden a record by ticket or tier. `judgeToken` receives the token
identity and consults a prevalidated lookup only after its existing fallback
and escalation paths decline to classify the token. `Report.Blocking` remains
true when any final row is `silent-descent`.

### Task 1: preserve exact dispatch identity through audit evidence

**Files:**

- Modify: `internal/audit/audit.go` (`dispatch`, `subagent`,
  `evidenceToken`, `readTranscripts`, `scanJSONL`, `parseLine`, and `Run`)
- Modify: `internal/audit/resolve_test.go`
- Modify: `internal/audit/audit_test.go`

**Consumes:** existing Claude session basename, `tool_use.id`, and linked
subagent sidecar `toolUseId`.

**Produces:** `evidenceToken.identity` for direct Claude dispatch evidence and
linked Claude subagent actuals, but no identity for malformed or unlinked
evidence.

- [ ] **Step 1: Write failing focused tests.** Add a helper that writes two
  Claude session files for one primary ticket. Assert their otherwise-identical
  routine tokens retain different `{source:"claude", session, dispatch}`
  identities at judgment time by giving the test-only ledger one matching
  record and observing that it can cover only the matching event.

- [ ] **Step 2: Run red.**

  Run: `go test ./internal/audit -run 'TestDiscardedClaudeIdentityIsPerDispatch' -count=1`

  Expected: FAIL because `evidenceToken` has no identity and no discarded
  record path exists.

- [ ] **Step 3: Implement the smallest identity plumbing.** In
  `readTranscripts`, pass the filename stem to `scanJSONL`; in `parseLine`,
  place `b.ID` on each direct `Task`, `Agent`, and recognized team-spawn
  dispatch. For a linked Claude subagent, use the parent session stem and its
  sidecar `ToolUseID`. Thread those fields to `evidenceToken` without changing
  ticket matching, models, `Actuals`, or detail text.

- [ ] **Step 4: Verify green.**

  Run: `go test ./internal/audit -run 'Test(DiscardedClaudeIdentityIsPerDispatch|SubagentTranscriptIsTheActual|SilentDescentBlocks)' -count=1`

  Expected: PASS.

- [ ] **Step 5: Commit the identity unit.**

  Run: `git add internal/audit/audit.go internal/audit/resolve_test.go internal/audit/audit_test.go && git commit -m 'feat(I078): retain dispatch identity in audit evidence'`

### Task 2: parse and apply discarded records, with red-green guards

**Files:**

- Modify: `internal/audit/audit.go` (`Verdict`, `severity`, `ledger`,
  `readLedger`, `judge`, `judgeToken`, and `Report.Blocking`)
- Modify: `internal/audit/audit_test.go`
- Modify: `cmd/spine/main_test.go`

**Consumes:** Task 1 exact Claude identities and existing `tiersOf`,
`pickTier`, fallback, escalation, warning, and CLI printer seams.

**Produces:** visible, advisory `discarded-with-reason` for exactly one
otherwise-lower token and diagnostics for unusable records.

- [ ] **Step 1: Write failing acceptance tests.** Cover all of these named
  cases with a primary ticket and routine evidence:

  - `TestDiscardedWithExactIdentityIsVisibleAndNonBlocking`: one exact record
    produces `VerdictDiscardedWithReason`, includes the reason, and
    `Report.Blocking()` is false.
  - `TestDiscardedAbsentKeepsSilentDescent`: remove the record and assert
    `VerdictSilentDescent` and a blocking report.
  - `TestDiscardedDoesNotExcuseLandedSibling`: two distinct lower routine
    events share the ticket, but the record names only the prototype. Assert
    final `silent-descent`, blocking exit, and the prototype's discarded
    reason remains available in the token-level test result or row detail.
  - `TestDiscardedWrongIdentityOrTierDoesNotExcuse`: table-test wrong source,
    session, dispatch, and tier. Each remains `silent-descent`.
  - `TestDiscardedMalformedDuplicateAndAmbiguousRecordsDoNotExcuse`: reject a
    missing `dispatch:`, reordered fields, empty reason, duplicate complete
    key, and one record matching two eligible events. Assert no suppression
    and one safe warning per bad key or line.
  - `TestDiscardedDoesNotChangeEscalationOrFallback`: retain the existing
    I201, I206, I210, and I211 fixture verdicts.

- [ ] **Step 2: Run red.**

  Run: `go test ./internal/audit -run 'TestDiscarded|TestEscalationRecordDoesNotExcuseUnrelatedDescent|TestReasonedDescentStaysAdvisory|TestFallbackCoverage' -count=1`

  Expected: FAIL because the enum, exact grammar parser, and correlation
  lookup do not exist.

- [ ] **Step 3: Implement the minimum parser and validator.** Parse only the
  exact six-token prefix and non-empty `reason:` suffix. Preserve line numbers
  for warnings. Build a key from ticket, source, session, dispatch, and tier;
  invalidate duplicate keys. After evidence collection, count each parsed
  record's eligible token matches. Keep only records with exactly one eligible
  lower token. Leave malformed, duplicate, zero-match, and one-to-many
  records unusable and append the specified warnings.

- [ ] **Step 4: Implement the verdict path.** Add
  `VerdictDiscardedWithReason` at advisory severity. In `judgeToken`, retain
  exact-match, fallback, and escalation ordering. Only then, when the actual
  ordered tier is lower than the declared tier, look up the token's complete
  identity and resolved tier. Return `discarded-with-reason` with model,
  actual tier, and reason only for the one validated match. Do not change
  `pickTier` or downgrade a worse sibling token.

- [ ] **Step 5: Verify green, including the command seam.**

  Run: `go test ./internal/audit -run 'Test(Discarded|Escalation|Fallback|SilentDescent)' -count=1`

  Expected: PASS.

  Run: `go test ./cmd/spine -run 'TestAuditRouting.*Discarded' -count=1`

  Expected: PASS and output contains `discarded-with-reason` plus the reason.

- [ ] **Step 6: Demonstrate the load-bearing landed-work guard.** Temporarily
  change the lookup to use only ticket plus tier. Re-run
  `TestDiscardedDoesNotExcuseLandedSibling`; it must fail because the sibling
  no longer blocks. Restore the complete identity lookup before continuing.

- [ ] **Step 7: Commit the judgment unit.**

  Run: `git add internal/audit/audit.go internal/audit/audit_test.go cmd/spine/main_test.go && git commit -m 'feat(I078): classify exact discarded dispatches'`

### Task 3: retain direct Codex call identities and fail closed for worker scans

**Files:**

- Modify: `internal/audit/codex.go` (`codexResponseItem`, `scanCodexLine`,
  `codexExecWorker`, `parseCodexBytes`, and `readCodexSessions`)
- Modify: `internal/audit/codex_test.go`
- Modify: `internal/audit/audit_test.go`

**Consumes:** `session_meta` root identity, raw `call_id`, and Task 2's
identity-scoped discarded lookup.

**Produces:** discardable direct Codex `spawn_agent` and explicit team-start
records only. Root-linked Codex worker actuals remain identity-less.

- [ ] **Step 1: Write failing tests.** Make Codex fixture helpers accept a
  distinct `call_id`. Assert an exact record for a direct `spawn_agent` or
  team-start routine event produces `discarded-with-reason`. Then add a
  root-linked worker actual on the same ticket and same root; assert the record
  cannot excuse it and the final result is `silent-descent`.

- [ ] **Step 2: Run red.**

  Run: `go test ./internal/audit -run 'TestCodexDiscarded(DirectDispatch|DoesNotExcuseRootOnlyWorker)' -count=1`

  Expected: FAIL because `call_id` is not retained and root-only evidence is
  not distinguishable from a direct dispatch at the discarded lookup.

- [ ] **Step 3: Implement the smallest source-specific plumbing.** Decode
  `CallID` in `codexResponseItem`; preserve the call that supplied a direct
  dispatch model, including a team worker's start command. In
  `readCodexSessions`, pair it with source `codex` and the resolved root id.
  Do not manufacture an event id for `turn_context` worker actuals or use the
  coarse `codex:<root>` linkage as an I078 identity.

- [ ] **Step 4: Verify green and preserve I102.**

  Run: `go test ./internal/audit -run 'Test(CodexDiscarded|CodexTeamSpawnUsesFirstPromptOnly|CodexHerdrDispatchRecordJudgesMatch)' -count=1`

  Expected: PASS.

- [ ] **Step 5: Commit the Codex unit.**

  Run: `git add internal/audit/codex.go internal/audit/codex_test.go internal/audit/audit_test.go && git commit -m 'feat(I078): correlate direct codex dispatch records'`

### Task 4: publish grammar through template generation 13

**Files:**

- Modify: `templates/VERSION`
- Modify: `templates/current/WORKFLOW.md.tmpl`
- Modify: `internal/scaffold/scaffold_test.go`
- Modify: `internal/tmpl/tmpl_test.go`
- Modify: `internal/update/update.go` (`supersededLines`)
- Create: `internal/update/gen12to13_test.go`
- Modify: `WORKFLOW.md` only via `spine update --write --dir .`

**Consumes:** exact grammar and scope from the PRD.

**Produces:** a generation-13 generated workflow contract and a safe update
migration from I050's generation-12 output that preserves its
approved-untested wording. It recognizes a predecessor line only if I078
actually replaces that rendered line.

- [ ] **Step 1: Write failing template and migration tests.** Assert a fresh
  scaffold contains the exact `DISCARDED` grammar, says one record covers one
  identified event rather than a ticket/tier, and stamps `template_version:
  13`. Build a generation-12 workflow fixture from I050's captured render,
  including its exact `Acceptance exceptions` and `APPROVED-UNTESTED` lines.
  Assert `update.Run` reports pending with no unrecognized lines, writes
  generation 13, preserves those I050 lines byte-for-byte, and is idempotent.
  Mutate one retained I050 line and assert ordinary update reports it as an
  unrecognized local edit. Do not treat a retained I050 line as a predecessor.
  If I078 replaces a different current rendered line, add a separate test for
  that exact predecessor line.

- [ ] **Step 2: Run red.**

  Run: `go test ./internal/{tmpl,scaffold,update} -run 'Test.*(Generation|Discarded|Gen12To13)' -count=1`

  Expected: FAIL because the version remains 12 after I050 and no discarded workflow
  contract or migration lock exists.

- [ ] **Step 3: Implement the documentation migration.** Bump
  `templates/VERSION` to `13`; add the exact grammar and scope paragraph to
  `templates/current/WORKFLOW.md.tmpl`; preserve I050's approved-untested
  lines verbatim; and make the new migration test permit only the stamp and
  documented generation-13 addition. Update current-version expectations in
  `internal/tmpl/tmpl_test.go` and scaffold assertions from 12 to 13, retain
  captured generation-12 inputs as history, and advance the future-generation
  refusal from 13 to 14. Search every current-version assertion before editing:

  ```bash
  rg -n 'template_version: 12|begin v12|generation 12|want 12|!= 12|future generation|template_version: 13' internal cmd templates --glob '*.go' --glob 'VERSION'
  ```

  Change only assertions about the compiled current version to 13. Keep the
  captured generation-12 fixture unchanged. Add an exact `supersededLines`
  entry only if this change actually replaces that rendered predecessor line.

- [ ] **Step 4: Render this repository instead of hand-editing it.**

  Run: `go run ./cmd/spine update --dir . --write`

  Expected: `WORKFLOW.md` updates from `template_version: 12` to 13 with the
  exact discarded-record documentation, retained I050 approved-untested
  wording, and no unrecognized local edit.

- [ ] **Step 5: Verify green.**

  Run: `go test ./internal/tmpl ./internal/scaffold ./internal/update -count=1`

  Expected: PASS.

- [ ] **Step 6: Commit the generated-contract unit.**

  Run: `git add templates/VERSION templates/current/WORKFLOW.md.tmpl internal/scaffold/scaffold_test.go internal/tmpl/tmpl_test.go internal/update/update.go internal/update/gen12to13_test.go WORKFLOW.md && git commit -m 'docs(I078): publish discarded dispatch grammar'`

### Task 5: integration, documentation, review, and verification

**Files:**

- Modify: `docs/issues/I078-discarded-dispatch-record-grammar-for-audit-routing.md`
- Modify: `CHANGELOG.md`
- Verify: `docs/specs/2026-08-29-discarded-dispatch-record-design.md`
- Verify: `docs/specs/2026-08-29-discarded-dispatch-record-plan.md`

- [ ] **Step 1: Run full checks.**

  Run: `gofmt -w internal/audit/audit.go internal/audit/codex.go internal/audit/audit_test.go internal/audit/codex_test.go internal/audit/resolve_test.go internal/scaffold/scaffold_test.go internal/tmpl/tmpl_test.go internal/update/update.go internal/update/gen12to13_test.go cmd/spine/main_test.go`

  Expected: files formatted; inspect `git diff` afterward to confirm no
  unrelated edits.

  Run: `go test ./... -count=1`

  Expected: PASS.

  Run: `go vet ./...`

  Expected: PASS.

  Run: `git diff --check`

  Expected: exit 0.

- [ ] **Step 2: Run functional audit probes.** Build a disposable fixture with
  an exact discarded record and verify CLI exit 0 and visible
  `discarded-with-reason`. Remove the record and verify exit 1 and
  `silent-descent`. Add the sibling routine event and verify exit 1 again.
  Record exact commands and output in the I078 implementation report.

- [ ] **Step 3: Update closure documentation.** Add a concise `CHANGELOG.md`
  item. Mark I078 `fixed` only with actual implementation and documentation
  SHAs. Its resolution must state the no-diff-attribution boundary, complete
  identity requirement, malformed/duplicate behavior, template generation 13,
  and test/review evidence.

- [ ] **Step 4: Commit ticket and changelog docs.**

  Run: `git add docs/issues/I078-discarded-dispatch-record-grammar-for-audit-routing.md CHANGELOG.md && git commit -m 'docs(I078): close discarded dispatch records'`

- [ ] **Step 5: Fresh spec review.** A fresh primary-tier reviewer reads the
  completed diff against the PRD and attacks: tier-only suppression, the
  landed sibling guard, Codex root-only attribution, parser grammar, duplicate
  and malformed records, I111/I102/D28 compatibility, output severity, and
  template migration. Resolve findings and rerun the affected red-green tests.

- [ ] **Step 6: Routing audit and full verification.** Run `spine audit routing
  --dir .` with `--transcripts <dir>` if the controlling transcript lives
  outside this repo; run `spine audit stages --dir .`, `spine doctor --dir .`,
  and `make verify`. Record expected pre-existing warnings separately from new
  findings. Run `spine audit routing` again after the final review commit.

- [ ] **Step 7: Final scope check and handoff.** Confirm `git status --short`
  lists only I078 paths before each I078 commit. Do not close or commit
  `.cache/`, the research stray, concurrent specs, or another ticket's work.
  Update the ticket, changelog, and required implementation report with actual
  SHAs, then create the normal handoff only after verification is complete.

## Plan self-review checklist

- [x] Each acceptance criterion maps to Tasks 1 through 5.
- [x] Every production task starts with a named failing test and an expected
  red command, then a minimal implementation and green command.
- [x] The landed-work guard is tested by a sibling lower-tier event, not a
  claimed diff heuristic.
- [x] Codex root-only evidence remains fail-closed.
- [x] Template generation, predecessor-line recognition, and hand-edited-line
  protection are explicit.
- [x] Placeholder and contradiction scan found no unfinished marker, omitted
  identity field, or tier-only exemption.
