# I032 correction report — spine-worker2

Date: 2026-08-29

## Correction

The correction is `910e421` (`test(I032): derive truncation fixture from naming cap`).
It changes only `internal/stages/stages_test.go` and adds the test-only
`internal/stages/export_test.go` seam. The prior I032 implementation remains
untouched; no existing commit was amended.

## Exact cap-coupling proof

`MaxNamedMissingIDsForTest` is declared in `export_test.go` as:

```go
const MaxNamedMissingIDsForTest = maxNamedMissingIDs
```

Therefore the test value is exactly the production value, while the symbol is
available only in test builds and does not expand the runtime package API.

Let `C = maxNamedMissingIDs`. The black-box fixture sets:

```go
rangeEnd := stages.MaxNamedMissingIDsForTest + 2
tickets: I001-I%03d   // rangeEnd
```

The inclusive resolved range contains `C + 2` ids. Only `I001` exists, so the
missing set contains exactly `C + 1` ids. `namedIDs` therefore emits the first
`C` ids followed by `+1 more`; the final id `I(C+2)` must not appear. With the
current `C = 5`, this is the original `I001-I007` shape: six missing ids,
five named, and `+1 more` for `I007`. If the production cap changes, the
fixture changes with it and still exercises the truncation boundary.

## Test evidence

RED, after changing the black-box fixture to use the intended seam but before
adding the seam:

```text
go test ./internal/stages -run '^TestTickedMissingTruncatesLongMissingSet$' -count=1
internal/stages/stages_test.go:562:21: undefined: stages.MaxNamedMissingIDsForTest
FAIL ... [build failed]
```

GREEN after adding the test-only seam:

```text
GOCACHE=/Users/ldh/Projects/github.com/spine/.cache/go-build go test ./internal/stages -run '^TestTickedMissingTruncatesLongMissingSet$' -count=1
ok   github.com/russellpope/spine/internal/stages  0.300s
```

The concurrent I111 test pattern was green before package-wide verification:

```text
GOCACHE=/Users/ldh/Projects/github.com/spine/.cache/go-build go test ./internal/audit -run 'Test(ClaudeLayout|D28StillRejectsUnqualifiedOpenweights|AmbiguousModelID|UnknownModelID)' -count=1
ok   github.com/russellpope/spine/internal/audit  2.505s
```

Required post-I111 verification:

```text
GOCACHE=/Users/ldh/Projects/github.com/spine/.cache/go-build go test ./internal/stages -count=1
ok   github.com/russellpope/spine/internal/stages  0.180s

GOCACHE=/Users/ldh/Projects/github.com/spine/.cache/go-build go test ./... -count=1
ok   all tested packages; exit_code=0
```

The correction diff also passed `git diff --check`.

## Commits and blockers

- Existing I032 implementation: `2e75d5e`.
- I032 correction: `910e421`.
- This ticket/report documentation update is committed separately after the
  correction commit; its SHA is reported by the worker handoff.
- No blockers remain. The default Go cache was not writable, so verification
  used the repository-local `.cache/go-build`; `.cache`, the known research
  stray, and all unrelated concurrent work were left unstaged.
