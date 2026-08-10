package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
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

// stageCurrentRepo starts with a generation-10 mirror so current-table
// refresh behavior is exercised without modifying historical fixtures.
func stageCurrentRepo(t *testing.T, mutate func(string) string) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if mutate != nil {
		content = mutate(content)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The captured bare-key rows carry both historical Claude defaults. Update
// refreshes each inherited pair to its current pair and itemizes both.
func TestInheritedBareRowsRefreshedAndItemized(t *testing.T) {
	dir := stageGen8Repo(t, nil)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("want Pending, got state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	if len(wf.ModelRefreshes) != 2 {
		t.Fatalf("ModelRefreshes = %+v, want routine and fallback refreshes", wf.ModelRefreshes)
	}
	routine := wf.ModelRefreshes[0]
	if routine.Key != "model_routing.claude.routine" || routine.Old != "claude-sonnet-5" || routine.New != "claude-opus-5 @ low" {
		t.Errorf("routine refresh = %+v, want {model_routing.claude.routine claude-sonnet-5 claude-opus-5 @ low}", routine)
	}
	m := wf.ModelRefreshes[1]
	// I036: refreshes itemize under the flavor-qualified dotted key.
	if m.Key != "model_routing.claude.fallback" || m.Old != "claude-opus-4-8" || m.New != "claude-opus-5" {
		t.Errorf("refresh item = %+v, want {model_routing.claude.fallback claude-opus-4-8 claude-opus-5}", m)
	}
	if len(wf.ModelOverrides) != 0 {
		t.Errorf("pristine fixture reported overrides: %+v", wf.ModelOverrides)
	}
	if !strings.Contains(wf.Diff, "claude.fallback:   claude-opus-5") {
		t.Errorf("diff does not carry the refreshed value:\n%s", wf.Diff)
	}
	if !strings.Contains(wf.Diff, "claude.routine:    claude-opus-5 @ low") {
		t.Errorf("diff does not carry the refreshed routine value:\n%s", wf.Diff)
	}

	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "claude.fallback:   claude-opus-5") {
		t.Error("written file does not carry the current fallback default")
	}
	if strings.Contains(string(got), "claude-opus-4-8") {
		t.Error("written file still carries the stale inherited default")
	}
	if strings.Contains(string(got), "claude-sonnet-5") {
		t.Error("written file still carries the stale inherited routine default")
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
	if len(wf.ModelRefreshes) != 1 || wf.ModelRefreshes[0].Key != "model_routing.claude.routine" {
		t.Errorf("ModelRefreshes = %+v, want only the inherited routine refresh", wf.ModelRefreshes)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Key != "model_routing.claude.fallback" ||
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
	if !strings.Contains(string(got), "claude.fallback: claude-opus-3-pinned") {
		t.Error("deliberate override did not survive the update")
	}
	if strings.Contains(string(got), "claude.fallback:   claude-opus-5") {
		t.Error("fallback override was clobbered by the current default")
	}
}

func TestCurrentMirrorStaleRoutineRefreshesAndIsItemized(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return mustReplace(t, content, "claude.routine:    claude-opus-5 @ low", "claude.routine:    claude-sonnet-5")
	})
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) != 0 {
		t.Fatalf("state=%v unrecognized=%v, want clean pending refresh", wf.State, wf.Unrecognized)
	}
	if len(wf.ModelRefreshes) != 1 {
		t.Fatalf("ModelRefreshes = %+v, want exactly routine refresh", wf.ModelRefreshes)
	}
	if got := wf.ModelRefreshes[0]; got.Key != "model_routing.claude.routine" || got.Old != "claude-sonnet-5" || got.New != "claude-opus-5 @ low" {
		t.Errorf("routine refresh = %+v, want {model_routing.claude.routine claude-sonnet-5 claude-opus-5 @ low}", got)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "claude.routine:    claude-opus-5 @ low") {
		t.Error("written current mirror did not refresh Claude routine to Opus low")
	}
}

func TestCurrentMirrorRoutineOverrideIsPreserved(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return mustReplace(t, content, "claude.routine:    claude-opus-5 @ low", "claude.routine:    local-llama-70b")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if len(wf.ModelRefreshes) != 0 {
		t.Errorf("ModelRefreshes = %+v, want none for a routine override", wf.ModelRefreshes)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Key != "model_routing.claude.routine" || wf.ModelOverrides[0].Value != "local-llama-70b" {
		t.Errorf("ModelOverrides = %+v, want preserved routine override", wf.ModelOverrides)
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "claude.routine: local-llama-70b") {
		t.Error("write update did not preserve the current-mirror routine override")
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf = report(t, reports, "WORKFLOW.md")
	if wf.State != UpToDate || len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Value != "local-llama-70b" {
		t.Errorf("second pass state=%v overrides=%+v, want an unchanged preserved routine override", wf.State, wf.ModelOverrides)
	}
}

func TestCurrentMirrorOpusLowIsIdempotent(t *testing.T) {
	dir := stageCurrentRepo(t, nil)
	before, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		reports, err := Run(Options{Dir: dir, Write: true})
		if err != nil {
			t.Fatal(err)
		}
		wf := report(t, reports, "WORKFLOW.md")
		if wf.State != UpToDate || len(wf.ModelRefreshes) != 0 || len(wf.ModelOverrides) != 0 {
			t.Errorf("run %d: state=%v refreshes=%+v overrides=%+v, want unchanged current mirror", i+1, wf.State, wf.ModelRefreshes, wf.ModelOverrides)
		}
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("already-current Opus-low mirror changed across write updates")
	}
}

// I063 AC5: spine itself carried the owner-selected Opus-low row before it
// became the estate default. Keep the checked-in WORKFLOW model mirror
// byte-stable under the real update path, so a future table change cannot be
// declared idempotent using only a freshly scaffolded, already-canonical repo.
func TestSpineOwnWorkflowModelMirrorIsByteStable(t *testing.T) {
	before, err := os.ReadFile(filepath.Join("..", "..", "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), before, 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		reports, err := Run(Options{Dir: dir, Write: true})
		if err != nil {
			t.Fatal(err)
		}
		wf := report(t, reports, "WORKFLOW.md")
		if wf.State != UpToDate || len(wf.ModelRefreshes) != 0 || len(wf.ModelOverrides) != 0 {
			t.Errorf("run %d: state=%v refreshes=%+v overrides=%+v, want spine WORKFLOW unchanged", i+1, wf.State, wf.ModelRefreshes, wf.ModelOverrides)
		}
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("spine WORKFLOW changed across write updates")
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
	extracted["model_routing.fallback"] = "some-strange-value"     // differs from every default
	extracted["model_routing.claude.fallback"] = "another-strange" // gen-10 dotted spelling
	extracted["effort"] = "xhigh"                                  // retired key, customized (D16)
	extracted["model_default"] = "some-strange-value"              // retired key, customized (I036 ruling)
	extracted["functional_harness"] = "rest"                       // real non-model choice (rust default is cli)
	choices, err := Choices(extracted, "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	for k := range choices {
		if strings.HasPrefix(k, "model_routing.") {
			t.Errorf("model key %q classified by the choice-vs-default rule", k)
		}
		if k == "effort" || k == "model_default" {
			t.Errorf("retired key %q classified by the choice-vs-default rule", k)
		}
	}
	if choices["functional_harness"] != "rest" {
		t.Errorf("non-model choice lost: %#v", choices)
	}
}
