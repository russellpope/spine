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
	for _, want := range []string{"# Workflow — demo", "profile: rust", "template_version: 2",
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
