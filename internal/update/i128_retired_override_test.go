package update

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/model"
)

// I128 item 2: an override whose id is a historical id of its harness can
// never launch (launch validation refuses historical ids byte-exactly), so
// preserving it verbatim preserves a dead value. Update migrates the id to
// its successor, keeps the effort the repo chose, and itemizes the change
// as a retired-override refresh — never as a preserved override.
func TestCurrentMirrorRetiredOverrideMigratesKeepingEffort(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return replaceRow(t, content, "claude.primary", "claude-fable-5 @ xhigh")
	})
	if _, err := model.ValidateLaunch(model.LaunchRequest{RepoDir: dir, Harness: "claude", Tier: "primary", MaxTemplateVersion: 99}); err == nil {
		t.Fatal("precondition: the stuck override must be refused before update")
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) != 0 {
		t.Fatalf("state=%v unrecognized=%v, want a clean pending migration", wf.State, wf.Unrecognized)
	}
	if len(wf.ModelRefreshes) != 1 {
		t.Fatalf("ModelRefreshes = %+v, want exactly the retired-override migration", wf.ModelRefreshes)
	}
	got := wf.ModelRefreshes[0]
	want := ModelRefresh{Key: "model_routing.claude.primary", Old: "claude-fable-5 @ xhigh", New: "claude-fable-5-1 @ xhigh", Retired: true}
	if got != want {
		t.Errorf("refresh = %+v, want %+v", got, want)
	}
	if len(wf.ModelOverrides) != 0 {
		t.Errorf("ModelOverrides = %+v, want none: a migrated retired override is not a preserved one", wf.ModelOverrides)
	}
	if !hasRow(wf.Diff, "claude.primary", "claude-fable-5-1 @ xhigh") {
		t.Errorf("diff does not carry the migrated value:\n%s", wf.Diff)
	}

	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasRow(string(after), "claude.primary", "claude-fable-5-1 @ xhigh") {
		t.Errorf("written mirror did not migrate the retired override:\n%s", after)
	}
	// The printed remedy alone reaches a validating state (I128 AC 2).
	entry, err := model.ValidateLaunch(model.LaunchRequest{RepoDir: dir, Harness: "claude", Tier: "primary", MaxTemplateVersion: 99})
	if err != nil {
		t.Fatalf("validate after the remedy: %v", err)
	}
	if entry.ID != "claude-fable-5-1" || entry.Effort != "xhigh" {
		t.Errorf("validated %s @ %s, want claude-fable-5-1 @ xhigh", entry.ID, entry.Effort)
	}
	// Second pass: the migrated value is now an ordinary current-id override,
	// preserved and reported as such, with nothing left to refresh.
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf = report(t, reports, "WORKFLOW.md")
	if wf.State != UpToDate || len(wf.ModelRefreshes) != 0 {
		t.Errorf("second pass state=%v refreshes=%+v, want up-to-date with nothing to refresh", wf.State, wf.ModelRefreshes)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Value != "claude-fable-5-1 @ xhigh" || wf.ModelOverrides[0].Migrated {
		t.Errorf("second pass overrides = %+v, want the migrated value preserved as a plain override", wf.ModelOverrides)
	}
}

// Negative control: the same effort on the CURRENT id is a deliberate
// override and stays exactly as it was — the migration keys off the id
// being historical, not off the effort suffix.
func TestCurrentMirrorCurrentIDOverrideAtXhighIsPreserved(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return replaceRow(t, content, "claude.primary", "claude-fable-5-1 @ xhigh")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if len(wf.ModelRefreshes) != 0 {
		t.Errorf("ModelRefreshes = %+v, want none for a current-id override", wf.ModelRefreshes)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Value != "claude-fable-5-1 @ xhigh" {
		t.Errorf("ModelOverrides = %+v, want the xhigh override preserved", wf.ModelOverrides)
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasRow(string(after), "claude.primary", "claude-fable-5-1 @ xhigh") {
		t.Error("write update did not preserve the current-id xhigh override")
	}
}

// A retired id with an edited alternate keeps the alternate: only the id
// token is replaced, the rest of the repo's spelling survives.
func TestCurrentMirrorRetiredOverrideKeepsAlternate(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return replaceRow(t, content, "claude.primary", "claude-fable-5 @ xhigh alt: claude-opus-5 @ high")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	want := ModelRefresh{Key: "model_routing.claude.primary", Old: "claude-fable-5 @ xhigh alt: claude-opus-5 @ high", New: "claude-fable-5-1 @ xhigh alt: claude-opus-5 @ high", Retired: true}
	if len(wf.ModelRefreshes) != 1 || wf.ModelRefreshes[0] != want {
		t.Errorf("ModelRefreshes = %+v, want %+v", wf.ModelRefreshes, want)
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasRow(string(after), "claude.primary", "claude-fable-5-1 @ xhigh alt: claude-opus-5 @ high") {
		t.Errorf("written mirror lost the alternate or the effort:\n%s", after)
	}
}

// I063's pair-aware rule still classifies claude-sonnet-5 @ low as an
// override (the resolver is unchanged); update now migrates it to the
// successor at the chosen effort instead of preserving a banned id. The
// result equals the current default, so afterwards it reads as inherited.
func TestCurrentMirrorRetiredRoutineOverrideMigratesToSuccessor(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return replaceRow(t, content, "claude.routine", "claude-sonnet-5 @ low")
	})
	live, err := model.Resolve(dir, "claude", "routine")
	if err != nil {
		t.Fatal(err)
	}
	if live.Provenance != model.Override {
		t.Fatalf("precondition: resolver classifies claude-sonnet-5 @ low as %s, want override (I063 pair-aware history)", live.Provenance)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	want := ModelRefresh{Key: "model_routing.claude.routine", Old: "claude-sonnet-5 @ low", New: "claude-opus-5 @ low", Retired: true}
	if len(wf.ModelRefreshes) != 1 || wf.ModelRefreshes[0] != want {
		t.Errorf("ModelRefreshes = %+v, want %+v", wf.ModelRefreshes, want)
	}
	if len(wf.ModelOverrides) != 0 {
		t.Errorf("ModelOverrides = %+v, want none", wf.ModelOverrides)
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf = report(t, reports, "WORKFLOW.md")
	if wf.State != UpToDate || len(wf.ModelRefreshes) != 0 || len(wf.ModelOverrides) != 0 {
		t.Errorf("second pass state=%v refreshes=%+v overrides=%+v, want an inherited current row", wf.State, wf.ModelRefreshes, wf.ModelOverrides)
	}
}

// The inherited path is untouched: a retired id at its shipped effort is
// still an inherited refresh, not a retired-override one.
func TestCurrentMirrorInheritedRetiredIDIsStillInheritedRefresh(t *testing.T) {
	dir := stageCurrentRepo(t, func(content string) string {
		return replaceRow(t, content, "claude.primary", "claude-fable-5")
	})
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	want := ModelRefresh{Key: "model_routing.claude.primary", Old: "claude-fable-5", New: "claude-fable-5-1"}
	if len(wf.ModelRefreshes) != 1 || wf.ModelRefreshes[0] != want {
		t.Errorf("ModelRefreshes = %+v, want %+v", wf.ModelRefreshes, want)
	}
}

// I128 item 4 (low): a retired model_default: equal to any id the primary
// row ever shipped was never a deliberate divergence from the lineage and
// retires quietly; a foreign value is still surfaced.
func TestModelDefaultDivergenceRetiresHistoricalPrimaryQuietly(t *testing.T) {
	dir := stageCurrentRepo(t, nil)
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"claude-fable-5", false},
		{"claude-fable-5-1", false},
		{"claude-opus-4-8", false}, // gen0's own shipped default
		{"local-llama-70b", true},
	} {
		msg, err := modelDefaultDivergence(dir, "gen0", map[string]string{"model_default": tc.value})
		if err != nil {
			t.Fatal(err)
		}
		if (msg != "") != tc.want {
			t.Errorf("model_default: %s => divergence %q, want surfaced=%v", tc.value, msg, tc.want)
		}
	}
}

// I128 item 4: the generation locks admit a mirror-row diff line only when
// the lock's own update report itemizes that exact old or new value.
func TestSanctionedRefreshLineFollowsTheReportNotAStaticAllowlist(t *testing.T) {
	refreshes := []ModelRefresh{
		{Key: "model_routing.claude.primary", Old: "claude-fable-5", New: "claude-fable-5-1"},
		{Key: "model_routing.claude.fallback", Old: "claude-opus-4-8", New: "claude-opus-5"},
	}
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"-  claude.primary:         claude-fable-5", true},
		{"+  claude.primary:         claude-fable-5-1", true},
		{"+  claude.primary: claude-fable-5-1", true},
		{"-fallback: claude-opus-4-8        # primary-refused or security-framed work", true},
		{"+  claude.fallback:        claude-opus-5", true},
		// negative controls
		{"+  claude.primary:         claude-fable-5-2", false},
		{"+  codex.primary:          gpt-5.6-sol @ xhigh", false},
		{"-  claude.routine:         claude-sonnet-5", false},
		{"   claude.primary:         claude-fable-5", false},
		{"+template_version: 14", false},
	} {
		if got := sanctionedRefreshLine(tc.line, refreshes); got != tc.want {
			t.Errorf("sanctionedRefreshLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
	if sanctionedRefreshLine("+  claude.primary: claude-fable-5-1", nil) {
		t.Error("an empty report sanctions nothing")
	}
}
