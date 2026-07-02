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
