# I024 report — cursor printer malformed wording

## Ticket

`spine cursor` printed `derivation: clean` on a malformed cursor (grammar
findings, zero parsed stage rows), incoherent with `spine audit stages`
blocking on the same input. Doctor D9 never surfaced grammar-level
`CursorFindings` either.

## Implemented

### 1. `cmd/spine/main.go` — `cmdCursor`

When `rep.CursorFindings` is non-empty (a cursor block with grammar
findings — malformed stage tokens, missing/duplicate/unknown keys, bad
YOU-ARE-HERE marker count, etc.), the derivation line now prints:

```
derivation: n/a (cursor malformed)
```

instead of `clean`. This check takes priority over the existing
`Blocking()` check — with zero parsed stage rows there is nothing coherent
to call "clean" or "blocking" about the stage table itself; the actual
grammar problems are already surfaced above as `finding: ...` lines. Exit
code is unchanged: `spine cursor` stays exit 0 always (verified by an
existing test and a new one). `--quiet` mode behavior is untouched — it
only ever gated the "nothing to report" case, never a found-but-malformed
cursor, and that logic sits earlier in the function and was not touched.

Doc comment above `cmdCursor` extended to name all three derivation-line
outcomes and why grammar findings win priority.

### 2. `internal/doctor/doctor.go` — `stagesCheck` (D9)

D9 now emits an additional warn finding when `rep.CursorFindings` is
non-empty:

```
D9    warn  .superpowers/sdd/progress.md: cursor block malformed — grammar findings: <findings joined with "; ">
```

Severity is `warn`, consistent with every other D9 finding — no
special-casing of the existing "any warn flips doctor's exit code to 1"
policy, per the brief. Placed before the stage-verdict loop, mirroring
`spine audit stages`' ordering (malformed-cursor row printed before the
stage table).

### 3. Untouched (verified, not just assumed)

- `cmdAuditStages` (`spine audit stages`) — zero diff, confirmed via
  `git diff` — its own `cursorMalformed` handling (already correct per
  the ticket) is unchanged.
- `spine cursor --quiet` — still silences only the "nothing to report"
  case; the SessionStart-hook contract (I021) is untouched.
- Doctor's existing warn-affects-exit-code rule — untouched, and the new
  D9 finding participates in it exactly like every other warn finding
  (no special-casing added).

## Files changed

- `cmd/spine/main.go` (implementation + doc comment)
- `cmd/spine/main_test.go` (new test)
- `internal/doctor/doctor.go` (implementation + doc comment)
- `internal/doctor/doctor_test.go` (new test)

No new fixture files were added — the exact "grammar findings, zero parsed
stage rows" scenario the ticket describes is already covered by the
existing, already-tracked fixture
`internal/stages/testdata/malformed-cursor/repo` (used today by
`TestAuditStagesMalformedCursorBlocks`), reused here via the existing
`stagesFixture()` helper for the new `spine cursor` test. The new doctor
test builds its fixture inline in `t.TempDir()` (same pattern as the
existing `seedCleanCursor` helper), so the repo's `.superpowers/` fixture
gitignore gotcha did not come into play — confirmed via `git status
--untracked-files=all` showing no new untracked files after the change,
and `git diff --stat` showing only the four modified files above.

## TDD evidence

### RED (before implementation)

```
$ go test ./cmd/spine/... -run TestCursorCommandMalformedGrammarPrintsNAWording -v
=== RUN   TestCursorCommandMalformedGrammarPrintsNAWording
    main_test.go:563: want the n/a (cursor malformed) wording, out="finding: malformed stage token \"???\"\nfinding: malformed stage token \"***\"\nfinding: malformed stage token \"!!!\"\neffort: fixture-effort\nprd: docs/specs/2026-01-01-fixture-design.md\ntickets: I001-I002\nderivation: clean\n"
    main_test.go:566: must not claim clean on a grammar-malformed cursor, out="...derivation: clean\n" errs=""
--- FAIL: TestCursorCommandMalformedGrammarPrintsNAWording (0.00s)
FAIL
```

```
$ go test ./internal/doctor/... -run TestD9WarnOnMalformedCursorGrammar -v
=== RUN   TestD9WarnOnMalformedCursorGrammar
    doctor_test.go:547: want a D9 warn naming the malformed cursor grammar, got []doctor.Finding{}
--- FAIL: TestD9WarnOnMalformedCursorGrammar (0.00s)
FAIL
```

Both failures confirm the exact bug described in the ticket (the printer
said "clean" on the fixture already used by `TestAuditStagesMalformedCursorBlocks`
to prove `audit stages` blocks; doctor produced zero D9 findings on it).

### GREEN (after implementation)

```
$ go test ./cmd/spine/... -run 'TestCursor|TestAuditStages' -v
...
--- PASS: TestAuditStagesMalformedCursorBlocks (0.00s)
--- PASS: TestCursorCommandStaysExitZeroOnMalformedCursor (0.00s)
--- PASS: TestCursorCommandMalformedGrammarPrintsNAWording (0.00s)
--- PASS: TestCursorCommandPrintsValidCursor (0.00s)
--- PASS: TestCursorCommandExitsZeroOnMalformedAndPrintsFindings (0.00s)
--- PASS: TestCursorQuietSilentWhenNoCursor (0.00s)
--- PASS: TestCursorQuietSilentWhenSpineRepoHasNoLedgerYet (0.00s)
--- PASS: TestCursorQuietDoesNotSuppressAPresentCursor (0.00s)
--- PASS: TestCursorCommandOnRealRepoLedger (0.00s)
PASS
ok  	github.com/russellpope/spine/cmd/spine	0.272s
```

```
$ go test ./internal/doctor/... -run TestD9 -v
=== RUN   TestD9SilentWithNoCursor
--- PASS: TestD9SilentWithNoCursor (0.00s)
=== RUN   TestD9SilentOnCleanCursor
--- PASS: TestD9SilentOnCleanCursor (0.00s)
=== RUN   TestD9WarnOnTickedMissingStage
--- PASS: TestD9WarnOnTickedMissingStage (0.00s)
=== RUN   TestD9WarnOnMalformedCursorGrammar
--- PASS: TestD9WarnOnMalformedCursorGrammar (0.00s)
PASS
ok  	github.com/russellpope/spine/internal/doctor	0.281s
```

`TestCursorCommandOnRealRepoLedger` (the dogfood test against this repo's
own live `.superpowers/sdd/progress.md`) passed unaffected — the live
ledger is well-formed, so `CursorFindings` is empty for it and the new
branch never fires; it was not weakened.

## Full-suite result

```
$ go build ./... && go vet ./... && go test ./...
ok  	github.com/russellpope/spine/cmd/spine	0.225s
ok  	github.com/russellpope/spine/internal/adopt	0.530s
ok  	github.com/russellpope/spine/internal/adr	(cached)
ok  	github.com/russellpope/spine/internal/audit	(cached)
ok  	github.com/russellpope/spine/internal/cursor	(cached)
ok  	github.com/russellpope/spine/internal/doctor	0.329s
ok  	github.com/russellpope/spine/internal/eval	(cached)
ok  	github.com/russellpope/spine/internal/fsutil	(cached)
ok  	github.com/russellpope/spine/internal/handoff	(cached)
ok  	github.com/russellpope/spine/internal/meta	(cached)
ok  	github.com/russellpope/spine/internal/scaffold	(cached)
ok  	github.com/russellpope/spine/internal/stages	(cached)
ok  	github.com/russellpope/spine/internal/tmpl	(cached)
ok  	github.com/russellpope/spine/internal/update	0.748s
?   	github.com/russellpope/spine/templates	[no test files]
```

`gofmt -l .` reported no files.

## Requirements-attack (self-review of the ticket itself)

Checked the Problem/Fix sections for internal contradictions before
judging the work against them — found none. The Fix section's condition
(`HasCursor && len(CursorFindings) > 0`) and the Problem section's example
(zero parsed stage rows) are consistent with each other and with the
existing `internal/stages` package doc's stated design ("CursorFindings
... never affects Report.Blocking()"), which this change deliberately does
not alter — only the printer's wording changed, not the derivation
semantics.

One latitude point worth flagging explicitly (not a contradiction, a
judgment call the ticket leaves open): the Fix section says the new
wording prints "instead of clean" without saying what happens when a
cursor is *both* grammar-malformed *and* independently blocking (e.g. its
grammar findings coexist with a real handoff-backstop failure — possible
in principle, since `Report.Blocking()` also checks the newest-handoff
independent of stage rows). I resolved this by giving the malformed-grammar
wording priority over "blocking" in all cases, on the reasoning that with
zero parsed stage rows there's nothing coherent left to call "blocking"
about (the "blocking" branch's detail lines iterate `rep.Stages`, which is
empty for a garbage `stages:` line) — printing "n/a (cursor malformed)"
unconditionally is strictly more informative than either alternative,
and the underlying grammar findings are still visible via the `finding:
...` lines printed just above regardless of which branch fires. This
doesn't weaken any existing behavior: the only fixture exercising this
combination pre-fix already printed "clean" (not "blocking") because
`rep.Blocking()` was false on it.

## Self-review

- **Completeness**: both Fix-section changes implemented (printer wording,
  D9 warn). `spine audit stages` untouched (verified via `git diff`
  showing zero changes to `cmdAuditStages`). `--quiet` mode: verified via
  existing passing tests (`TestCursorQuietSilentWhenNoCursor`,
  `TestCursorQuietSilentWhenSpineRepoHasNoLedgerYet`,
  `TestCursorQuietDoesNotSuppressAPresentCursor`) that its behavior is
  unchanged beyond the derivation wording it would print when a cursor is
  found — no `--quiet`-specific test needed a code change.
- **Test quality**: both new tests were run RED against the pre-fix code
  and failed with output demonstrating exactly the bug described (see TDD
  evidence above) — they would fail on the old code.
- **No scope creep**: no new fixtures added; reused the existing, already
  git-tracked `internal/stages/testdata/malformed-cursor` fixture for the
  cursor-command test; the doctor test builds its input inline rather than
  adding a checked-in fixture. No unrelated files touched. `PICKUP.md`
  (pre-existing untracked file in the working tree, unrelated to this
  ticket) was left alone and not committed.
- **Concerns**: none blocking. The one judgment call (grammar-malformed
  wording takes priority over "blocking" wording in the hypothetical
  case where both would otherwise apply) is documented above and in the
  code comment; I believe it's the correct reading of "instead of clean"
  but flagging it since the ticket's Fix section doesn't explicitly
  address that combination.
