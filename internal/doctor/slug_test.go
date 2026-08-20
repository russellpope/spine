package doctor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/scaffold"
)

// slugFixture scaffolds a git repo carrying docs/issues/, with the global git
// config redirected to an empty file so the developer's own settings cannot
// reach the check.
func slugFixture(t *testing.T, slug string) string {
	t.Helper()
	globalCfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(globalCfg, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", globalCfg)
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "library-cli", "demo"); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	if slug != "" {
		run("config", scaffold.SlugKey, slug)
	}
	return dir
}

func slugFindings(t *testing.T, dir string) []doctor.Finding {
	t.Helper()
	fs, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []doctor.Finding
	for _, f := range fs {
		if f.ID == "D12" {
			out = append(out, f)
		}
	}
	return out
}

func TestD12WarnsOnMissingSlug(t *testing.T) {
	fs := slugFindings(t, slugFixture(t, ""))
	if len(fs) != 1 || fs[0].Severity != "warn" {
		t.Fatalf("want one D12 warn, got %#v", fs)
	}
	if !strings.Contains(fs[0].Message, scaffold.SlugRemedy) {
		t.Errorf("message lacks the remedy command: %q", fs[0].Message)
	}
}

func TestD12WarnsOnMalformedSlug(t *testing.T) {
	for _, bad := range []string{"acme", "acme/", "/x", "acme/-x", "ac me/x", "acme/x/y"} {
		fs := slugFindings(t, slugFixture(t, bad))
		if len(fs) != 1 || fs[0].Severity != "warn" {
			t.Errorf("slug %q: want one D12 warn, got %#v", bad, fs)
			continue
		}
		if !strings.Contains(fs[0].Message, scaffold.SlugRemedy) || !strings.Contains(fs[0].Message, bad) {
			t.Errorf("slug %q: message %q lacks the value or the remedy", bad, fs[0].Message)
		}
	}
}

// TestD12SilentOnWellFormedSlug is the negative control: the check must not
// fire on a healthy repo, or it would warn on the whole fleet.
func TestD12SilentOnWellFormedSlug(t *testing.T) {
	for _, ok := range []string{"acme/x", "a-c.m_e/some.repo-1"} {
		if fs := slugFindings(t, slugFixture(t, ok)); len(fs) != 0 {
			t.Errorf("slug %q: want no D12 findings, got %#v", ok, fs)
		}
	}
}

// TestD12SilentOutsideGit: a scaffolded directory that is not a git repo has
// nowhere to hold the key, so the check stays quiet.
func TestD12SilentOutsideGit(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "library-cli", "demo"); err != nil {
		t.Fatal(err)
	}
	if fs := slugFindings(t, dir); len(fs) != 0 {
		t.Fatalf("want no D12 findings outside git, got %#v", fs)
	}
}
