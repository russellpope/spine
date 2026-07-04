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
	if len(res.Created) != 6 || len(res.Skipped) != 0 {
		t.Fatalf("created=%v skipped=%v", res.Created, res.Skipped)
	}
	wf, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Workflow — demo", "profile: rust", "template_version: 4",
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
	for _, rel := range []string{"WORKFLOW.md", "CLAUDE.md", "docs/adr/README.md", "docs/adr", "docs/handoffs"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}
