# spine CLI v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `spine` Go CLI (init / update / adr / doctor) that absorbs workflow-init, per the approved spec at `docs/specs/2026-07-01-spine-cli-design.md`.

**Architecture:** Single static binary; templates (current generation + gen-0 legacy) compile in via `go:embed`; four flat subcommands dispatched from a `run()` function that tests call directly. Update implements ownership-split + config-preserving regeneration with the choice-vs-default rule.

**Tech Stack:** Go 1.26 (installed: go1.26.4 darwin/arm64), standard library ONLY.

## Global Constraints

- Module path: `github.com/russellpope/spine`. Repo root: `/Users/ldh/Projects/github.com/spine`.
- **No third-party imports anywhere.** stdlib only (spec decision 2).
- No network calls. No file deletion. All writes via `fsutil.WriteFileAtomic`.
- Exit codes, uniform: 0 = clean/success · 1 = findings or pending/skipped changes · 2 = hard error / bad usage.
- Errors → stderr; diffs/output/JSON → stdout.
- TDD: every code task writes the failing test first. Run tests with `go test ./internal/... ./cmd/... -run <Name> -v` from the repo root.
- Commit after every task (messages given per task). The repo already exists with `origin` set — commit locally, do NOT push.
- Template placeholders are literal `{{PROJECT}}`, `{{PROFILE}}`, `{{REVIEWERS}}`, `{{HARNESS}}`, `{{VERSION}}` strings replaced with `strings.NewReplacer` — not Go text/template.
- The deepthought repo (gen-0 source) is at `/Users/ldh/Projects/github.com/deepthought`; gen-0 = commit `f6bca64^` of `skills/workflow-init/templates/`.
- hbmview (live acceptance target) is at `/Users/ldh/Projects/github.com/hbmview`; its values: project `hbmview`, profile `rust`, reviewers `rust-reviewer, security-review`, harness `cli`.

## File Structure

```
go.mod, .gitignore, Makefile
templates/embed.go                 # package templates; go:embed VERSION current gen0
templates/VERSION                  # "1"
templates/current/                 # CLAUDE.md.tmpl WORKFLOW.md.tmpl harness-interface.md
                                   # issues-README.md issue.tmpl.md adr-README.md adr.tmpl.md
templates/gen0/                    # WORKFLOW.md.tmpl CLAUDE.md.tmpl harness-interface.md (legacy claim set)
internal/tmpl/tmpl.go(+_test)      # Render, Version, Defaults, Profiles
internal/fsutil/fsutil.go(+_test)  # WriteFileAtomic
internal/scaffold/scaffold.go(+_test)  # DetectProfile, Init
internal/update/keys.go            # ExtractKeys, Choices, ProjectFromWorkflow, setKey helpers
internal/update/diff.go            # Diff (LCS line diff)
internal/update/update.go(+_test)  # FileReport, Options, Run + per-file planners
internal/adr/adr.go(+_test)        # Entry, List, New (+ supersede flip)
internal/doctor/doctor.go(+_test)  # Finding, Run (D1–D6)
cmd/spine/main.go(+main_test.go)   # run() dispatch + cmdInit/cmdUpdate/cmdADR/cmdDoctor
internal/update/testdata/hbmview/  # copies of hbmview's real drifted files (Task 9)
```

---

### Task 1: Module bootstrap + embedded templates + tmpl package

**Files:**
- Create: `go.mod`, `.gitignore`, `Makefile`, `templates/embed.go`, `templates/VERSION`, `templates/current/*` (7 files), `templates/gen0/*` (3 files)
- Create: `internal/tmpl/tmpl.go`
- Test: `internal/tmpl/tmpl_test.go`

**Interfaces:**
- Produces: `templates.FS embed.FS`; `tmpl.Values{Project, Profile, Reviewers, Harness string; Version int}`; `tmpl.Render(gen, name string, v Values) (string, error)`; `tmpl.Version() int`; `tmpl.Defaults(profile string) (reviewers, harness string, err error)`; `tmpl.Profiles() []string`

- [ ] **Step 1: Bootstrap files**

```bash
cd /Users/ldh/Projects/github.com/spine
printf 'module github.com/russellpope/spine\n\ngo 1.26\n' > go.mod
printf 'bin/\n' > .gitignore
```

Create `Makefile` (recipe lines MUST be tab-indented):

```make
BIN := $(HOME)/bin

.PHONY: build test install

build:
	go build -o bin/spine ./cmd/spine

test:
	go test ./...

install:
	mkdir -p $(BIN)
	go build -o $(BIN)/spine ./cmd/spine
```

- [ ] **Step 2: Copy unchanged templates + gen-0 from deepthought**

```bash
cd /Users/ldh/Projects/github.com/spine
mkdir -p templates/current templates/gen0
DT=/Users/ldh/Projects/github.com/deepthought
cp "$DT/skills/workflow-init/templates/harness-interface.md" templates/current/
cp "$DT/skills/workflow-init/templates/issues-README.md"     templates/current/
cp "$DT/skills/workflow-init/templates/issue.tmpl.md"        templates/current/
git -C "$DT" show 'f6bca64^:skills/workflow-init/templates/WORKFLOW.md.tmpl'    > templates/gen0/WORKFLOW.md.tmpl
git -C "$DT" show 'f6bca64^:skills/workflow-init/templates/CLAUDE.md.tmpl'      > templates/gen0/CLAUDE.md.tmpl
git -C "$DT" show 'f6bca64^:skills/workflow-init/templates/harness-interface.md' > templates/gen0/harness-interface.md
echo 1 > templates/VERSION
```

- [ ] **Step 3: Write the two modified current templates**

`templates/current/WORKFLOW.md.tmpl` — deepthought's current template plus the `template_version` line (line 4):

```
# Workflow — {{PROJECT}}

profile: {{PROFILE}}
template_version: {{VERSION}}
reviewers: [{{REVIEWERS}}]
functional_harness: {{HARNESS}}    # cli | rest | framebuffer | none
gates: [grill, verify]             # mandatory; everything else advisory. verify = fresh-context verifier subagent(s) against the PRD/spec, not self-review
model_routing:
  primary: claude-fable-5          # long-horizon, ambiguous, or first-shot-complex work (design, plan, implement, orchestrate)
  fallback: claude-opus-4-8        # auto on stop_reason: refusal (cyber/bio/reasoning-extraction); also context/usage exhaustion
  routine: claude-sonnet-5         # mechanical subagent roles: doc edits, plan-transcription implementers, build fixers, simple reviews
effort: high                       # default; xhigh for security-critical analysis + final verification; medium/low for routine subagents
model_default: claude-fable-5      # swappable; re-evaluate on major model/platform releases
security_routing: quality-framing-opus-4-8
stages: [grill, prd, issues, implement, functional-test, review, verify, ship, deploy, docs, handoff]

See `docs/harness-interface.md` for the functional-test harness contract.
Mandatory gates: a PRD up front (grill-with-docs -> to-prd) and verification before completion.
Execution mode per plan: live-system mutation, secrets, or interactive steps -> inline with the human; otherwise subagent-driven.
```

`templates/current/CLAUDE.md.tmpl` — marker-wrapped, with the specs-absorb-plans line (spec decision 6):

```
<!-- spine:begin v{{VERSION}} -->
# {{PROJECT}}

Uses the **unified workflow** — see `WORKFLOW.md` for the active profile (`{{PROFILE}}`) and stages.

- Specs / PRDs / plans -> `docs/specs/` (pairs: `<date>-<topic>-design.md` + `-plan.md`)
- Decisions (ADRs) -> `docs/adr/` (convention in `docs/adr/README.md`)
- Issue / bug ledger -> `docs/issues/` (dependency convention in `docs/issues/README.md`)
- Handoffs -> `docs/handoffs/`

**Mandatory gates:** a PRD up front (run `/grill-with-docs` -> `/to-prd`) and verification before completion.
**Model:** see `WORKFLOW.md` `model_routing` (primary / fallback-on-refusal / routine; swappable).
<!-- spine:end -->
```

- [ ] **Step 4: Write the two new ADR templates**

`templates/current/adr-README.md`:

```
# Architecture Decision Records — convention

One decision per file: `NNNN-short-slug.md` (numbering starts at 0001; `spine adr new` picks
the next number). Front-matter fields: `id`, `title`, `status`, `date`, optional `supersedes`.

Statuses: `Accepted` (default) or `Superseded by NNNN`.

ADRs are immutable once accepted. Reversing or amending a decision means a NEW ADR that
supersedes the old one (`spine adr new "..." --supersedes NNNN`) — the only permitted edit to
an existing ADR is the status flip that supersede performs. If resolving an issue changes the
architecture, record the change as an ADR and link it from the issue.
```

`templates/current/adr.tmpl.md`:

```
---
id: {{ADR_ID}}
title: {{ADR_TITLE}}
status: Accepted
date: {{ADR_DATE}}{{ADR_SUPERSEDES}}
---

# {{ADR_ID}}: {{ADR_TITLE}}

## Context

## Decision

## Consequences
```

`templates/embed.go`:

```go
// Package templates embeds the workflow template generations that compile
// into the spine binary. gen0 is the pre-versioning 2026-06-28 generation,
// kept only so update can claim legacy files.
package templates

import "embed"

//go:embed VERSION current gen0
var FS embed.FS
```

- [ ] **Step 5: Write the failing test**

`internal/tmpl/tmpl_test.go`:

```go
package tmpl_test

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/tmpl"
)

func TestVersionIsOne(t *testing.T) {
	if got := tmpl.Version(); got != 1 {
		t.Fatalf("Version() = %d, want 1", got)
	}
}

func TestRenderFillsAllPlaceholders(t *testing.T) {
	for _, gen := range []string{"current", "gen0"} {
		got, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", tmpl.Values{
			Project: "demo", Profile: "rust",
			Reviewers: "rust-reviewer, security-review", Harness: "cli", Version: 1,
		})
		if err != nil {
			t.Fatalf("%s: %v", gen, err)
		}
		for _, want := range []string{"# Workflow — demo", "profile: rust", "functional_harness: cli"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q", gen, want)
			}
		}
		if strings.Contains(got, "{{") {
			t.Errorf("%s: unfilled placeholder:\n%s", gen, got)
		}
	}
}

func TestCurrentWorkflowIsStamped(t *testing.T) {
	got, err := tmpl.Render("current", "WORKFLOW.md.tmpl", tmpl.Values{Profile: "rust", Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "template_version: 1") {
		t.Error("current WORKFLOW template lacks template_version stamp")
	}
	if !strings.Contains(got, "primary: claude-fable-5") {
		t.Error("current WORKFLOW template lacks model_routing")
	}
}

func TestCurrentClaudeHasMarkers(t *testing.T) {
	got, err := tmpl.Render("current", "CLAUDE.md.tmpl", tmpl.Values{Project: "p", Profile: "rust", Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "<!-- spine:begin v1 -->") || !strings.Contains(got, "<!-- spine:end -->") {
		t.Errorf("markers missing:\n%s", got)
	}
}

func TestDefaults(t *testing.T) {
	rev, harness, err := tmpl.Defaults("rust")
	if err != nil || rev != "rust-reviewer, security-review" || harness != "cli" {
		t.Fatalf("rust defaults = %q %q %v", rev, harness, err)
	}
	if _, _, err := tmpl.Defaults("nope"); err == nil {
		t.Fatal("unknown profile should error")
	}
	if len(tmpl.Profiles()) != 6 {
		t.Fatalf("Profiles() = %v, want 6 entries", tmpl.Profiles())
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd /Users/ldh/Projects/github.com/spine && go test ./internal/tmpl/ -v`
Expected: FAIL — package `internal/tmpl` does not exist / does not compile.

- [ ] **Step 7: Implement `internal/tmpl/tmpl.go`**

```go
// Package tmpl renders the embedded workflow templates.
package tmpl

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/russellpope/spine/templates"
)

// Values fills the {{KEY}} placeholders in a template.
type Values struct {
	Project   string
	Profile   string
	Reviewers string
	Harness   string
	Version   int
}

type profileDefaults struct{ reviewers, harness string }

var profiles = map[string]profileDefaults{
	"go-service":   {"go-reviewer, security-review", "rest"},
	"py-tool":      {"python-reviewer, security-review", "cli"},
	"rust":         {"rust-reviewer, security-review", "cli"},
	"library-cli":  {"go-reviewer, python-reviewer", "cli"},
	"presentation": {"", "none"},
	"ui":           {"typescript-reviewer", "framebuffer"},
}

// Profiles lists the known profile names, sorted.
func Profiles() []string {
	out := make([]string, 0, len(profiles))
	for p := range profiles {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Defaults returns the reviewers/harness pair the profile map assigns.
func Defaults(profile string) (reviewers, harness string, err error) {
	d, ok := profiles[profile]
	if !ok {
		return "", "", fmt.Errorf("unknown profile %q (known: %s)", profile, strings.Join(Profiles(), ", "))
	}
	return d.reviewers, d.harness, nil
}

// Version returns the compiled template generation from templates/VERSION.
func Version() int {
	raw, err := templates.FS.ReadFile("VERSION")
	if err != nil {
		panic("templates/VERSION missing from embed: " + err.Error())
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n < 1 {
		panic("templates/VERSION must be a positive integer")
	}
	return n
}

// Render fills placeholders in templates/<gen>/<name>; gen is "current" or "gen0".
func Render(gen, name string, v Values) (string, error) {
	raw, err := templates.FS.ReadFile(gen + "/" + name)
	if err != nil {
		return "", fmt.Errorf("template %s/%s: %w", gen, name, err)
	}
	r := strings.NewReplacer(
		"{{PROJECT}}", v.Project,
		"{{PROFILE}}", v.Profile,
		"{{REVIEWERS}}", v.Reviewers,
		"{{HARNESS}}", v.Harness,
		"{{VERSION}}", strconv.Itoa(v.Version),
	)
	return r.Replace(string(raw)), nil
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/tmpl/ -v`
Expected: PASS (5 tests).

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "feat: module bootstrap, embedded templates (current + gen0), tmpl package"
```

---

### Task 2: fsutil + scaffold (init engine)

**Files:**
- Create: `internal/fsutil/fsutil.go`, `internal/scaffold/scaffold.go`
- Test: `internal/fsutil/fsutil_test.go`, `internal/scaffold/scaffold_test.go`

**Interfaces:**
- Consumes: `tmpl.Render`, `tmpl.Defaults`, `tmpl.Version`, `tmpl.Values`
- Produces: `fsutil.WriteFileAtomic(path string, data []byte) error`; `scaffold.Result{Created, Skipped []string}`; `scaffold.DetectProfile(dir string) (string, bool)`; `scaffold.Init(dir, profile, name string) (Result, error)`

- [ ] **Step 1: Write the failing tests**

`internal/fsutil/fsutil_test.go`:

```go
package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/fsutil"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	if err := fsutil.WriteFileAtomic(path, []byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "hello\n" {
		t.Fatalf("read back %q, %v", got, err)
	}
	// overwrite works and leaves no temp litter
	if err := fsutil.WriteFileAtomic(path, []byte("two\n")); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp file litter: %v", entries)
	}
}
```

`internal/scaffold/scaffold_test.go`:

```go
package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectProfile(t *testing.T) {
	cases := []struct {
		file, content, want string
	}{
		{"Cargo.toml", "[package]", "rust"},
		{"go.mod", "module x", "go-service"},
		{"pyproject.toml", "[project]", "py-tool"},
		{"deck.pptx", "x", "presentation"},
		{"package.json", `{"dependencies":{"react":"19"}}`, "ui"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		write(t, dir, c.file, c.content)
		got, ok := scaffold.DetectProfile(dir)
		if !ok || got != c.want {
			t.Errorf("%s: got %q ok=%v, want %q", c.file, got, ok, c.want)
		}
	}
	if _, ok := scaffold.DetectProfile(t.TempDir()); ok {
		t.Error("empty dir should not detect")
	}
}

func TestInitCreatesAndStamps(t *testing.T) {
	dir := t.TempDir()
	res, err := scaffold.Init(dir, "rust", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 6 || len(res.Skipped) != 0 {
		t.Fatalf("created=%v skipped=%v", res.Created, res.Skipped)
	}
	wf, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Workflow — demo", "profile: rust", "template_version: 1",
		"reviewers: [rust-reviewer, security-review]", "functional_harness: cli"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("WORKFLOW.md missing %q", want)
		}
	}
	for _, d := range []string{"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"} {
		if fi, err := os.Stat(filepath.Join(dir, d)); err != nil || !fi.IsDir() {
			t.Errorf("missing dir %s", d)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "docs/adr/README.md")); err != nil {
		t.Error("missing docs/adr/README.md")
	}
}

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	res, err := scaffold.Init(dir, "rust", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 0 || len(res.Skipped) != 6 {
		t.Fatalf("second run created=%v skipped=%v", res.Created, res.Skipped)
	}
}

func TestInitUnknownProfile(t *testing.T) {
	if _, err := scaffold.Init(t.TempDir(), "nope", ""); err == nil {
		t.Fatal("want error for unknown profile")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/fsutil/ ./internal/scaffold/ -v`
Expected: FAIL — packages do not exist.

- [ ] **Step 3: Implement**

`internal/fsutil/fsutil.go`:

```go
// Package fsutil holds the one write primitive every spine command uses.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data via temp-file + rename in the same directory,
// so a crash never leaves a partial file.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".spine-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}
```

`internal/scaffold/scaffold.go`:

```go
// Package scaffold implements spine init: profile detection and first-time
// emission of the workflow file set.
package scaffold

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/tmpl"
)

// Result reports what Init did, as repo-relative paths.
type Result struct {
	Created []string
	Skipped []string
}

// Files is the scaffolded set, in emission order. Shared with update.
var Files = []struct{ TmplName, RelPath string }{
	{"CLAUDE.md.tmpl", "CLAUDE.md"},
	{"WORKFLOW.md.tmpl", "WORKFLOW.md"},
	{"harness-interface.md", "docs/harness-interface.md"},
	{"issues-README.md", "docs/issues/README.md"},
	{"issue.tmpl.md", "docs/issues/_template.md"},
	{"adr-README.md", "docs/adr/README.md"},
}

// DetectProfile inspects dir and returns a profile when signals are unambiguous.
func DetectProfile(dir string) (string, bool) {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	switch {
	case has("Cargo.toml"):
		return "rust", true
	case has("go.mod"):
		return "go-service", true
	case has("pyproject.toml"), has("setup.py"):
		return "py-tool", true
	}
	for _, pat := range []string{"*.pptx", "*.key"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pat)); len(m) > 0 {
			return "presentation", true
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		s := string(raw)
		for _, fw := range []string{`"react"`, `"vue"`, `"svelte"`, `"next"`} {
			if strings.Contains(s, fw) {
				return "ui", true
			}
		}
	}
	return "", false
}

// Init scaffolds dir with the current-generation file set; existing files are
// skipped, never overwritten.
func Init(dir, profile, name string) (Result, error) {
	reviewers, harness, err := tmpl.Defaults(profile)
	if err != nil {
		return Result{}, err
	}
	if name == "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return Result{}, err
		}
		name = filepath.Base(abs)
	}
	for _, d := range []string{"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return Result{}, err
		}
	}
	v := tmpl.Values{Project: name, Profile: profile, Reviewers: reviewers, Harness: harness, Version: tmpl.Version()}
	var res Result
	for _, f := range Files {
		dst := filepath.Join(dir, f.RelPath)
		if _, err := os.Stat(dst); err == nil {
			res.Skipped = append(res.Skipped, f.RelPath)
			continue
		}
		content, err := tmpl.Render("current", f.TmplName, v)
		if err != nil {
			return res, err
		}
		if err := fsutil.WriteFileAtomic(dst, []byte(content)); err != nil {
			return res, err
		}
		res.Created = append(res.Created, f.RelPath)
	}
	return res, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/fsutil/ ./internal/scaffold/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: fsutil atomic writes + scaffold init engine with profile detection"
```

---

### Task 3: CLI skeleton + `spine init` wiring

**Files:**
- Create: `cmd/spine/main.go`
- Test: `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: `scaffold.DetectProfile`, `scaffold.Init`, `tmpl.Version`, `tmpl.Profiles`
- Produces: `run(args []string, stdout, stderr io.Writer) int` — every later task registers its command here; `cmdUpdate`/`cmdADR`/`cmdDoctor` stubs return 2 with "not implemented" until their tasks land.

- [ ] **Step 1: Write the failing test**

`cmd/spine/main_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCmd(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestNoArgsShowsUsage(t *testing.T) {
	code, _, errs := runCmd(t)
	if code != 2 || !strings.Contains(errs, "usage: spine") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errs := runCmd(t, "bogus")
	if code != 2 || !strings.Contains(errs, "unknown command") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

func TestVersionCommand(t *testing.T) {
	code, out, _ := runCmd(t, "version")
	if code != 0 || !strings.Contains(out, "1") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestInitEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "init", "--dir", dir, "--name", "demo")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "create: WORKFLOW.md") || !strings.Contains(out, "done: rust") {
		t.Errorf("out=%q", out)
	}
}

func TestInitUndetectableNeedsProfile(t *testing.T) {
	code, _, errs := runCmd(t, "init", "--dir", t.TempDir())
	if code != 2 || !strings.Contains(errs, "--profile") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/spine/ -v`
Expected: FAIL — `run` undefined.

- [ ] **Step 3: Implement `cmd/spine/main.go`**

```go
// Command spine is the unified-workflow runtime companion.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
)

const usage = `usage: spine <command> [flags]

commands:
  init     scaffold the unified workflow into a repo
  update   regenerate machine-owned workflow files (dry-run by default; --write applies)
  adr      manage architecture decision records (new, list)
  doctor   read-only workflow health checks
  version  print the compiled template generation
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "init":
		return cmdInit(args[1:], stdout, stderr)
	case "update":
		return cmdUpdate(args[1:], stdout, stderr)
	case "adr":
		return cmdADR(args[1:], stdout, stderr)
	case "doctor":
		return cmdDoctor(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "spine template generation %d\n", tmpl.Version())
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return 2
	}
}

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "", "profile: "+strings.Join(tmpl.Profiles(), " | "))
	dir := fs.String("dir", ".", "repo root")
	name := fs.String("name", "", "project name (default: basename of dir)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	p := *profile
	if p == "" {
		detected, ok := scaffold.DetectProfile(*dir)
		if !ok {
			fmt.Fprintln(stderr, "cannot detect profile; pass --profile")
			return 2
		}
		p = detected
	}
	res, err := scaffold.Init(*dir, p, *name)
	if err != nil {
		fmt.Fprintln(stderr, "init:", err)
		return 2
	}
	for _, f := range res.Created {
		fmt.Fprintln(stdout, "create:", f)
	}
	for _, f := range res.Skipped {
		fmt.Fprintln(stdout, "skip (exists):", f)
	}
	fmt.Fprintf(stdout, "done: %s -> %s (template_version %d)\n", p, *dir, tmpl.Version())
	return 0
}

func cmdUpdate(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "update: not implemented yet")
	return 2
}

func cmdADR(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "adr: not implemented yet")
	return 2
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "doctor: not implemented yet")
	return 2
}
```

- [ ] **Step 4: Run tests, verify pass, and build**

Run: `go test ./cmd/spine/ -v && make build && ./bin/spine version`
Expected: PASS; binary prints `spine template generation 1`.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: CLI skeleton with init command wired"
```

---

### Task 4: update — key extraction and choice-vs-default

**Files:**
- Create: `internal/update/keys.go`
- Test: `internal/update/keys_test.go`

**Interfaces:**
- Consumes: `tmpl.Render`, `tmpl.Defaults`, `tmpl.Version`
- Produces: `update.ExtractKeys(content string) map[string]string` (dotted `model_routing.primary` for sub-keys); `update.Choices(extracted map[string]string, gen, project string) (map[string]string, error)`; `update.ProjectFromWorkflow(content, fallback string) string`; `setKey(content, dottedKey, val string) string` (unexported, used by Task 5)

- [ ] **Step 1: Write the failing test**

`internal/update/keys_test.go`:

```go
package update

import (
	"testing"
)

const gen0Hbmview = `# Workflow — hbmview

profile: rust
reviewers: [rust-reviewer, security-review]
functional_harness: cli    # cli | rest | framebuffer | none
gates: [grill, verify]             # mandatory; everything else advisory
model_default: claude-opus-4-8     # swappable; re-evaluate on major model/platform releases
security_routing: quality-framing-opus-4-8
stages: [grill, prd, issues, implement, functional-test, review, verify, ship, deploy, docs, handoff]

See ` + "`docs/harness-interface.md`" + ` for the functional-test harness contract.
Mandatory gates: a PRD up front (grill-with-docs -> to-prd) and verification before completion.
`

func TestExtractKeys(t *testing.T) {
	keys := ExtractKeys(gen0Hbmview)
	want := map[string]string{
		"profile":            "rust",
		"reviewers":          "[rust-reviewer, security-review]",
		"functional_harness": "cli",
		"gates":              "[grill, verify]",
		"model_default":      "claude-opus-4-8",
		"security_routing":   "quality-framing-opus-4-8",
	}
	for k, v := range want {
		if keys[k] != v {
			t.Errorf("keys[%q] = %q, want %q", k, keys[k], v)
		}
	}
	if _, ok := keys["template_version"]; ok {
		t.Error("gen0 file must have no template_version")
	}
}

func TestExtractKeysRoutingSubBlock(t *testing.T) {
	content := "model_routing:\n  primary: claude-fable-5   # x\n  fallback: claude-opus-4-8\n  routine: claude-sonnet-5\neffort: high\n"
	keys := ExtractKeys(content)
	if keys["model_routing.primary"] != "claude-fable-5" ||
		keys["model_routing.routine"] != "claude-sonnet-5" ||
		keys["effort"] != "high" {
		t.Errorf("keys = %#v", keys)
	}
}

func TestProjectFromWorkflow(t *testing.T) {
	if got := ProjectFromWorkflow(gen0Hbmview, "fb"); got != "hbmview" {
		t.Errorf("got %q", got)
	}
	if got := ProjectFromWorkflow("no title", "fb"); got != "fb" {
		t.Errorf("fallback got %q", got)
	}
}

// The heart of un-stranding: values equal to their own generation's defaults
// are NOT choices; hbmview's gen0 model_default must not survive.
func TestChoicesDropsGenerationDefaults(t *testing.T) {
	choices, err := Choices(ExtractKeys(gen0Hbmview), "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	if choices["profile"] != "rust" {
		t.Errorf("profile must always be preserved, got %#v", choices)
	}
	if _, ok := choices["model_default"]; ok {
		t.Errorf("gen0-default model_default wrongly kept as a choice: %#v", choices)
	}
	if _, ok := choices["reviewers"]; ok {
		t.Errorf("profile-derived reviewers wrongly kept: %#v", choices)
	}
}

func TestChoicesKeepsRealChoices(t *testing.T) {
	custom := ExtractKeys(gen0Hbmview)
	custom["functional_harness"] = "rest" // user overrode the rust default (cli)
	choices, err := Choices(custom, "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	if choices["functional_harness"] != "rest" {
		t.Errorf("real choice dropped: %#v", choices)
	}
}

func TestSetKey(t *testing.T) {
	content := "profile: rust\nmodel_routing:\n  primary: claude-fable-5          # comment\neffort: high    # c2\n"
	got := setKey(content, "model_routing.primary", "custom-model")
	if want := "  primary: custom-model    # comment"; !containsLine(got, want) {
		t.Errorf("sub-key: got\n%s", got)
	}
	got = setKey(content, "effort", "xhigh")
	if want := "effort: xhigh    # c2"; !containsLine(got, want) {
		t.Errorf("top key: got\n%s", got)
	}
}

func containsLine(content, line string) bool {
	for _, l := range splitLines(content) {
		if l == line {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/update/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement `internal/update/keys.go`**

```go
// Package update regenerates machine-owned workflow files from the compiled
// templates, preserving deliberate per-repo choices (spec: ownership split +
// config-preserving regeneration + choice-vs-default rule).
package update

import (
	"fmt"
	"strings"

	"github.com/russellpope/spine/internal/tmpl"
)

var topKeys = []string{
	"profile", "template_version", "reviewers", "functional_harness", "gates",
	"effort", "model_default", "security_routing", "stages",
}

var routingKeys = []string{"primary", "fallback", "routine"}

func splitLines(s string) []string { return strings.Split(s, "\n") }

// cutKey returns the value of "key: value  # comment" with comment stripped.
func cutKey(line, key string) (string, bool) {
	rest, ok := strings.CutPrefix(line, key+":")
	if !ok {
		return "", false
	}
	if i := strings.Index(rest, "#"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest), true
}

// ExtractKeys pulls known config keys out of WORKFLOW.md content. Sub-keys of
// model_routing come back dotted: "model_routing.primary".
func ExtractKeys(content string) map[string]string {
	keys := map[string]string{}
	inRouting := false
	for _, line := range splitLines(content) {
		if strings.HasPrefix(line, "model_routing:") {
			inRouting = true
			continue
		}
		if inRouting {
			if strings.HasPrefix(line, "  ") {
				trimmed := strings.TrimSpace(line)
				for _, k := range routingKeys {
					if v, ok := cutKey(trimmed, k); ok {
						keys["model_routing."+k] = v
					}
				}
				continue
			}
			inRouting = false
		}
		for _, k := range topKeys {
			if v, ok := cutKey(line, k); ok {
				keys[k] = v
			}
		}
	}
	return keys
}

// ProjectFromWorkflow reads the project name from the "# Workflow — X" title.
func ProjectFromWorkflow(content, fallback string) string {
	for _, line := range splitLines(content) {
		if rest, ok := strings.CutPrefix(line, "# Workflow — "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return fallback
}

// Choices filters extracted keys down to deliberate user choices: values that
// differ from what the file's own generation would have rendered by default.
// profile is always a choice; template_version never is.
func Choices(extracted map[string]string, gen, project string) (map[string]string, error) {
	profile := extracted["profile"]
	if profile == "" {
		return nil, fmt.Errorf("no profile: key found in WORKFLOW.md")
	}
	reviewers, harness, err := tmpl.Defaults(profile)
	if err != nil {
		return nil, err
	}
	rendered, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", tmpl.Values{
		Project: project, Profile: profile, Reviewers: reviewers, Harness: harness, Version: tmpl.Version(),
	})
	if err != nil {
		return nil, err
	}
	defaults := ExtractKeys(rendered)
	choices := map[string]string{"profile": profile}
	for k, v := range extracted {
		if k == "profile" || k == "template_version" {
			continue
		}
		if defaults[k] != v {
			choices[k] = v
		}
	}
	return choices, nil
}

// setKey rewrites the value of a top-level or model_routing.* key in rendered
// WORKFLOW.md content, keeping the template's trailing comment.
func setKey(content, dotted, val string) string {
	top, sub, isSub := strings.Cut(dotted, ".")
	lines := splitLines(content)
	inBlock := false
	for i, line := range lines {
		if isSub {
			if strings.HasPrefix(line, top+":") {
				inBlock = true
				continue
			}
			if !inBlock {
				continue
			}
			if !strings.HasPrefix(line, "  ") {
				inBlock = false
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), sub+":") {
				lines[i] = replaceValue(line, sub, val)
				return strings.Join(lines, "\n")
			}
			continue
		}
		if strings.HasPrefix(line, top+":") {
			lines[i] = replaceValue(line, top, val)
			return strings.Join(lines, "\n")
		}
	}
	return strings.Join(lines, "\n")
}

func replaceValue(line, key, val string) string {
	indent := line[:strings.Index(line, key)]
	comment := ""
	if i := strings.Index(line, "#"); i >= 0 {
		comment = "    " + strings.TrimRight(line[i:], " ")
	}
	return indent + key + ": " + val + comment
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/update/ -v`
Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: update key extraction with choice-vs-default rule"
```

---

### Task 5: update — diff, planners, Run

**Files:**
- Create: `internal/update/diff.go`, `internal/update/update.go`
- Test: `internal/update/update_test.go` (append to package; `diff` covered here too)

**Interfaces:**
- Consumes: Task 4 helpers; `tmpl.*`; `fsutil.WriteFileAtomic`; `scaffold.Init` (tests only)
- Produces: `update.FileState` (`UpToDate`, `Pending`, `SkippedUnrecognized`); `update.FileReport{Path string; State FileState; Diff string; Unrecognized []string}` (+ unexported `newContent string`); `update.Options{Dir string; Write, Force bool}`; `update.Run(opts Options) ([]FileReport, error)`; `update.Diff(path, a, b string) string`

- [ ] **Step 1: Write the failing test**

`internal/update/update_test.go`:

```go
package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

const gen0HbmviewClaude = `# hbmview

Uses the **unified workflow** — see ` + "`WORKFLOW.md`" + ` for the active profile (` + "`rust`" + `) and stages.

- Specs / PRDs -> ` + "`docs/specs/`" + `
- Decisions (ADRs) -> ` + "`docs/adr/`" + `
- Issue / bug ledger -> ` + "`docs/issues/`" + ` (dependency convention in ` + "`docs/issues/README.md`" + `)
- Handoffs -> ` + "`docs/handoffs/`" + `

**Mandatory gates:** a PRD up front (grill-with-docs -> to-prd) and verification before completion.
**Model:** see ` + "`WORKFLOW.md`" + ` ` + "`model_default`" + ` (swappable).
`

func writeRepo(t *testing.T, workflow, claude string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if workflow != "" {
		if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if claude != "" {
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claude), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func report(t *testing.T, reports []FileReport, path string) FileReport {
	t.Helper()
	for _, r := range reports {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("no report for %s in %#v", path, reports)
	return FileReport{}
}

func TestDiffEmptyWhenEqual(t *testing.T) {
	if d := Diff("x", "a\nb", "a\nb"); d != "" {
		t.Errorf("got %q", d)
	}
	d := Diff("x", "a\nb\nc", "a\nB\nc")
	if !strings.Contains(d, "- b") || !strings.Contains(d, "+ B") || !strings.Contains(d, "  a") {
		t.Errorf("diff:\n%s", d)
	}
}

func TestFreshInitIsUpToDate(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("%s: state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestGen0CleanClaim(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("WORKFLOW state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	for _, want := range []string{"template_version: 1", "primary: claude-fable-5",
		"model_default: claude-fable-5", "profile: rust", "functional_harness: cli"} {
		if !strings.Contains(wf.newContent, want) {
			t.Errorf("regenerated WORKFLOW missing %q", want)
		}
	}
	if strings.Contains(wf.newContent, "model_default: claude-opus-4-8") {
		t.Error("stale gen0 default survived regeneration")
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != Pending {
		t.Fatalf("CLAUDE state=%v", cl.State)
	}
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin v1 -->") {
		t.Error("claimed CLAUDE.md lacks markers")
	}
	if got := strings.Count(cl.newContent, "# hbmview"); got != 1 {
		t.Errorf("clean claim duplicated content, %d title lines", got)
	}
}

func TestGen0WriteThenUpToDate(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("after write, %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestUnrecognizedEditsSkipUnlessForce(t *testing.T) {
	custom := gen0Hbmview + "custom_rule: never deploy on fridays\n"
	dir := writeRepo(t, custom, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != SkippedUnrecognized || len(wf.Unrecognized) != 1 ||
		!strings.Contains(wf.Unrecognized[0], "custom_rule") {
		t.Fatalf("state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	// force regenerates (dropping the line) and write applies it
	if _, err := Run(Options{Dir: dir, Write: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if strings.Contains(string(got), "custom_rule") {
		t.Error("force did not drop unrecognized line")
	}
	if !strings.Contains(string(got), "template_version: 1") {
		t.Error("force did not regenerate")
	}
}

func TestCustomChoiceSurvivesUpdate(t *testing.T) {
	custom := strings.Replace(gen0Hbmview, "functional_harness: cli", "functional_harness: rest", 1)
	dir := writeRepo(t, custom, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	if !strings.Contains(wf.newContent, "functional_harness: rest") {
		t.Error("user harness choice lost")
	}
}

func TestLegacyClaudeWithUserContentPreserved(t *testing.T) {
	userClaude := gen0HbmviewClaude + "\n## Local invariants\n\n- verify with `lms ps --json`\n"
	dir := writeRepo(t, gen0Hbmview, userClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != Pending {
		t.Fatalf("state=%v", cl.State)
	}
	if !strings.Contains(cl.newContent, "lms ps --json") {
		t.Error("user content dropped")
	}
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin") {
		t.Error("markers not inserted at top")
	}
}

func TestMarkerBlockReplacedUserTailKept(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(raw, []byte("\n## Notes\n\n- remote is github\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != UpToDate {
		t.Fatalf("same-version block should be up-to-date, state=%v diff:\n%s", cl.State, cl.Diff)
	}
}

func TestUnbalancedMarkersSkipped(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, _ := os.ReadFile(path)
	broken := strings.Replace(string(raw), "<!-- spine:end -->", "", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != SkippedUnrecognized {
		t.Fatalf("unbalanced markers must skip even with force, state=%v", cl.State)
	}
}

func TestMissingWorkflowIsHardError(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want error when WORKFLOW.md missing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/update/ -v`
Expected: FAIL — `Diff`, `Run`, `FileReport` undefined.

- [ ] **Step 3: Implement `internal/update/diff.go`**

```go
package update

import (
	"fmt"
	"strings"
)

// Diff returns a minimal LCS line diff of a -> b, or "" when equal. Files
// here are small (tens of lines), so the O(n*m) table is fine.
func Diff(path, a, b string) string {
	if a == b {
		return ""
	}
	al, bl := splitLines(a), splitLines(b)
	m, n := len(al), len(bl)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			switch {
			case al[i] == bl[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s (on disk)\n+++ %s (regenerated)\n", path, path)
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case al[i] == bl[j]:
			sb.WriteString("  " + al[i] + "\n")
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			sb.WriteString("- " + al[i] + "\n")
			i++
		default:
			sb.WriteString("+ " + bl[j] + "\n")
			j++
		}
	}
	for ; i < m; i++ {
		sb.WriteString("- " + al[i] + "\n")
	}
	for ; j < n; j++ {
		sb.WriteString("+ " + bl[j] + "\n")
	}
	return sb.String()
}
```

- [ ] **Step 4: Implement `internal/update/update.go`**

```go
package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/tmpl"
)

// FileState classifies what update would do to one file.
type FileState int

const (
	UpToDate FileState = iota
	Pending
	SkippedUnrecognized
)

// FileReport is the per-file outcome. newContent stays unexported: only Run
// writes it, and only for Pending files.
type FileReport struct {
	Path         string
	State        FileState
	Diff         string
	Unrecognized []string
	newContent   string
}

// Options configures Run. Zero value = dry-run on ".".
type Options struct {
	Dir   string
	Write bool
	Force bool
}

const (
	markerBegin = "<!-- spine:begin"
	markerEnd   = "<!-- spine:end -->"
)

// simple machine-owned files: regenerate wholesale, no key extraction.
// inGen0 marks files whose gen0 content differs from current.
var simpleFiles = []struct {
	tmplName, relPath string
	inGen0            bool
}{
	{"harness-interface.md", "docs/harness-interface.md", true},
	{"issues-README.md", "docs/issues/README.md", false},
	{"issue.tmpl.md", "docs/issues/_template.md", false},
	{"adr-README.md", "docs/adr/README.md", false},
}

// Run plans (and with opts.Write, applies) regeneration of every managed file.
func Run(opts Options) ([]FileReport, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	wf, vals, gen, err := planWorkflow(opts.Dir)
	if err != nil {
		return nil, err
	}
	reports := []FileReport{wf}
	cl, err := planClaude(opts.Dir, gen, vals)
	if err != nil {
		return nil, err
	}
	reports = append(reports, cl)
	for _, f := range simpleFiles {
		r, err := planSimple(opts.Dir, gen, f.tmplName, f.relPath, f.inGen0, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	// policy: unrecognized edits skip the file unless --force; files with no
	// regenerable content (nil newContent) stay skipped regardless.
	for i := range reports {
		r := &reports[i]
		if len(r.Unrecognized) > 0 {
			if opts.Force && r.newContent != "" {
				r.State = Pending
			} else {
				r.State = SkippedUnrecognized
			}
		}
	}
	if opts.Write {
		for i := range reports {
			r := &reports[i]
			if r.State != Pending {
				continue
			}
			dst := filepath.Join(opts.Dir, r.Path)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return reports, err
			}
			if err := fsutil.WriteFileAtomic(dst, []byte(r.newContent)); err != nil {
				return reports, err
			}
		}
	}
	return reports, nil
}

func planWorkflow(dir string) (FileReport, tmpl.Values, string, error) {
	report := FileReport{Path: "WORKFLOW.md"}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return report, tmpl.Values{}, "", fmt.Errorf("read %s (run spine init first?): %w", path, err)
	}
	old := string(raw)
	keys := ExtractKeys(old)
	gen := "gen0"
	if keys["template_version"] != "" {
		gen = "current"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	project := ProjectFromWorkflow(old, filepath.Base(abs))
	profile := keys["profile"]
	if profile == "" {
		return report, tmpl.Values{}, "", fmt.Errorf("%s has no profile: line", path)
	}
	defRev, defHarness, err := tmpl.Defaults(profile)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	vals := tmpl.Values{Project: project, Profile: profile, Reviewers: defRev, Harness: defHarness, Version: tmpl.Version()}

	// unrecognized detection: what the old generation would look like with
	// every extracted key applied — anything beyond that is a local edit.
	expectedOld, err := tmpl.Render(gen, "WORKFLOW.md.tmpl", vals)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	for k, v := range keys {
		expectedOld = setKey(expectedOld, k, v)
	}
	report.Unrecognized = unrecognizedLines(old, expectedOld)

	choices, err := Choices(keys, gen, project)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	newContent, err := tmpl.Render("current", "WORKFLOW.md.tmpl", vals)
	if err != nil {
		return report, tmpl.Values{}, "", err
	}
	for k, v := range choices {
		if k == "profile" {
			continue
		}
		newContent = setKey(newContent, k, v)
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, vals, gen, nil
}

func planClaude(dir, gen string, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: "CLAUDE.md"}
	block, err := tmpl.Render("current", "CLAUDE.md.tmpl", vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
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
			// unbalanced markers: never force-droppable, no newContent.
			report.Unrecognized = []string{err.Error()}
			return report, nil
		}
		newContent = replaced
	} else {
		gen0, err := tmpl.Render("gen0", "CLAUDE.md.tmpl", vals)
		if err != nil {
			return report, err
		}
		if strings.TrimSpace(old) == strings.TrimSpace(gen0) {
			newContent = block // pristine legacy file: clean claim
		} else {
			newContent = block + "\n" + old // claim on top, preserve everything
		}
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

func replaceMarkerBlock(old, block string) (string, error) {
	if strings.Count(old, markerBegin) != 1 || strings.Count(old, markerEnd) != 1 {
		return "", fmt.Errorf("CLAUDE.md spine markers unbalanced; fix by hand")
	}
	begin := strings.Index(old, markerBegin)
	end := strings.Index(old, markerEnd)
	if end < begin {
		return "", fmt.Errorf("CLAUDE.md spine markers out of order; fix by hand")
	}
	return old[:begin] + strings.TrimSuffix(block, "\n") + old[end+len(markerEnd):], nil
}

func planSimple(dir, gen, tmplName, relPath string, inGen0 bool, vals tmpl.Values) (FileReport, error) {
	report := FileReport{Path: relPath}
	newContent, err := tmpl.Render("current", tmplName, vals)
	if err != nil {
		return report, err
	}
	path := filepath.Join(dir, relPath)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		report.State = Pending
		report.Diff = Diff(relPath, "", newContent)
		report.newContent = newContent
		return report, nil
	}
	if err != nil {
		return report, err
	}
	old := string(raw)
	expectedGen := "current"
	if gen == "gen0" && inGen0 {
		expectedGen = "gen0"
	}
	expectedOld, err := tmpl.Render(expectedGen, tmplName, vals)
	if err != nil {
		return report, err
	}
	report.Unrecognized = unrecognizedLines(old, expectedOld)
	if d := Diff(relPath, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, nil
}

// unrecognizedLines returns non-blank lines of got that expected does not
// contain anywhere (order-insensitive, trailing-space-insensitive).
func unrecognizedLines(got, expected string) []string {
	want := map[string]bool{}
	for _, l := range splitLines(expected) {
		want[strings.TrimRight(l, " ")] = true
	}
	var extra []string
	for _, l := range splitLines(got) {
		t := strings.TrimRight(l, " ")
		if t == "" || want[t] {
			continue
		}
		extra = append(extra, t)
	}
	return extra
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/update/ -v`
Expected: PASS (all Task 4 + Task 5 tests).

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: update engine — diff, planners, ownership-split regeneration"
```

---

### Task 6: `spine update` CLI wiring

**Files:**
- Modify: `cmd/spine/main.go` (replace the `cmdUpdate` stub)
- Test: append to `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: `update.Run`, `update.Options`, `update.FileState` constants; test also imports `github.com/russellpope/spine/internal/tmpl`
- Produces: `spine update [--dir D] [--write] [--force]` with exit 0 (all current) / 1 (pending or skipped) / 2 (hard error)

- [ ] **Step 1: Write the failing test** (append to `cmd/spine/main_test.go`)

```go
func TestUpdateDryRunThenWrite(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo"); code != 0 {
		t.Fatal(errs)
	}
	// fresh scaffold: nothing pending
	code, out, _ := runCmd(t, "update", "--dir", dir)
	if code != 0 || !strings.Contains(out, "up-to-date") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	// regress the repo to a TRUE gen0 state (rendering gen0 templates) —
	// merely deleting the stamp line would leave current-only lines that
	// read as unrecognized edits against gen0, i.e. Skipped, not Pending.
	vals := tmpl.Values{Project: "demo", Profile: "rust",
		Reviewers: "rust-reviewer, security-review", Harness: "cli", Version: 1}
	for tmplName, rel := range map[string]string{
		"WORKFLOW.md.tmpl": "WORKFLOW.md",
		"CLAUDE.md.tmpl":   "CLAUDE.md",
	} {
		gen0, err := tmpl.Render("gen0", tmplName, vals)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(gen0), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, out, _ = runCmd(t, "update", "--dir", dir)
	if code != 1 || !strings.Contains(out, "+ template_version: 1") {
		t.Fatalf("dry-run code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "update", "--dir", dir, "--write")
	if code != 0 || !strings.Contains(out, "updated: WORKFLOW.md") {
		t.Fatalf("write code=%d out=%q", code, out)
	}
	code, _, _ = runCmd(t, "update", "--dir", dir)
	if code != 0 {
		t.Fatalf("after write, code=%d", code)
	}
}

func TestUpdateMissingWorkflowExits2(t *testing.T) {
	code, _, errs := runCmd(t, "update", "--dir", t.TempDir())
	if code != 2 || !strings.Contains(errs, "spine init") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/spine/ -run TestUpdate -v`
Expected: FAIL — stub returns 2 with "not implemented".

- [ ] **Step 3: Replace the `cmdUpdate` stub in `cmd/spine/main.go`**

Add imports: `bytes`, `os/exec`, `github.com/russellpope/spine/internal/update`.

```go
func cmdUpdate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	write := fs.Bool("write", false, "apply changes (default: dry-run diff)")
	force := fs.Bool("force", false, "regenerate files with unrecognized local edits (diff shows what gets dropped)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *write {
		warnDirty(*dir, stderr)
	}
	reports, err := update.Run(update.Options{Dir: *dir, Write: *write, Force: *force})
	if err != nil {
		fmt.Fprintln(stderr, "update:", err)
		return 2
	}
	outstanding := 0
	for _, r := range reports {
		switch r.State {
		case update.UpToDate:
			fmt.Fprintf(stdout, "up-to-date: %s\n", r.Path)
		case update.Pending:
			if *write {
				fmt.Fprintf(stdout, "updated: %s\n", r.Path)
			} else {
				outstanding++
				fmt.Fprint(stdout, r.Diff)
			}
		case update.SkippedUnrecognized:
			outstanding++
			fmt.Fprintf(stderr, "skipped %s — unrecognized local edits (use --force to drop):\n", r.Path)
			for _, l := range r.Unrecognized {
				fmt.Fprintf(stderr, "  %s\n", l)
			}
		}
	}
	if outstanding > 0 {
		return 1
	}
	return 0
}

// warnDirty nudges the user to review post-write diffs with git; git being
// absent or dir not being a repo is fine and silent.
func warnDirty(dir string, stderr io.Writer) {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		fmt.Fprintln(stderr, "warning: repo has uncommitted changes — review the update with git diff afterwards")
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/spine/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: spine update command (dry-run default, --write, --force)"
```

---

### Task 7: adr package + CLI

**Files:**
- Create: `internal/adr/adr.go`
- Modify: `cmd/spine/main.go` (replace `cmdADR` stub)
- Test: `internal/adr/adr_test.go`, append to `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: `templates.FS` (adr.tmpl.md), `fsutil.WriteFileAtomic`
- Produces: `adr.Entry{ID int; Title, Status, Path string}`; `adr.List(dir string) ([]Entry, error)`; `adr.New(dir, title string, supersedes int) (string, error)`

- [ ] **Step 1: Write the failing test**

`internal/adr/adr_test.go`:

```go
package adr_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/adr"
)

func adrDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewNumbersFromOne(t *testing.T) {
	dir := adrDir(t)
	path, err := adr.New(dir, "Go with stdlib only", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "0001-go-with-stdlib-only.md" {
		t.Fatalf("path = %s", path)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"id: 0001", "title: Go with stdlib only", "status: Accepted",
		"# 0001: Go with stdlib only", "## Decision"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %q in:\n%s", want, raw)
		}
	}
	path2, err := adr.New(dir, "Second decision", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path2) != "0002-second-decision.md" {
		t.Fatalf("path2 = %s", path2)
	}
}

func TestListSorted(t *testing.T) {
	dir := adrDir(t)
	adr.New(dir, "First", 0)
	adr.New(dir, "Second", 0)
	entries, err := adr.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].ID != 1 || entries[1].Title != "Second" ||
		entries[0].Status != "Accepted" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestSupersedeFlipsStatus(t *testing.T) {
	dir := adrDir(t)
	first, _ := adr.New(dir, "Old way", 0)
	_, err := adr.New(dir, "New way", 1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(first)
	if !strings.Contains(string(raw), "status: Superseded by 0002") {
		t.Errorf("old ADR not flipped:\n%s", raw)
	}
	entries, _ := adr.List(dir)
	if entries[1].Status != "Accepted" {
		t.Errorf("new ADR status = %q", entries[1].Status)
	}
	raw2, _ := os.ReadFile(filepath.Join(dir, "docs", "adr", "0002-new-way.md"))
	if !strings.Contains(string(raw2), "supersedes: 0001") {
		t.Errorf("new ADR missing supersedes line:\n%s", raw2)
	}
}

func TestSupersedeMissingTarget(t *testing.T) {
	if _, err := adr.New(adrDir(t), "X", 9); err == nil {
		t.Fatal("want error for missing supersede target")
	}
}

func TestSlugStripsPunctuation(t *testing.T) {
	dir := adrDir(t)
	path, err := adr.New(dir, "docs/specs absorbs plans, fleet-wide!", 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "0001-docs-specs-absorbs-plans-fleet-wide.md" {
		t.Fatalf("path = %s", path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adr/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement `internal/adr/adr.go`**

```go
// Package adr manages the docs/adr/ ledger: immutable decisions, supersede
// status flips being the single permitted mutation.
package adr

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/templates"
)

// Entry is one parsed ADR file.
type Entry struct {
	ID     int
	Title  string
	Status string
	Path   string
}

var fileRe = regexp.MustCompile(`^(\d{4})-.+\.md$`)

// List parses docs/adr/ under dir, sorted by ID. Files not matching
// NNNN-slug.md (e.g. README.md) are ignored.
func List(dir string) ([]Entry, error) {
	adrDir := filepath.Join(dir, "docs", "adr")
	des, err := os.ReadDir(adrDir)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, de := range des {
		m := fileRe.FindStringSubmatch(de.Name())
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		e := Entry{ID: id, Path: filepath.Join(adrDir, de.Name())}
		raw, err := os.ReadFile(e.Path)
		if err != nil {
			return nil, err
		}
		e.Title, e.Status = parseFrontMatter(string(raw))
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func parseFrontMatter(content string) (title, status string) {
	for _, line := range strings.Split(content, "\n") {
		if t, ok := strings.CutPrefix(line, "title: "); ok {
			title = strings.TrimSpace(t)
		}
		if s, ok := strings.CutPrefix(line, "status: "); ok {
			status = strings.TrimSpace(s)
		}
	}
	return title, status
}

// New writes the next-numbered ADR; supersedes > 0 also flips that ADR's
// status line. Returns the new file's path.
func New(dir, title string, supersedes int) (string, error) {
	entries, err := List(dir)
	if err != nil {
		return "", err
	}
	next := 1
	for _, e := range entries {
		if e.ID >= next {
			next = e.ID + 1
		}
	}
	var target *Entry
	if supersedes > 0 {
		for i := range entries {
			if entries[i].ID == supersedes {
				target = &entries[i]
			}
		}
		if target == nil {
			return "", fmt.Errorf("supersedes target %04d not found", supersedes)
		}
	}
	raw, err := templates.FS.ReadFile("current/adr.tmpl.md")
	if err != nil {
		return "", err
	}
	sup := ""
	if supersedes > 0 {
		sup = fmt.Sprintf("\nsupersedes: %04d", supersedes)
	}
	id := fmt.Sprintf("%04d", next)
	content := strings.NewReplacer(
		"{{ADR_ID}}", id,
		"{{ADR_TITLE}}", title,
		"{{ADR_DATE}}", time.Now().Format("2006-01-02"),
		"{{ADR_SUPERSEDES}}", sup,
	).Replace(string(raw))
	path := filepath.Join(dir, "docs", "adr", id+"-"+slugify(title)+".md")
	if err := fsutil.WriteFileAtomic(path, []byte(content)); err != nil {
		return "", err
	}
	if target != nil {
		if err := flipStatus(target.Path, next); err != nil {
			return "", err
		}
	}
	return path, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b []rune
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b = append(b, r)
		default:
			if len(b) > 0 && b[len(b)-1] != '-' {
				b = append(b, '-')
			}
		}
	}
	return strings.Trim(string(b), "-")
}

func flipStatus(path string, by int) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "status: ") {
			lines[i] = fmt.Sprintf("status: Superseded by %04d", by)
			return fsutil.WriteFileAtomic(path, []byte(strings.Join(lines, "\n")))
		}
	}
	return fmt.Errorf("no status line in %s", path)
}
```

- [ ] **Step 4: Replace the `cmdADR` stub in `cmd/spine/main.go`**

Add import `github.com/russellpope/spine/internal/adr`.

```go
func cmdADR(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine adr <new|list> [flags]  (adr new [--dir D] [--supersedes N] "Title")`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("adr new", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		supersedes := fs.Int("supersedes", 0, "ADR number this decision supersedes")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, `usage: spine adr new [--dir D] [--supersedes N] "Title" (flags before title)`)
			return 2
		}
		path, err := adr.New(*dir, fs.Arg(0), *supersedes)
		if err != nil {
			fmt.Fprintln(stderr, "adr new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "list":
		fs := flag.NewFlagSet("adr list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		entries, err := adr.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "adr list:", err)
			return 2
		}
		for _, e := range entries {
			fmt.Fprintf(stdout, "%04d  %-22s  %s\n", e.ID, e.Status, e.Title)
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown adr subcommand %q\n", args[0])
		return 2
	}
}
```

Append to `cmd/spine/main_test.go`:

```go
func TestADRNewAndList(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "adr", "new", "--dir", dir, "Go with stdlib only")
	if code != 0 || !strings.Contains(out, "0001-go-with-stdlib-only.md") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = runCmd(t, "adr", "list", "--dir", dir)
	if code != 0 || !strings.Contains(out, "0001  Accepted") {
		t.Fatalf("list code=%d out=%q", code, out)
	}
	code, _, errs = runCmd(t, "adr", "new", "--dir", dir, "--supersedes", "9", "X")
	if code != 2 || !strings.Contains(errs, "not found") {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/adr/ ./cmd/spine/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: spine adr new/list with supersede status flip"
```

---

### Task 8: doctor package + CLI

**Files:**
- Create: `internal/doctor/doctor.go`
- Modify: `cmd/spine/main.go` (replace `cmdDoctor` stub)
- Test: `internal/doctor/doctor_test.go`, append to `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: `update.Run`, `update.Options`, `update.FileState`; `adr.List`; `tmpl.Version`
- Produces: `doctor.Finding{ID, Severity, Path, Message string}` (json tags lowercase); `doctor.Run(dir string) ([]Finding, error)`

- [ ] **Step 1: Write the failing test**

`internal/doctor/doctor_test.go`:

```go
package doctor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/scaffold"
)

func ids(fs []doctor.Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.ID]++
	}
	return m
}

func TestCleanScaffoldNoFindings(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("want clean, got %#v", fs)
	}
}

func TestMissingPiecesD1(t *testing.T) {
	fs, err := doctor.Run(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ids(fs)["D1"] == 0 {
		t.Fatalf("want D1 findings, got %#v", fs)
	}
}

func TestStaleGen0D2AndD3(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	// regress to a TRUE gen0 repo by rendering the gen0 templates (stripping
	// the stamp from a current file would read as unrecognized edits instead)
	vals := tmpl.Values{Project: "demo", Profile: "rust",
		Reviewers: "rust-reviewer, security-review", Harness: "cli", Version: 1}
	for tmplName, rel := range map[string]string{
		"WORKFLOW.md.tmpl": "WORKFLOW.md",
		"CLAUDE.md.tmpl":   "CLAUDE.md",
	} {
		gen0, err := tmpl.Render("gen0", tmplName, vals)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(gen0), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := ids(fs)
	if got["D2"] == 0 || got["D3"] == 0 {
		t.Fatalf("want D2 (stale, pending update) + D3 (no markers), got %#v", fs)
	}
}

func TestSuperpowersDriftD5(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(dir, "docs", "superpowers", "plans")
	os.MkdirAll(sp, 0o755)
	os.WriteFile(filepath.Join(sp, "old-plan.md"), []byte("x"), 0o644)
	fs, _ := doctor.Run(dir)
	if ids(fs)["D5"] != 1 {
		t.Fatalf("want one D5, got %#v", fs)
	}
}

func TestADRProblemsD6(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	adr.New(dir, "Real one", 0)
	// duplicate number + bogus status
	os.WriteFile(filepath.Join(dir, "docs", "adr", "0001-dupe.md"),
		[]byte("---\nid: 0001\ntitle: Dupe\nstatus: Draft\ndate: 2026-07-01\n---\n"), 0o644)
	fs, _ := doctor.Run(dir)
	got := ids(fs)
	if got["D6"] < 2 {
		t.Fatalf("want duplicate+status D6 findings, got %#v", fs)
	}
}

```

Test-file imports for the above: `os`, `path/filepath`, `testing`, plus
`github.com/russellpope/spine/internal/{adr,doctor,scaffold,tmpl}`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/doctor/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement `internal/doctor/doctor.go`**

```go
// Package doctor runs read-only workflow health checks (spec D1–D6).
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/update"
)

// Finding is one doctor result; Severity is error | warn | info.
type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

var required = []string{
	"WORKFLOW.md", "CLAUDE.md", "docs/harness-interface.md",
	"docs/specs", "docs/adr", "docs/issues", "docs/handoffs",
}

// Run executes all checks. It never writes.
func Run(dir string) ([]Finding, error) {
	fs := []Finding{}
	missingCore := false
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			fs = append(fs, Finding{"D1", "error", rel, "missing — run spine init"})
			if rel == "WORKFLOW.md" {
				missingCore = true
			}
		}
	}
	if !missingCore {
		fs = append(fs, updateChecks(dir)...)
	}
	fs = append(fs, markerCheck(dir)...)
	fs = append(fs, superpowersCheck(dir)...)
	fs = append(fs, adrCheck(dir)...)
	return fs, nil
}

// updateChecks maps a dry-run of update onto D2 (stale) and D4 (unrecognized).
func updateChecks(dir string) []Finding {
	var fs []Finding
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		return []Finding{{"D2", "error", "WORKFLOW.md", "update cannot run: " + err.Error()}}
	}
	pending := 0
	for _, r := range reports {
		switch r.State {
		case update.Pending:
			pending++
		case update.SkippedUnrecognized:
			fs = append(fs, Finding{"D4", "warn", r.Path,
				fmt.Sprintf("%d unrecognized local edit(s) in a machine-owned file — reconcile or spine update --force", len(r.Unrecognized))})
		}
	}
	if pending > 0 {
		fs = append(fs, Finding{"D2", "warn", "WORKFLOW.md",
			fmt.Sprintf("%d file(s) behind template generation — run spine update", pending)})
	}
	return fs
}

func markerCheck(dir string) []Finding {
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		return nil // D1 already reported it
	}
	begins := strings.Count(string(raw), "<!-- spine:begin")
	ends := strings.Count(string(raw), "<!-- spine:end -->")
	switch {
	case begins == 1 && ends == 1:
		return nil
	case begins == 0 && ends == 0:
		return []Finding{{"D3", "info", "CLAUDE.md", "no spine markers (legacy file) — spine update will claim it"}}
	default:
		return []Finding{{"D3", "error", "CLAUDE.md",
			fmt.Sprintf("unbalanced spine markers (%d begin / %d end) — fix by hand", begins, ends)}}
	}
}

func superpowersCheck(dir string) []Finding {
	var fs []Finding
	for _, sub := range []string{"specs", "plans"} {
		glob := filepath.Join(dir, "docs", "superpowers", sub, "*.md")
		if m, _ := filepath.Glob(glob); len(m) > 0 {
			fs = append(fs, Finding{"D5", "info", "docs/superpowers/" + sub,
				fmt.Sprintf("%d artifact(s) in legacy location — new work goes to docs/specs/", len(m))})
		}
	}
	return fs
}

func adrCheck(dir string) []Finding {
	entries, err := adr.List(dir)
	if err != nil {
		return nil // no docs/adr — D1 covers structural absence
	}
	var fs []Finding
	seen := map[int]bool{}
	for _, e := range entries {
		if seen[e.ID] {
			fs = append(fs, Finding{"D6", "error", e.Path, fmt.Sprintf("duplicate ADR number %04d", e.ID)})
		}
		seen[e.ID] = true
		if e.Status != "Accepted" && !strings.HasPrefix(e.Status, "Superseded by ") {
			fs = append(fs, Finding{"D6", "warn", e.Path, fmt.Sprintf("invalid status %q", e.Status)})
		}
	}
	return fs
}
```

- [ ] **Step 4: Replace the `cmdDoctor` stub in `cmd/spine/main.go`**

Add imports: `encoding/json`, `github.com/russellpope/spine/internal/doctor`.

```go
func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	findings, err := doctor.Run(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "doctor:", err)
		return 2
	}
	if *asJSON {
		payload := struct {
			Findings []doctor.Finding `json:"findings"`
		}{Findings: findings}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintln(stderr, "doctor:", err)
			return 2
		}
	} else if len(findings) == 0 {
		fmt.Fprintln(stdout, "ok — workflow healthy")
	} else {
		for _, f := range findings {
			fmt.Fprintf(stdout, "%s %-5s %s: %s\n", f.ID, f.Severity, f.Path, f.Message)
		}
	}
	if len(findings) > 0 {
		return 1
	}
	return 0
}
```

Append to `cmd/spine/main_test.go`:

```go
func TestDoctorCleanAndJSON(t *testing.T) {
	dir := t.TempDir()
	runCmd(t, "init", "--dir", dir, "--profile", "rust", "--name", "demo")
	code, out, _ := runCmd(t, "doctor", "--dir", dir)
	if code != 0 || !strings.Contains(out, "ok") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "doctor", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"findings":[]`) {
		t.Fatalf("json code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "doctor", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(out, "D1") {
		t.Fatalf("empty-dir code=%d out=%q", code, out)
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS across all packages.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: spine doctor with D1-D6 checks and --json"
```

---

### Task 9: hbmview fixtures — the un-stranding proof

**Files:**
- Create: `internal/update/testdata/hbmview/{WORKFLOW.md,CLAUDE.md,harness-interface.md}` (copies of the real repo's files)
- Test: `internal/update/hbmview_test.go`

**Interfaces:**
- Consumes: `update.Run`; `scaffold` not needed — fixture repo is assembled by hand to mirror hbmview's layout.

- [ ] **Step 1: Copy the real drifted files as fixtures**

```bash
cd /Users/ldh/Projects/github.com/spine
mkdir -p internal/update/testdata/hbmview
HB=/Users/ldh/Projects/github.com/hbmview
cp "$HB/WORKFLOW.md" "$HB/CLAUDE.md" internal/update/testdata/hbmview/
cp "$HB/docs/harness-interface.md" internal/update/testdata/hbmview/harness-interface.md
```

- [ ] **Step 2: Write the failing test**

`internal/update/hbmview_test.go`:

```go
package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end against copies of hbmview's REAL stranded files (gen0, pristine).
func TestHbmviewUnstranding(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFixture := func(src, dst string) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join("testdata", "hbmview", src))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyFixture("WORKFLOW.md", "WORKFLOW.md")
	copyFixture("CLAUDE.md", "CLAUDE.md")
	copyFixture("harness-interface.md", filepath.Join("docs", "harness-interface.md"))

	// dry run: everything claimable, nothing skipped
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State == SkippedUnrecognized {
			t.Fatalf("%s skipped: %v", r.Path, r.Unrecognized)
		}
	}

	// write, then verify the outcome
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	wf, _ := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	for _, want := range []string{"# Workflow — hbmview", "profile: rust", "template_version: 1",
		"primary: claude-fable-5", "model_default: claude-fable-5",
		"reviewers: [rust-reviewer, security-review]", "functional_harness: cli",
		"Execution mode per plan"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("WORKFLOW.md missing %q", want)
		}
	}
	if strings.Contains(string(wf), "model_default: claude-opus-4-8") {
		t.Error("stale model_default survived")
	}
	cl, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.HasPrefix(string(cl), "<!-- spine:begin v1 -->") ||
		strings.Count(string(cl), "# hbmview") != 1 {
		t.Errorf("CLAUDE.md claim wrong:\n%s", cl)
	}
	hi, _ := os.ReadFile(filepath.Join(dir, "docs", "harness-interface.md"))
	if !strings.Contains(string(hi), "fresh-context") {
		t.Error("harness-interface.md not upgraded to current generation")
	}

	// idempotence: second run is all up-to-date
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("second pass %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}
```

- [ ] **Step 3: Run test**

Run: `go test ./internal/update/ -run TestHbmview -v`
Expected: PASS if Tasks 4–5 are correct. If it FAILS, the fixture caught a divergence between the synthetic tests and reality — fix `update`, not the test. (Note: fixture `WORKFLOW.md`/`CLAUDE.md` should byte-match the `gen0Hbmview` constants; a mismatch means the constants were transcribed wrong — trust the fixtures.)

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "test: hbmview real-file fixtures prove the un-stranding end to end"
```

---

### Task 10: Acceptance — install, dogfood, live hbmview upgrade

**Execution mode: INLINE with the human** (live-system mutation: PATH change, writes to the hbmview repo). Do not delegate this task to a subagent.

**Files:**
- Create (via the tool itself): `CLAUDE.md`, `WORKFLOW.md`, `docs/harness-interface.md`, `docs/issues/*`, `docs/adr/README.md` + 4 ADRs in this repo; upgraded files in `/Users/ldh/Projects/github.com/hbmview`

- [ ] **Step 1: Full test suite + install**

```bash
cd /Users/ldh/Projects/github.com/spine
make test && make install
fish -c 'fish_add_path -U ~/bin'   # ~/bin did not exist before; persists for fish
~/bin/spine version
```
Expected: tests green; `spine template generation 1`.

- [ ] **Step 2: Self-scaffold (dogfood)**

```bash
cd /Users/ldh/Projects/github.com/spine
~/bin/spine init --profile library-cli --name spine
~/bin/spine doctor
```
Expected: `create:` lines for all 6 files (docs/specs already exists with the spec+plan — dirs are fine); doctor exits 0 `ok`. Review the generated `CLAUDE.md`/`WORKFLOW.md`, then append any spine-specific notes BELOW the CLAUDE.md marker block.

- [ ] **Step 3: Record the design decisions as ADRs**

```bash
~/bin/spine adr new "Go with stdlib only; cobra reconsidered if v2 nests commands"
~/bin/spine adr new "Ownership split with config-preserving regeneration and choice-vs-default"
~/bin/spine adr new "docs/specs absorbs plans as the fleet convention"
~/bin/spine adr new "Templates compile into the binary; single integer template generation"
~/bin/spine adr list
```
Fill in Context/Decision/Consequences in each from the spec, then commit:

```bash
git add -A && git commit -m "chore: self-scaffold + ADRs 0001-0004 via spine itself"
```

- [ ] **Step 4: Live hbmview un-stranding (the acceptance bar)**

```bash
cd /Users/ldh/Projects/github.com/hbmview
git status --short   # confirm clean before touching it
~/bin/spine update --dir .          # DRY RUN — human reviews every diff
```
Human gate: review the printed diffs together. Then:

```bash
~/bin/spine update --dir . --write
~/bin/spine doctor --dir .
git diff                            # human reviews the real diff
```
Expected: doctor exits 0 (or lists only D5 info if hbmview has legacy superpowers artifacts); WORKFLOW.md now carries `template_version: 1` + model routing; CLAUDE.md has markers with hbmview's hand-written content (if any) intact. Commit in hbmview only after the human approves the diff:

```bash
git add -A && git commit -m "chore: spine update — adopt template generation 1 (model routing, verify gate, markers)"
```

- [ ] **Step 5: Verification before completion**

Confirm all four acceptance criteria from the spec explicitly, with command output pasted into the session: (1) self-scaffold done, (2) ADRs 0001–0004 exist via `spine adr list`, (3) hbmview doctor clean + diff approved, (4) `make test` green + `~/bin/spine version` works.

---

### Task 11: deepthought handover — workflow-init becomes a shim

**Execution mode: INLINE with the human** (edits another repo the human also uses live).

**Files:**
- Modify: `/Users/ldh/Projects/github.com/deepthought/skills/workflow-init/SKILL.md`, `.../INSTALL.md`
- Delete: `.../scaffold.sh`, `.../templates/` (5 files), `.../tests/` (superseded by spine's Go tests)

- [ ] **Step 1: Rewrite SKILL.md as a shim**

Replace the body of `skills/workflow-init/SKILL.md` with (keep the front-matter name/description unchanged — it is the model-invocation trigger):

```markdown
---
name: workflow-init
description: Scaffold a project for the unified workflow — lean CLAUDE.md, docs/{specs,adr,issues,handoffs}, WORKFLOW.md, and the issue-ledger convention. Use when starting a new repo or adopting the workflow in an existing one.
---

# workflow-init

Thin shim over the `spine` binary (canonical repo: ~/Projects/github.com/spine; install with
`make install` there → `~/bin/spine`).

1. Run `spine init --dir <repo-root>`; it detects the profile (go-service | py-tool | rust |
   library-cli | presentation | ui). If detection fails it exits 2 — ask the user one question
   and re-run with `--profile <p>`.
2. Report created/skipped files. For previously scaffolded repos, `spine update` (dry-run diff,
   then `--write`) upgrades to the current template generation; `spine doctor` checks health.
3. Recommend the user run the spec front-end themselves: `/grill-with-docs`, then `/to-prd`
   (interactive, user-driven skills — do NOT invoke them via the Skill tool).
4. Record decisions with `spine adr new "Title" [--supersedes NNNN]`.

Mandatory gates for every project: a PRD up front, and verification before completion.
If `spine` is not on PATH: `cd ~/Projects/github.com/spine && make install`.
```

- [ ] **Step 2: Remove the absorbed implementation**

```bash
cd /Users/ldh/Projects/github.com/deepthought
git rm -r skills/workflow-init/scaffold.sh skills/workflow-init/templates skills/workflow-init/tests
```
Update `skills/workflow-init/INSTALL.md`: replace scaffold.sh references with the spine repo + `make install` instructions (keep the symlink documentation — the skill still installs via the existing `~/.claude/skills/workflow-init` symlink, which now serves the shim).

- [ ] **Step 3: Verify the shim end-to-end**

```bash
ls -l ~/.claude/skills/workflow-init   # symlink still -> deepthought skills/workflow-init
~/bin/spine init --dir "$(mktemp -d)" --profile rust --name smoke && echo SHIM-OK
```
Expected: symlink intact; smoke init succeeds from a location with no templates on disk (proves embedding).

- [ ] **Step 4: Commit (deepthought)**

```bash
git add -A && git commit -m "refactor: workflow-init absorbed by spine — SKILL.md becomes a shim over ~/bin/spine

Templates + scaffold.sh + tests now live in ~/Projects/github.com/spine
(embedded in the binary). See spine's docs/specs/2026-07-01-spine-cli-design.md.

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Plan Self-Review Record

- **Spec coverage:** decisions 1–7 → Tasks 11, 1, 3–8, 10 (make install), 10.4 (hbmview), 1.3 (specs line), 4–5 (ownership+choice rule). Non-goals respected: no network (only git subprocess for a warning), no deletions by spine (Task 11's `git rm` is the human-gated handover, not the tool). D1–D6 all implemented (Task 8). Acceptance 1–4 → Task 10.
- **Placeholder scan:** clean — every code step ships complete code. Review round 2 fixed two test bugs: gen0 regression in Tasks 6/8 now renders true gen0 templates instead of stamp-stripping (which would classify as unrecognized edits → Skipped, not Pending).
- **Type consistency:** `run(args, stdout, stderr) int` used by all cmd tests; `update.Run(Options) ([]FileReport, error)` consumed by doctor Task 8 exactly as produced in Task 5; `scaffold.Files` shared name checked; `tmpl.Values` fields consistent across Tasks 1/2/5.
