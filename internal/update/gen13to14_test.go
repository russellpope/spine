package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The captured predecessor comes from spine at template generation 13, before
// I073. It protects the actual generation boundary instead of synthesizing a
// near-current workflow in the test.
func stageGen13Repo(t *testing.T, mutate func(string) string) string {
	t.Helper()
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("testdata", "spine-gen13", "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(raw)
	if mutate != nil {
		workflow = mutate(workflow)
	}
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestGen13To14PristineUpdatesAndStaysStable(t *testing.T) {
	dir := stageGen13Repo(t, nil)
	before, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	beforeKeys := ExtractKeys(string(before))
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) != 0 {
		t.Fatalf("pristine generation-13 dry run state=%v unrecognized=%v, want clean pending migration", wf.State, wf.Unrecognized)
	}
	written, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	// A mirror row may change only as an itemized table refresh (D6): the
	// captured fixture carries the Fable 5 primary that later became history.
	refreshed := map[string]ModelRefresh{}
	for _, m := range report(t, written, "WORKFLOW.md").ModelRefreshes {
		refreshed[m.Key] = m
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	afterKeys := ExtractKeys(string(after))
	if afterKeys["template_version"] != "14" {
		t.Fatalf("template_version = %q, want 14", afterKeys["template_version"])
	}
	if afterKeys["functional_harness"] != beforeKeys["functional_harness"] {
		t.Fatalf("functional_harness changed from %q to %q", beforeKeys["functional_harness"], afterKeys["functional_harness"])
	}
	for key, value := range beforeKeys {
		if !strings.HasPrefix(key, "model_routing.") || afterKeys[key] == value {
			continue
		}
		if m, ok := refreshed[key]; ok && m.Old == value && m.New == afterKeys[key] {
			continue
		}
		t.Errorf("mirror %s changed from %q to %q without an itemized refresh", key, value, afterKeys[key])
	}
	// The converse (I128): every itemized refresh is a real before/after
	// change of that row, and the captured Fable 5 primary is itemized —
	// a table reverted to Fable 5 with empty history produces no refresh
	// and must fail here rather than pass vacuously.
	for key, m := range refreshed {
		if beforeKeys[key] != m.Old || afterKeys[key] != m.New {
			t.Errorf("itemized refresh %+v does not match the on-disk change %q -> %q", m, beforeKeys[key], afterKeys[key])
		}
	}
	if m, ok := refreshed["model_routing.claude.primary"]; !ok || m.Old != "claude-fable-5" || m.New != "claude-fable-5-1" {
		t.Errorf("primary refresh = %+v, want the captured Fable 5 row itemized as claude-fable-5 -> claude-fable-5-1", m)
	}
	if !strings.Contains(string(after), "owns the public harness migration;") {
		t.Error("generation-14 workflow lacks canonical harness migration wording")
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(after) {
		t.Error("second generation-14 write changed WORKFLOW.md")
	}
}

func TestGen13To14KeepsDottedOverrideAndRefusesLocalEditWithoutForce(t *testing.T) {
	t.Run("dotted override", func(t *testing.T) {
		dir := stageGen13Repo(t, func(workflow string) string {
			return replaceRow(t, workflow, "codex.primary", "local-codex-override @ deliberate")
		})
		if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !hasRow(string(raw), "codex.primary", "local-codex-override @ deliberate") {
			t.Error("generation-13 dotted override was overwritten")
		}
	})
	t.Run("local edit", func(t *testing.T) {
		dir := stageGen13Repo(t, func(workflow string) string {
			return strings.Replace(workflow, "owns the public flavor-to-harness rename.", "owns a local routing note.", 1)
		})
		path := filepath.Join(dir, "WORKFLOW.md")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		reports, err := Run(Options{Dir: dir, Write: true})
		if err != nil {
			t.Fatal(err)
		}
		wf := report(t, reports, "WORKFLOW.md")
		if wf.State != SkippedUnrecognized || len(wf.Unrecognized) == 0 {
			t.Fatalf("local edit state=%v unrecognized=%v, want refusal", wf.State, wf.Unrecognized)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Error("unforced write changed locally edited generation-13 WORKFLOW.md")
		}
	})
}
