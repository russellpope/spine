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

func TestAddRunSecondSameNameFails(t *testing.T) {
	dir := t.TempDir()
	evalDir, err := New(dir, "collision eval")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddRun(dir, filepath.Base(evalDir), "run1"); err != nil {
		t.Fatal(err)
	}
	_, err = AddRun(dir, filepath.Base(evalDir), "run1")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want already-exists error, got %v", err)
	}
}
