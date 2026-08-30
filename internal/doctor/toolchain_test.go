package doctor_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/doctor"
	"github.com/russellpope/spine/internal/scaffold"
)

func TestD14IgnoresPatchToolchainSkew(t *testing.T) {
	major, minor := runtimeMajorMinor(t)
	dir := cleanDoctorRepo(t)
	setPathGoVersion(t, fmt.Sprintf("go%d.%d.999", major, minor))

	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := d14Findings(findings); len(got) != 0 {
		t.Fatalf("patch-only toolchain skew must stay quiet, got %#v", got)
	}
}

func TestD14WarnsOnMinorToolchainSkew(t *testing.T) {
	major, minor := runtimeMajorMinor(t)
	dir := cleanDoctorRepo(t)
	pathVersion := fmt.Sprintf("go%d.%d.0", major, minor+1)
	setPathGoVersion(t, pathVersion)

	findings, err := doctor.Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := d14Findings(findings)
	if len(got) != 1 {
		t.Fatalf("want one D14 toolchain advisory, got %#v (all: %#v)", got, findings)
	}
	if got[0].Severity != "warn" || got[0].Path != "go" {
		t.Errorf("D14 finding = %#v, want warn on go", got[0])
	}
	for _, want := range []string{runtime.Version(), pathVersion, "binary toolchain", "toolchain on PATH", "make install"} {
		if !strings.Contains(got[0].Message, want) {
			t.Errorf("D14 message missing %q: %q", want, got[0].Message)
		}
	}
}

func cleanDoctorRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func setPathGoVersion(t *testing.T, version string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "go")
	content := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' %q\n", version)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func runtimeMajorMinor(t *testing.T) (int, int) {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(runtime.Version(), "go"), ".")
	if len(parts) < 2 {
		t.Fatalf("runtime version %q has no major/minor", runtime.Version())
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		t.Fatalf("runtime major in %q: %v", runtime.Version(), err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("runtime minor in %q: %v", runtime.Version(), err)
	}
	return major, minor
}

func d14Findings(findings []doctor.Finding) []doctor.Finding {
	var got []doctor.Finding
	for _, finding := range findings {
		if finding.ID == "D14" {
			got = append(got, finding)
		}
	}
	return got
}
