package main

// I120: --supersedes went through flag.Int, whose base-0 parse read
// zero-padded ADR ids as octal ("0011" -> 9) and silently flipped the wrong
// ADR. The flag is now parsed as a base-10 string; these tests pin that and
// the success-output line naming the flipped target.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adrLedger creates docs/adr under a temp dir and scaffolds n ADRs via the
// real command, returning the repo root.
func adrLedger(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		if code, _, errs := runCmd(t, "adr", "new", "--dir", dir, fmt.Sprintf("Decision %d", i)); code != 0 {
			t.Fatalf("scaffold adr %d: %s", i, errs)
		}
	}
	return dir
}

// adrContent reads the ADR with the given zero-padded id prefix.
func adrContent(t *testing.T, dir, id string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "docs", "adr", id+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("adr %s: matches=%v err=%v", id, matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestADRSupersedesZeroPaddedIsBase10(t *testing.T) {
	dir := adrLedger(t, 11)

	code, out, errs := runCmd(t, "adr", "new", "--dir", dir, "--supersedes", "0011", "New direction")
	if code != 0 {
		t.Fatalf("code=%d out=%q stderr=%q", code, out, errs)
	}
	if !strings.Contains(out, "0012-new-direction.md") {
		t.Errorf("new ADR path missing from output: %q", out)
	}
	// Success names the flipped target so a wrong one is visible immediately.
	if !strings.Contains(out, "superseded: 0011") {
		t.Errorf("output does not name the flipped ADR: %q", out)
	}
	if !strings.Contains(adrContent(t, dir, "0012"), `supersedes: "0011"`) {
		t.Errorf("new ADR does not record supersedes 0011")
	}
	if !strings.Contains(adrContent(t, dir, "0011"), "Superseded by 0012") {
		t.Errorf("ADR 0011 not flipped")
	}
	// Negative control: the octal misparse of "0011" flipped ADR 0009.
	if got := adrContent(t, dir, "0009"); strings.Contains(got, "Superseded") {
		t.Errorf("ADR 0009 touched (octal parse lives):\n%s", got)
	}
}

func TestADRSupersedesRejectsNonDigits(t *testing.T) {
	dir := adrLedger(t, 1)
	code, _, errs := runCmd(t, "adr", "new", "--dir", dir, "--supersedes", "0x11", "X")
	if code != 2 || !strings.Contains(errs, "base-10") {
		t.Fatalf("code=%d stderr=%q; want exit 2 naming the base-10 rule", code, errs)
	}
	// The reject happens before any write: no new ADR scaffolded.
	if matches, _ := filepath.Glob(filepath.Join(dir, "docs", "adr", "0002-*.md")); len(matches) != 0 {
		t.Errorf("rejected invocation still scaffolded: %v", matches)
	}
}

func TestADRSupersedesRejectsZero(t *testing.T) {
	dir := adrLedger(t, 1)
	for _, v := range []string{"0", "0000"} {
		code, _, errs := runCmd(t, "adr", "new", "--dir", dir, "--supersedes", v, "X")
		if code != 2 || !strings.Contains(errs, "start at 0001") {
			t.Fatalf("--supersedes %s: code=%d stderr=%q; want exit 2 naming the id floor", v, code, errs)
		}
	}
}

func TestADRSupersedesRejectsExplicitEmpty(t *testing.T) {
	// An explicitly passed empty value must not silently mean "no supersede" —
	// that is the same failure class as the octal bug (a supersede the user
	// asked for that silently doesn't happen).
	dir := adrLedger(t, 1)
	code, _, errs := runCmd(t, "adr", "new", "--dir", dir, "--supersedes", "", "X")
	if code != 2 || !strings.Contains(errs, "--supersedes") {
		t.Fatalf("code=%d stderr=%q; want exit 2 naming the flag", code, errs)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "docs", "adr", "0002-*.md")); len(matches) != 0 {
		t.Errorf("rejected invocation still scaffolded: %v", matches)
	}
}

func TestADRSupersedesOverflowNamesTheRule(t *testing.T) {
	dir := adrLedger(t, 1)
	code, _, errs := runCmd(t, "adr", "new", "--dir", dir, "--supersedes", "99999999999999999999", "X")
	if code != 2 || !strings.Contains(errs, "out of range") || strings.Contains(errs, "strconv") {
		t.Fatalf("code=%d stderr=%q; want exit 2 naming the range rule without stdlib leakage", code, errs)
	}
}

func TestADRNewWithoutSupersedesPrintsNoFlipLine(t *testing.T) {
	dir := adrLedger(t, 1)
	code, out, errs := runCmd(t, "adr", "new", "--dir", dir, "Plain decision")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if strings.Contains(out, "superseded:") {
		t.Errorf("flip line printed with no supersede: %q", out)
	}
}
