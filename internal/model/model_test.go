package model

import (
	"os"
	"path/filepath"
	"testing"
)

func writeWorkflow(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// AC: resolution with no repo context returns embedded defaults for every
// (flavor, tier).
func TestResolve_NoRepoContext_ReturnsDefaultsForEveryFlavorTier(t *testing.T) {
	want := map[string]map[string]struct{ id, effort string }{
		"claude": {
			"primary":    {"claude-fable-5", "high"},
			"routine":    {"claude-sonnet-5", "medium"},
			"mechanical": {"claude-haiku-4-5", "low"},
			"fallback":   {"claude-opus-5", "high"},
		},
		"codex": {
			"primary":    {"gpt-5.6-sol", "xhigh"},
			"routine":    {"gpt-5.6-terra", "medium"},
			"mechanical": {"gpt-5.6-luna", "low"},
			"fallback":   {"gpt-5.6-terra", "xhigh"},
		},
	}
	for _, repoDir := range []string{"", "/nonexistent/not-a-repo"} {
		for flavor, tiers := range want {
			for tier, exp := range tiers {
				entry, err := Resolve(repoDir, flavor, tier)
				if err != nil {
					t.Fatalf("Resolve(%q, %q, %q): %v", repoDir, flavor, tier, err)
				}
				if entry.ID != exp.id || entry.Effort != exp.effort {
					t.Errorf("Resolve(%q, %q, %q) = {%s, %s}, want {%s, %s}",
						repoDir, flavor, tier, entry.ID, entry.Effort, exp.id, exp.effort)
				}
				if entry.Provenance != Default {
					t.Errorf("Resolve(%q, %q, %q).Provenance = %s, want %s", repoDir, flavor, tier, entry.Provenance, Default)
				}
			}
		}
	}
}

// AC: a repo carrying no override resolves to defaults.
func TestResolve_RepoWithNoOverride_ResolvesToDefault(t *testing.T) {
	dir := writeWorkflow(t, "profile: library-cli\n")
	entry, err := Resolve(dir, "claude", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "claude-fable-5" || entry.Provenance != Default {
		t.Errorf("got %+v, want id=claude-fable-5 provenance=default", entry)
	}
}

// AC: a repo carrying an override resolves to the override, reported as such.
func TestResolve_RepoWithOverride_ResolvesToOverride(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.primary:    claude-custom-model\n")
	entry, err := Resolve(dir, "claude", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "claude-custom-model" {
		t.Errorf("ID = %q, want claude-custom-model", entry.ID)
	}
	if entry.Provenance != Override {
		t.Errorf("Provenance = %s, want %s", entry.Provenance, Override)
	}
}

// AC: an entry omitting effort resolves to its tier default.
func TestResolve_OmittedEffort_ResolvesToTierDefault(t *testing.T) {
	entry, err := Resolve("", "claude", "mechanical")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Effort != "low" {
		t.Errorf("Effort = %q, want low (mechanical tier default)", entry.Effort)
	}

	dir := writeWorkflow(t, "model_routing:\n  claude.mechanical: claude-custom\n")
	entry, err = Resolve(dir, "claude", "mechanical")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Effort != "low" {
		t.Errorf("override with omitted effort: Effort = %q, want low", entry.Effort)
	}
}

// AC: an entry carrying effort resolves to that effort.
func TestResolve_ExplicitEffort_ResolvesToThatEffort(t *testing.T) {
	entry, err := Resolve("", "codex", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Effort != "xhigh" {
		t.Errorf("Effort = %q, want xhigh (codex primary override)", entry.Effort)
	}

	dir := writeWorkflow(t, "model_routing:\n  claude.primary: claude-custom @ xhigh\n")
	entry, err = Resolve(dir, "claude", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Effort != "xhigh" || entry.ID != "claude-custom" {
		t.Errorf("got %+v, want id=claude-custom effort=xhigh", entry)
	}
}

// AC: a value matching any historical default reports as inherited.
func TestResolve_ValueMatchingHistory_ReportsInherited(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.fallback: claude-opus-4-8\n")
	entry, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Inherited {
		t.Errorf("Provenance = %s, want %s (claude-opus-4-8 is fallback's shipped history)", entry.Provenance, Inherited)
	}
	if entry.ID != "claude-opus-4-8" {
		t.Errorf("ID = %q, want claude-opus-4-8", entry.ID)
	}
}

// AC (converse of the above): an unrelated value reports as override.
func TestResolve_UnrelatedValue_ReportsOverride(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.fallback: claude-opus-3\n")
	entry, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Override {
		t.Errorf("Provenance = %s, want %s (claude-opus-3 matches no shipped default)", entry.Provenance, Override)
	}
}

// A value equal to the *current* default is also inherited, not a fresh
// default report — the entry set span is "everything ever shipped",
// current default included, per D5/D6.
func TestResolve_ValueMatchingCurrentDefault_ReportsInherited(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.fallback: claude-opus-5\n")
	entry, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Inherited {
		t.Errorf("Provenance = %s, want %s", entry.Provenance, Inherited)
	}
}

// AC: unknown flavor is rejected with a clear error, not silently defaulted.
func TestResolve_UnknownFlavor_Rejected(t *testing.T) {
	_, err := Resolve("", "gemini", "primary")
	if err == nil {
		t.Fatal("expected an error for unknown flavor, got nil")
	}
}

// AC: unknown tier is rejected with a clear error, not silently defaulted.
func TestResolve_UnknownTier_Rejected(t *testing.T) {
	_, err := Resolve("", "claude", "senior")
	if err == nil {
		t.Fatal("expected an error for unknown tier, got nil")
	}
}

// AC: flavors are data-driven — Flavors() reflects the embedded table's
// keys rather than a hardcoded enum, so a third flavor added to
// models/defaults.json needs no change here.
func TestFlavors_DataDriven(t *testing.T) {
	flavors := Flavors()
	if len(flavors) != 2 || flavors[0] != "claude" || flavors[1] != "codex" {
		t.Errorf("Flavors() = %v, want [claude codex]", flavors)
	}
}

// Aliases ship as data on every entry, ready for a later ticket to consume
// (nothing in this ticket reads them beyond this smoke check).
func TestResolve_CarriesAliases(t *testing.T) {
	entry, err := Resolve("", "claude", "routine")
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Aliases) == 0 {
		t.Error("expected non-empty Aliases")
	}
}

// A repo dir that exists but has no WORKFLOW.md at all behaves exactly like
// no repo context (D11's "outside a spine repo" case).
func TestResolve_RepoDirWithoutWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	entry, err := Resolve(dir, "codex", "routine")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Default || entry.ID != "gpt-5.6-terra" {
		t.Errorf("got %+v, want default gpt-5.6-terra", entry)
	}
}

// A dotted override for one (flavor, tier) must not leak into a sibling
// tier or flavor's resolution.
func TestResolve_OverrideScopedToItsOwnFlavorAndTier(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.primary: claude-custom\n")

	if entry, err := Resolve(dir, "claude", "routine"); err != nil || entry.Provenance != Default {
		t.Errorf("sibling tier claude.routine leaked override: %+v, err=%v", entry, err)
	}
	if entry, err := Resolve(dir, "codex", "primary"); err != nil || entry.Provenance != Default {
		t.Errorf("sibling flavor codex.primary leaked override: %+v, err=%v", entry, err)
	}
}

// Regression for task review Important #1: a deliberate effort override on
// an id that otherwise matches the current default must not be misreported
// as Inherited — I036's refresh rule would silently revert exactly this
// choice (user stories 6/14).
func TestResolve_EffortOverrideOnDefaultID_ReportsOverride(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.primary: claude-fable-5 @ low\n")
	entry, err := Resolve(dir, "claude", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Override {
		t.Errorf("Provenance = %s, want %s (id matches default but effort was deliberately overridden)", entry.Provenance, Override)
	}
	if entry.Effort != "low" {
		t.Errorf("Effort = %q, want low", entry.Effort)
	}
}

// Regression for task review Important #1's converse: an override that
// repeats the default id but omits effort, where the default itself carries
// an explicit effort (codex primary's xhigh), resolves to the *tier*
// default effort (high) per D3 — which disagrees with what codex primary
// actually ships. That disagreement must be reported as Override, not
// Inherited: an "inherited" entry that silently resolves to a different
// effort than the default it claims to inherit is the bug.
func TestResolve_DefaultIDOmittedEffortDivergesFromShipped_ReportsOverride(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  codex.primary: gpt-5.6-sol\n")
	entry, err := Resolve(dir, "codex", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Override {
		t.Errorf("Provenance = %s, want %s (resolves to effort %q, not shipped xhigh)", entry.Provenance, Override, entry.Effort)
	}
	if entry.Effort != "high" {
		t.Errorf("Effort = %q, want high (tier default for an entry with no effort suffix)", entry.Effort)
	}
}

// Regression for task review Important #2: a flavor present in the table
// but missing one of the four tiers (the shape a partially-populated third
// flavor would take) must error, never resolve to a zero-value Entry with
// an empty id. Exercised via resolveFrom against a synthetic table, since
// the real models/defaults.json is validated complete at load time and
// can't itself be made partial without touching the shipped data.
func TestResolveFrom_PartialTable_ReturnsError(t *testing.T) {
	partial := table{
		TierDefaultEffort: map[string]string{"primary": "high", "routine": "medium", "mechanical": "low", "fallback": "high"},
		Flavors: map[string]map[string]tableEntry{
			"local": {
				"primary": {ID: "local-big-model"},
				// routine, mechanical, fallback deliberately absent.
			},
		},
	}

	entry, err := resolveFrom(partial, "", "local", "routine")
	if err == nil {
		t.Fatalf("expected an error for a missing tier entry, got %+v", entry)
	}
	if entry.ID != "" {
		t.Errorf("entry.ID = %q on error, want zero value", entry.ID)
	}

	// The present tier still resolves normally.
	entry, err = resolveFrom(partial, "", "local", "primary")
	if err != nil {
		t.Fatalf("resolveFrom(local, primary): %v", err)
	}
	if entry.ID != "local-big-model" {
		t.Errorf("ID = %q, want local-big-model", entry.ID)
	}
}

// Load-time validation (the fix vehicle task review Minor #6 names for
// Important #2) must reject an incomplete table before Resolve ever sees
// it: a flavor missing a tier's id fails fast.
func TestValidateTable_PanicsOnFlavorMissingTierID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected validateTable to panic on a flavor missing a tier id")
		}
	}()
	validateTable(table{
		TierDefaultEffort: map[string]string{"primary": "high", "routine": "medium", "mechanical": "low", "fallback": "high"},
		Flavors: map[string]map[string]tableEntry{
			"local": {"primary": {ID: "local-big-model"}},
		},
	})
}

// Load-time validation must also reject a tierDefaultEffort map missing a
// tier — the same silent-empty-value failure mode, one field over.
func TestValidateTable_PanicsOnMissingTierDefaultEffort(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected validateTable to panic on a missing tierDefaultEffort entry")
		}
	}()
	complete := map[string]tableEntry{}
	for _, tier := range Tiers {
		complete[tier] = tableEntry{ID: "id-for-" + tier}
	}
	validateTable(table{
		TierDefaultEffort: map[string]string{"primary": "high"}, // routine/mechanical/fallback missing
		Flavors:           map[string]map[string]tableEntry{"claude": complete},
	})
}
