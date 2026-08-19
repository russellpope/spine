package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// checkpointRepo makes a tempdir git repo with one commit, so `spine
// checkpoint new` has a HEAD to record.
func checkpointRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "spine test"},
		{"add", "seed.txt"},
		{"commit", "-q", "-m", "seed"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func checkpointHeadSHA(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// narrativeFile writes a valid model region: all three sections, non-empty.
func narrativeFile(t *testing.T, dir, task string) string {
	t.Helper()
	path := filepath.Join(dir, "narrative.md")
	body := fmt.Sprintf("## Task\n%s\n\n## Conclusions\nThe parser was the culprit.\n\n## Next moves\nWire the CLI seam.\n", task)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckpointNewLatestListRoundTrip(t *testing.T) {
	dir := checkpointRepo(t)
	from := narrativeFile(t, dir, "Ship the checkpoint document")

	code, out, errs := runCmd(t, "checkpoint", "new", "--dir", dir, "--from", from,
		"--touched", "b.go,a.go", "--gate", "pass", "--effort", "high")
	if code != 0 {
		t.Fatalf("new: code=%d stderr=%q", code, errs)
	}
	path := strings.TrimSpace(out)
	want := filepath.Join(dir, ".superpowers", "sdd", "checkpoints", "001-ship-the-checkpoint-document.md")
	if path != want {
		t.Fatalf("path = %q want %q", path, want)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	for _, want := range []string{
		"ordinal: 1\n",
		"effort: high\n",
		"narrative: present\n",
		"<!-- spine:checkpoint:model -->\n## Task\nShip the checkpoint document",
		"<!-- /spine:checkpoint:model -->\n<!-- spine:checkpoint:facts -->\n",
		// touched preserves caller order, not sorted order
		"touched:\n- b.go\n- a.go\n",
		"gate: pass\n",
		"sha: " + checkpointHeadSHA(t, dir) + "\n",
		"effort_recommended: high\n",
		"<!-- /spine:checkpoint:facts -->\n",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("document missing %q; got:\n%s", want, doc)
		}
	}

	// A second checkpoint takes the next ordinal.
	code, out, errs = runCmd(t, "checkpoint", "new", "--dir", dir, "--from", from,
		"--touched", "", "--gate", "none", "--effort", "low", "--slug", "second pass")
	if code != 0 {
		t.Fatalf("new #2: code=%d stderr=%q", code, errs)
	}
	if got := filepath.Base(strings.TrimSpace(out)); got != "002-second-pass.md" {
		t.Fatalf("second checkpoint = %q", got)
	}

	code, out, errs = runCmd(t, "checkpoint", "list", "--dir", dir)
	if code != 0 {
		t.Fatalf("list: code=%d stderr=%q", code, errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "001  ship-the-checkpoint-document") || !strings.HasPrefix(lines[1], "002  second-pass") {
		t.Fatalf("list output:\n%s", out)
	}

	code, latest1, errs := runCmd(t, "checkpoint", "latest", "--dir", dir)
	if code != 0 {
		t.Fatalf("latest: code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(latest1, "ordinal: 2") {
		t.Errorf("latest did not print the newest checkpoint:\n%s", latest1)
	}
	// Byte-identical across invocations: the reload prefix must be cacheable.
	_, latest2, _ := runCmd(t, "checkpoint", "latest", "--dir", dir)
	if latest1 != latest2 {
		t.Error("latest output differs across two invocations")
	}
}

// Ordinals increase and are never reused: a crash-left reservation marker
// (the exclusive-marker technique handoffs use) consumes its ordinal even
// though no checkpoint file carries it.
func TestCheckpointOrdinalsNeverReused(t *testing.T) {
	dir := checkpointRepo(t)
	from := narrativeFile(t, dir, "First leg")
	for i := 1; i <= 2; i++ {
		if code, _, errs := runCmd(t, "checkpoint", "new", "--dir", dir, "--from", from,
			"--touched", "a.go", "--gate", "none", "--effort", "low", "--slug", fmt.Sprintf("leg-%d", i)); code != 0 {
			t.Fatalf("new #%d: code=%d stderr=%q", i, code, errs)
		}
	}
	home := filepath.Join(dir, ".superpowers", "sdd", "checkpoints")
	reservation := filepath.Join(home, ".spine-checkpoint-ordinal-reservations", "3")
	if err := os.WriteFile(reservation, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runCmd(t, "checkpoint", "new", "--dir", dir, "--from", from,
		"--touched", "a.go", "--gate", "none", "--effort", "low", "--slug", "leg-4")
	if code != 0 {
		t.Fatalf("new #4: code=%d stderr=%q", code, errs)
	}
	if got := filepath.Base(strings.TrimSpace(out)); got != "004-leg-4.md" {
		t.Fatalf("ordinal consumed by a crash-left reservation was reused: got %q", got)
	}
}

func TestCheckpointNewRefusesHollowNarrative(t *testing.T) {
	dir := checkpointRepo(t)
	cases := []struct {
		name, body, wantSection string
	}{
		{"missing section", "## Task\nx\n\n## Conclusions\ny\n", "## Next moves"},
		{"empty section", "## Task\nx\n\n## Conclusions\n\n## Next moves\nz\n", "## Conclusions"},
		{"whitespace-only section", "## Task\n   \n\n## Conclusions\ny\n\n## Next moves\nz\n", "## Task"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			from := filepath.Join(t.TempDir(), "n.md")
			if err := os.WriteFile(from, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			code, out, errs := runCmd(t, "checkpoint", "new", "--dir", dir, "--from", from,
				"--touched", "a.go", "--gate", "pass", "--effort", "high")
			if code != 2 {
				t.Fatalf("code=%d out=%q", code, out)
			}
			if !strings.Contains(errs, tc.wantSection) {
				t.Errorf("stderr %q does not name %q", errs, tc.wantSection)
			}
			if out != "" {
				t.Errorf("stdout not pristine: %q", out)
			}
		})
	}
	// Nothing was written for any refused checkpoint.
	if _, err := os.Stat(filepath.Join(dir, ".superpowers", "sdd", "checkpoints")); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(filepath.Join(dir, ".superpowers", "sdd", "checkpoints"))
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".md") {
				t.Fatalf("refused checkpoint still wrote %s", e.Name())
			}
		}
	}
}

func TestCheckpointFactsOnly(t *testing.T) {
	dir := checkpointRepo(t)
	code, out, errs := runCmd(t, "checkpoint", "new", "--dir", dir,
		"--touched", "a.go", "--gate", "fail", "--effort", "medium", "--facts-only")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	path := strings.TrimSpace(out)
	if got := filepath.Base(path); got != "001-facts-only.md" {
		t.Fatalf("filename = %q", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	if !strings.Contains(doc, "narrative: missing\n") {
		t.Errorf("frontmatter does not flag narrative: missing:\n%s", doc)
	}
	if !strings.Contains(doc, "<!-- spine:checkpoint:model -->\n<!-- /spine:checkpoint:model -->\n") {
		t.Errorf("model region is not empty:\n%s", doc)
	}
	if !strings.Contains(doc, "gate: fail\n") {
		t.Errorf("facts region missing gate: fail:\n%s", doc)
	}
}

func TestCheckpointNewRejectsBadFlags(t *testing.T) {
	dir := checkpointRepo(t)
	from := narrativeFile(t, dir, "Task line")
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"bad gate", []string{"--from", from, "--gate", "maybe", "--effort", "high"}},
		{"missing effort", []string{"--from", from, "--gate", "pass"}},
		{"missing from", []string{"--gate", "pass", "--effort", "high"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"checkpoint", "new", "--dir", dir}, tc.args...)
			if code, out, _ := runCmd(t, args...); code != 2 || out != "" {
				t.Fatalf("code=%d out=%q", code, out)
			}
		})
	}
}

func TestCheckpointLatestAndListOnEmptyHome(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := runCmd(t, "checkpoint", "latest", "--dir", dir)
	if code != 1 || out != "" || !strings.Contains(errs, "no checkpoints found") {
		t.Fatalf("latest: code=%d out=%q stderr=%q", code, out, errs)
	}
	if code, out, _ := runCmd(t, "checkpoint", "list", "--dir", dir); code != 0 || out != "" {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
}

// The reload preamble is embedded (no per-repo rendering) and states the
// trust split plus the facts-only case.
func TestCheckpointLatestPrintsPreamble(t *testing.T) {
	dir := checkpointRepo(t)
	if code, _, errs := runCmd(t, "checkpoint", "new", "--dir", dir,
		"--touched", "a.go", "--gate", "none", "--effort", "low", "--facts-only"); code != 0 {
		t.Fatalf("new: code=%d stderr=%q", code, errs)
	}
	code, out, _ := runCmd(t, "checkpoint", "latest", "--dir", dir)
	if code != 0 {
		t.Fatalf("latest: code=%d", code)
	}
	preamble, _, ok := strings.Cut(out, "---\n")
	if !ok {
		t.Fatalf("latest output has no checkpoint after the preamble:\n%s", out)
	}
	for _, want := range []string{
		"model region",
		"own prior claims",
		"never evidence",
		"facts region",
		"harness-written evidence",
		"narrative: missing",
		"reconstruct intent from facts",
	} {
		if !strings.Contains(preamble, want) {
			t.Errorf("preamble missing %q:\n%s", want, preamble)
		}
	}
}

func TestCheckpointUsage(t *testing.T) {
	code, _, errs := runCmd(t, "checkpoint")
	if code != 2 || !strings.Contains(errs, "usage: spine checkpoint") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	code, _, errs = runCmd(t, "checkpoint", "bogus")
	if code != 2 || !strings.Contains(errs, "unknown checkpoint subcommand") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if _, out, _ := runCmd(t, "help"); !strings.Contains(out, "checkpoint") {
		t.Error("top-level usage does not document checkpoint")
	}
}
