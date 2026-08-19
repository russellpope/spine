package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// I081: `spine handoff new` embeds the newest checkpoint after the cursor
// block, and `spine doctor` gains the D11 checkpoint advisory.

// newCheckpoint writes one checkpoint through the CLI and returns its path.
func newCheckpoint(t *testing.T, dir, slug string) string {
	t.Helper()
	from := narrativeFile(t, dir, "Ship the "+slug)
	code, out, errs := runCmd(t, "checkpoint", "new", "--dir", dir, "--from", from,
		"--touched", "b.go,a.go", "--gate", "pass", "--effort", "high", "--slug", slug)
	if code != 0 {
		t.Fatalf("checkpoint new: code=%d stderr=%q", code, errs)
	}
	return strings.TrimSpace(out)
}

func newHandoff(t *testing.T, dir, topic string) string {
	t.Helper()
	code, out, errs := runCmd(t, "handoff", "new", "--dir", dir, topic)
	if code != 0 {
		t.Fatalf("handoff new: code=%d stderr=%q", code, errs)
	}
	path := strings.Split(strings.TrimSpace(out), "\n")[0]
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestHandoffEmbedsNewestCheckpoint(t *testing.T) {
	dir := checkpointRepo(t)
	newCheckpoint(t, dir, "first pass")
	cpPath := newCheckpoint(t, dir, "second pass")
	cpRaw, err := os.ReadFile(cpPath)
	if err != nil {
		t.Fatal(err)
	}
	facts := factsRegion(t, string(cpRaw))

	doc := newHandoff(t, dir, "checkpoint embed")
	// The newest checkpoint, not the first.
	if !strings.Contains(doc, "## Checkpoint (newest): 002-second-pass.md") {
		t.Fatalf("handoff does not name the newest checkpoint:\n%s", doc)
	}
	if strings.Contains(doc, "001-first-pass.md") {
		t.Errorf("handoff embedded a checkpoint other than the newest:\n%s", doc)
	}
	// Facts region verbatim, markers included.
	if !strings.Contains(doc, facts) {
		t.Errorf("facts region not embedded verbatim; want:\n%s\ngot:\n%s", facts, doc)
	}
	// Model region under the fixed heading, markers not carried over.
	heading := "### Prior narrative (model-authored, not evidence)"
	idx := strings.Index(doc, heading)
	if idx < 0 {
		t.Fatalf("missing narrative heading:\n%s", doc)
	}
	if !strings.Contains(doc[idx:], "## Task\nShip the second pass") {
		t.Errorf("narrative not embedded under the heading:\n%s", doc[idx:])
	}
	if strings.Contains(doc, "<!-- spine:checkpoint:model -->") {
		t.Errorf("model markers must not be carried into the handoff:\n%s", doc)
	}
	if strings.Index(doc, facts) > idx {
		t.Errorf("facts region must precede the prior-narrative heading:\n%s", doc)
	}
}

// factsRegion returns the checkpoint's facts region with its markers.
func factsRegion(t *testing.T, raw string) string {
	t.Helper()
	_, after, ok := strings.Cut(raw, "<!-- spine:checkpoint:facts -->")
	if !ok {
		t.Fatal("checkpoint has no facts region")
	}
	body, _, ok := strings.Cut(after, "<!-- /spine:checkpoint:facts -->")
	if !ok {
		t.Fatal("checkpoint facts region is unterminated")
	}
	return "<!-- spine:checkpoint:facts -->" + body + "<!-- /spine:checkpoint:facts -->"
}

// Negative control: no checkpoints, no embed — the handoff is byte-identical
// to what the same repo produced before this feature existed.
func TestHandoffWithoutCheckpointsUnchanged(t *testing.T) {
	dir := checkpointRepo(t)
	doc := newHandoff(t, dir, "no checkpoints here")
	for _, unwanted := range []string{"## Checkpoint (newest)", "Prior narrative", "spine:checkpoint"} {
		if strings.Contains(doc, unwanted) {
			t.Fatalf("handoff without checkpoints must not mention %q:\n%s", unwanted, doc)
		}
	}
}

// A facts-only checkpoint (narrative: missing) still embeds, and says so.
func TestHandoffEmbedsFactsOnlyCheckpoint(t *testing.T) {
	dir := checkpointRepo(t)
	code, _, errs := runCmd(t, "checkpoint", "new", "--dir", dir, "--facts-only",
		"--touched", "a.go", "--gate", "fail", "--effort", "low", "--slug", "facts only")
	if code != 0 {
		t.Fatalf("checkpoint new --facts-only: code=%d stderr=%q", code, errs)
	}
	doc := newHandoff(t, dir, "facts only embed")
	if !strings.Contains(doc, "### Prior narrative (model-authored, not evidence)\n\nnarrative: missing — reconstruct intent from facts\n") {
		t.Fatalf("facts-only checkpoint must state the missing narrative:\n%s", doc)
	}
	if !strings.Contains(doc, "gate: fail\n") {
		t.Errorf("facts region not embedded:\n%s", doc)
	}
}

// A hand-mangled checkpoint never fails the handoff: what exists is embedded
// and the gap is noted in place (doctor D11 reports the drift).
func TestHandoffWithMalformedCheckpointStillWrites(t *testing.T) {
	dir := checkpointRepo(t)
	path := newCheckpoint(t, dir, "mangled")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mangled := strings.Replace(string(raw), "<!-- spine:checkpoint:facts -->", "", 1)
	if err := os.WriteFile(path, []byte(mangled), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := newHandoff(t, dir, "mangled embed")
	if !strings.Contains(doc, "facts region: missing or malformed — run spine doctor") {
		t.Fatalf("handoff must note the missing facts region:\n%s", doc)
	}
	if !strings.Contains(doc, "## Task\nShip the mangled") {
		t.Errorf("the surviving model region must still be embedded:\n%s", doc)
	}
}

// d11 returns the D11 lines of `spine doctor` for dir.
func d11(t *testing.T, dir string) []string {
	t.Helper()
	_, out, errs := runCmd(t, "doctor", "--dir", dir)
	if errs != "" {
		t.Fatalf("doctor stderr=%q", errs)
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "D11 ") {
			lines = append(lines, l)
			if !strings.HasPrefix(l, "D11 warn ") {
				t.Errorf("D11 is advisory — warn only, got %q", l)
			}
		}
	}
	return lines
}

// Negative control: canonical checkpoints in an ignored working home are
// silent.
func TestD11SilentOnCanonicalCheckpoints(t *testing.T) {
	dir := checkpointRepo(t)
	ignoreSuperpowers(t, dir)
	newCheckpoint(t, dir, "first pass")
	newCheckpoint(t, dir, "second pass")
	if got := d11(t, dir); len(got) != 0 {
		t.Fatalf("want no D11 findings, got %#v", got)
	}
}

// Second negative control: a repo with no working home at all is silent.
func TestD11SilentWithoutWorkingHome(t *testing.T) {
	if got := d11(t, checkpointRepo(t)); len(got) != 0 {
		t.Fatalf("want no D11 findings, got %#v", got)
	}
}

func TestD11HandMutatedFactsBlock(t *testing.T) {
	dir := checkpointRepo(t)
	ignoreSuperpowers(t, dir)
	path := newCheckpoint(t, dir, "mutated")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A hand edit that still parses: extra blank-ish spacing drifts the bytes.
	mutated := strings.Replace(string(raw), "gate: pass\n", "gate:  pass\n", 1)
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	got := d11(t, dir)
	if len(got) != 1 || !strings.Contains(got[0], "facts region") || !strings.Contains(got[0], "001-mutated.md") {
		t.Fatalf("want one D11 about the mutated facts region, got %#v", got)
	}
}

// Byte drift that still parses: the block re-renders differently, so it is
// not canonical even though every key is present and legal.
func TestD11NonCanonicalFactsBlock(t *testing.T) {
	dir := checkpointRepo(t)
	ignoreSuperpowers(t, dir)
	path := newCheckpoint(t, dir, "drifted")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	drifted := strings.Replace(string(raw), "- a.go\n", "- a.go \n", 1)
	if err := os.WriteFile(path, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	got := d11(t, dir)
	if len(got) != 1 || !strings.Contains(got[0], "not canonical") {
		t.Fatalf("want one D11 non-canonical finding, got %#v", got)
	}
}

func TestD11OrdinalGap(t *testing.T) {
	dir := checkpointRepo(t)
	ignoreSuperpowers(t, dir)
	newCheckpoint(t, dir, "first pass")
	newCheckpoint(t, dir, "second pass")
	newCheckpoint(t, dir, "third pass")
	home := filepath.Join(dir, ".superpowers", "sdd", "checkpoints")
	if err := os.Remove(filepath.Join(home, "002-second-pass.md")); err != nil {
		t.Fatal(err)
	}
	got := d11(t, dir)
	if len(got) != 1 || !strings.Contains(got[0], "ordinal gap: no checkpoint 002 between 001 and 003") {
		t.Fatalf("want one D11 ordinal-gap finding, got %#v", got)
	}
}

func TestD11UnignoredSuperpowers(t *testing.T) {
	dir := checkpointRepo(t)
	newCheckpoint(t, dir, "first pass")
	got := d11(t, dir)
	if len(got) != 1 || !strings.Contains(got[0], "not gitignored") {
		t.Fatalf("want one D11 gitignore advisory, got %#v", got)
	}
	// Negative control: ignoring it silences the advisory.
	ignoreSuperpowers(t, dir)
	if got := d11(t, dir); len(got) != 0 {
		t.Fatalf("want no D11 findings once .superpowers/ is ignored, got %#v", got)
	}
}

func ignoreSuperpowers(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".superpowers/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
