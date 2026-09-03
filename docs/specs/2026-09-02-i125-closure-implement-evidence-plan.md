# I125 Closure Implement Evidence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a closed ticket file (`status: fixed`, SHA-shaped `commits:`) evidence the implement stage, OR'd with the progress-ledger scan, and name the real rule on a zero-evidence implement miss.

**Architecture:** One package, `internal/stages`. The issues scan already reads every ticket's frontmatter for `id:`; it now also records whether the file is a closure record. The implement presence vector ORs the ledger scan with that map. The judge takes its zero-evidence hint text from the caller so each row owns its wording.

**Tech Stack:** Go standard library, existing temp-repo test style.

**Spec:** `docs/specs/2026-09-02-i125-closure-implement-evidence-design.md`

## Global constraints

- Ticket `I125`; execution mode inline (single tightly-coupled package change, one session); tier primary; review-tier primary.
- Conservative rule holds: no new error paths, no new verdicts.
- Stage explicit paths only. `maipipe run full --wait` green at the final SHA before push.
- Every commit cites I125.

## File map

| File | Responsibility |
|---|---|
| `internal/stages/stages.go` | Frontmatter walk returns fields; closure predicate; OR into implement presence; row-owned zero-evidence hints; label; package doc. |
| `internal/stages/stages_test.go` | Black-box derivation tests for the positive path, negative controls, mixed OR, pending direction, rule wording. |
| `internal/stages/implement_evidence_internal_test.go` | White-box table test for the closure predicate. |
| `cmd/spine/main_test.go` | Compiled-CLI byte-exact expectation for the renamed label. |
| `CHANGELOG.md` | Unreleased/Fixed entry. |
| `CONTEXT.md` | Glossary terms (done in the grill). |
| `docs/issues/I125-…md` | Close: status, commits, acceptance boxes. |

### Task 1: closure predicate (white-box)

- [ ] Write `TestClosureRecord` table in the internal test file: fixed+one SHA → true; fixed+several → true; fixed+`[]` → false; fixed+absent → false; fixed+`[pending]` → false; open+SHA → false; in-progress+SHA → false; wontfix+SHA → false; superseded+SHA → false; `Fixed` (case) → false; 6-char token → false; 40-char token → true.
- [ ] Run red: `go test ./internal/stages -run TestClosureRecord -count=1`.
- [ ] Implement `frontmatterFields` (fence walk, trims, strips matching quotes), keep `frontmatterID` as a wrapper, add `closureRecord(fm) bool` and `commitSHARe`.
- [ ] Run green.

### Task 2: derivation ORs closure records

- [ ] Write black-box tests: `TestImplementClosureRecordEvidencesTickedStage` (implement[x], zero ledger lines, fixed+commits → match, detail `1/1 implement evidence present`); `TestImplementClosureNegativeControls` subtests for open, fixed-empty-commits, fixed-no-commits, wontfix, superseded, placeholder token → ticked-missing; `TestImplementLedgerAndClosureOr` (I001 by ledger line, I002 by closure → match 2/2); `TestImplementAnchoredNoDoneWordClosureWins`; `TestImplementPendingWithClosureRecordIsPresentUnticked`.
- [ ] Run red.
- [ ] Refactor the issues scan into one pass returning per-id facts (present, closure); OR into `implPresent`; rename the label.
- [ ] Run green; run `go test ./internal/stages -count=1` for regressions.

### Task 3: row-owned zero-evidence wording

- [ ] Write `TestImplementZeroEvidenceNoAnchoredLinesNamesBothSources`: open ticket, no ledger line → detail contains `no progress-ledger implement line` and `closure record`, not `typo`, not `tickets:`. Extend the existing I117 anchored test to assert the new text is absent there.
- [ ] Run red.
- [ ] Replace `judgeSet`'s `ticketsRaw` + `anchoredNoEvidence` with a caller-supplied `zeroHint string`; issues row builds the typo hint, implement row picks wording vs sources message, prd passes "".
- [ ] Run green; update the package doc comment and `implementEvidence` comment.

### Task 4: CLI expectation and full lane

- [ ] Update the byte-exact label in `cmd/spine/main_test.go`; `go test ./... -count=1`; `go vet ./...`.
- [ ] Negative control on the live repo: temporarily rename the closure predicate's status check and confirm the new tests fail; restore.
- [ ] Live: `go build -o bin/spine ./cmd/spine && bin/spine audit stages` derives the prior effort 21/21 unchanged.

### Task 5: docs, close, gates

- [ ] CHANGELOG entry; tick acceptance boxes in I125; set `status: fixed`, `commits:`.
- [ ] `/spec-review` against the design; `/code-review`; fix findings.
- [ ] `maipipe run full --wait` at the final SHA; install both binaries; push.
