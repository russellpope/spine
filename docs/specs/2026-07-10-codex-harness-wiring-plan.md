# Codex Harness Wiring — Workstream A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `spine` emit a Codex-facing `AGENTS.md` (twin of the CLAUDE.md spine block) so Codex sessions follow the same workflow, gates, and model routing as Claude — shipping at template generation 7.

**Architecture:** Reuse the existing marker-surgery machinery. A new `templates/current/AGENTS.md.tmpl` renders a Codex-tuned block bounded by the same `<!-- spine:begin v{{VERSION}} -->` markers. `scaffold.Files` gains the file so `spine init` emits it; a `planAgents` clone of `planClaude` handles `spine update` (create-if-missing / marker-replace / claim-on-top). Bumping `templates/VERSION` 6→7 stamps the generation.

**Tech Stack:** Go 1.x, `text`-free string templating (`strings.NewReplacer` in `internal/tmpl`), stdlib `testing`. Templates are `go:embed`'d (`templates/embed.go`).

**Scope note:** This plan covers **workstream A only** (design §A). Workstream B (Codex global config hygiene) and C (project skills as a local Codex marketplace, spike-gated) are separate follow-on plans. A is a complete, testable deliverable on its own — every spine repo gains `AGENTS.md` on its next `spine update`.

## Global Constraints

- Design doc: `docs/specs/2026-07-10-codex-harness-wiring-design.md` (locked decisions there).
- Machine-owned block markers are exactly `<!-- spine:begin v{{VERSION}} -->` and `<!-- spine:end -->` (constants `markerBegin`/`markerEnd`, update.go:53-54). Templates render the version from `{{VERSION}}`; never hardcode a generation integer in a template.
- `AGENTS.md` is owned by **all** profiles (including `knowledge`) — Codex-awareness is universal. `ProfileOwns` must return `true` for it.
- The `AGENTS.md` block is **Codex-tuned**: same workflow facts as `CLAUDE.md.tmpl`, but no Claude-only `/slash-command` invocations (Codex can't resolve them until workstream C). Reference stages/gates by name and `spine` subcommands instead.
- Generation bump is **VERSION-file only** (`templates/current/*.tmpl` already render `{{VERSION}}`). Bumping to 7 breaks a fixed set of version-coupled test assertions listed verbatim in Task 3 — the suite is red until all are updated; that is expected mid-task, green by task end.
- Commit trailers (house rule, every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_01To5QfgzXk3oZ2gNRM8QidH
  ```
- Commits go to a short-lived feature branch reviewed before merge (or `main` directly for trivial changes); merge/push only when the owner asks.
- Build/test: `go test ./...` from repo root. Install for manual verify: `make install` (builds to `~/bin/spine`).

---

### Task 1: `AGENTS.md.tmpl` template + scaffold registration

Adds the template and wires `spine init` to emit it. This task keeps VERSION at 6, so the marker renders `v6` here; Task 3 bumps it. To keep the suite green within this task, the scaffold count assertions are updated here **as part of adding the file** (they'd otherwise go red the moment `AGENTS.md` joins `scaffold.Files`).

**Files:**
- Create: `templates/current/AGENTS.md.tmpl`
- Modify: `internal/scaffold/scaffold.go:23-30` (add to `Files`)
- Test: `internal/scaffold/scaffold_test.go` (new test + count fixes)

**Interfaces:**
- Consumes: `tmpl.Render("current", "AGENTS.md.tmpl", tmpl.Values)`, `tmpl.ProfileOwns` (unchanged — already returns `true` for non-`knowledge`; verify it returns `true` for `AGENTS.md` under `knowledge` too, which it does since the path isn't in its exclusion switch).
- Produces: an emitted `AGENTS.md` at repo root for every profile; `scaffold.Init` `Created` count rises from 6 to 7.

- [ ] **Step 1: Write the failing test** — assert `init` emits a Codex-tuned, marker-bounded `AGENTS.md`.

Add to `internal/scaffold/scaffold_test.go`:

```go
func TestInitEmitsCodexAgentsMd(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not emitted: %v", err)
	}
	content := string(raw)
	for _, want := range []string{
		"<!-- spine:begin v", "<!-- spine:end -->",
		"read by **Codex**", // Codex-tuned framing
		"WORKFLOW.md",
		"docs/specs/", "docs/adr/", "docs/issues/", "docs/handoffs",
		"Mandatory gates", "verification before completion",
		"model_routing", "primary / routine / mechanical / fallback",
		"spine audit routing",
		"multi_agent", "spawn_agent",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("AGENTS.md missing %q", want)
		}
	}
	// Codex-tuned: no Claude-only slash-command invocations in the block.
	for _, banned := range []string{"/grill-with-docs", "/to-spec", "/spec-review", "/wayfinder"} {
		if strings.Contains(content, banned) {
			t.Errorf("AGENTS.md must not carry Claude-only invocation %q", banned)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scaffold/ -run TestInitEmitsCodexAgentsMd -v`
Expected: FAIL — `AGENTS.md not emitted: ... no such file`.

- [ ] **Step 3: Create the template**

Create `templates/current/AGENTS.md.tmpl` (exact content):

```markdown
<!-- spine:begin v{{VERSION}} -->
# {{PROJECT}} — Codex working brief

This file is read by **Codex**; the `CLAUDE.md` twin carries the same facts for Claude. Both are machine-owned by `spine` between the markers — edit only outside them.

Uses the **unified workflow** — see `WORKFLOW.md` for the active profile (`{{PROFILE}}`) and stages.

- Specs / PRDs / plans -> `docs/specs/` (pairs: `<date>-<topic>-design.md` + `-plan.md`)
- Decisions (ADRs) -> `docs/adr/` (convention in `docs/adr/README.md`)
- Issue / bug ledger -> `docs/issues/` (dependency convention in `docs/issues/README.md`)
- Handoffs -> `docs/handoffs/`

**Mandatory gates:** a PRD up front, a spec-review of the finished diff against the PRD, and verification before completion — these are workflow stages (grill -> prd -> ... -> verify), not optional.

**Model routing:** `WORKFLOW.md` `model_routing` maps tiers (primary / routine / mechanical / fallback) to model ids — reference tiers, never ids. Every subagent dispatch carries an explicit model and the ticket-id token; `spine audit routing` checks this at the verify gate.

**Subagents:** the Codex `multi_agent` feature is enabled, so `spawn_agent` / `wait_agent` / `close_agent` are available for parallel and subagent-driven work; close agents once their work is done. Detect worktree/branch state with read-only git before creating branches (see superpowers `codex-tools` environment detection).

Workflow operations run through `spine`: `spine adr`, `spine handoff`, `spine doctor`, `spine audit routing`, `spine update`.
<!-- spine:end -->
```

- [ ] **Step 4: Register in the scaffold manifest**

Modify `internal/scaffold/scaffold.go`, the `Files` slice (currently scaffold.go:23-30). Add the `AGENTS.md` entry immediately after `CLAUDE.md`:

```go
var Files = []struct{ TmplName, RelPath string }{
	{"CLAUDE.md.tmpl", "CLAUDE.md"},
	{"AGENTS.md.tmpl", "AGENTS.md"},
	{"WORKFLOW.md.tmpl", "WORKFLOW.md"},
	{"harness-interface.md", "docs/harness-interface.md"},
	{"issues-README.md", "docs/issues/README.md"},
	{"issue.tmpl.md", "docs/issues/_template.md"},
	{"adr-README.md", "docs/adr/README.md"},
}
```

- [ ] **Step 5: Fix the count assertions this file addition breaks**

The new file makes `init` create 7 files, not 6. Update `internal/scaffold/scaffold_test.go`:

- Line ~61 (`TestInitCreatesAndStamps`): `if len(res.Created) != 6 || len(res.Skipped) != 0 {` → `if len(res.Created) != 7 || len(res.Skipped) != 0 {`
- Line ~195 (`TestInitIdempotent`): `if len(res.Created) != 0 || len(res.Skipped) != 6 {` → `if len(res.Created) != 0 || len(res.Skipped) != 7 {`

Also extend `TestInitKnowledgeManifest`'s existence list (line ~287) to prove `knowledge` gets it too — add `"AGENTS.md"`:

```go
	for _, rel := range []string{"WORKFLOW.md", "CLAUDE.md", "AGENTS.md", "docs/adr/README.md", "docs/adr", "docs/handoffs"} {
```

- [ ] **Step 6: Run scaffold tests to verify pass**

Run: `go test ./internal/scaffold/ -v`
Expected: PASS — including `TestInitEmitsCodexAgentsMd`, `TestInitCreatesAndStamps`, `TestInitIdempotent`, `TestInitKnowledgeManifest`.

- [ ] **Step 7: Commit**

```bash
git add templates/current/AGENTS.md.tmpl internal/scaffold/scaffold.go internal/scaffold/scaffold_test.go
git commit  # message: "feat(spine): emit Codex-facing AGENTS.md on init"
```

---

### Task 2: `planAgents` — `spine update` manages the AGENTS.md block

Mirrors `planClaude` (update.go:248-293): create when missing, replace the marker block when present, claim-on-top when markers absent, surface unbalanced markers as `Unrecognized`. Wire it into `Run`. Still at VERSION 6 (markers render `v6`); Task 3 bumps.

**Files:**
- Modify: `internal/update/update.go` (add `planAgents`; call it in `Run` after `planClaude`, ~update.go:82-86)
- Test: `internal/update/update_test.go` (new tests)

**Interfaces:**
- Consumes: `tmpl.Render("current", "AGENTS.md.tmpl", vals)`, `replaceMarkerBlock` (update.go:295), `Diff` (update.go), the `vals` produced by `planWorkflow` (no `gen` — AGENTS.md has no generation-specific predecessor).
- Produces: `planAgents(dir string, vals tmpl.Values) (FileReport, error)` returning a `FileReport{Path: "AGENTS.md"}`. `Run` appends its report to `reports`.

- [ ] **Step 1: Write the failing tests** — created when missing; block replaced when present; hand-authored content outside markers preserved; unbalanced markers flagged.

Add to `internal/update/update_test.go`:

```go
func TestAgentsMdCreatedWhenMissing(t *testing.T) {
	// A repo with WORKFLOW.md + CLAUDE.md but no AGENTS.md (the state every
	// existing gen-6 repo is in) gains AGENTS.md on update.
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a := report(t, reports, "AGENTS.md")
	if a.State != Pending || !a.Created {
		t.Fatalf("AGENTS.md state=%v created=%v", a.State, a.Created)
	}
	if !strings.Contains(a.Diff, "read by **Codex**") {
		t.Errorf("AGENTS.md diff missing Codex-tuned body:\n%s", a.Diff)
	}
}

func TestAgentsMdMarkerReplacePreservesHandContent(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	// pre-place an AGENTS.md with a stale block + hand-authored tail.
	stale := "<!-- spine:begin v5 -->\nold codex brief\n<!-- spine:end -->\n\n## Local notes\nkeep me\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = reports
	got, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if strings.Contains(s, "old codex brief") {
		t.Error("stale block survived marker replacement")
	}
	if !strings.Contains(s, "## Local notes") || !strings.Contains(s, "keep me") {
		t.Error("hand-authored content outside markers was lost")
	}
	if strings.Count(s, "<!-- spine:begin") != 1 {
		t.Error("marker replacement duplicated the block")
	}
}

func TestAgentsMdUnbalancedMarkersFlagged(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	bad := "<!-- spine:begin v6 -->\nbody\n<!-- spine:begin v6 -->\n<!-- spine:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a := report(t, reports, "AGENTS.md")
	if len(a.Unrecognized) == 0 {
		t.Fatal("unbalanced markers should be flagged, not clobbered")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/update/ -run 'TestAgentsMd' -v`
Expected: FAIL — `report` helper can't find `"AGENTS.md"` (planAgents not wired).

- [ ] **Step 3: Add `planAgents` and wire it into `Run`**

In `internal/update/update.go`, add after `planClaude` (a direct structural clone, differing only in `Path`, template name, and gen0-fallback handling — AGENTS.md has no gen0 predecessor, so the no-marker branch always claims-on-top rather than clean-claiming a pristine legacy file):

```go
func planAgents(dir string, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: "AGENTS.md"}
	block, err := tmpl.Render("current", "AGENTS.md.tmpl", vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, "AGENTS.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", block)
		report.newContent = block
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	var newContent string
	if strings.Contains(old, markerBegin) {
		replaced, err := replaceMarkerBlock(old, block)
		if err != nil {
			report.Unrecognized = []string{err.Error()}
			return report, nil
		}
		newContent = replaced
	} else {
		// No spine-owned block yet: claim on top, preserve everything below.
		// (AGENTS.md has no gen0 template, so there is no pristine-legacy
		// clean-claim case as CLAUDE.md has.)
		newContent = block + "\n" + old
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}
```

Note `replaceMarkerBlock` (update.go:296) counts `markerBegin`/`markerEnd`; it already trims the trailing newline of `block`. Reused unchanged.

Then in `Run`, right after the `planClaude` block is appended (update.go:82-86):

```go
	cl, err := planClaude(opts.Dir, gen, vals)
	if err != nil {
		return nil, err
	}
	reports = append(reports, cl)
	ag, err := planAgents(opts.Dir, vals)
	if err != nil {
		return nil, err
	}
	reports = append(reports, ag)
```

(`planAgents` doesn't need `gen`; AGENTS.md has no generation-specific predecessor.)

- [ ] **Step 4: Run the update tests to verify pass**

Run: `go test ./internal/update/ -run 'TestAgentsMd' -v`
Expected: PASS — all three.

- [ ] **Step 5: Add the anti-drift guard test** — both machine blocks must reference the same gate + routing vocabulary (design risk mitigation).

Add to `internal/update/update_test.go`:

```go
func TestClaudeAndAgentsBlocksShareVocabulary(t *testing.T) {
	vals := tmpl.Values{Project: "demo", Profile: "go-service", Version: tmpl.Version()}
	claude, err := tmpl.Render("current", "CLAUDE.md.tmpl", vals)
	if err != nil {
		t.Fatal(err)
	}
	agents, err := tmpl.Render("current", "AGENTS.md.tmpl", vals)
	if err != nil {
		t.Fatal(err)
	}
	// The two harness briefs must not drift on the load-bearing vocabulary.
	for _, term := range []string{
		"WORKFLOW.md", "docs/specs/", "docs/adr/", "docs/issues/", "docs/handoffs",
		"primary", "routine", "mechanical", "fallback",
		"verification before completion",
	} {
		if !strings.Contains(claude, term) {
			t.Errorf("CLAUDE.md.tmpl missing shared term %q", term)
		}
		if !strings.Contains(agents, term) {
			t.Errorf("AGENTS.md.tmpl missing shared term %q", term)
		}
	}
}
```

- [ ] **Step 6: Run it to verify pass**

Run: `go test ./internal/update/ -run TestClaudeAndAgentsBlocksShareVocabulary -v`
Expected: PASS. (If it fails, the two templates have drifted — reconcile the wording, don't weaken the test.)

- [ ] **Step 7: Commit**

```bash
git add internal/update/update.go internal/update/update_test.go
git commit  # message: "feat(spine): manage AGENTS.md block on spine update"
```

---

### Task 3: Bump template generation 6 → 7

Single content edit (`templates/VERSION`) plus the known, enumerated set of version-coupled test assertions it breaks. The suite is red between Step 1 and Step 3; that's expected. This is one atomic task — a reviewer approves or rejects the whole generation bump.

**Files:**
- Modify: `templates/VERSION`
- Modify (assertion updates): `internal/tmpl/tmpl_test.go`, `cmd/spine/main_test.go`, `internal/update/update_test.go`, `internal/update/gen5to6_test.go`, `internal/update/hbmview_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `tmpl.Version() == 7`; all emitted markers/stamps render `v7` / `template_version: 7`.

- [ ] **Step 1: Bump the VERSION file**

Edit `templates/VERSION`: change the single line `6` to `7`.

- [ ] **Step 2: Confirm the whole suite is now red on version assertions (sanity)**

Run: `go test ./... 2>&1 | grep -E 'FAIL|template_version|v6|Version'`
Expected: failures in the files listed below (proves the assertions are the only coupling).

- [ ] **Step 3: Update every version-coupled assertion (exact list)**

Apply each edit verbatim:

- `internal/tmpl/tmpl_test.go:11`: `if got := tmpl.Version(); got != 6 {` → `got != 7`
- `cmd/spine/main_test.go:131`: `"+ template_version: 6"` → `"+ template_version: 7"`
- `cmd/spine/main_test.go:346`: `"+ template_version: 6"` → `"+ template_version: 7"`
- `cmd/spine/main_test.go:353`: `"+ template_version: 6\n"` → `"+ template_version: 7\n"`
- `internal/update/update_test.go:105`: `"template_version: 6"` → `"template_version: 7"`
- `internal/update/update_test.go:118`: `"<!-- spine:begin v6 -->"` → `"<!-- spine:begin v7 -->"`
- `internal/update/update_test.go:162`: `"template_version: 6"` → `"template_version: 7"`
- `internal/update/update_test.go:321`: `"template_version: 6"` → `"template_version: 7"`
- `internal/update/gen5to6_test.go:143`: `"template_version: 6"` → `"template_version: 7"`
- `internal/update/gen5to6_test.go:163-164`: `"<!-- spine:begin v6 -->"` and the `"missing v6 marker"` message → `v7`
- `internal/update/hbmview_test.go:46`: `"template_version: 6"` → `"template_version: 7"`
- `internal/update/hbmview_test.go:58`: `v6` marker → `v7`

**Downgrade-guard test (special — must go to 8, not 7).** `internal/update/update_test.go:398-418` (`TestVersionDowngradeGuard`) fabricates a *newer-than-compiled* stamp. Post-bump the scaffolded file is `template_version: 7`, so bump the fabricated value to 8:

- Line 408: `bumped := strings.Replace(string(raw), "template_version: 7", "template_version: 8", 1)`
- Line 410: message `"template_version: 7 not found in scaffolded WORKFLOW.md to bump"`

(The scaffolded file is now v7 because Step 1 bumped VERSION; the `strings.Replace` source string must match the new scaffold output.)

**Not changed:** `internal/audit/testdata/*/WORKFLOW.md` (`template_version: 5`/`6`) are deliberate historical fixtures representing older repos for the audit; they stay. `internal/audit/gen6_scaffold_test.go` scaffolds and checks routing/tickets, not the version literal — it stays green at v7.

- [ ] **Step 4: Run the full suite to verify green**

Run: `go test ./...`
Expected: PASS — all packages. If any `v6`/`template_version: 6` failure remains, it's a site missed above; fix it the same way and note it.

- [ ] **Step 5: Commit**

```bash
git add templates/VERSION internal/tmpl/tmpl_test.go cmd/spine/main_test.go internal/update/update_test.go internal/update/gen5to6_test.go internal/update/hbmview_test.go
git commit  # message: "feat(spine)!: bump template generation 6 -> 7 (AGENTS.md)"
```

---

### Task 4: gen-6 → gen-7 upgrade proof

An explicit end-to-end test that a repo already at generation 6 (the real-world state of deepthought, spine, praxis, etc.) gains `AGENTS.md` and advances to v7 on a plain `spine update` — the migration story the design promises with no migration code.

**Files:**
- Test: `internal/update/update_test.go` (new test; may reuse `writeRepo`/`report` helpers)

**Interfaces:**
- Consumes: `Run`, `scaffold.Init`. No production changes.

- [ ] **Step 1: Write the failing test**

Add to `internal/update/update_test.go`:

```go
// A repo scaffolded at the *previous* generation must, on a plain update,
// gain AGENTS.md and advance its stamp — with no hand-written migration code.
func TestGen6RepoGainsAgentsMdOnUpdate(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	// Simulate a repo stamped at the prior generation.
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	downgraded := strings.Replace(string(raw), "template_version: 7", "template_version: 6", 1)
	if downgraded == string(raw) {
		t.Fatal("could not stage a gen-6 fixture")
	}
	if err := os.WriteFile(wfPath, []byte(downgraded), 0o644); err != nil {
		t.Fatal(err)
	}
	// Remove AGENTS.md so we're truly in the pre-A state.
	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("update did not create AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), "<!-- spine:begin v7 -->") {
		t.Error("AGENTS.md not stamped at v7")
	}
	wfAfter, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wfAfter), "template_version: 7") {
		t.Error("WORKFLOW.md did not advance to gen 7")
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/update/ -run TestGen6RepoGainsAgentsMdOnUpdate -v`
Expected: PASS (behavior already implemented by Tasks 1-3; this test locks the migration guarantee).

- [ ] **Step 3: Commit**

```bash
git add internal/update/update_test.go
git commit  # message: "test(spine): prove gen-6 repos gain AGENTS.md at v7 on update"
```

---

### Task 5: Verification — install and drive it against real repos

The verify gate: prove the change works end-to-end on the actual repos, not just in `t.TempDir()`. Dry-run first (no writes), inspect, then this is where the owner decides to `--write` deepthought.

**Files:** none (verification only).

- [ ] **Step 1: Full suite + vet**

Run: `go test ./... && go vet ./...`
Expected: all PASS, no vet complaints.

- [ ] **Step 2: Install the new binary**

Run: `make install`
Expected: builds `~/bin/spine`. Confirm: `spine version` prints generation `7`.

- [ ] **Step 3: Dry-run against spine itself**

Run: `cd ~/Projects/github.com/spine && spine update`
Expected: reports `AGENTS.md` as **would-create**, `WORKFLOW.md` stamp `6 -> 7`, `CLAUDE.md` marker `v6 -> v7`. No other file shows unexpected content churn. Do **not** `--write` yet.

- [ ] **Step 4: Dry-run against deepthought (the proving ground)**

Run: `cd ~/Projects/github.com/deepthought && spine update`
Expected: same shape — `AGENTS.md` would-create, version stamps advance, no hand-authored content in CLAUDE.md flagged as `SkippedUnrecognized`. If anything reads as unrecognized, stop and reconcile before writing.

- [ ] **Step 5: Owner gate — apply to deepthought**

Present the two dry-run outputs to the owner. On approval:

Run: `cd ~/Projects/github.com/deepthought && spine update --write`
Then open a Codex session in deepthought and confirm it can state, from `AGENTS.md`, the mandatory gates and the model-routing tiers. Record the result in the task report.

- [ ] **Step 6: Handoff note**

Note for follow-on: workstreams B (fix `~/.codex/AGENTS.md` stub + `~/.Codex/` path bug) and C (deepthought `skills/` as a local Codex marketplace — spike the manifest format first) are their own plans. Reference the design doc §B/§C.

---

## Self-Review

**Spec coverage (design §A):**
- A.1 new `AGENTS.md.tmpl`, Codex-tuned → Task 1 Step 3 (+ banned-invocation assertions Step 1).
- A.2 `scaffold.Files` registration → Task 1 Step 4.
- A.3 `planAgents` + `Run` wiring → Task 2 Steps 3.
- A.4 generation bump 6→7 (VERSION only) → Task 3.
- A.5 tests (scaffold, update marker/preserve/unbalanced, version) → Tasks 1, 2, 3, 4.
- Risk "gen-bump blast radius" → Task 3 (enumerated sites) + Task 5 dry-runs.
- Risk "two files drift" → Task 2 Step 5 anti-drift guard.
- Migration guarantee ("existing repos pick it up") → Task 4.
- §B/§C are explicitly out of scope (Scope note) with a handoff in Task 5 Step 6. ✅ No §A gap.

**Placeholder scan:** No TBD/TODO; every code step carries complete code; the version-bump collateral is an exact line list, not "update the tests." ✅

**Type consistency:** `planAgents(dir string, vals tmpl.Values) (FileReport, error)` — the signature in the Interfaces block, the Step-3 definition, and the `Run` call site (`planAgents(opts.Dir, vals)`) all agree (no `gen` param, intentionally). `report(t, reports, "AGENTS.md")` and `FileReport{Path: "AGENTS.md"}` agree. `writeRepo`/`report`/`gen0Hbmview`/`gen0HbmviewClaude` are existing helpers/fixtures in `update_test.go`. ✅
