# Flag-ordering generalization (I119) Implementation Plan

> **For agentic workers:** solo inline execution in the owning session — TDD
> per task, stage gates as normal. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** No spine subcommand silently discards input — trailing flags and
stray positionals error naming the rule (I116 shape), unknown cursor
sub-subcommands error like every other dispatcher, gate joins the
flags-first house grammar; then retire the workaround guidance everywhere
it is taught.

**Architecture:** One ticket, serial tasks, one session (execution-mode
inline, tier primary, review-tier n/a). All code in `cmd/spine/main.go` +
`main_test.go`. Branch off main, ff-merge at ship. Never claude-sonnet-5.

**Tech Stack:** Go standard library only (ADR 0001).

**Spec:** `docs/specs/2026-08-28-flag-ordering-generalization-design.md`

## Global Constraints

- All commits cite I119; batch commits so one `maipipe run full --wait`
  covers each HEAD move.
- Every negative control **observed red** (command + output recorded).
- `spine cursor` is the only cursor writer; never write the literal cursor
  marker in prose; never tick the handoff stage.
- Stage explicit paths only. Read exit codes unpiped under fish.
- Behavior contracts, unchanged: the three I116 model tests pass
  unmodified; flag-only `spine cursor` invocations exit 0 always;
  `spine gate go@1 <check>` (maipipe form) behaves identically;
  `version`/`help` stay lenient; documented flags-first invocations behave
  identically.

### Task 1: shared parse helper + total conversion

**Files:**
- Modify: `cmd/spine/main.go` (new `parseArgs`-style helper; every `cmd*`
  FlagSet site converted, `cmdModel` included with byte-identical output)
- Modify: `cmd/spine/main_test.go`

**Interfaces:**
- Produces: helper owning `fs.Parse` + ordering guard
  (`flagAmongPositionals` on `fs.Args()`, first-position `--` exemption
  intact) + exact-arity check (`wantN`; -1 skips) + usage printing.
  Messages: `<cmd>: flags must precede positionals (saw %q after %q)` and
  `<cmd>: unexpected argument %q`, each + usage, exit 2. Called with
  post-`takeForce` args where takeForce applies.

- [ ] **Step 1: Failing tests.** Table-driven ordering sweep (one
  trailing-flag invocation per converted subcommand ⇒ exit 2, rule + token
  named; the table is the conversion checklist); arity sweep
  (`doctor foo`, `update junk`, `cursor start` extras ⇒ exit 2,
  `unexpected argument`); first-position exemption green control
  (`adr new -- "-Title"` reaches adr's own logic); takeForce green control
  (`cursor tick <stage> --force` in a scratch repo still works).
- [ ] **Step 2: Verify red.** Record command + output.
- [ ] **Step 3: Implement the minimum**; convert every site.
- [ ] **Step 4: Verify green** + the three I116 model tests unmodified;
  `gofmt -l`, `go vet ./...`.
- [ ] **Step 5: Negative control.** Disable the guard call inside the
  helper (keep tests); observe the sweep red; restore.

### Task 2: cursor unknown-subcommand + gate flags-first

**Files:**
- Modify: `cmd/spine/main.go` (`cmdCursor` dispatch + doc-comment contract
  amendment; `cmdGate` parse rework)
- Modify: `cmd/spine/main_test.go`

**Interfaces:**
- Produces: `cursor <unknown-positional>` ⇒
  `unknown cursor subcommand %q` + usage listing `start|tick|here|set`,
  exit 2; flag-only cursor invocations byte-identical to today (exit 0
  always). `cmdGate` parses flags first via the helper (wantN 2), pack and
  check from the returned positionals; exit-code contract 0/1/2 unchanged.

- [ ] **Step 1: Failing tests.** `cursor show` ⇒ exit 2 naming the unknown
  subcommand; flag-only `cursor --dir <empty>` ⇒ exit 0 + no-cursor
  message (green control on the narrowed contract);
  `gate --dir X go@1 <check>` resolves as the same check run in X;
  `gate go@1 <check> --dir X` ⇒ exit 2 naming the rule.
- [ ] **Step 2: Verify red.** Record command + output.
- [ ] **Step 3: Implement**; amend the `cmdCursor` doc comment in the same
  change.
- [ ] **Step 4: Verify green**; live green control:
  `spine gate go@1 binary-hygiene` at repo root exits 0 (record it).
- [ ] **Step 5: Negative control.** Revert the dispatch hunk; `cursor show`
  test red; restore.

### Task 3: guidance sweep

**Files:**
- Modify (in-repo, committed): `README.md` / `WORKFLOW.md` / docs where
  examples or gotcha prose would now be wrong.
- Out-of-repo (NOT committed here; listed in the handoff): `~/.claude`
  skills invoking spine (model-eval and grep hits), `~/.claude/CLAUDE.md`,
  auto-memory entries teaching the "silently ignores" warning.

- [ ] Grep repo docs + `~/.claude` for spine invocations with trailing
  flags and for flag-ordering workaround prose; record the hit list.
- [ ] Fix in-repo examples; retire "silently ignores" warnings (the
  flags-first rule itself stays documented — now enforced with a helpful
  error).
- [ ] Apply out-of-repo edits; list them in the handoff.

### Task 4: functional test, review, verify, ship, docs

- [ ] `go test ./...`, `gofmt -l`, `go vet ./...` — exit 0, record output.
- [ ] Functional pass on the installed binary (`make install` to `~/bin`):
  the original repro `spine cursor show --dir X` now errors; a trailing
  flag on a converted subcommand names the rule; flag-only cursor hook
  form still exit 0.
- [ ] Spec-review of the finished diff against the PRD (mandatory gate),
  requirements-attack step first — attack the spec itself for internal
  contradictions; surface with proposed resolutions, never silently
  resolve.
- [ ] ff-merge to main; `maipipe run full --wait` green at the merge SHA
  (the six gate lanes double as the gate-rework green control).
- [ ] `make install`; `spine version` build-provenance line recorded as
  the drift baseline.
- [ ] Ledger close: I119 status fixed, `commits:` written — resolution
  line carries a done-word.
- [ ] CHANGELOG entry; the handoff stops re-teaching the flag-order
  gotcha (it is now code behavior everywhere).
