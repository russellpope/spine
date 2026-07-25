package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// modelRefreshDiffLines are the diff-line bodies (TrimSpace'd, matching the
// isGenNContentDiffLine convention) that the I035 model-table refresh (design
// D6) adds to any migration diff whose fixture carries a prior shipped
// default: the inherited old value's rendered line ("-") and the current
// default's rendered line ("+"). Every strict generation lock consults this
// set — sanctioned 2026-07-24 (I035, owner-approved): before I035 those
// locks were pinning the propagation bug itself, in which a stale inherited
// default was classified as a choice, carried forward, and produced no diff
// line at all.
var modelRefreshDiffLines = map[string]bool{
	// gen 6–9 rendered claude fallback row, refreshed away ("-").
	"fallback: claude-opus-4-8        # primary-refused or security-framed work": true,
	// the current table default's rendered row ("+").
	"fallback: claude-opus-5          # primary-refused or security-framed work": true,
}

// isModelRefreshDiffLine reports whether a unified-diff line carries the
// sanctioned model-table refresh above.
func isModelRefreshDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	return modelRefreshDiffLines[strings.TrimSpace(line[1:])]
}

// stageGen8Repo copies the ccq-gen8 capture — a realistic repo whose mirror
// carries BARE tier keys (`fallback: claude-opus-4-8`), the on-disk format
// every real gen ≤9 repo has — into a temp dir, with mutate applied to
// WORKFLOW.md first ("" = pristine).
func stageGen8Repo(t *testing.T, mutate func(string) string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen8", name))
		if err != nil {
			t.Fatal(err)
		}
		content := string(raw)
		if name == "WORKFLOW.md" && mutate != nil {
			content = mutate(content)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// AC (I035): a repo whose bare-key fallback carries the previous shipped
// default is refreshed to the current default, and the refresh is itemized
// with old value, new value, and inherited provenance — not left to be
// inferred from the content diff.
func TestInheritedBareFallbackRefreshedAndItemized(t *testing.T) {
	dir := stageGen8Repo(t, nil)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("want Pending, got state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	if len(wf.ModelRefreshes) != 1 {
		t.Fatalf("ModelRefreshes = %+v, want exactly the fallback refresh", wf.ModelRefreshes)
	}
	m := wf.ModelRefreshes[0]
	if m.Key != "model_routing.fallback" || m.Old != "claude-opus-4-8" || m.New != "claude-opus-5" {
		t.Errorf("refresh item = %+v, want {model_routing.fallback claude-opus-4-8 claude-opus-5}", m)
	}
	if len(wf.ModelOverrides) != 0 {
		t.Errorf("pristine fixture reported overrides: %+v", wf.ModelOverrides)
	}
	if !strings.Contains(wf.Diff, "fallback: claude-opus-5") {
		t.Errorf("diff does not carry the refreshed value:\n%s", wf.Diff)
	}

	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "fallback: claude-opus-5") {
		t.Error("written file does not carry the current fallback default")
	}
	if strings.Contains(string(got), "claude-opus-4-8") {
		t.Error("written file still carries the stale inherited default")
	}
}

// AC (I035): a repo whose bare-key fallback carries a value no default ever
// shipped keeps it untouched and has it reported as an override.
func TestOverrideBareFallbackPreservedAndReported(t *testing.T) {
	dir := stageGen8Repo(t, func(content string) string {
		out := strings.Replace(content, "fallback: claude-opus-4-8", "fallback: claude-opus-3-pinned", 1)
		if out == content {
			t.Fatal("fixture fallback line not found to replace")
		}
		return out
	})
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State == SkippedUnrecognized {
		t.Fatalf("override misread as unrecognized local edit: %v", wf.Unrecognized)
	}
	if len(wf.ModelRefreshes) != 0 {
		t.Errorf("override wrongly scheduled for refresh: %+v", wf.ModelRefreshes)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Key != "model_routing.fallback" ||
		wf.ModelOverrides[0].Value != "claude-opus-3-pinned" {
		t.Errorf("ModelOverrides = %+v, want the pinned fallback reported", wf.ModelOverrides)
	}

	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "fallback: claude-opus-3-pinned") {
		t.Error("deliberate override did not survive the update")
	}
	if strings.Contains(string(got), "fallback: claude-opus-5") {
		t.Error("override was clobbered by the current default")
	}
}

// AC (I035): nothing is written without the write flag — the refresh is
// plan-only until --write.
func TestRefreshWritesNothingWithoutWriteFlag(t *testing.T) {
	dir := stageGen8Repo(t, nil)
	before, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("dry-run modified WORKFLOW.md")
	}
}

// AC (I035, design D7): model-routing keys no longer appear in generic
// choice extraction — a routing value that differs from the rendered default
// must NOT come back as a choice (that misclassification was the propagation
// trap), while non-model keys keep the unchanged choice-vs-default rule.
func TestChoicesExcludesModelRoutingKeys(t *testing.T) {
	extracted := ExtractKeys(gen0Hbmview)
	extracted["model_routing.fallback"] = "some-strange-value" // differs from every default
	extracted["functional_harness"] = "rest"                   // real non-model choice (rust default is cli)
	choices, err := Choices(extracted, "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	for k := range choices {
		if strings.HasPrefix(k, "model_routing.") {
			t.Errorf("model key %q classified by the choice-vs-default rule", k)
		}
	}
	if choices["functional_harness"] != "rest" {
		t.Errorf("non-model choice lost: %#v", choices)
	}
}
