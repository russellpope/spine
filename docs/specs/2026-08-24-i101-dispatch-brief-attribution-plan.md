# I101 dispatch-brief attribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Attribute a claude-team spawn whose brief is delivered by file
reference, using the brief write the lead's own transcript recorded — never the
file on disk — and make a worktree-built effort's transcripts discoverable by
default.

**Architecture:** The claude transcript reader gains a per-session brief table
(normalized path → body, in scan order). `teamspawn.go` gains reference-form
recognition and textual path normalization. The dispatch record gains a resolved
brief so `matches()` sees a first line for the token and a body for repo
qualification. `DefaultTranscriptsDir` gains a union-returning sibling.

**Tech Stack:** Go standard library only (ADR 0001).

**Spec:** `docs/specs/2026-08-24-i101-dispatch-brief-attribution-design.md`
**Decision:** `docs/adr/0020-dispatch-brief-attribution-reads-the-lead-s-transcript-never-the-file-on-disk.md`

## Global Constraints

- I101 is routine-tier, subagent-driven; all commits cite I101.
- Zero third-party dependencies. The audit never invokes a shell and never
  opens a referenced brief path — path handling is string work only.
- Under-attribution is always the acceptable failure. No step may make the
  audit attribute a ticket it cannot resolve from a recorded write.
- Tests drive `audit.Run` and the CLI seam with real temp repos and hand-built
  JSONL fixtures; no source-text assertions.
- Stage explicit paths only. `spine cursor` is the only cursor writer.

### Task 1: brief table and path normalization

**Files:**
- Modify: `internal/audit/teamspawn.go`
- Create: `internal/audit/brief_test.go`

**Interfaces:**
- Produces: a per-session brief table type — record a heredoc write, resolve a
  referenced path to a body at or before a given position (D29, D30, D32).
- Consumes: the existing `stripHeredocBodies` scanner, extended to also yield
  each heredoc's target and body rather than only discarding them.

- [ ] **Step 1: Failing tests.** Table records `cat > $WS/a.md <<'EOF'` under
  the normalized absolute path given `WS=` recorded earlier and a session cwd;
  `cat >>` appends; a reference by any of the three D31 forms and by any of the
  three path spellings (`$WS/a.md`, relative expanded, absolute) resolves to the
  same entry; an unexpanded variable does not resolve; a rewrite of the same
  path resolves by position (D32).
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum.** Extend the heredoc scanner to emit
  (target, body) alongside the stripped command; add normalization and the
  positional lookup.
- [ ] **Step 4: Verify green**, then `gofmt`.
- [ ] **Step 5: Negative control.** Temporarily drop the position bound in the
  lookup; the D32 rewrite test must fail. Restore.

### Task 2: resolve a brief onto the dispatch record

**Files:**
- Modify: `internal/audit/teamspawn.go`, `internal/audit/audit.go`
- Modify: `internal/audit/i090_test.go` (regression guard stays green)
- Create: `internal/audit/testdata/brief/` fixture

**Interfaces:**
- Consumes: Task 1's table; the existing `parseTeamSpawn` / `parseTeamPrompt` /
  `attributeTeamPrompt` pairing.
- Produces: a resolved brief on the dispatch record — first line for the ticket
  token, body for repo qualification (D33) — with precedence per D34.

- [ ] **Step 1: Failing tests.** The design's team fixture: brief first line
  names one ticket and the repo path, body names three others → exactly one
  attributed, three untouched. A spawn naming a ticket in its own command keeps
  it (D34). A brief whose first line names the ticket but not the repo, body
  naming the repo → attributed (D33). Unresolvable reference → unmatched, with
  today's listing text.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum.** Thread the table through `scanJSONL` /
  `parseLine`; resolve on the spawn and on its paired prompt; carry the brief on
  `dispatch` so `matches()` reads first line for the token and body for
  `repoQualifies`.
- [ ] **Step 4: Verify green.** Confirm I090's existing tests, including its
  `recognizeTeamSpawns` negative control, are untouched and green.
- [ ] **Step 5: Negative control.** Add the `recognizeBriefFiles` package var;
  flipped off, the brief fixture must fall back to unattributed. Assert it.
- [ ] **Step 6: C1 regression guard.** Assert a heredoc body that nothing
  references still contributes no attribution — I090's ruling is intact.

### Task 3: disclosure and footer

**Files:**
- Modify: `internal/audit/audit.go`
- Modify: `internal/audit/audit_test.go`

- [ ] **Step 1: Failing tests.** A verdict attributed via a brief names the
  brief path on the line (D35). The unmatched footer no longer says `see I101`
  and keeps its count and reason (D37).
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum**, then `gofmt`.
- [ ] **Step 4: Verify green.**

### Task 4: transcript discovery union

**Files:**
- Modify: `internal/audit/audit.go` (`DefaultTranscriptsDir` and its caller)
- Modify: `cmd/spine/main.go` (audit wiring, help text if it names the default)
- Create/modify: `internal/audit/resolve_test.go`

**Interfaces:**
- Produces: a union-returning discovery — repo slug, `git worktree list` slugs,
  `<repo-slug>-*` slugs (D36) — with scanned dirs in `rep.Warnings`.
- Consumes: `--transcripts`, which still overrides the union entirely.

- [ ] **Step 1: Failing tests.** Over a temp `HOME`: slug dirs for the repo, a
  live worktree, and a removed worktree are all scanned; a decoy sibling repo's
  records are swept in but land unmatched via D28; a non-git dir degrades to the
  slug scan without error; `--transcripts` overrides; scanned dirs appear in
  warnings.
- [ ] **Step 2: Verify red.**
- [ ] **Step 3: Implement the minimum**, then `gofmt`.
- [ ] **Step 4: Verify green.**
- [ ] **Step 5: Negative control.** Temporarily drop the D28 gate; the decoy
  sibling test must fail by attributing. Restore.

### Task 5: verification against the live corpus

**Files:**
- Modify: `docs/issues/I101-audit-routing-attribution-from-brief-file.md`

- [ ] **Step 1:** `gofmt -l .` clean, `go vet ./...` clean,
  `SPINE_REQUIRE_MAIPIPE=1 make test` green. Paste the commands and output.
- [ ] **Step 2:** `make install`, then run `spine audit routing` with **no**
  `--transcripts` from the repo root. Record: how many of the 27 local-harness
  spawns are now attributed, which tickets among I079–I087 changed verdict, and
  which spawns remain unmatched and why.
- [ ] **Step 3:** Confirm the expected verdict churn from ADR 0020's
  consequences is what appears (reviewer spawns on `claude-fable-5 @ high`
  against `tier: routine`). Report it; do not silence it.
- [ ] **Step 4:** Record the measured numbers in the ticket and set
  `status: fixed`.

## Verification (branch level)

- [ ] `gofmt -l .` empty; `go vet ./...` clean.
- [ ] `SPINE_REQUIRE_MAIPIPE=1 make test` green.
- [ ] `spine doctor` exit 0 (two pre-existing D4 notes on the two READMEs are
  expected and unrelated).
- [ ] `spine audit stages` derivation clean.
- [ ] Every negative control above demonstrated failing before restore, with the
  failure output quoted.
- [ ] `/spec-review` of the finished diff against the design doc.
