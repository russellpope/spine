package scaffold_test

import (
	"os"
	"os/exec"
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

// A dir with BOTH Cargo.toml and go.mod is a real signal collision (e.g. a
// Rust project vendoring a Go tool); Cargo.toml wins by detection order.
func TestDetectProfilePriorityRustOverGo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Cargo.toml", "[package]")
	write(t, dir, "go.mod", "module x")
	got, ok := scaffold.DetectProfile(dir)
	if !ok || got != "rust" {
		t.Errorf("got %q ok=%v, want rust", got, ok)
	}
}

func TestInitCreatesAndStamps(t *testing.T) {
	dir := t.TempDir()
	res, err := scaffold.Init(dir, "rust", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 7 || len(res.Skipped) != 0 {
		t.Fatalf("created=%v skipped=%v", res.Created, res.Skipped)
	}
	wf, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Workflow — demo", "profile: rust", "template_version: 6",
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

// Acceptance (I003): the gen-6 WORKFLOW.md carries the full dispatch
// contract — four tiers with the tier->id mapping, the "auto" fallback
// wording is gone, tier effort defaults are stated, the escalation ledger
// grammar is exact, and the verify stage names the audit command.
func TestInitGen6DispatchContract(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	wf, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(wf)

	for _, want := range []string{
		// four tiers present in the model_routing mapping
		"primary: claude-fable-5",
		"routine: claude-sonnet-5",
		"mechanical: claude-haiku-4-5",
		"fallback: claude-opus-4-8",
		// effort defaults
		"primary=high, routine=medium, mechanical=low, fallback=high",
		"xhigh reserved for final verification",
		// escalation ledger grammar (exact, unspaced arrow)
		"ESCALATION <ticket-id> <from-tier>-><to-tier> reason: <one line>",
		"ESCALATION <ticket-id> effort <from>-><to> reason: <one line>",
		"FALLBACK <ticket-id> reason: <one line>",
		".superpowers/sdd/progress.md",
		// malformed records excuse nothing — the general clause, not just
		// the spaced-arrow example
		"Any record not matching the grammar exactly excuses nothing",
		// reviewer floor + named risk triggers + inline n/a nuance
		"review-tier is never below tier",
		"review-tier: n/a",
		"cross-task-integration",
		"concurrency-subtle-state",
		"security-surface",
		"plan-flagged-ambiguity",
		// fallback semantics (proactive + reactive), security_routing folded in
		"security-framed work",
		"push-notifies the owner",
		// execution modes
		"subagent-driven is the default",
		"ultracode",
		"inline is the rare justified exception",
		// verify-stage audit line
		"spine audit routing",
		"--transcripts <dir>",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("WORKFLOW.md missing %q", want)
		}
	}
	if strings.Contains(content, "auto") {
		t.Errorf("WORKFLOW.md must not contain \"auto\" (fallback must not read as automatic)")
	}
	if strings.Contains(content, "security_routing:") {
		t.Errorf("WORKFLOW.md must not carry the standalone security_routing key")
	}
}

// Acceptance (I003): the ticket template gains the optional model-routing
// annotation fields, and issues-README documents them.
func TestIssueTemplateHasAnnotationFields(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	tmplContent, err := os.ReadFile(filepath.Join(dir, "docs/issues/_template.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"execution-mode:", "tier:", "effort:", "risk-triggers: []", "review-tier:"} {
		if !strings.Contains(string(tmplContent), want) {
			t.Errorf("issue template missing %q", want)
		}
	}
	// The annotation fields must be optional additions after the pre-gen-6
	// fields (id/title/severity/status/affects/blocked-by), not a reordering
	// of them — plain bug issues written against the old field order stay
	// valid ticket files. (Whether an issue missing the fields entirely is
	// still audited correctly — unannotated, never judged — is proven
	// end-to-end in internal/audit's gen-6 proof tests.)
	for _, want := range []string{"id: I000", "title:", "severity: med", "status: open", "affects: []", "blocked-by: []"} {
		if !strings.Contains(string(tmplContent), want) {
			t.Errorf("issue template lost a pre-gen-6 field: missing %q", want)
		}
	}

	readme, err := os.ReadFile(filepath.Join(dir, "docs/issues/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"execution-mode", "tier", "effort", "risk-triggers", "review-tier",
		"review-tier: n/a"} { // inline tickets: no per-task review cycle, verify-stage gates still apply
		if !strings.Contains(string(readme), want) {
			t.Errorf("issues/README.md missing field doc for %q", want)
		}
	}
}

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

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	res, err := scaffold.Init(dir, "rust", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Created) != 0 || len(res.Skipped) != 7 {
		t.Fatalf("second run created=%v skipped=%v", res.Created, res.Skipped)
	}
}

func TestInitUnknownProfile(t *testing.T) {
	if _, err := scaffold.Init(t.TempDir(), "nope", ""); err == nil {
		t.Fatal("want error for unknown profile")
	}
}

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
		got, ok := scaffold.DetectProfile(mk(t, c.paths...))
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
	got, ok := scaffold.DetectProfile(dir)
	if !ok || got != "knowledge" {
		t.Fatalf("got %q,%v want knowledge", got, ok)
	}
}

func TestInitKnowledgeManifest(t *testing.T) {
	dir := t.TempDir()
	res, err := scaffold.Init(dir, "knowledge", "vault")
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
	for _, rel := range []string{"WORKFLOW.md", "CLAUDE.md", "AGENTS.md", "docs/adr/README.md", "docs/adr", "docs/handoffs"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}
