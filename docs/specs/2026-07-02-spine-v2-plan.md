# spine v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship spine generation 2: `adopt` (retrofit pre-spine repos), `handoff new|list|latest [--fleet]`, `eval new|add-run|list` + the `docs/evals/` convention, and three new profiles (swift, knowledge, infra) — per the approved spec `docs/specs/2026-07-02-spine-v2-design.md`.

**Architecture:** Everything composes v1 machinery: adopt is `update`'s planners plus profile detection and dir creation; eval/handoff are new packages using a shared front-matter helper (`internal/meta`) extracted from `adr`; profiles stay centralized in `internal/tmpl` and gain a per-profile file/dir manifest consulted by scaffold, update, and doctor. Templates gain handoff/eval skeletons; `templates/VERSION` bumps 1→2 with **no content edits to any existing template** (so gen-1 repos diff only on the stamp and marker line).

**Tech Stack:** Go 1.26, stdlib only (ADR 0001). `go:embed` templates. Table-driven unit tests + integration tests through `run()` in `cmd/spine/main_test.go`.

## Global Constraints

- **Stdlib only.** No third-party imports anywhere (ADR 0001).
- **Exit contract, uniform:** 0 = clean/current/created, 1 = findings-or-pending, 2 = hard error. Errors → stderr; data/diffs/JSON → stdout.
- **No network. No deletion. No config file.** All writes via `fsutil.WriteFileAtomic`. `new`/`add-run` never overwrite.
- **No content edits to existing templates** (`templates/current/*`, `templates/gen0/*`). v2 only ADDS template files and bumps `templates/VERSION` to `2`. (Rationale: only gen0 is archived — ADR 0004 — so editing current-template content would corrupt choice-vs-default for gen-1 stamps.)
- **Repo:** `~/Projects/github.com/spine`. Commits local only — never push. Commit after every task.
- **TDD:** red → green per task. `go test ./...` green before every commit.
- The Bash tool runs bash but the login shell is fish — quote globs, no `cd` in compound commands.
- Stage/score values in eval run records are **opaque strings** — no Go code may branch on their contents (ADR 0007, to be written in Task 13).

---

## File Structure (end state)

```
cmd/spine/main.go                 # + cmdAdopt, cmdHandoff, cmdEval; adr list --json; usage text
internal/meta/meta.go             # NEW: front-matter Bounds/Parse + Slugify (moved from adr)
internal/tmpl/tmpl.go             # + swift/knowledge/infra profiles; ProfileDirs, ProfileOwns
internal/scaffold/scaffold.go     # detection: swift/infra/knowledge; Init uses ProfileDirs/Owns
internal/update/update.go         # per-profile simpleFiles filter; evals README; adopt mode
internal/adopt/adopt.go           # NEW: orchestration, plan, infos
internal/handoff/handoff.go       # NEW: New/List/Latest/Fleet/ParseName
internal/eval/eval.go             # NEW: New/AddRun/List
internal/doctor/doctor.go         # D1 profile-aware; D7 evals; D8 handoff naming
templates/current/handoff.tmpl.md # NEW
templates/current/evals-README.md # NEW
templates/current/eval.tmpl.md    # NEW
templates/current/run.tmpl.md     # NEW
templates/VERSION                 # 1 → 2
internal/adopt/testdata/          # NEW: praxis/ home-lab-admin/ obsidian-ep-vault/ moo-clone/
internal/update/testdata/ccq/     # NEW: gen-1 real files for the 1→2 stamp-only fixture
```

---

### Task 1: Generation 2 foundations

**Files:**
- Modify: `templates/VERSION` (content `1` → `2`)
- Modify: `cmd/spine/main_test.go:43-48` (TestVersionCommand)

**Interfaces:**
- Produces: `tmpl.Version() == 2` for every subsequent task. All golden content in later tasks bakes `template_version: 2` and marker `<!-- spine:begin v2 -->`.

- [ ] **Step 1: Tighten the version test so it fails on the bump being missing**

Replace TestVersionCommand in `cmd/spine/main_test.go`:

```go
func TestVersionCommand(t *testing.T) {
	code, out, _ := runCmd(t, "version")
	if code != 0 || !strings.Contains(out, "spine template generation 2") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd ~/Projects/github.com/spine && go test ./cmd/... -run TestVersionCommand -v`
Expected: FAIL (`out="spine template generation 1\n"`)

- [ ] **Step 3: Bump the generation**

`templates/VERSION` becomes exactly:

```
2
```

- [ ] **Step 4: Full suite — fix any other hardcoded generation-1 assertions**

Run: `go test ./...`
Expected: PASS. If anything else asserts `template_version: 1` or `v1` markers (grep `go test` failures; `internal/update/update_test.go` uses `Version: 1` in gen0-regression values — that is *deliberate gen0 rendering input*, leave it; only assertions about CURRENT output may need `2`). Fix only genuine current-generation assertions.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: bump template generation to 2"
```

---

### Task 2: Extract `internal/meta` (front matter + slugify)

**Files:**
- Create: `internal/meta/meta.go`, `internal/meta/meta_test.go`
- Modify: `internal/adr/adr.go` (delete `frontMatterBounds`, `parseFrontMatter` body logic, `slugify`; call meta)

**Interfaces:**
- Produces (consumed by adr now; handoff/eval/doctor later):
  - `meta.Bounds(lines []string) (start, end int)` — first `---` fence pair, `-1,-1` if none.
  - `meta.Parse(content string) (kv map[string]string, has bool)` — all `key: value` pairs inside the fence block; first occurrence of a key wins; keys present with empty values map to `""`. `has=false` and nil map when no block.
  - `meta.Slugify(s string) string` — exact behavior of adr's current `slugify` (ASCII letters/digits kept, everything else collapses to single `-`, trimmed; may return `""`).
- **No behavior change to adr** — its existing tests are the lock.

- [ ] **Step 1: Write the failing tests**

`internal/meta/meta_test.go`:

```go
package meta

import "testing"

func TestParse(t *testing.T) {
	content := "---\ntitle: My Title\nstatus: Accepted\nempty:\n---\n\nstatus: Body Decoy\n"
	kv, has := Parse(content)
	if !has {
		t.Fatal("want has=true")
	}
	if kv["title"] != "My Title" || kv["status"] != "Accepted" {
		t.Fatalf("kv=%v", kv)
	}
	if v, ok := kv["empty"]; !ok || v != "" {
		t.Fatalf("empty key: ok=%v v=%q", ok, v)
	}
}

func TestParseFirstOccurrenceWins(t *testing.T) {
	kv, _ := Parse("---\ntitle: First\ntitle: Second\n---\n")
	if kv["title"] != "First" {
		t.Fatalf("kv=%v", kv)
	}
}

func TestParseNoBlock(t *testing.T) {
	if kv, has := Parse("# Just a doc\n"); has || kv != nil {
		t.Fatalf("want nil,false; got %v,%v", kv, has)
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Go with stdlib only": "go-with-stdlib-only",
		"qwen 3.6 (27b)!":     "qwen-3-6-27b",
		"日本語":                 "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q)=%q want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/meta/ -v` — Expected: FAIL (package does not exist)

- [ ] **Step 3: Implement `internal/meta/meta.go`**

```go
// Package meta holds the shared artifact-file helpers: front-matter parsing
// (the first "---" ... "---" fence block) and title slugification. adr,
// handoff, eval, and doctor all consume it.
package meta

import "strings"

// Bounds returns the line indices of the first "---" ... "---" block: start
// is the opening fence, end the closing fence. -1, -1 if no block exists.
func Bounds(lines []string) (start, end int) {
	start, end = -1, -1
	for i, line := range lines {
		if line != "---" {
			continue
		}
		if start == -1 {
			start = i
			continue
		}
		end = i
		break
	}
	return start, end
}

// Parse returns every "key: value" (and bare "key:") pair inside the front-
// matter block, first occurrence winning. has is false when no block exists;
// body content outside the block can never contribute keys.
func Parse(content string) (kv map[string]string, has bool) {
	lines := strings.Split(content, "\n")
	start, end := Bounds(lines)
	if start == -1 || end == -1 {
		return nil, false
	}
	kv = map[string]string{}
	for _, line := range lines[start+1 : end] {
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) == "" || strings.ContainsAny(k, " \t") {
			continue
		}
		if _, seen := kv[k]; !seen {
			kv[k] = strings.TrimSpace(v)
		}
	}
	return kv, true
}

// Slugify lowercases s and keeps only ASCII letters/digits, collapsing every
// other rune into a single '-' separator (trimmed at both ends). Lossy for
// non-ASCII by design; may return "" — callers must reject that.
func Slugify(s string) string {
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
```

- [ ] **Step 4: Refactor adr onto meta — no behavior change**

In `internal/adr/adr.go`:
- Add import `"github.com/russellpope/spine/internal/meta"`.
- Delete `frontMatterBounds` and `slugify` entirely; delete the body of `parseFrontMatter` and reimplement as:

```go
func parseFrontMatter(content string) (title, status string, hasFrontMatter bool) {
	kv, has := meta.Parse(content)
	if !has {
		return "", "", false
	}
	return kv["title"], kv["status"], true
}
```

- In `New`, replace `slug := slugify(title)` with `slug := meta.Slugify(title)`.
- In `flippedContent`, replace `frontMatterBounds(lines)` with `meta.Bounds(lines)`.

Note one deliberate nuance: old `parseFrontMatter` required `"title: "` with a trailing space; `meta.Parse` also accepts `"title:"` bare (empty value). For adr this widens acceptance of an empty-value line from "missing" to "empty string" — both render as empty Title/Status in `list` and both fail the D6 status validation identically. Acceptable; the adr test suite is the referee.

- [ ] **Step 5: Run the full suite — adr tests are the behavior lock**

Run: `go test ./...`
Expected: PASS, zero adr test edits needed.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor: extract internal/meta (front matter + slugify) from adr"
```

---

### Task 3: New profiles — swift, knowledge, infra (defaults, manifest, detection)

**Files:**
- Modify: `internal/tmpl/tmpl.go` (profiles map + ProfileDirs/ProfileOwns)
- Modify: `internal/scaffold/scaffold.go` (DetectProfile + Init uses manifest)
- Test: `internal/tmpl/tmpl_test.go`, `internal/scaffold/scaffold_test.go`

**Interfaces:**
- Produces:
  - `tmpl.Defaults("swift") → ("swift-reviewer, security-review", "framebuffer")`; `("infra") → ("security-review", "none")`; `("knowledge") → ("", "none")`.
  - `tmpl.ProfileDirs(profile string) []string` — knowledge: `["docs/adr","docs/handoffs"]`; every other profile: `["docs/specs","docs/adr","docs/issues","docs/handoffs"]`.
  - `tmpl.ProfileOwns(profile, relPath string) bool` — false only for knowledge × {`docs/harness-interface.md`, `docs/issues/README.md`, `docs/issues/_template.md`}; true otherwise.
  - `scaffold.DetectProfile` additionally detects: `Package.swift` or `*.xcodeproj` → swift; `ansible/ansible.cfg` | `ansible/playbooks` | `helm/` | `terraform/` | `k8s/` → infra; `.obsidian/` | ≥80% git-tracked `.md` → knowledge. Precedence: existing code signals → swift → presentation/ui (unchanged) → infra → knowledge.
- Consumes: nothing new.

- [ ] **Step 1: Failing tests — tmpl**

Append to `internal/tmpl/tmpl_test.go`:

```go
func TestNewProfileDefaults(t *testing.T) {
	cases := map[string][2]string{
		"swift":     {"swift-reviewer, security-review", "framebuffer"},
		"infra":     {"security-review", "none"},
		"knowledge": {"", "none"},
	}
	for p, want := range cases {
		rev, harness, err := Defaults(p)
		if err != nil || rev != want[0] || harness != want[1] {
			t.Errorf("Defaults(%q) = %q,%q,%v", p, rev, harness, err)
		}
	}
}

func TestProfileManifest(t *testing.T) {
	if d := ProfileDirs("knowledge"); len(d) != 2 || d[0] != "docs/adr" || d[1] != "docs/handoffs" {
		t.Fatalf("knowledge dirs=%v", d)
	}
	if d := ProfileDirs("go-service"); len(d) != 4 {
		t.Fatalf("go-service dirs=%v", d)
	}
	for _, rel := range []string{"docs/harness-interface.md", "docs/issues/README.md", "docs/issues/_template.md"} {
		if ProfileOwns("knowledge", rel) {
			t.Errorf("knowledge should not own %s", rel)
		}
		if !ProfileOwns("swift", rel) {
			t.Errorf("swift should own %s", rel)
		}
	}
	if !ProfileOwns("knowledge", "WORKFLOW.md") || !ProfileOwns("knowledge", "CLAUDE.md") || !ProfileOwns("knowledge", "docs/adr/README.md") {
		t.Error("knowledge must own WORKFLOW.md, CLAUDE.md, docs/adr/README.md")
	}
}
```

- [ ] **Step 2: Run to verify failure** — `go test ./internal/tmpl/ -v` → FAIL (unknown profile / undefined funcs)

- [ ] **Step 3: Implement in `internal/tmpl/tmpl.go`**

Add to the `profiles` map:

```go
	"swift":     {"swift-reviewer, security-review", "framebuffer"},
	"infra":     {"security-review", "none"},
	"knowledge": {"", "none"},
```

Append:

```go
// ProfileDirs is the directory set init/adopt create for a profile.
// knowledge repos center on decisions + handoffs; specs/issues are opt-in.
func ProfileDirs(profile string) []string {
	if profile == "knowledge" {
		return []string{"docs/adr", "docs/handoffs"}
	}
	return []string{"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"}
}

// ProfileOwns reports whether a machine-owned file belongs to the profile's
// manifest. knowledge has no build/test harness and no issue ledger.
func ProfileOwns(profile, relPath string) bool {
	if profile != "knowledge" {
		return true
	}
	switch relPath {
	case "docs/harness-interface.md", "docs/issues/README.md", "docs/issues/_template.md":
		return false
	}
	return true
}
```

- [ ] **Step 4: Failing tests — detection + init manifest**

Append to `internal/scaffold/scaffold_test.go`:

```go
func TestDetectNewProfiles(t *testing.T) {
	mk := func(t *testing.T, paths ...string) string {
		dir := t.TempDir()
		for _, p := range paths {
			full := filepath.Join(dir, p)
			if strings.HasSuffix(p, "/") {
				if err := os.MkdirAll(full, 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		return dir
	}
	cases := []struct {
		name, want string
		paths      []string
	}{
		{"package-swift", "swift", []string{"Package.swift"}},
		{"xcodeproj", "swift", []string{"App.xcodeproj/"}},
		{"ansible-cfg", "infra", []string{"ansible/ansible.cfg"}},
		{"ansible-playbooks", "infra", []string{"ansible/playbooks/"}},
		{"helm", "infra", []string{"helm/"}},
		{"terraform", "infra", []string{"terraform/"}},
		{"k8s", "infra", []string{"k8s/"}},
		{"obsidian", "knowledge", []string{".obsidian/"}},
		{"code-beats-infra", "go-service", []string{"go.mod", "ansible/ansible.cfg"}},
		{"infra-beats-knowledge", "infra", []string{"helm/", ".obsidian/"}},
	}
	for _, c := range cases {
		got, ok := DetectProfile(mk(t, c.paths...))
		if !ok || got != c.want {
			t.Errorf("%s: got %q,%v want %q", c.name, got, ok, c.want)
		}
	}
}

func TestDetectKnowledgeByMdMajority(t *testing.T) {
	dir := t.TempDir()
	files := []string{"a.md", "b.md", "c.md", "d.md", "notes/e.md", "x.txt"}
	for _, f := range files {
		full := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	got, ok := DetectProfile(dir)
	if !ok || got != "knowledge" {
		t.Fatalf("got %q,%v want knowledge", got, ok)
	}
}

func TestInitKnowledgeManifest(t *testing.T) {
	dir := t.TempDir()
	res, err := Init(dir, "knowledge", "vault")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Created {
		if f == "docs/harness-interface.md" || f == "docs/issues/README.md" {
			t.Errorf("knowledge must not create %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "specs")); !os.IsNotExist(err) {
		t.Error("knowledge must not create docs/specs")
	}
	for _, rel := range []string{"WORKFLOW.md", "CLAUDE.md", "docs/adr/README.md", "docs/adr", "docs/handoffs"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}
```

Add imports `"os/exec"`, `"strings"` to the test file as needed.

- [ ] **Step 5: Run to verify failure** — `go test ./internal/scaffold/ -v` → FAIL

- [ ] **Step 6: Implement detection + manifest wiring in `internal/scaffold/scaffold.go`**

Replace `DetectProfile` with:

```go
// DetectProfile inspects dir and returns a profile when signals are
// unambiguous. Precedence: code signals, then infra, then knowledge — a repo
// with go.mod AND ansible/ is a go service that carries some automation.
func DetectProfile(dir string) (string, bool) {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	hasDir := func(name string) bool {
		fi, err := os.Stat(filepath.Join(dir, name))
		return err == nil && fi.IsDir()
	}
	switch {
	case has("Cargo.toml"):
		return "rust", true
	case has("go.mod"):
		return "go-service", true
	case has("pyproject.toml"), has("setup.py"):
		return "py-tool", true
	case has("Package.swift"):
		return "swift", true
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
	if m, _ := filepath.Glob(filepath.Join(dir, "*.xcodeproj")); len(m) > 0 {
		return "swift", true
	}
	// infra signals live one level below root (the home-lab-admin lesson).
	if has(filepath.Join("ansible", "ansible.cfg")) || hasDir(filepath.Join("ansible", "playbooks")) ||
		hasDir("helm") || hasDir("terraform") || hasDir("k8s") {
		return "infra", true
	}
	if hasDir(".obsidian") || mdMajority(dir) {
		return "knowledge", true
	}
	return "", false
}

// mdMajority reports whether ≥80% of git-tracked files are .md. Repos
// without git (or with nothing tracked) never qualify — .obsidian is then
// the only knowledge signal.
func mdMajority(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "ls-files").Output()
	if err != nil {
		return false
	}
	var md, total int
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f == "" {
			continue
		}
		total++
		if strings.HasSuffix(strings.ToLower(f), ".md") {
			md++
		}
	}
	return total > 0 && md*100 >= total*80
}
```

Add import `"os/exec"`.

In `Init`, replace the hardcoded dirs loop and the Files loop's render/skip body:

```go
	for _, d := range tmpl.ProfileDirs(profile) {
		target := filepath.Join(dir, d)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return Result{}, fmt.Errorf("mkdir %s: %w", target, err)
		}
	}
```

and inside the `for _, f := range Files` loop, first line:

```go
		if !tmpl.ProfileOwns(profile, f.RelPath) {
			continue
		}
```

- [ ] **Step 7: Run full suite** — `go test ./...` → PASS

- [ ] **Step 8: Commit**

```bash
git add -A && git commit -m "feat: swift/knowledge/infra profiles with per-profile manifest and detection"
```

---

### Task 4: update learns the profile manifest + opt-in evals README

**Files:**
- Modify: `internal/update/update.go` (`Run`: filter simpleFiles by profile; conditional evals README)
- Create: `templates/current/evals-README.md`
- Test: `internal/update/update_test.go`

**Interfaces:**
- Consumes: `tmpl.ProfileOwns` (Task 3).
- Produces: `update.Run` on a knowledge repo plans only WORKFLOW.md, CLAUDE.md, docs/adr/README.md; on any repo where `docs/evals/` exists it also manages `docs/evals/README.md` (template `evals-README.md`, no gen0).

- [ ] **Step 1: Failing tests**

Append to `internal/update/update_test.go`:

```go
func TestUpdateKnowledgeManifest(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "knowledge", "vault"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "docs/harness-interface.md" || r.Path == "docs/issues/README.md" || r.Path == "docs/issues/_template.md" {
			t.Errorf("knowledge update must not manage %s", r.Path)
		}
		if r.State != UpToDate {
			t.Errorf("%s not up-to-date after fresh init", r.Path)
		}
	}
}

func TestUpdateManagesEvalsReadmeOnlyWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "docs/evals/README.md" {
			t.Fatal("evals README managed without docs/evals/")
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "evals"), 0o755); err != nil {
		t.Fatal(err)
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range reports {
		if r.Path == "docs/evals/README.md" {
			found = true
			if r.State != Pending || !r.Created {
				t.Errorf("want Pending+Created, got state=%v created=%v", r.State, r.Created)
			}
		}
	}
	if !found {
		t.Fatal("evals README not planned despite docs/evals/ existing")
	}
}
```

Add imports to the test file: `"github.com/russellpope/spine/internal/scaffold"`, `"os"`, `"path/filepath"` (as missing).

- [ ] **Step 2: Run to verify failure** — `go test ./internal/update/ -run 'TestUpdateKnowledge|TestUpdateManagesEvals' -v` → FAIL

- [ ] **Step 3: Create `templates/current/evals-README.md`** (no placeholders — static convention doc):

```markdown
# Evals

Machine-checkable convention (owned by `spine`; see `spine eval --help`):

- One directory per eval: `YYYY-MM-DD-<slug>/`, created by `spine eval new "<title>"`.
- `eval.md` — front matter `title`, `created`, `prompt` (path), `rubric` (path); prose body free.
- `runs/<name>.md` — one record per run, created by `spine eval add-run --eval E --name N`.
  Front matter: `name`, `created`, `model`, `stage`, `score`.

`stage` and `score` are written by the process driving the eval (the
/model-eval skill) and read back verbatim by `spine eval list` — spine never
interprets them. The canonical loop stages are the run template's body
sections: Wire, Audit, Score, Compare, Remediate, Rescore.

`spine doctor` (D7) validates structure only: parseable front matter with the
required keys present. Values — including empty ones — are yours.
```

- [ ] **Step 4: Implement in `internal/update/update.go`**

In `Run`, after `wf, vals, gen, err := planWorkflow(...)` succeeds, filter the simple files and append the conditional evals README (replace the existing `for _, f := range simpleFiles` loop):

```go
	for _, f := range simpleFiles {
		if !tmpl.ProfileOwns(vals.Profile, f.relPath) {
			continue
		}
		r, err := planSimple(opts.Dir, gen, f.tmplName, f.relPath, f.inGen0, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	// docs/evals/README.md is opt-in machine-owned: managed only where the
	// convention is in use (the directory exists); never created by init/adopt.
	if fi, err := os.Stat(filepath.Join(opts.Dir, "docs", "evals")); err == nil && fi.IsDir() {
		r, err := planSimple(opts.Dir, gen, "evals-README.md", "docs/evals/README.md", false, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
```

- [ ] **Step 5: Run full suite** — `go test ./...` → PASS

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: update consults profile manifest; opt-in docs/evals README"
```

---

### Task 5: update adopt mode (WORKFLOW.md may be absent)

**Files:**
- Modify: `internal/update/update.go` (`Options` + `planWorkflow`)
- Test: `internal/update/update_test.go`

**Interfaces:**
- Produces: `update.Options{AdoptProfile, AdoptName string}` — when `AdoptProfile` is set and `WORKFLOW.md` does not exist, `Run` synthesizes it: report `{Path: "WORKFLOW.md", State: Pending, Created: true}` with current-template content rendered from the profile defaults (project = AdoptName, else basename of dir), `gen = "current"`. All other planners then run normally (CLAUDE.md claim etc.). Without `AdoptProfile`, missing WORKFLOW.md stays the existing hard error.
- Consumed by: Task 9 (`internal/adopt`).

- [ ] **Step 1: Failing test**

```go
func TestAdoptModeSynthesizesWorkflow(t *testing.T) {
	dir := t.TempDir()
	// pre-existing hand-authored CLAUDE.md, praxis-style
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("## Repo invariants\n\n- push with git push github main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, AdoptProfile: "go-service", AdoptName: "praxis"})
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]FileReport{}
	for _, r := range reports {
		byPath[r.Path] = r
	}
	wf := byPath["WORKFLOW.md"]
	if wf.State != Pending || !wf.Created {
		t.Fatalf("WORKFLOW.md state=%v created=%v", wf.State, wf.Created)
	}
	if !strings.Contains(wf.Diff, "profile: go-service") || !strings.Contains(wf.Diff, "template_version: 2") || !strings.Contains(wf.Diff, "# Workflow — praxis") {
		t.Errorf("diff=%q", wf.Diff)
	}
	cl := byPath["CLAUDE.md"]
	if cl.State != Pending || cl.Created {
		t.Fatalf("CLAUDE.md state=%v created=%v (want claim of existing file)", cl.State, cl.Created)
	}
	if !strings.Contains(cl.Diff, "spine:begin") || !strings.Contains(cl.Diff, "Repo invariants") {
		t.Errorf("claim must insert markers and keep hand content; diff=%q", cl.Diff)
	}
}

func TestMissingWorkflowStillErrorsWithoutAdoptMode(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want error")
	}
}
```

- [ ] **Step 2: Run to verify failure** — `go test ./internal/update/ -run TestAdoptMode -v` → FAIL

- [ ] **Step 3: Implement**

`Options` becomes:

```go
// Options configures Run. Zero value = dry-run on ".". AdoptProfile switches
// on adopt mode: a missing WORKFLOW.md is synthesized from that profile's
// defaults (project name = AdoptName, else the dir basename) instead of
// being a hard error. Set only by spine adopt.
type Options struct {
	Dir          string
	Write        bool
	Force        bool
	AdoptProfile string
	AdoptName    string
}
```

`planWorkflow` signature changes from `planWorkflow(dir string)` to `planWorkflow(opts Options)` (update the call in `Run`: `planWorkflow(opts)`); its read block becomes:

```go
	report := FileReport{Path: "WORKFLOW.md"}
	path := filepath.Join(opts.Dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && opts.AdoptProfile != "" {
		project := opts.AdoptName
		if project == "" {
			abs, aerr := filepath.Abs(opts.Dir)
			if aerr != nil {
				return report, tmpl.Values{}, "", aerr
			}
			project = filepath.Base(abs)
		}
		defRev, defHarness, derr := tmpl.Defaults(opts.AdoptProfile)
		if derr != nil {
			return report, tmpl.Values{}, "", derr
		}
		vals := tmpl.Values{Project: project, Profile: opts.AdoptProfile,
			Reviewers: defRev, Harness: defHarness, Version: tmpl.Version()}
		newContent, rerr := tmpl.Render("current", "WORKFLOW.md.tmpl", vals)
		if rerr != nil {
			return report, tmpl.Values{}, "", rerr
		}
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", newContent)
		report.newContent = newContent
		return report, vals, "current", nil
	}
	if err != nil {
		return report, tmpl.Values{}, "", fmt.Errorf("read %s (run spine init first?): %w", path, err)
	}
```

(the rest of the existing function body follows unchanged, with every `dir` reference now `opts.Dir`).

- [ ] **Step 4: Run full suite** — `go test ./...` → PASS

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: update adopt mode — synthesize missing WORKFLOW.md from a profile"
```

---

### Task 6: doctor D1 goes profile-aware

**Files:**
- Modify: `internal/doctor/doctor.go` (required-set computation)
- Test: `internal/doctor/doctor_test.go`

**Interfaces:**
- Consumes: `update.ExtractKeys`, `tmpl.ProfileDirs`, `tmpl.ProfileOwns`.
- Produces: on a repo whose WORKFLOW.md stamps `profile: knowledge`, D1 does not require `docs/specs`, `docs/issues`, `docs/harness-interface.md`. Missing/unreadable WORKFLOW.md or unknown profile → v1 behavior (full required list).

- [ ] **Step 1: Failing test**

Append to `internal/doctor/doctor_test.go`:

```go
func TestD1ProfileAwareKnowledge(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "knowledge", "vault"); err != nil {
		t.Fatal(err)
	}
	findings, err := Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.ID == "D1" {
			t.Errorf("unexpected D1 on fresh knowledge repo: %+v", f)
		}
	}
}
```

Add import `"github.com/russellpope/spine/internal/scaffold"` if missing.

- [ ] **Step 2: Run to verify failure** — `go test ./internal/doctor/ -run TestD1Profile -v` → FAIL (D1 errors for docs/specs etc.)

- [ ] **Step 3: Implement**

In `internal/doctor/doctor.go`, delete the package-level `required` var and compute it inside `Run`:

```go
	required := []string{"WORKFLOW.md", "CLAUDE.md", "docs/harness-interface.md",
		"docs/specs", "docs/adr", "docs/issues", "docs/handoffs"}
	if raw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md")); err == nil {
		if p := update.ExtractKeys(string(raw))["profile"]; p != "" {
			if _, _, err := tmpl.Defaults(p); err == nil {
				required = []string{"WORKFLOW.md", "CLAUDE.md"}
				required = append(required, tmpl.ProfileDirs(p)...)
				if tmpl.ProfileOwns(p, "docs/harness-interface.md") {
					required = append(required, "docs/harness-interface.md")
				}
			}
		}
	}
```

Add import `"github.com/russellpope/spine/internal/tmpl"`.

- [ ] **Step 4: Run full suite** — `go test ./...` → PASS
- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: doctor D1 consults the stamped profile's manifest"
```

---

### Task 7: `spine handoff new | list | latest` (per repo, --json)

**Files:**
- Create: `internal/handoff/handoff.go`, `internal/handoff/handoff_test.go`, `templates/current/handoff.tmpl.md`
- Modify: `cmd/spine/main.go` (dispatch + `cmdHandoff` + usage), `cmd/spine/main_test.go`

**Interfaces:**
- Produces (consumed by Task 8 fleet + Task 10 doctor D8):
  - `handoff.ParseName(filename string) (date time.Time, topic string, ok bool)` — `YYYY-MM-DD-<topic>.md` with a real calendar date.
  - `handoff.Entry{Date time.Time; Topic, Title, Path string}` — `Title` from front matter `title:` if present, else `Topic`.
  - `handoff.New(dir, topic string) (string, error)` — writes `docs/handoffs/<today>-<slug>.md` from `handoff.tmpl.md`; errors: empty slug, file exists, docs/handoffs missing (created via MkdirAll — adopt/init normally provide it).
  - `handoff.List(dir string) ([]Entry, error)` — newest first (date desc, then filename desc); missing `docs/handoffs` → `(nil, nil)`.
  - `handoff.Latest(dir string) (Entry, bool, error)`.
- JSON shapes (marshaled in main.go): entry = `{"path":"docs/handoffs/...","date":"2026-07-02","topic":"spine-v2-brainstorm","title":"..."}`; `list --json` = array of entry; `latest --json` = one entry.

- [ ] **Step 1: Create `templates/current/handoff.tmpl.md`**

```markdown
---
title: {{HANDOFF_TITLE}}
created: {{HANDOFF_DATE}}
---

# Handoff — {{HANDOFF_TITLE}} ({{HANDOFF_DATE}})

## Context

## State (verify before relying)

## Next steps

## Gotchas
```

- [ ] **Step 2: Failing tests — `internal/handoff/handoff_test.go`**

```go
package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseName(t *testing.T) {
	d, topic, ok := ParseName("2026-07-02-spine-v2-brainstorm.md")
	if !ok || topic != "spine-v2-brainstorm" || d.Format("2006-01-02") != "2026-07-02" {
		t.Fatalf("d=%v topic=%q ok=%v", d, topic, ok)
	}
	for _, bad := range []string{"README.md", "2026-13-45-x.md", "2026-07-02-x.txt", "notes.md"} {
		if _, _, ok := ParseName(bad); ok {
			t.Errorf("ParseName(%q) should fail", bad)
		}
	}
}

func TestNewListLatest(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "docs", "handoffs", "2020-01-01-ancient-work.md")
	if err := os.MkdirAll(filepath.Dir(older), 0o755); err != nil {
		t.Fatal(err)
	}
	// legacy handoff: no front matter at all
	if err := os.WriteFile(older, []byte("# some legacy handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := New(dir, "spine v2 spec")
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	if !strings.HasSuffix(path, today+"-spine-v2-spec.md") {
		t.Fatalf("path=%q", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"title: spine v2 spec", "created: " + today, "## Context", "## Gotchas"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %q in %q", want, raw)
		}
	}
	if _, err := New(dir, "spine v2 spec"); err == nil {
		t.Fatal("same-day collision must error")
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Topic != "spine-v2-spec" || entries[1].Topic != "ancient-work" {
		t.Fatalf("order wrong: %v", entries)
	}
	if entries[0].Title != "spine v2 spec" {
		t.Errorf("title from front matter, got %q", entries[0].Title)
	}
	if entries[1].Title != "ancient-work" {
		t.Errorf("legacy title falls back to topic, got %q", entries[1].Title)
	}
	latest, ok, err := Latest(dir)
	if err != nil || !ok || latest.Topic != "spine-v2-spec" {
		t.Fatalf("latest=%v ok=%v err=%v", latest, ok, err)
	}
}

func TestListMissingDirIsEmpty(t *testing.T) {
	entries, err := List(t.TempDir())
	if err != nil || entries != nil {
		t.Fatalf("want nil,nil got %v,%v", entries, err)
	}
}
```

- [ ] **Step 3: Run to verify failure** — `go test ./internal/handoff/ -v` → FAIL (package missing)

- [ ] **Step 4: Implement `internal/handoff/handoff.go`**

```go
// Package handoff manages docs/handoffs/: date-named session handoff notes.
// spine owns the naming and skeleton; the /handoff skill owns the content.
package handoff

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/meta"
	"github.com/russellpope/spine/templates"
)

// Entry is one handoff file. Title comes from front matter when present
// (spine-scaffolded files); legacy handoffs fall back to the filename topic.
type Entry struct {
	Date  time.Time
	Topic string
	Title string
	Path  string
}

var nameRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)\.md$`)

// ParseName validates a handoff filename: date-prefixed, .md, real date.
func ParseName(filename string) (date time.Time, topic string, ok bool) {
	m := nameRe.FindStringSubmatch(filename)
	if m == nil {
		return time.Time{}, "", false
	}
	d, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, "", false
	}
	return d, m[2], true
}

// New scaffolds docs/handoffs/<today>-<slug>.md. It never overwrites.
func New(dir, topic string) (string, error) {
	slug := meta.Slugify(topic)
	if slug == "" {
		return "", fmt.Errorf("topic %q produces an empty slug — use at least one ASCII letter or digit", topic)
	}
	if strings.ContainsAny(topic, "\n\r") {
		return "", fmt.Errorf("topic %q contains a newline, which would inject fake front matter", topic)
	}
	today := time.Now().Format("2006-01-02")
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(hdir, today+"-"+slug+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists — pick a more specific topic", path)
	}
	raw, err := templates.FS.ReadFile("current/handoff.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{HANDOFF_TITLE}}", topic,
		"{{HANDOFF_DATE}}", today,
	).Replace(string(raw))
	if err := fsutil.WriteFileAtomic(path, []byte(content)); err != nil {
		return "", err
	}
	return path, nil
}

// List returns entries newest-first (date desc, filename desc as tiebreak).
// A missing docs/handoffs dir lists as empty, not an error.
func List(dir string) ([]Entry, error) {
	hdir := filepath.Join(dir, "docs", "handoffs")
	des, err := os.ReadDir(hdir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		d, topic, ok := ParseName(de.Name())
		if !ok {
			continue
		}
		e := Entry{Date: d, Topic: topic, Title: topic, Path: filepath.Join(hdir, de.Name())}
		if raw, err := os.ReadFile(e.Path); err == nil {
			if kv, has := meta.Parse(string(raw)); has && kv["title"] != "" {
				e.Title = kv["title"]
			}
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].Path > out[j].Path
	})
	return out, nil
}

// Latest returns the newest entry; ok is false when there are none.
func Latest(dir string) (Entry, bool, error) {
	entries, err := List(dir)
	if err != nil || len(entries) == 0 {
		return Entry{}, false, err
	}
	return entries[0], true, nil
}
```

- [ ] **Step 5: Run package tests** — `go test ./internal/handoff/ -v` → PASS

- [ ] **Step 6: Failing integration test, then wire `cmdHandoff`**

Append to `cmd/spine/main_test.go`:

```go
func TestHandoffEndToEnd(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := runCmd(t, "handoff", "new", "--dir", dir, "spine v2 wrap")
	if code != 0 || !strings.Contains(out, "-spine-v2-wrap.md") {
		t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = runCmd(t, "handoff", "list", "--dir", dir)
	if code != 0 || !strings.Contains(out, "spine-v2-wrap") {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "handoff", "latest", "--dir", dir)
	if code != 0 || !strings.HasSuffix(strings.TrimSpace(out), "-spine-v2-wrap.md") {
		t.Fatalf("latest: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "handoff", "latest", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"topic":"spine-v2-wrap"`) {
		t.Fatalf("latest --json: code=%d out=%q", code, out)
	}
	code, _, _ = runCmd(t, "handoff", "latest", "--dir", t.TempDir())
	if code != 1 {
		t.Fatalf("latest on empty repo: want exit 1, got %d", code)
	}
}
```

Run: `go test ./cmd/... -run TestHandoffEndToEnd -v` → FAIL (unknown command).

Then in `cmd/spine/main.go`: add `case "handoff": return cmdHandoff(args[1:], stdout, stderr)` to the dispatch switch; add to the `usage` const after the adr line: `  handoff  manage docs/handoffs (new, list, latest [--fleet DIR])`; add import `"github.com/russellpope/spine/internal/handoff"` (`encoding/json` is already present; do NOT import `time` yet — nothing in this task uses it and Go rejects unused imports; Task 8 adds it); and:

```go
type handoffJSON struct {
	Path  string `json:"path"`
	Date  string `json:"date"`
	Topic string `json:"topic"`
	Title string `json:"title"`
}

func handoffToJSON(e handoff.Entry) handoffJSON {
	return handoffJSON{Path: e.Path, Date: e.Date.Format("2006-01-02"), Topic: e.Topic, Title: e.Title}
}

func cmdHandoff(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine handoff <new|list|latest> [flags]  (handoff new [--dir D] "Topic")`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("handoff new", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, `usage: spine handoff new [--dir D] "Topic" (flags before topic)`)
			return 2
		}
		path, err := handoff.New(*dir, fs.Arg(0))
		if err != nil {
			fmt.Fprintln(stderr, "handoff new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "list":
		fs := flag.NewFlagSet("handoff list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		entries, err := handoff.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "handoff list:", err)
			return 2
		}
		if *asJSON {
			out := make([]handoffJSON, 0, len(entries))
			for _, e := range entries {
				out = append(out, handoffToJSON(e))
			}
			if err := json.NewEncoder(stdout).Encode(out); err != nil {
				fmt.Fprintln(stderr, "handoff list:", err)
				return 2
			}
			return 0
		}
		for _, e := range entries {
			fmt.Fprintf(stdout, "%s  %s\n", e.Date.Format("2006-01-02"), e.Topic)
		}
		return 0
	case "latest":
		return cmdHandoffLatest(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown handoff subcommand %q\n", args[0])
		return 2
	}
}

func cmdHandoffLatest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("handoff latest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	asJSON := fs.Bool("json", false, "machine-readable output")
	fleet := fs.String("fleet", "", "scan every child repo of DIR instead of one repo")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *fleet != "" {
		return handoffFleet(*fleet, *asJSON, stdout, stderr) // Task 8
	}
	e, ok, err := handoff.Latest(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "handoff latest:", err)
		return 2
	}
	if !ok {
		fmt.Fprintln(stderr, "no handoffs found")
		return 1
	}
	if *asJSON {
		if err := json.NewEncoder(stdout).Encode(handoffToJSON(e)); err != nil {
			fmt.Fprintln(stderr, "handoff latest:", err)
			return 2
		}
		return 0
	}
	fmt.Fprintln(stdout, e.Path)
	return 0
}
```

Until Task 8, stub `handoffFleet`:

```go
func handoffFleet(parent string, asJSON bool, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "handoff latest --fleet: not implemented yet")
	return 2
}
```

- [ ] **Step 7: Run full suite** — `go test ./...` → PASS
- [ ] **Step 8: Commit**

```bash
git add -A && git commit -m "feat: spine handoff new/list/latest with --json"
```

---

### Task 8: `spine handoff latest --fleet DIR`

**Files:**
- Modify: `internal/handoff/handoff.go` (+`Fleet`), `cmd/spine/main.go` (real `handoffFleet`)
- Test: `internal/handoff/handoff_test.go`, `cmd/spine/main_test.go`

**Interfaces:**
- Produces: `handoff.Fleet(parent string) ([]FleetEntry, error)`; `FleetEntry{Repo string; Entry}` — one per immediate child dir of `parent` that has ≥1 valid handoff; sorted date desc, repo asc as tiebreak. Missing/unreadable `parent` → error (exit 2). JSON row: `{"repo":"spine","path":"...","date":"2026-07-02","topic":"...","title":"...","age_days":0}`.

- [ ] **Step 1: Failing test**

Append to `internal/handoff/handoff_test.go`:

```go
func TestFleet(t *testing.T) {
	parent := t.TempDir()
	mk := func(repo, name string) {
		p := filepath.Join(parent, repo, "docs", "handoffs")
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("alpha", "2026-07-01-older.md")
	mk("beta", "2026-07-02-newer.md")
	if err := os.MkdirAll(filepath.Join(parent, "no-handoffs-repo"), 0o755); err != nil {
		t.Fatal(err)
	}
	rows, err := Fleet(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Repo != "beta" || rows[1].Repo != "alpha" {
		t.Fatalf("rows=%v", rows)
	}
	if _, err := Fleet(filepath.Join(parent, "does-not-exist")); err == nil {
		t.Fatal("missing parent must error")
	}
}
```

- [ ] **Step 2: Run to verify failure** — `go test ./internal/handoff/ -run TestFleet -v` → FAIL

- [ ] **Step 3: Implement `Fleet` in `internal/handoff/handoff.go`**

```go
// FleetEntry is one repo's latest handoff in a --fleet scan.
type FleetEntry struct {
	Repo string
	Entry
}

// Fleet scans every immediate child dir of parent for docs/handoffs and
// returns each repo's latest handoff, newest first (repo name as tiebreak).
// Children without handoffs are silently skipped; a missing parent errors.
func Fleet(parent string) ([]FleetEntry, error) {
	des, err := os.ReadDir(parent)
	if err != nil {
		return nil, err
	}
	var out []FleetEntry
	for _, de := range des {
		if !de.IsDir() || strings.HasPrefix(de.Name(), ".") {
			continue
		}
		e, ok, err := Latest(filepath.Join(parent, de.Name()))
		if err != nil || !ok {
			continue
		}
		out = append(out, FleetEntry{Repo: de.Name(), Entry: e})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].Repo < out[j].Repo
	})
	return out, nil
}
```

- [ ] **Step 4: Replace the `handoffFleet` stub in `cmd/spine/main.go`** (add the `"time"` import here — `ageDays` is its first user)

```go
func handoffFleet(parent string, asJSON bool, stdout, stderr io.Writer) int {
	rows, err := handoff.Fleet(parent)
	if err != nil {
		fmt.Fprintln(stderr, "handoff latest --fleet:", err)
		return 2
	}
	if asJSON {
		type row struct {
			Repo string `json:"repo"`
			handoffJSON
			AgeDays int `json:"age_days"`
		}
		out := make([]row, 0, len(rows))
		for _, r := range rows {
			out = append(out, row{Repo: r.Repo, handoffJSON: handoffToJSON(r.Entry), AgeDays: ageDays(r.Date)})
		}
		if err := json.NewEncoder(stdout).Encode(out); err != nil {
			fmt.Fprintln(stderr, "handoff latest --fleet:", err)
			return 2
		}
		return 0
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-24s %4dd  %s\n", r.Repo, ageDays(r.Date), r.Path)
	}
	return 0
}

func ageDays(d time.Time) int {
	age := int(time.Since(d).Hours() / 24)
	if age < 0 {
		return 0
	}
	return age
}
```

And an integration test in `cmd/spine/main_test.go`:

```go
func TestHandoffFleet(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "demo")
	if err := os.MkdirAll(filepath.Join(repo, "docs", "handoffs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "handoffs", "2026-07-01-x.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "handoff", "latest", "--fleet", parent)
	if code != 0 || !strings.Contains(out, "demo") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, _, _ = runCmd(t, "handoff", "latest", "--fleet", filepath.Join(parent, "nope"))
	if code != 2 {
		t.Fatalf("want 2, got %d", code)
	}
}
```

- [ ] **Step 5: Run full suite** — `go test ./...` → PASS
- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: spine handoff latest --fleet"
```

---

### Task 9: `spine eval new | add-run | list`

**Files:**
- Create: `internal/eval/eval.go`, `internal/eval/eval_test.go`, `templates/current/eval.tmpl.md`, `templates/current/run.tmpl.md`
- Modify: `cmd/spine/main.go` (dispatch + `cmdEval` + usage), `cmd/spine/main_test.go`

**Interfaces:**
- Produces (consumed by doctor D7 in Task 10):
  - `eval.Eval{Name, Path string; Runs []Run}`, `eval.Run{Name, Stage, Score, Path string}`, `eval.Problem{Path, Message string}`.
  - `eval.New(dir, title string) (string, error)` — creates `docs/evals/` + README (only if README absent), `docs/evals/<today>-<slug>/eval.md` + `runs/`; existing eval dir → error.
  - `eval.AddRun(dir, evalRef, name string) (string, error)` — evalRef matches a `docs/evals/` child dir exactly or by suffix after the `YYYY-MM-DD-` prefix; zero matches or >1 matches → error listing candidates; run name must not contain `/`, `\`, whitespace, or a leading `.`; existing run file → error.
  - `eval.List(dir string) ([]Eval, []Problem, error)` — missing `docs/evals/` → all nil. Problems: eval dir without parseable `eval.md` front matter or missing keys {title, created, prompt, rubric}; run file without parseable front matter or missing keys {name, created, model, stage, score}. Stage/score VALUES are never inspected.
- Template placeholders (own replacer, adr-style): `{{EVAL_TITLE}}`, `{{EVAL_DATE}}`, `{{RUN_NAME}}`, `{{RUN_DATE}}`.

- [ ] **Step 1: Create the two templates**

`templates/current/eval.tmpl.md`:

```markdown
---
title: {{EVAL_TITLE}}
created: {{EVAL_DATE}}
prompt:
rubric:
---

# Eval — {{EVAL_TITLE}}

Point `prompt:` and `rubric:` at the task prompt and audit rubric (repo-relative
paths). Describe the task, bar, and environment here. Runs live in `runs/`,
one record per model/attempt: `spine eval add-run --eval {{EVAL_DATE}}-… --name <model>`.
```

`templates/current/run.tmpl.md`:

```markdown
---
name: {{RUN_NAME}}
created: {{RUN_DATE}}
model:
stage:
score:
---

# Run — {{RUN_NAME}}

## Wire

## Audit

## Score

## Compare

## Remediate

## Rescore
```

- [ ] **Step 2: Failing tests — `internal/eval/eval_test.go`**

```go
package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAndAddRunAndList(t *testing.T) {
	dir := t.TempDir()
	evalPath, err := New(dir, "govmomi cli")
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	wantDir := filepath.Join(dir, "docs", "evals", today+"-govmomi-cli")
	if evalPath != wantDir {
		t.Fatalf("path=%q want %q", evalPath, wantDir)
	}
	for _, rel := range []string{"eval.md", "runs"} {
		if _, err := os.Stat(filepath.Join(wantDir, rel)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "evals", "README.md")); err != nil {
		t.Fatal("README must be created on first eval new")
	}
	if _, err := New(dir, "govmomi cli"); err == nil {
		t.Fatal("duplicate eval must error")
	}

	runPath, err := AddRun(dir, "govmomi-cli", "qwen-3.6-27b")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: qwen-3.6-27b", "stage:", "score:", "## Rescore"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("missing %q", want)
		}
	}
	if _, err := AddRun(dir, "govmomi-cli", "qwen-3.6-27b"); err == nil {
		t.Fatal("duplicate run must error")
	}
	if _, err := AddRun(dir, "no-such-eval", "m"); err == nil {
		t.Fatal("unknown eval must error")
	}
	if _, err := AddRun(dir, "govmomi-cli", "bad/name"); err == nil {
		t.Fatal("path separator in run name must error")
	}

	evals, problems, err := List(dir)
	if err != nil || len(problems) != 0 {
		t.Fatalf("problems=%v err=%v", problems, err)
	}
	if len(evals) != 1 || len(evals[0].Runs) != 1 || evals[0].Runs[0].Name != "qwen-3.6-27b" {
		t.Fatalf("evals=%+v", evals)
	}
	// stage/score read back verbatim after the driving process edits them
	edited := strings.Replace(string(raw), "stage:", "stage: rescored", 1)
	edited = strings.Replace(edited, "score:", "score: 71/100", 1)
	if err := os.WriteFile(runPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	evals, _, _ = List(dir)
	if evals[0].Runs[0].Stage != "rescored" || evals[0].Runs[0].Score != "71/100" {
		t.Fatalf("runs=%+v", evals[0].Runs)
	}
}

func TestListMissingEvalsDir(t *testing.T) {
	evals, problems, err := List(t.TempDir())
	if evals != nil || problems != nil || err != nil {
		t.Fatalf("want all nil, got %v %v %v", evals, problems, err)
	}
}

func TestListFlagsMalformedRun(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(dir, "demo"); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	bad := filepath.Join(dir, "docs", "evals", today+"-demo", "runs", "broken.md")
	if err := os.WriteFile(bad, []byte("no front matter here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, problems, err := List(dir)
	if err != nil || len(problems) != 1 || !strings.Contains(problems[0].Message, "front matter") {
		t.Fatalf("problems=%v err=%v", problems, err)
	}
}
```

- [ ] **Step 3: Run to verify failure** — `go test ./internal/eval/ -v` → FAIL (package missing)

- [ ] **Step 4: Implement `internal/eval/eval.go`**

```go
// Package eval manages the docs/evals/ convention: spine owns the structure
// (dirs, eval.md, run records); the process driving the eval (/model-eval)
// owns every value. Stage and score are opaque strings here — no code in
// this package may branch on their contents (ADR 0007).
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/russellpope/spine/internal/fsutil"
	"github.com/russellpope/spine/internal/meta"
	"github.com/russellpope/spine/templates"
)

// Run is one parsed run record.
type Run struct {
	Name  string
	Stage string
	Score string
	Path  string
}

// Eval is one eval dir with its runs.
type Eval struct {
	Name string
	Path string
	Runs []Run
}

// Problem is a structural defect List found (doctor surfaces these as D7).
type Problem struct {
	Path    string
	Message string
}

var evalKeys = []string{"title", "created", "prompt", "rubric"}
var runKeys = []string{"name", "created", "model", "stage", "score"}

// New scaffolds docs/evals/<today>-<slug>/{eval.md,runs/}, plus the
// convention README on first use. It never overwrites.
func New(dir, title string) (string, error) {
	slug := meta.Slugify(title)
	if slug == "" {
		return "", fmt.Errorf("title %q produces an empty slug — use at least one ASCII letter or digit", title)
	}
	if strings.ContainsAny(title, "\n\r") {
		return "", fmt.Errorf("title %q contains a newline, which would inject fake front matter", title)
	}
	today := time.Now().Format("2006-01-02")
	root := filepath.Join(dir, "docs", "evals")
	evalDir := filepath.Join(root, today+"-"+slug)
	if _, err := os.Stat(evalDir); err == nil {
		return "", fmt.Errorf("%s already exists", evalDir)
	}
	if err := os.MkdirAll(filepath.Join(evalDir, "runs"), 0o755); err != nil {
		return "", err
	}
	readme := filepath.Join(root, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		raw, rerr := templates.FS.ReadFile("current/evals-README.md")
		if rerr != nil {
			return "", rerr
		}
		if werr := fsutil.WriteFileAtomic(readme, raw); werr != nil {
			return "", werr
		}
	}
	raw, err := templates.FS.ReadFile("current/eval.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{EVAL_TITLE}}", title,
		"{{EVAL_DATE}}", today,
	).Replace(string(raw))
	if err := fsutil.WriteFileAtomic(filepath.Join(evalDir, "eval.md"), []byte(content)); err != nil {
		return "", err
	}
	return evalDir, nil
}

// resolveEval matches ref against docs/evals/ children: exact dir name, or
// the name with its YYYY-MM-DD- prefix stripped. Ambiguity is an error.
func resolveEval(dir, ref string) (string, error) {
	root := filepath.Join(dir, "docs", "evals")
	des, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("no docs/evals/ in %s (run spine eval new first): %w", dir, err)
	}
	var matches []string
	for _, de := range des {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		stripped := name
		if len(name) > 11 && name[4] == '-' && name[7] == '-' && name[10] == '-' {
			stripped = name[11:]
		}
		if name == ref || stripped == ref {
			matches = append(matches, name)
		}
	}
	switch len(matches) {
	case 1:
		return filepath.Join(root, matches[0]), nil
	case 0:
		return "", fmt.Errorf("no eval matches %q under %s", ref, root)
	default:
		return "", fmt.Errorf("eval ref %q is ambiguous: %s", ref, strings.Join(matches, ", "))
	}
}

// AddRun scaffolds runs/<name>.md inside the resolved eval. Never overwrites.
func AddRun(dir, evalRef, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "/\\ \t\n\r") || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("run name %q must be a plain filename fragment (no separators, whitespace, or leading dot)", name)
	}
	evalDir, err := resolveEval(dir, evalRef)
	if err != nil {
		return "", err
	}
	runsDir := filepath.Join(evalDir, "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(runsDir, name+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists", path)
	}
	raw, err := templates.FS.ReadFile("current/run.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{RUN_NAME}}", name,
		"{{RUN_DATE}}", time.Now().Format("2006-01-02"),
	).Replace(string(raw))
	if err := fsutil.WriteFileAtomic(path, []byte(content)); err != nil {
		return "", err
	}
	return path, nil
}

// List parses docs/evals/. A missing tree is empty, not an error. Structural
// defects come back as Problems; values are returned verbatim.
func List(dir string) ([]Eval, []Problem, error) {
	root := filepath.Join(dir, "docs", "evals")
	des, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var evals []Eval
	var problems []Problem
	for _, de := range des {
		if !de.IsDir() {
			continue
		}
		e := Eval{Name: de.Name(), Path: filepath.Join(root, de.Name())}
		problems = append(problems, checkDoc(filepath.Join(e.Path, "eval.md"), evalKeys)...)
		runsDir := filepath.Join(e.Path, "runs")
		rdes, err := os.ReadDir(runsDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
		if os.IsNotExist(err) {
			problems = append(problems, Problem{Path: runsDir, Message: "missing runs/ directory"})
		}
		for _, rde := range rdes {
			if rde.IsDir() || !strings.HasSuffix(rde.Name(), ".md") {
				continue
			}
			rpath := filepath.Join(runsDir, rde.Name())
			raw, err := os.ReadFile(rpath)
			if err != nil {
				return nil, nil, err
			}
			kv, has := meta.Parse(string(raw))
			if probs := checkKeys(rpath, kv, has, runKeys); len(probs) > 0 {
				problems = append(problems, probs...)
				continue
			}
			e.Runs = append(e.Runs, Run{Name: kv["name"], Stage: kv["stage"], Score: kv["score"], Path: rpath})
		}
		sort.Slice(e.Runs, func(i, j int) bool { return e.Runs[i].Name < e.Runs[j].Name })
		evals = append(evals, e)
	}
	sort.Slice(evals, func(i, j int) bool { return evals[i].Name < evals[j].Name })
	return evals, problems, nil
}

func checkDoc(path string, keys []string) []Problem {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []Problem{{Path: path, Message: "missing eval.md"}}
	}
	kv, has := meta.Parse(string(raw))
	return checkKeys(path, kv, has, keys)
}

func checkKeys(path string, kv map[string]string, has bool, keys []string) []Problem {
	if !has {
		return []Problem{{Path: path, Message: "no front matter block"}}
	}
	var missing []string
	for _, k := range keys {
		if _, ok := kv[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return []Problem{{Path: path, Message: "front matter missing key(s): " + strings.Join(missing, ", ")}}
	}
	return nil
}
```

- [ ] **Step 5: Run package tests** — `go test ./internal/eval/ -v` → PASS

- [ ] **Step 6: Wire `cmdEval` into main.go with an integration test**

Test first (append to `cmd/spine/main_test.go`):

```go
func TestEvalEndToEnd(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := runCmd(t, "eval", "new", "--dir", dir, "govmomi cli")
	if code != 0 || !strings.Contains(out, "-govmomi-cli") {
		t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, errs = runCmd(t, "eval", "add-run", "--dir", dir, "--eval", "govmomi-cli", "--name", "qwen-3.6-27b")
	if code != 0 || !strings.Contains(out, "qwen-3.6-27b.md") {
		t.Fatalf("add-run: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = runCmd(t, "eval", "list", "--dir", dir)
	if code != 0 || !strings.Contains(out, "qwen-3.6-27b") {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
	code, out, _ = runCmd(t, "eval", "list", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"name":"qwen-3.6-27b"`) {
		t.Fatalf("list --json: code=%d out=%q", code, out)
	}
	code, _, errs = runCmd(t, "eval", "add-run", "--dir", dir, "--eval", "nope", "--name", "m")
	if code != 2 || !strings.Contains(errs, "no eval matches") {
		t.Fatalf("code=%d errs=%q", code, errs)
	}
}
```

Run to verify FAIL, then add to main.go dispatch: `case "eval": return cmdEval(args[1:], stdout, stderr)`; usage line `  eval     manage docs/evals (new, add-run, list)`; import `"github.com/russellpope/spine/internal/eval"`; and:

```go
func cmdEval(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, `usage: spine eval <new|add-run|list> [flags]  (eval new [--dir D] "Title"; eval add-run --eval E --name N)`)
		return 2
	}
	switch args[0] {
	case "new":
		fs := flag.NewFlagSet("eval new", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, `usage: spine eval new [--dir D] "Title" (flags before title)`)
			return 2
		}
		path, err := eval.New(*dir, fs.Arg(0))
		if err != nil {
			fmt.Fprintln(stderr, "eval new:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "add-run":
		fs := flag.NewFlagSet("eval add-run", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		evalRef := fs.String("eval", "", "eval dir name (date prefix optional)")
		name := fs.String("name", "", "run name (becomes runs/<name>.md)")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *evalRef == "" || *name == "" {
			fmt.Fprintln(stderr, "eval add-run: --eval and --name are required")
			return 2
		}
		path, err := eval.AddRun(*dir, *evalRef, *name)
		if err != nil {
			fmt.Fprintln(stderr, "eval add-run:", err)
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "list":
		fs := flag.NewFlagSet("eval list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dir := fs.String("dir", ".", "repo root")
		asJSON := fs.Bool("json", false, "machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		evals, problems, err := eval.List(*dir)
		if err != nil {
			fmt.Fprintln(stderr, "eval list:", err)
			return 2
		}
		for _, p := range problems {
			fmt.Fprintf(stderr, "warning: %s: %s\n", p.Path, p.Message)
		}
		if *asJSON {
			type runJSON struct {
				Name  string `json:"name"`
				Stage string `json:"stage"`
				Score string `json:"score"`
				Path  string `json:"path"`
			}
			type evalJSON struct {
				Name string    `json:"name"`
				Path string    `json:"path"`
				Runs []runJSON `json:"runs"`
			}
			out := make([]evalJSON, 0, len(evals))
			for _, e := range evals {
				ej := evalJSON{Name: e.Name, Path: e.Path, Runs: []runJSON{}}
				for _, r := range e.Runs {
					ej.Runs = append(ej.Runs, runJSON{Name: r.Name, Stage: r.Stage, Score: r.Score, Path: r.Path})
				}
				out = append(out, ej)
			}
			if err := json.NewEncoder(stdout).Encode(out); err != nil {
				fmt.Fprintln(stderr, "eval list:", err)
				return 2
			}
			return 0
		}
		for _, e := range evals {
			if len(e.Runs) == 0 {
				fmt.Fprintf(stdout, "%-30s  %-20s  %-10s  %s\n", e.Name, "-", "-", "-")
			}
			for _, r := range e.Runs {
				fmt.Fprintf(stdout, "%-30s  %-20s  %-10s  %s\n", e.Name, r.Name, r.Stage, r.Score)
			}
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown eval subcommand %q\n", args[0])
		return 2
	}
}
```

- [ ] **Step 7: Run full suite** — `go test ./...` → PASS
- [ ] **Step 8: Commit**

```bash
git add -A && git commit -m "feat: spine eval new/add-run/list and the docs/evals convention"
```

---

### Task 10: doctor D7 (eval structure) + D8 (handoff naming)

**Files:**
- Modify: `internal/doctor/doctor.go`
- Test: `internal/doctor/doctor_test.go`

**Interfaces:**
- Consumes: `eval.List` Problems (Task 9), `handoff.ParseName` (Task 7).
- Produces: D7 warn per eval Problem (only when `docs/evals/` exists — List already behaves that way); D8 info per non-conforming file in `docs/handoffs/` (subdirectories ignored).

- [ ] **Step 1: Failing tests**

```go
func TestD7EvalStructure(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := eval.New(dir, "demo eval"); err != nil {
		t.Fatal(err)
	}
	// well-formed: no D7
	findings, _ := Run(dir)
	for _, f := range findings {
		if f.ID == "D7" {
			t.Fatalf("unexpected D7: %+v", f)
		}
	}
	// malformed run: D7 warn
	today := time.Now().Format("2006-01-02")
	bad := filepath.Join(dir, "docs", "evals", today+"-demo-eval", "runs", "broken.md")
	if err := os.WriteFile(bad, []byte("no front matter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, _ = Run(dir)
	found := false
	for _, f := range findings {
		if f.ID == "D7" && f.Severity == "warn" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want D7 warn, findings=%+v", findings)
	}
}

func TestD8HandoffNaming(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "handoffs", "notes.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, _ := Run(dir)
	found := false
	for _, f := range findings {
		if f.ID == "D8" {
			found = true
			if f.Severity != "info" {
				t.Errorf("D8 must be info, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("want D8, findings=%+v", findings)
	}
}
```

Add imports as needed: `"github.com/russellpope/spine/internal/eval"`, `"time"`.

- [ ] **Step 2: Run to verify failure** — `go test ./internal/doctor/ -run 'TestD7|TestD8' -v` → FAIL

- [ ] **Step 3: Implement in `internal/doctor/doctor.go`**

Add to `Run` before the final return: `findings = append(findings, evalCheck(dir)...)` and `findings = append(findings, handoffCheck(dir)...)`. Then:

```go
// evalCheck maps eval.List structural problems onto D7. Values (stage,
// score) are never validated — structure only (ADR 0007).
func evalCheck(dir string) []Finding {
	_, problems, err := eval.List(dir)
	if err != nil {
		return []Finding{{"D7", "error", "docs/evals", "evals tree unreadable: " + err.Error()}}
	}
	var findings []Finding
	for _, p := range problems {
		findings = append(findings, Finding{"D7", "warn", p.Path, p.Message})
	}
	return findings
}

// handoffCheck flags files in docs/handoffs that don't follow the
// YYYY-MM-DD-<topic>.md convention. Info only — legacy is legal.
func handoffCheck(dir string) []Finding {
	des, err := os.ReadDir(filepath.Join(dir, "docs", "handoffs"))
	if err != nil {
		return nil // D1 covers structural absence
	}
	var findings []Finding
	for _, de := range des {
		if de.IsDir() {
			continue
		}
		if _, _, ok := handoff.ParseName(de.Name()); !ok {
			findings = append(findings, Finding{"D8", "info", "docs/handoffs/" + de.Name(),
				"does not match YYYY-MM-DD-<topic>.md — spine handoff new produces conforming names"})
		}
	}
	return findings
}
```

Add imports `"github.com/russellpope/spine/internal/eval"`, `"github.com/russellpope/spine/internal/handoff"`.

- [ ] **Step 4: Run full suite** — `go test ./...` → PASS
- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: doctor D7 eval structure + D8 handoff naming"
```

---

### Task 11: `spine adopt`

**Files:**
- Create: `internal/adopt/adopt.go`, `internal/adopt/adopt_test.go`
- Modify: `cmd/spine/main.go` (dispatch + `cmdAdopt` + usage), `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: `scaffold.DetectProfile` (Task 3), `update.Run` with `AdoptProfile` (Task 5), `tmpl.ProfileDirs`, `adr.List`.
- Produces:
  - `adopt.Options{Dir, Profile, Name string; Write, Force bool}`
  - `adopt.Info{Path, Message string}`
  - `adopt.Result{Profile string; DirsToCreate []string; Reports []update.FileReport; Infos []Info}`
  - `adopt.Run(opts) (Result, error)` — detects profile when empty (error if undetectable), computes missing `tmpl.ProfileDirs`, runs `update.Run` in adopt mode, gathers infos. With `Write`: MkdirAll the dirs, then update with Write.
  - `Result.Pending() bool` — true if any dir missing or any report Pending/SkippedUnrecognized.
- CLI contract: dry-run prints plan, exit 1 if `Pending()`, 0 if not (idempotent no-op); `--write` applies, exit 0 on success (2 on error). `--json` emits `{"profile":..., "dirs":[...], "files":[{"path":...,"action":"create|update|up-to-date|skip"}], "infos":[{"path":...,"message":...}]}`.

- [ ] **Step 1: Failing tests — `internal/adopt/adopt_test.go`**

```go
package adopt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/update"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAdoptPraxisShape(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod":                          "module praxis\n",
		"CLAUDE.md":                       "## Repo invariants\n\n- remote is github, not origin\n",
		"docs/adr/0001-legacy.md":         "# 0001: legacy decision\n\nno front matter\n",
		"docs/superpowers/specs/a.md":     "old spec\n",
		"docs/decisions/2026-Q3-nonce.md": "quarterly recheck\n",
	})
	res, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile != "go-service" {
		t.Fatalf("profile=%q", res.Profile)
	}
	if !res.Pending() {
		t.Fatal("fresh adopt must be pending")
	}
	joined := strings.Join(res.DirsToCreate, " ")
	for _, d := range []string{"docs/specs", "docs/issues", "docs/handoffs"} {
		if !strings.Contains(joined, d) {
			t.Errorf("missing dir %s in %q", d, joined)
		}
	}
	if strings.Contains(joined, "docs/adr") {
		t.Error("docs/adr exists; must not be in DirsToCreate")
	}
	var infoText string
	for _, i := range res.Infos {
		infoText += i.Path + ": " + i.Message + "\n"
	}
	for _, want := range []string{"docs/superpowers/specs", "pre-spine", "docs/decisions"} {
		if !strings.Contains(infoText, want) {
			t.Errorf("infos missing %q:\n%s", want, infoText)
		}
	}

	// apply, then idempotency: second adopt is a clean no-op
	res, err = Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "spine:begin") || !strings.Contains(string(raw), "Repo invariants") {
		t.Fatalf("claim failed: %q", raw)
	}
	res, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pending() {
		for _, r := range res.Reports {
			t.Logf("%s state=%v", r.Path, r.State)
		}
		t.Fatal("adopted repo must be a no-op")
	}
	// post-condition: update agrees
	reports, err := update.Run(update.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != update.UpToDate {
			t.Errorf("update not no-op: %s state=%v", r.Path, r.State)
		}
	}
}

func TestAdoptUndetectableErrors(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want detection error")
	}
}
```

- [ ] **Step 2: Run to verify failure** — `go test ./internal/adopt/ -v` → FAIL

- [ ] **Step 3: Implement `internal/adopt/adopt.go`**

```go
// Package adopt retrofits a pre-spine repo: compose init's dir creation and
// update's claim/regenerate machinery under one dry-runnable plan. It maps
// nothing and migrates nothing: legacy trees stay put and are reported as
// info (ADR 0008).
package adopt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/russellpope/spine/internal/adr"
	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
	"github.com/russellpope/spine/internal/update"
)

// Options configures Run. Zero value = dry-run detection on ".".
type Options struct {
	Dir     string
	Profile string
	Name    string
	Write   bool
	Force   bool
}

// Info is a transparency note in the plan (never affects exit codes).
type Info struct {
	Path    string
	Message string
}

// Result is the adopt plan (and with Write, what was applied).
type Result struct {
	Profile      string
	DirsToCreate []string
	Reports      []update.FileReport
	Infos        []Info
}

// Pending reports whether applying would change anything.
func (r Result) Pending() bool {
	if len(r.DirsToCreate) > 0 {
		return true
	}
	for _, rep := range r.Reports {
		if rep.State != update.UpToDate {
			return true
		}
	}
	return false
}

// dirsKnown is every docs/ entry spine has a concept of; anything else in
// docs/ is reported as "not spine's" for plan transparency.
var dirsKnown = map[string]bool{
	"specs": true, "adr": true, "issues": true, "handoffs": true,
	"evals": true, "superpowers": true, "harness-interface.md": true,
}

// Run plans (and with opts.Write, applies) the retrofit.
func Run(opts Options) (Result, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	profile := opts.Profile
	if profile == "" {
		detected, ok := scaffold.DetectProfile(opts.Dir)
		if !ok {
			return Result{}, fmt.Errorf("cannot detect profile for %s; pass --profile", opts.Dir)
		}
		profile = detected
	}
	if _, _, err := tmpl.Defaults(profile); err != nil {
		return Result{}, err
	}
	res := Result{Profile: profile}
	for _, d := range tmpl.ProfileDirs(profile) {
		if fi, err := os.Stat(filepath.Join(opts.Dir, d)); err != nil || !fi.IsDir() {
			res.DirsToCreate = append(res.DirsToCreate, d)
		}
	}
	if opts.Write {
		for _, d := range res.DirsToCreate {
			if err := os.MkdirAll(filepath.Join(opts.Dir, d), 0o755); err != nil {
				return res, err
			}
		}
	}
	reports, err := update.Run(update.Options{
		Dir: opts.Dir, Write: opts.Write, Force: opts.Force,
		AdoptProfile: profile, AdoptName: opts.Name,
	})
	if err != nil {
		return res, err
	}
	res.Reports = reports
	res.Infos = gatherInfos(opts.Dir)
	return res, nil
}

func gatherInfos(dir string) []Info {
	var infos []Info
	for _, sub := range []string{"specs", "plans"} {
		glob := filepath.Join(dir, "docs", "superpowers", sub, "*.md")
		if m, _ := filepath.Glob(glob); len(m) > 0 {
			infos = append(infos, Info{Path: "docs/superpowers/" + sub,
				Message: fmt.Sprintf("%d artifact(s) in legacy location — left alone; new work goes to docs/specs/", len(m))})
		}
	}
	if entries, err := adr.List(dir); err == nil {
		preSpine := 0
		for _, e := range entries {
			if !e.HasFrontMatter {
				preSpine++
			}
		}
		if preSpine > 0 {
			infos = append(infos, Info{Path: "docs/adr",
				Message: fmt.Sprintf("%d pre-spine ADR(s) (no front matter) — left alone; spine conventions apply to new ADRs", preSpine)})
		}
	}
	if des, err := os.ReadDir(filepath.Join(dir, "docs")); err == nil {
		var unknown []string
		for _, de := range des {
			if !dirsKnown[de.Name()] {
				unknown = append(unknown, "docs/"+de.Name())
			}
		}
		sort.Strings(unknown)
		if len(unknown) > 0 {
			infos = append(infos, Info{Path: strings.Join(unknown, ", "),
				Message: "not spine's — left alone"})
		}
	}
	return infos
}
```

- [ ] **Step 4: Run package tests** — `go test ./internal/adopt/ -v` → PASS

- [ ] **Step 5: Wire `cmdAdopt` with an integration test**

Test (append to `cmd/spine/main_test.go`):

```go
func TestAdoptEndToEnd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("## Invariants\n- keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCmd(t, "adopt", "--dir", dir)
	if code != 1 || !strings.Contains(out, "profile: go-service") || !strings.Contains(out, "WORKFLOW.md") {
		t.Fatalf("dry-run: code=%d out=%q", code, out)
	}
	code, out, errs := runCmd(t, "adopt", "--dir", dir, "--write")
	if code != 0 {
		t.Fatalf("write: code=%d out=%q err=%q", code, out, errs)
	}
	code, _, _ = runCmd(t, "adopt", "--dir", dir)
	if code != 0 {
		t.Fatalf("idempotency: want 0, got %d", code)
	}
	code, _, _ = runCmd(t, "doctor", "--dir", dir)
	if code != 0 {
		t.Fatalf("doctor after adopt: want 0, got %d", code)
	}
	code, out, _ = runCmd(t, "adopt", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"profile":"go-service"`) {
		t.Fatalf("json: code=%d out=%q", code, out)
	}
}
```

Run to verify FAIL, then in `cmd/spine/main.go`: dispatch `case "adopt": return cmdAdopt(args[1:], stdout, stderr)`; usage line `  adopt    retrofit a pre-spine repo (dry-run by default; --write applies)`; import `"github.com/russellpope/spine/internal/adopt"`; and:

```go
func cmdAdopt(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repo root")
	profile := fs.String("profile", "", "override profile detection")
	name := fs.String("name", "", "project name (default: basename of dir)")
	write := fs.Bool("write", false, "apply the plan (default: dry-run)")
	force := fs.Bool("force", false, "regenerate files with unrecognized local edits")
	asJSON := fs.Bool("json", false, "machine-readable plan output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *write {
		warnDirty(*dir, stderr)
	}
	res, err := adopt.Run(adopt.Options{Dir: *dir, Profile: *profile, Name: *name, Write: *write, Force: *force})
	if err != nil {
		fmt.Fprintln(stderr, "adopt:", err)
		return 2
	}
	action := func(r update.FileReport) string {
		switch r.State {
		case update.UpToDate:
			return "up-to-date"
		case update.SkippedUnrecognized:
			return "skip"
		default:
			if r.Created {
				return "create"
			}
			return "update"
		}
	}
	if *asJSON {
		type fileJSON struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		}
		type infoJSON struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		}
		payload := struct {
			Profile string     `json:"profile"`
			Dirs    []string   `json:"dirs"`
			Files   []fileJSON `json:"files"`
			Infos   []infoJSON `json:"infos"`
		}{Profile: res.Profile, Dirs: res.DirsToCreate, Files: []fileJSON{}, Infos: []infoJSON{}}
		if payload.Dirs == nil {
			payload.Dirs = []string{}
		}
		for _, r := range res.Reports {
			payload.Files = append(payload.Files, fileJSON{Path: r.Path, Action: action(r)})
		}
		for _, i := range res.Infos {
			payload.Infos = append(payload.Infos, infoJSON{Path: i.Path, Message: i.Message})
		}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintln(stderr, "adopt:", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "profile: %s\n", res.Profile)
		fmt.Fprintln(stdout, "plan:")
		for _, d := range res.DirsToCreate {
			fmt.Fprintf(stdout, "  create dir  %s\n", d)
		}
		for _, r := range res.Reports {
			fmt.Fprintf(stdout, "  %-11s %s\n", action(r), r.Path)
			if r.State == update.SkippedUnrecognized {
				for _, l := range r.Unrecognized {
					fmt.Fprintf(stderr, "    unrecognized: %s\n", l)
				}
			}
		}
		if len(res.Infos) > 0 {
			fmt.Fprintln(stdout, "info:")
			for _, i := range res.Infos {
				fmt.Fprintf(stdout, "  %s: %s\n", i.Path, i.Message)
			}
		}
	}
	if !*write && res.Pending() {
		fmt.Fprintln(stdout, "rerun with --write to apply")
		return 1
	}
	skipped := false
	for _, r := range res.Reports {
		if r.State == update.SkippedUnrecognized {
			skipped = true
		}
	}
	if skipped {
		return 1
	}
	return 0
}
```

- [ ] **Step 6: Run full suite** — `go test ./...` → PASS
- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "feat: spine adopt — compose init+update into one dry-runnable retrofit"
```

---

### Task 12: `adr list --json` + usage text sweep

**Files:**
- Modify: `cmd/spine/main.go` (`cmdADR` list branch; final usage review)
- Test: `cmd/spine/main_test.go`

**Interfaces:**
- Produces: `adr list --json` → array of `{"id":1,"title":"...","status":"Accepted","path":"docs/adr/0001-....md","has_front_matter":true}`.

- [ ] **Step 1: Failing test**

```go
func TestADRListJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := runCmd(t, "adr", "new", "--dir", dir, "Some Decision"); code != 0 {
		t.Fatal(errs)
	}
	code, out, _ := runCmd(t, "adr", "list", "--dir", dir, "--json")
	if code != 0 || !strings.Contains(out, `"title":"Some Decision"`) || !strings.Contains(out, `"has_front_matter":true`) {
		t.Fatalf("code=%d out=%q", code, out)
	}
}
```

- [ ] **Step 2: Run to verify failure**, then implement in `cmdADR`'s `list` branch: add `asJSON := fs.Bool("json", false, "machine-readable output")` and after `adr.List`:

```go
		if *asJSON {
			type entryJSON struct {
				ID             int    `json:"id"`
				Title          string `json:"title"`
				Status         string `json:"status"`
				Path           string `json:"path"`
				HasFrontMatter bool   `json:"has_front_matter"`
			}
			out := make([]entryJSON, 0, len(entries))
			for _, e := range entries {
				out = append(out, entryJSON{e.ID, e.Title, e.Status, e.Path, e.HasFrontMatter})
			}
			if err := json.NewEncoder(stdout).Encode(out); err != nil {
				fmt.Fprintln(stderr, "adr list:", err)
				return 2
			}
			return 0
		}
```

- [ ] **Step 3: Verify the full usage text now reads** (in the `usage` const):

```
usage: spine <command> [flags]

commands:
  init     scaffold the unified workflow into a repo
  adopt    retrofit a pre-spine repo (dry-run by default; --write applies)
  update   regenerate machine-owned workflow files (dry-run by default; --write applies)
  adr      manage architecture decision records (new, list)
  handoff  manage docs/handoffs (new, list, latest [--fleet DIR])
  eval     manage docs/evals (new, add-run, list)
  doctor   read-only workflow health checks
  version  print the compiled template generation
```

- [ ] **Step 4: Run full suite** — `go test ./...` → PASS
- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: adr list --json; usage text covers the v2 surface"
```

---

### Task 13: Real-file fixtures (praxis, home-lab-admin, obsidian-ep-vault, moo-clone, ccq gen-1)

**Files:**
- Create: `internal/adopt/testdata/{praxis,home-lab-admin,obsidian-ep-vault,moo-clone}/…` (copied from the live repos)
- Create: `internal/update/testdata/ccq/{WORKFLOW.md,CLAUDE.md}` (copied from live ccq)
- Test: `internal/adopt/realfixtures_test.go`, `internal/update/gen1to2_test.go`

**Interfaces:** none new — these tests lock the composition against reality (the v1 lesson: an hbmview real-file fixture caught an inverted constant all unit tests missed).

- [ ] **Step 1: Copy real files into fixtures** (bash; fish quoting not an issue inside the Bash tool):

```bash
cd ~/Projects/github.com/spine
mkdir -p internal/adopt/testdata/praxis/docs/superpowers/specs internal/adopt/testdata/praxis/docs/decisions internal/adopt/testdata/praxis/docs/adr
cp ~/Projects/github.com/praxis/CLAUDE.md internal/adopt/testdata/praxis/CLAUDE.md
cp ~/Projects/github.com/praxis/docs/adr/0001-*.md internal/adopt/testdata/praxis/docs/adr/
cp ~/Projects/github.com/praxis/docs/adr/0002-*.md internal/adopt/testdata/praxis/docs/adr/
ls ~/Projects/github.com/praxis/docs/superpowers/specs/ | head -1 | xargs -I{} cp ~/Projects/github.com/praxis/docs/superpowers/specs/{} internal/adopt/testdata/praxis/docs/superpowers/specs/
ls ~/Projects/github.com/praxis/docs/decisions/ | head -1 | xargs -I{} cp ~/Projects/github.com/praxis/docs/decisions/{} internal/adopt/testdata/praxis/docs/decisions/
printf 'module praxis\n\ngo 1.26\n' > internal/adopt/testdata/praxis/go.mod

mkdir -p internal/adopt/testdata/home-lab-admin/ansible internal/adopt/testdata/home-lab-admin/helm
cp ~/Projects/github.com/home-lab-admin/ansible/ansible.cfg internal/adopt/testdata/home-lab-admin/ansible/
cp ~/Projects/github.com/home-lab-admin/helm/repositories.yaml internal/adopt/testdata/home-lab-admin/helm/

mkdir -p internal/adopt/testdata/obsidian-ep-vault/.obsidian
printf '{}\n' > internal/adopt/testdata/obsidian-ep-vault/.obsidian/app.json

mkdir -p internal/adopt/testdata/moo-clone/Moo.xcodeproj
printf '// fixture marker\n' > internal/adopt/testdata/moo-clone/Moo.xcodeproj/project.pbxproj

mkdir -p internal/update/testdata/ccq
cp ~/Projects/github.com/ccq/WORKFLOW.md internal/update/testdata/ccq/
cp ~/Projects/github.com/ccq/CLAUDE.md internal/update/testdata/ccq/
```

**Check the praxis CLAUDE.md copy for anything sensitive before committing** (it is an internal-invariants doc; expected clean, but eyeball it).

- [ ] **Step 2: Write the fixture tests**

`internal/adopt/realfixtures_test.go`:

```go
package adopt

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/update"
)

// copyTree copies testdata/<name> into a temp dir so --write can mutate it.
func copyTree(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", name)
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestRealFixtureAdopts(t *testing.T) {
	cases := []struct {
		fixture, wantProfile string
	}{
		{"praxis", "go-service"},
		{"home-lab-admin", "infra"},
		{"obsidian-ep-vault", "knowledge"},
		{"moo-clone", "swift"},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			dir := copyTree(t, c.fixture)
			res, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if res.Profile != c.wantProfile {
				t.Fatalf("profile=%q want %q", res.Profile, c.wantProfile)
			}
			if !res.Pending() {
				t.Fatal("fresh adopt must be pending")
			}
			if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
				t.Fatal(err)
			}
			// post-condition: doctor clean (info-only) and update a no-op
			findings, err := doctor.Run(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range findings {
				if f.Severity == "warn" || f.Severity == "error" {
					t.Errorf("doctor %s %s %s: %s", f.ID, f.Severity, f.Path, f.Message)
				}
			}
			reports, err := update.Run(update.Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range reports {
				if r.State != update.UpToDate {
					t.Errorf("update not no-op: %s", r.Path)
				}
			}
		})
	}
}

func TestPraxisClaimPreservesInvariants(t *testing.T) {
	dir := copyTree(t, "praxis")
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "spine:begin") {
		t.Error("marker block missing")
	}
	// the load-bearing praxis invariant must survive the claim verbatim
	if !strings.Contains(content, "github") || !strings.Contains(content, "NOT `origin`") {
		t.Error("praxis remote invariant lost in claim")
	}
}
```

`internal/update/gen1to2_test.go`:

```go
package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The ccq fixture is that repo's actual gen-1 WORKFLOW.md and CLAUDE.md.
// Updating 1→2 must be exactly the stamp + marker-version diff — v2 ships
// no content edits to existing templates, so anything else here is a bug.
func TestGen1To2IsStampOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		switch r.Path {
		case "WORKFLOW.md", "CLAUDE.md":
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			for _, line := range strings.Split(r.Diff, "\n") {
				if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
					continue
				}
				if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
					continue
				}
				if strings.Contains(line, "template_version") || strings.Contains(line, "spine:begin") {
					continue
				}
				t.Errorf("%s: unexpected changed line %q — gen 1→2 must be stamp-only", r.Path, line)
			}
		}
	}
}
```

Note: the ccq fixture lacks the simple machine-owned files, so their reports will be Pending+Created — the test only asserts on WORKFLOW.md/CLAUDE.md deliberately.

- [ ] **Step 3: Run** — `go test ./internal/adopt/ ./internal/update/ -v` → PASS. If `TestGen1To2IsStampOnly` fails on extra lines, an existing template was edited somewhere in Tasks 1–12 — revert that edit; it violates the global constraint.

- [ ] **Step 4: Full suite + commit**

```bash
go test ./...
git add -A && git commit -m "test: real-file fixtures — 4 adopt targets + ccq gen1→2 stamp-only lock"
```

---

### Task 14: Dogfood — install, self-update, ADRs 0006–0008

**Files:**
- Modify (via the tool, not by hand): spine's own `WORKFLOW.md`/`CLAUDE.md` (gen 2 stamp), `docs/adr/0006-*.md`, `0007-*.md`, `0008-*.md`

- [ ] **Step 1: Install and verify**

```bash
cd ~/Projects/github.com/spine && make test && make install
~/bin/spine version
```
Expected: `spine template generation 2`

- [ ] **Step 2: Self-update the spine repo to gen 2**

```bash
~/bin/spine update --dir ~/Projects/github.com/spine; echo "exit: $?"
```
Expected: exit 1, diff shows ONLY `template_version: 1` → `2` and marker `v1` → `v2`. Then:

```bash
~/bin/spine update --dir ~/Projects/github.com/spine --write && ~/bin/spine doctor --dir ~/Projects/github.com/spine
```
Expected: doctor `ok — workflow healthy` (or info-only).

- [ ] **Step 3: Record the three ADRs with real content via the tool**

```bash
cd ~/Projects/github.com/spine
~/bin/spine adr new "stdlib dispatch holds for the two-level v2 command tree"
~/bin/spine adr new "eval seam: schema in spine, process in the model-eval skill"
~/bin/spine adr new "adopt composes init and update; no mapping, no migration"
```

Fill each body (Context/Decision/Consequences) from the spec's Decisions §1/§2/§5 — 0006: two-level tree = adr-new/list shape, cobra trigger moves to three levels or persistent flags; 0007: stage/score opaque, loop changes are template bumps, no Go branching on values; 0008: legacy trees stay put as info, fleet converges on one shape, post-condition doctor-clean/update-no-op.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: dogfood gen 2 — self-update + ADRs 0006-0008"
```

---

### Task 15: Live acceptance (INLINE WITH RUSSELL — live-system mutation)

No subagents for this task: it mutates real repos and needs human-reviewed diffs between steps (v1's hbmview precedent).

- [ ] **Step 1: Dry-run adopt plans on all four targets, present to Russell**

```bash
~/bin/spine adopt --dir ~/Projects/github.com/praxis
~/bin/spine adopt --dir ~/Projects/github.com/moo-clone
~/bin/spine adopt --dir ~/Projects/github.com/home-lab-admin
~/bin/spine adopt --dir ~/Projects/github.com/obsidian-ep-vault
```
Expected: exit 1 each; profiles go-service / swift / infra / knowledge. **STOP — human reviews each plan before any --write.**

- [ ] **Step 2: Apply per approved target, verify post-condition, review diff**

Per target (praxis shown):

```bash
~/bin/spine adopt --dir ~/Projects/github.com/praxis --write
~/bin/spine doctor --dir ~/Projects/github.com/praxis
~/bin/spine update --dir ~/Projects/github.com/praxis; echo "exit: $?"
git -C ~/Projects/github.com/praxis diff --stat
```
Expected: doctor exit 0; update exit 0 (no-op); Russell reviews `git diff`; commit in that repo only on his OK (praxis pushes use remote `github`, and only on his say-so).

- [ ] **Step 3: Eval retrofit on local-model-evaluation**

```bash
~/bin/spine eval new --dir ~/Projects/github.com/local-model-evaluation "govmomi vsphere inventory cli"
for m in claude-code-opus-4.7 gemma-4-12b gemma-4-31b orinth-1.0-35b-fp16 qwen-3.6-27b qwen-agentworld-35b-a3b qwen3-coder-next qwen3.6-35b-a3b-ud-mxfp8_k_xl-mlx; do ~/bin/spine eval add-run --dir ~/Projects/github.com/local-model-evaluation --eval govmomi-vsphere-inventory-cli --name "$m"; done
```

Then hand-fill the 8 run records once from the repo's README results table + each model dir's REVIEW.md (model, stage, score front matter + section summaries). Verify:

```bash
~/bin/spine eval list --dir ~/Projects/github.com/local-model-evaluation
~/bin/spine doctor --dir ~/Projects/github.com/local-model-evaluation --json
```
Expected: 8 rows with real scores; no D7 findings.

- [ ] **Step 4: Fleet scan + gen-1 fleet updates**

```bash
~/bin/spine handoff latest --fleet ~/Projects/github.com
~/bin/spine update --dir ~/Projects/github.com/ccq --write && ~/bin/spine doctor --dir ~/Projects/github.com/ccq
~/bin/spine update --dir ~/Projects/github.com/hbmview --write && ~/bin/spine doctor --dir ~/Projects/github.com/hbmview
```
Expected: fleet table ordered newest-first (~9 repos); both updates stamp-only; doctors clean. Russell reviews + commits per repo.

- [ ] **Step 5: Wrap** — final whole-branch review of the spine repo (fresh-context reviewer against spec + plan), then `spine handoff new` for the session wrap. Push nothing anywhere unless Russell says.

---

## Plan Self-Review (performed while writing)

- **Spec coverage:** adopt (T5, T11, fixtures T13, live T15) · handoff new/list/latest/--fleet (T7, T8) · eval + docs/evals + opt-in README (T4, T9), D7/D8 (T10) · profiles + manifest + detection (T3), D1 profile-aware (T6) · gen 2 + stamp-only upgrade (T1, locked by T13) · --json everywhere structured: handoff (T7/T8), eval (T9), adopt (T11), adr (T12), doctor (already v1) · ADRs 0006–0008 (T14) · acceptance targets (T15). Non-goals respected: no mapping keys, no migration, no stage-value validation anywhere.
- **Type consistency:** `update.Options.{AdoptProfile,AdoptName}` (T5) consumed by `adopt.Run` (T11); `handoff.Entry/ParseName` (T7) consumed by `Fleet` (T8) and doctor D8 (T10); `eval.List → ([]Eval, []Problem, error)` (T9) consumed by doctor D7 (T10); `tmpl.ProfileDirs/ProfileOwns` (T3) consumed by scaffold (T3), update (T4), doctor (T6), adopt (T11).
- **Known judgment calls:** `meta.Parse` accepting bare `key:` (needed for the empty `stage:`/`score:` template lines) slightly widens adr front-matter acceptance — documented in T2 with rationale. `eval list` text output prints dash rows for run-less evals. `handoff latest` on an empty repo exits 1 (nothing found ≠ error).
