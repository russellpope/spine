package model

import (
	"os"
	"path/filepath"
	"strings"
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

// The consolidated block rule (I037): a whitespace-only line ends the
// model_routing block, for every consumer of the shared parser. The mirror
// is machine-rendered contiguous, so an override stranded after a stray
// blank line is not attributable to the block — it is ignored and the entry
// resolves to its default, rather than indented prose further down the file
// being read as routing entries.
func TestResolve_BlankLineEndsRoutingBlock(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  primary: claude-fable-5\n\n  fallback: stranded-override\n")
	entry, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Default {
		t.Errorf("got %+v, want the embedded default — a value after a blank line is outside the block", entry)
	}
}

// TRANSITIONAL bare-tier affordance (I035): a gen ≤9 mirror's bare tier key
// (`fallback: claude-opus-4-8`) is read as a claude-flavored value — this is
// what every real repo carries today, so the refresh rule must see it. A
// historical value reports Inherited, exactly as its dotted equivalent would.
func TestResolve_BareTierKey_ReadAsClaudeFlavored(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  fallback: claude-opus-4-8        # primary-refused or security-framed work\n")
	entry, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "claude-opus-4-8" || entry.Provenance != Inherited {
		t.Errorf("got %+v, want id=claude-opus-4-8 provenance=inherited", entry)
	}
}

// Bare tier keys carrying a value no default ever shipped report Override —
// the preserve side of D6, against the on-disk format real repos carry.
func TestResolve_BareTierKey_UnknownValue_ReportsOverride(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  fallback: claude-opus-3-pinned\n")
	entry, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "claude-opus-3-pinned" || entry.Provenance != Override {
		t.Errorf("got %+v, want id=claude-opus-3-pinned provenance=override", entry)
	}
}

// Bare tier keys are claude-only: the flavor they imply is the one every
// gen ≤9 mirror rendered. They must stay invisible to any other flavor.
func TestResolve_BareTierKey_InvisibleToOtherFlavors(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  fallback: claude-opus-4-8\n")
	entry, err := Resolve(dir, "codex", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Default || entry.ID != "gpt-5.6-terra" {
		t.Errorf("bare claude key leaked into codex: %+v", entry)
	}
}

// When both a dotted and a bare key name the same tier, the dotted (gen-10,
// D8) spelling wins regardless of order — the newer format is authoritative.
func TestResolve_DottedKeyWinsOverBare(t *testing.T) {
	for _, content := range []string{
		"model_routing:\n  claude.fallback: claude-dotted\n  fallback: claude-bare\n",
		"model_routing:\n  fallback: claude-bare\n  claude.fallback: claude-dotted\n",
	} {
		dir := writeWorkflow(t, content)
		entry, err := Resolve(dir, "claude", "fallback")
		if err != nil {
			t.Fatal(err)
		}
		if entry.ID != "claude-dotted" {
			t.Errorf("content %q: ID = %q, want claude-dotted", content, entry.ID)
		}
	}
}

// I035 carried item from I033 review: history entries are (id, effort)
// pairs, not bare ids. A value matching a historical id is Inherited only at
// the effort that pair actually shipped with; the same id at a different
// effective effort is an Override. Exercised via resolveFrom with a
// synthetic table because the real table's history entries all ship at tier
// default effort today.
func TestResolveFrom_HistoryPairMatchesOnEffortToo(t *testing.T) {
	synth := table{
		TierDefaultEffort: map[string]string{"primary": "high", "routine": "medium", "mechanical": "low", "fallback": "high"},
		Flavors: map[string]map[string]tableEntry{
			"claude": {
				"primary":    {ID: "new-model"},
				"routine":    {ID: "r"},
				"mechanical": {ID: "m"},
				"fallback": {
					ID:      "new-fb",
					History: []historyEntry{{ID: "old-fb", Effort: "xhigh"}},
				},
			},
		},
	}

	// Same id, same effort as the shipped pair -> Inherited.
	dir := writeWorkflow(t, "model_routing:\n  claude.fallback: old-fb @ xhigh\n")
	entry, err := resolveFrom(synth, dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Inherited {
		t.Errorf("historical (id, effort) pair: Provenance = %s, want %s", entry.Provenance, Inherited)
	}

	// Same historical id but effort omitted -> effective effort is the tier
	// default (high), which is NOT what the pair shipped with (xhigh) ->
	// Override, never a silent refresh candidate.
	dir = writeWorkflow(t, "model_routing:\n  claude.fallback: old-fb\n")
	entry, err = resolveFrom(synth, dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Provenance != Override {
		t.Errorf("historical id at non-shipped effort: Provenance = %s, want %s", entry.Provenance, Override)
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

// I036 (D9): a mirror value parses in every emitted shape — bare id (neither
// effort suffix nor comment), id with comment only, id with effort only, and
// id with both — with the id and effort landing intact in each case.
func TestResolve_MirrorValueShapes(t *testing.T) {
	cases := []struct {
		line, id, effort string
	}{
		{"  claude.primary: pinned-model", "pinned-model", "high"},
		{"  claude.primary: pinned-model    # local pin", "pinned-model", "high"},
		{"  claude.primary: pinned-model @ xhigh", "pinned-model", "xhigh"},
		{"  claude.primary: pinned-model @ xhigh    # local pin", "pinned-model", "xhigh"},
	}
	for _, c := range cases {
		dir := writeWorkflow(t, "model_routing:\n"+c.line+"\n")
		entry, err := Resolve(dir, "claude", "primary")
		if err != nil {
			t.Fatalf("%q: %v", c.line, err)
		}
		if entry.ID != c.id || entry.Effort != c.effort {
			t.Errorf("%q resolved to {%s, %s}, want {%s, %s}", c.line, entry.ID, entry.Effort, c.id, c.effort)
		}
	}
}

// I036 (D8/D9): MirrorValue renders the effort suffix exactly when the
// entry's effective effort deviates from its tier default — codex primary
// carries " @ xhigh", claude primary stays a bare id.
func TestMirrorValue_EffortSuffixOnlyOnDeviation(t *testing.T) {
	codex, err := Resolve("", "codex", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if got := MirrorValue(codex); got != "gpt-5.6-sol @ xhigh" {
		t.Errorf("MirrorValue(codex primary) = %q, want %q", got, "gpt-5.6-sol @ xhigh")
	}
	claude, err := Resolve("", "claude", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if got := MirrorValue(claude); got != "claude-fable-5" {
		t.Errorf("MirrorValue(claude primary) = %q, want %q", got, "claude-fable-5")
	}
}

// I036 (D8): MirrorRows renders one row per (flavor, tier) of the embedded
// table — flavors sorted, tiers in Tiers order — and every row round-trips
// through Resolve: written to a WORKFLOW.md, each row resolves back to the
// table's own (id, effort), so the rendered mirror and the reader can never
// disagree on the emitted format.
func TestMirrorRows_CoverEveryFlavorTierAndRoundTrip(t *testing.T) {
	rows := MirrorRows()
	if want := len(Flavors()) * len(Tiers); len(rows) != want {
		t.Fatalf("MirrorRows() = %d rows, want %d (every flavor x tier)", len(rows), want)
	}
	content := "model_routing:\n"
	for _, row := range rows {
		content += row + "\n"
	}
	dir := writeWorkflow(t, content)
	i := 0
	for _, flavor := range Flavors() {
		for _, tier := range Tiers {
			key := flavor + "." + tier + ":"
			if !strings.Contains(rows[i], key) {
				t.Errorf("rows[%d] = %q, want key %q (flavor-sorted, tier-fixed order)", i, rows[i], key)
			}
			i++
			def, err := Resolve("", flavor, tier)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Resolve(dir, flavor, tier)
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != def.ID || got.Effort != def.Effort {
				t.Errorf("round-trip %s.%s = {%s, %s}, want {%s, %s}", flavor, tier, got.ID, got.Effort, def.ID, def.Effort)
			}
			// A rendered default read back from disk reports Inherited, the
			// provenance the refresh rule keys on.
			if got.Provenance != Inherited {
				t.Errorf("round-trip %s.%s provenance = %s, want %s", flavor, tier, got.Provenance, Inherited)
			}
		}
	}
}
