package model

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/russellpope/spine/models"
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
// (harness, tier).
func TestResolve_NoRepoContext_ReturnsDefaultsForEveryHarnessTier(t *testing.T) {
	want := map[string]map[string]struct{ id, effort string }{
		"claude": {
			"primary":    {"claude-fable-5", "high"},
			"routine":    {"claude-opus-5", "low"},
			"mechanical": {"claude-haiku-4-5", "low"},
			"fallback":   {"claude-opus-5", "high"},
		},
		"codex": {
			"primary":    {"gpt-5.6-sol", "xhigh"},
			"routine":    {"gpt-5.6-terra", "medium"},
			"mechanical": {"gpt-5.6-luna", "low"},
			"fallback":   {"gpt-5.6-terra", "xhigh"},
		},
		// I110. Every tier resolves at effort "high", including routine and
		// mechanical — the two the global tierDefaultEffort would otherwise
		// give "medium" and "low". That is what tierDefaultEffortByHarness is
		// for, and asserting those two specifically is the point of this
		// block. fallback deliberately shares primary's id: the harness exists
		// to measure open-weights models, so a refusal re-run must not
		// silently leave open weights.
		"openweights": {
			"primary":    {"FW-Kimi-K3", "high"},
			"routine":    {"DeepSeek-V4-Pro", "high"},
			"mechanical": {"FW-GLM-5.2", "high"},
			"fallback":   {"FW-Kimi-K3", "high"},
		},
		"pi": {
			"primary":    {"qwen3.8-27b-q8_0", "xhigh"},
			"routine":    {"qwen3.8-27b-q8_0", "medium"},
			"mechanical": {"qwen3.8-27b-q8_0", "low"},
			"fallback":   {"qwen3.8-27b-q8_0", "xhigh"},
		},
	}
	for _, repoDir := range []string{"", "/nonexistent/not-a-repo"} {
		for _, harness := range Harnesses() {
			tiers, ok := want[harness]
			if !ok {
				t.Fatalf("Harnesses() includes %q but the default-resolution test has no expectations for it", harness)
			}
			for _, tier := range Tiers {
				exp, ok := tiers[tier]
				if !ok {
					t.Fatalf("default-resolution expectations for %q omit tier %q", harness, tier)
				}
				entry, err := Resolve(repoDir, harness, tier)
				if err != nil {
					t.Fatalf("Resolve(%q, %q, %q): %v", repoDir, harness, tier, err)
				}
				if entry.ID != exp.id || entry.Effort != exp.effort {
					t.Errorf("Resolve(%q, %q, %q) = {%s, %s}, want {%s, %s}",
						repoDir, harness, tier, entry.ID, entry.Effort, exp.id, exp.effort)
				}
				if entry.Provenance != Default {
					t.Errorf("Resolve(%q, %q, %q).Provenance = %s, want %s", repoDir, harness, tier, entry.Provenance, Default)
				}
			}
		}
	}
}

func TestDefaultModelTokensAreDisjointAcrossHarnesses(t *testing.T) {
	seen := map[string]string{}
	for _, harness := range Harnesses() {
		for _, tier := range Tiers {
			entry := defaults.Harnesses[harness][tier]
			tokens := append([]string{entry.ID}, entry.Aliases...)
			for _, historical := range entry.History {
				tokens = append(tokens, historical.ID)
			}
			for _, token := range tokens {
				if token == "" {
					continue
				}
				if prior, ok := seen[token]; ok && prior != harness {
					t.Errorf("model token %q is declared under both %s and %s", token, prior, harness)
					continue
				}
				seen[token] = harness
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

func TestResolveDispatchTargetUsesFinalResolvedEffortWhenOmitted(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  pi.routine: qwen3.8-27b-q8_0 @ low\n")
	want, err := Resolve(dir, "pi", "routine")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveDispatchTarget(DispatchTargetRequest{RepoDir: dir, Harness: "pi", Tier: "routine"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveDispatchTarget() = %+v, want unchanged final target %+v", got, want)
	}
}

func TestResolveDispatchTargetOverridesOnlyEffortAfterSelection(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  pi.routine: qwen3.8-27b-q8_0 @ low\n")
	want, err := Resolve(dir, "pi", "routine")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveDispatchTarget(DispatchTargetRequest{RepoDir: dir, Harness: "pi", Tier: "routine", RequestedEffort: "xhigh"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Provenance != want.Provenance || !reflect.DeepEqual(got.Aliases, want.Aliases) {
		t.Fatalf("override changed target metadata: got %+v, want target %+v", got, want)
	}
	if got.Effort != "xhigh" {
		t.Fatalf("Effort = %q, want byte-exact xhigh", got.Effort)
	}
}

func TestResolveDispatchTargetRejectsWhitespaceAndInvalidSelectedHarnessEffort(t *testing.T) {
	for _, requested := range []string{" ", "high"} {
		t.Run(strconv.Quote(requested), func(t *testing.T) {
			_, err := ResolveDispatchTarget(DispatchTargetRequest{Harness: "pi", Tier: "routine", RequestedEffort: requested})
			if err == nil {
				t.Fatal("ResolveDispatchTarget() unexpectedly succeeded")
			}
		})
	}
}

func TestApplyDispatchEffortRejectsExplicitEmptyValue(t *testing.T) {
	entry, err := Resolve("", "pi", "routine")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyDispatchEffort(entry, "")
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("ApplyDispatchEffort(empty) error = %v, want explicit empty rejection", err)
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

// Claude routine's displaced Sonnet default is historical at medium effort,
// while its current Opus default is explicitly low. The same historical id
// at low must remain a deliberate override rather than a refresh candidate.
func TestResolve_ClaudeRoutineHistoryIsPairAware(t *testing.T) {
	for _, tc := range []struct {
		name       string
		value      string
		provenance Provenance
	}{
		{"shipped medium pair", "claude-sonnet-5", Inherited},
		{"unshipped low pair", "claude-sonnet-5 @ low", Override},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeWorkflow(t, "model_routing:\n  claude.routine: "+tc.value+"\n")
			entry, err := Resolve(dir, "claude", "routine")
			if err != nil {
				t.Fatal(err)
			}
			if entry.Provenance != tc.provenance {
				t.Errorf("Provenance = %s, want %s for %q", entry.Provenance, tc.provenance, tc.value)
			}
		})
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

// AC: unknown harness is rejected with a clear error, not silently defaulted.
func TestResolve_UnknownHarness_Rejected(t *testing.T) {
	_, err := Resolve("", "gemini", "primary")
	if err == nil {
		t.Fatal("expected an error for unknown harness, got nil")
	}
}

// AC: unknown tier is rejected with a clear error, not silently defaulted.
func TestResolve_UnknownTier_Rejected(t *testing.T) {
	_, err := Resolve("", "claude", "senior")
	if err == nil {
		t.Fatal("expected an error for unknown tier, got nil")
	}
}

// I110. A repo may pin different open models without waiting on a spine
// release, exactly as it may for any other harness. Guards that the new harness
// went in as data and did not acquire a special resolution path.
func TestResolve_OpenweightsRowOverriddenByRepo(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  openweights.routine:    some-other-open-model\n")
	entry, err := Resolve(dir, "openweights", "routine")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "some-other-open-model" || entry.Provenance != Override {
		t.Errorf("got %+v, want id=some-other-open-model provenance=override", entry)
	}
	// The override is scoped: its sibling tiers still resolve to the table.
	sibling, err := Resolve(dir, "openweights", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if sibling.ID != "FW-Kimi-K3" || sibling.Provenance != Default {
		t.Errorf("sibling tier = %+v, want id=FW-Kimi-K3 provenance=default", sibling)
	}
}

// AC: harnesses are data-driven — Harnesses() reflects the embedded table's
// keys rather than a hardcoded enum, so a third harness added to
// models/defaults.json needs no change here.
func TestHarnesses_DataDriven(t *testing.T) {
	harnesses := Harnesses()
	want := []string{"claude", "codex", "openweights", "pi"}
	if len(harnesses) != len(want) {
		t.Fatalf("Harnesses() = %v, want %v", harnesses, want)
	}
	for i, f := range want {
		if harnesses[i] != f {
			t.Errorf("Harnesses() = %v, want %v", harnesses, want)
			break
		}
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

// A dotted override for one (harness, tier) must not leak into a sibling
// tier or harness's resolution.
func TestResolve_OverrideScopedToItsOwnHarnessAndTier(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  claude.primary: claude-custom\n")

	if entry, err := Resolve(dir, "claude", "routine"); err != nil || entry.Provenance != Default {
		t.Errorf("sibling tier claude.routine leaked override: %+v, err=%v", entry, err)
	}
	if entry, err := Resolve(dir, "codex", "primary"); err != nil || entry.Provenance != Default {
		t.Errorf("sibling harness codex.primary leaked override: %+v, err=%v", entry, err)
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

// Regression for task review Important #2: a harness present in the table
// but missing one of the four tiers (the shape a partially-populated third
// harness would take) must error, never resolve to a zero-value Entry with
// an empty id. Exercised via resolveFrom against a synthetic table, since
// the real models/defaults.json is validated complete at load time and
// can't itself be made partial without touching the shipped data.
func TestResolveFrom_PartialTable_ReturnsError(t *testing.T) {
	partial := table{
		TierDefaultEffort: map[string]string{"primary": "high", "routine": "medium", "mechanical": "low", "fallback": "high"},
		Harnesses: map[string]map[string]tableEntry{
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

// Provenance-scoped aliases (I037 fix round 1): a deliberate Override means
// exactly its on-disk id — the shipped entry's aliases are withheld, so a
// downstream consumer (the routing audit) cannot match a dispatch on the
// displaced default id through the override's entry. Inherited entries keep
// the current default's aliases: same tier lineage, and the fleet's real
// pre-sweep repos (inherited claude-opus-4-8) depend on it.
func TestResolve_OverrideCarriesNoAliases(t *testing.T) {
	dir := writeWorkflow(t, "model_routing:\n  primary: bespoke-x\n  fallback: claude-opus-4-8\n")
	ov, err := Resolve(dir, "claude", "primary")
	if err != nil {
		t.Fatal(err)
	}
	if ov.Provenance != Override || len(ov.Aliases) != 0 {
		t.Errorf("override entry = %+v, want provenance=override with no aliases", ov)
	}
	inh, err := Resolve(dir, "claude", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if inh.Provenance != Inherited || len(inh.Aliases) == 0 {
		t.Errorf("inherited entry = %+v, want provenance=inherited keeping the current default's aliases", inh)
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
// (`fallback: claude-opus-4-8`) is read as a claude-harnessed value — this is
// what every real repo carries today, so the refresh rule must see it. A
// historical value reports Inherited, exactly as its dotted equivalent would.
func TestResolve_BareTierKey_ReadAsClaudeHarnessed(t *testing.T) {
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

// Bare tier keys are claude-only: the harness they imply is the one every
// gen ≤9 mirror rendered. They must stay invisible to any other harness.
func TestResolve_BareTierKey_InvisibleToOtherHarnesses(t *testing.T) {
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
		Harnesses: map[string]map[string]tableEntry{
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
// it: a harness missing a tier's id fails fast.
func TestValidateTable_PanicsOnHarnessMissingTierID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected validateTable to panic on a harness missing a tier id")
		}
	}()
	validateTable(table{
		TierDefaultEffort: map[string]string{"primary": "high", "routine": "medium", "mechanical": "low", "fallback": "high"},
		Harnesses: map[string]map[string]tableEntry{
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
		Harnesses:         map[string]map[string]tableEntry{"claude": complete},
	})
}

func cloneDefaultTable(t *testing.T) table {
	t.Helper()
	raw, err := json.Marshal(defaults)
	if err != nil {
		t.Fatal(err)
	}
	var cloned table
	if err := json.Unmarshal(raw, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

// Each case names a build invariant whose removal would make an ambiguous or
// unsafe selector launchable. The policy is compiled into the binary, so a
// malformed policy must panic during table validation rather than survive to
// a runtime request.
func TestValidateTableModelValidationRejectsInvalidPolicy(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*table)
	}{
		{
			name: "missing id pattern",
			mutate: func(tbl *table) {
				tbl.ModelValidation.IDPattern = ""
			},
		},
		{
			name: "invalid id pattern",
			mutate: func(tbl *table) {
				tbl.ModelValidation.IDPattern = "["
			},
		},
		{
			name: "empty token",
			mutate: func(tbl *table) {
				tbl.ModelValidation.ForbiddenTokens = append(tbl.ModelValidation.ForbiddenTokens, "")
			},
		},
		{
			name: "duplicate token",
			mutate: func(tbl *table) {
				tbl.ModelValidation.ForbiddenTokens = append(tbl.ModelValidation.ForbiddenTokens, "opus")
			},
		},
		{
			name: "empty pattern name",
			mutate: func(tbl *table) {
				tbl.ModelValidation.ForbiddenPatterns[0].Name = ""
			},
		},
		{
			name: "duplicate pattern name",
			mutate: func(tbl *table) {
				tbl.ModelValidation.ForbiddenPatterns[1].Name = tbl.ModelValidation.ForbiddenPatterns[0].Name
			},
		},
		{
			name: "invalid RE2",
			mutate: func(tbl *table) {
				tbl.ModelValidation.ForbiddenPatterns[0].RE = "["
			},
		},
		{
			name: "current id syntax failure",
			mutate: func(tbl *table) {
				entry := tbl.Harnesses["codex"]["primary"]
				entry.ID = "bad id"
				tbl.Harnesses["codex"]["primary"] = entry
			},
		},
		{
			name: "current id deny overlap",
			mutate: func(tbl *table) {
				tbl.ModelValidation.ForbiddenTokens = append(tbl.ModelValidation.ForbiddenTokens, "DeepSeek-V4-Pro")
			},
		},
		{
			name: "historical id syntax failure",
			mutate: func(tbl *table) {
				entry := tbl.Harnesses["claude"]["routine"]
				entry.History[0].ID = "bad id"
				tbl.Harnesses["claude"]["routine"] = entry
			},
		},
		{
			name: "shorthand alias absent from forbidden tokens",
			mutate: func(tbl *table) {
				var kept []string
				for _, token := range tbl.ModelValidation.ForbiddenTokens {
					if token != "opus" {
						kept = append(kept, token)
					}
				}
				tbl.ModelValidation.ForbiddenTokens = kept
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tbl := cloneDefaultTable(t)
			tc.mutate(&tbl)
			defer func() {
				if recover() == nil {
					t.Errorf("validateTable accepted invalid modelValidation policy")
				}
			}()
			validateTable(tbl)
		})
	}
}

func TestValidateTableModelValidationRejectsUnknownJSONMembers(t *testing.T) {
	_, err := decodeTable([]byte(`{
		"harnesses": {},
		"modelValidation": {
			"idPattern": "^[a-z]+$",
			"forbiddenTokens": ["auto"],
			"forbiddenPatterns": [{"name": "selector", "re": "^auto$", "unknown": true}]
		}
	}`))
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("decodeTable unknown policy member error = %v, want strict rejection", err)
	}
}

func TestDecodeTableRejectsDuplicateJSONMembersRecursively(t *testing.T) {
	raw, err := models.FS.ReadFile("defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, old, replacement string
	}{
		{
			name:        "root modelValidation",
			old:         `"modelValidation": {`,
			replacement: `"modelValidation": {}, "modelValidation": {`,
		},
		{
			name:        "modelValidation idPattern",
			old:         `"idPattern": "^[A-Za-z0-9][A-Za-z0-9._/:+-]{0,127}$",`,
			replacement: `"idPattern": "shadow", "idPattern": "^[A-Za-z0-9][A-Za-z0-9._/:+-]{0,127}$",`,
		},
		{
			name:        "modelValidation forbiddenTokens",
			old:         `"forbiddenTokens": [`,
			replacement: `"forbiddenTokens": [], "forbiddenTokens": [`,
		},
		{
			name:        "pattern name",
			old:         `{"name": "generic-selector",`,
			replacement: `{"name": "shadow", "name": "generic-selector",`,
		},
		{
			name:        "harness member",
			old:         `"claude": {`,
			replacement: `"claude": {}, "claude": {`,
		},
		{
			name:        "tier member",
			old:         `"primary": {`,
			replacement: `"primary": {}, "primary": {`,
		},
		{
			name:        "entry id",
			old:         `"id": "claude-fable-5",`,
			replacement: `"id": "shadow", "id": "claude-fable-5",`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := strings.Replace(string(raw), tc.old, tc.replacement, 1)
			if mutated == string(raw) {
				t.Fatalf("mutation anchor %q was not found", tc.old)
			}
			if _, err := decodeTable([]byte(mutated)); err == nil || !strings.Contains(err.Error(), "duplicate JSON member") {
				t.Fatalf("decodeTable error = %v, want recursive duplicate-member rejection", err)
			}
		})
	}
}

// A generation-14 defaults file writes harnesses, while the one-release
// compatibility reader still accepts the legacy flavors spelling.  A later
// removal of either branch, or an ambiguous input that silently wins by JSON
// decoder order, must fail this boundary test.
func TestDecodeTableAcceptsOneHarnessesOrLegacyFlavorsObject(t *testing.T) {
	raw, err := models.FS.ReadFile("defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	canonical := raw
	if _, err := decodeTable(canonical); err != nil {
		t.Fatalf("decodeTable canonical harnesses: %v", err)
	}
	legacy := []byte(strings.Replace(string(raw), `"harnesses":`, `"flavors":`, 1))
	if _, err := decodeTable(legacy); err != nil {
		t.Fatalf("decodeTable legacy flavors: %v", err)
	}
	for _, input := range [][]byte{
		[]byte(`{"harnesses":{},"flavors":{}}`),
		[]byte(`{"harnesses":{},"flavors":null}`),
		[]byte(`{"harnesses":null,"flavors":{}}`),
		[]byte(`{"harnesses":null}`),
		[]byte(`{"flavors":null}`),
		[]byte(`{}`),
		[]byte(`{"Harnesses":{}}`),
		[]byte(`{"harnesses":{},"Harnesses":{}}`),
	} {
		if _, err := decodeTable(input); err == nil {
			t.Fatalf("decodeTable(%s) accepted a missing, null, ambiguous, or case-variant model table", input)
		}
	}
}

const testMaxTemplateVersion = 12

func validateWorkflow(t *testing.T, content, harness, tier, expected string) (Entry, error) {
	t.Helper()
	return ValidateLaunch(LaunchRequest{
		RepoDir:            writeWorkflow(t, content),
		Harness:            harness,
		Tier:               tier,
		Expected:           expected,
		MaxTemplateVersion: testMaxTemplateVersion,
	})
}

func requireLaunchRefusal(t *testing.T, err error, reason LaunchReason, key, value, rule string) *LaunchRefusal {
	t.Helper()
	var refusal *LaunchRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("error = %T %v, want *LaunchRefusal", err, err)
	}
	if refusal.Reason != reason || refusal.Key != key || refusal.Value != value || refusal.Rule != rule {
		t.Fatalf("refusal = %#v, want reason=%q key=%q value=%q rule=%q", refusal, reason, key, value, rule)
	}
	if strings.Contains(refusal.Error(), "\n") || !strings.Contains(refusal.Error(), strconv.Quote(value)) {
		t.Fatalf("refusal Error() = %q, want one-line quoted value", refusal.Error())
	}
	return refusal
}

func requireConfigurationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("validation succeeded, want configuration error")
	}
	var refusal *LaunchRefusal
	if errors.As(err, &refusal) {
		t.Fatalf("error = %#v, want ordinary configuration error", refusal)
	}
}

func TestValidateLaunchPositiveRoutes(t *testing.T) {
	t.Run("embedded default without repo", func(t *testing.T) {
		entry, err := ValidateLaunch(LaunchRequest{Harness: "codex", Tier: "primary", MaxTemplateVersion: testMaxTemplateVersion})
		if err != nil {
			t.Fatal(err)
		}
		if entry.ID != "gpt-5.6-sol" || entry.Provenance != Default {
			t.Fatalf("entry = %#v, want embedded default", entry)
		}
	})

	t.Run("absent workflow", func(t *testing.T) {
		entry, err := ValidateLaunch(LaunchRequest{RepoDir: t.TempDir(), Harness: "codex", Tier: "routine", MaxTemplateVersion: testMaxTemplateVersion})
		if err != nil || entry.ID != "gpt-5.6-terra" || entry.Provenance != Default {
			t.Fatalf("entry=%#v err=%v", entry, err)
		}
	})

	cases := []struct {
		name, content, harness, tier, wantID string
		wantProvenance                       Provenance
	}{
		{"current dotted", "template_version: 12\nmodel_routing:\n  codex.primary: gpt-5.6-sol @ xhigh\n", "codex", "primary", "gpt-5.6-sol", Inherited},
		{"current legacy bare", "model_routing:\n  routine: claude-opus-5 @ low\n", "claude", "routine", "claude-opus-5", Inherited},
		{"current id changed effort", "model_routing:\n  claude.routine: claude-opus-5 @ xhigh\n", "claude", "routine", "claude-opus-5", Override},
		{"custom dotted override", "model_routing:\n  codex.primary: bespoke-safe\n", "codex", "primary", "bespoke-safe", Override},
		{"custom legacy bare override", "model_routing:\n  primary: claude-bespoke-safe\n", "claude", "primary", "claude-bespoke-safe", Override},
		{"auto substring negative control", "model_routing:\n  codex.primary: automatic-model\n", "codex", "primary", "automatic-model", Override},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := validateWorkflow(t, tc.content, tc.harness, tc.tier, "")
			if err != nil {
				t.Fatal(err)
			}
			if entry.ID != tc.wantID || entry.Provenance != tc.wantProvenance {
				t.Fatalf("entry = %#v, want id=%q provenance=%q", entry, tc.wantID, tc.wantProvenance)
			}
		})
	}
}

func TestActiveIDMatchesUsesByteEquality(t *testing.T) {
	if !ActiveIDMatches("gpt-5.6-sol", "gpt-5.6-sol") {
		t.Fatal("exact active id did not match")
	}
	for _, candidate := range []string{" gpt-5.6-sol", "gpt-5.6-sol ", "GPT-5.6-SOL", "sol"} {
		if ActiveIDMatches("gpt-5.6-sol", candidate) {
			t.Errorf("candidate %q matched exact active id", candidate)
		}
	}
}

func TestValidateHostPinForLaunchPreservesAuditInvariant(t *testing.T) {
	if err := ValidateHostPinForLaunch("codex.primary", "gpt-5.6-sol", "gpt-5.6-sol"); err != nil {
		t.Fatalf("byte-identical host pin: %v", err)
	}
	err := ValidateHostPinForLaunch("codex.primary", "gpt-5.6-sol", "bespoke-host-safe")
	if err == nil {
		t.Fatal("divergent host pin validated before I074")
	}
	if !strings.Contains(err.Error(), "not auditable until I074") || strings.Contains(err.Error(), "bespoke-host-safe") {
		t.Fatalf("divergent host pin error = %q, want redacted pin and I074 gate", err)
	}
	var refusal *LaunchRefusal
	if errors.As(err, &refusal) {
		t.Fatalf("divergent host pin error = %#v, want exit-2 configuration error seam", refusal)
	}
}

func TestResolveForHostPreservesPreferenceTrailAndRequiresExactReachability(t *testing.T) {
	repo := writeWorkflow(t, "model_routing:\n  codex.primary: repository-safe @ xhigh\n")
	configPath := writeHostConfig(t, `{
  "schema_version": 1, "host_id": "test-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"host-safe": {"efforts": ["high"]}, "repository-safe": {"efforts": ["xhigh"]}}}
  }, "pins": {"codex.primary": {"model": "host-safe", "effort": "high", "evidence_refs": ["owner:I068"]}}
}`)
	resolution, err := ResolveForHost(repo, configPath, "codex", "primary", func(string) (string, error) { return "/bin/codex", nil })
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Entry.ID != "host-safe" || resolution.Entry.Effort != "high" {
		t.Fatalf("final entry = %#v", resolution.Entry)
	}
	if resolution.Requested.ID != "repository-safe" || resolution.Requested.Provenance != Override {
		t.Fatalf("requested entry = %#v", resolution.Requested)
	}
	if resolution.Host.Status != HostPinned || resolution.Host.ID != "test-host" || resolution.Pin == nil || resolution.Pin.Model != "host-safe" {
		t.Fatalf("host trail = %#v pin = %#v", resolution.Host, resolution.Pin)
	}

	unreachablePath := writeHostConfig(t, `{
  "schema_version": 1, "host_id": "test-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"other": {"efforts": ["high"]}}}
  }, "pins": {}}
`)
	_, err = ResolveForHost(repo, unreachablePath, "codex", "primary", func(string) (string, error) { return "/bin/codex", nil })
	if err == nil || !strings.Contains(err.Error(), "not reachable") {
		t.Fatalf("unreachable preference error = %v", err)
	}
}

func TestResolveForHostAbsentConfigIsRepositoryCompatible(t *testing.T) {
	repo := writeWorkflow(t, "model_routing:\n  codex.primary: repository-safe @ xhigh\n")
	want, err := Resolve(repo, "codex", "primary")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveForHost(repo, filepath.Join(t.TempDir(), "absent.json"), "codex", "primary", func(string) (string, error) { t.Fatal("lookup called without config"); return "", nil })
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Entry, want) || !reflect.DeepEqual(got.Requested, want) || got.Host.Status != HostUnconfigured {
		t.Fatalf("host resolution = %#v, want compatible %#v", got, want)
	}
}

func TestValidateLaunchForHostRejectsDivergentPinButPermitsIdenticalPin(t *testing.T) {
	repo := writeWorkflow(t, "template_version: 12\nmodel_routing:\n  codex.primary: repository-safe\n")
	request := LaunchRequest{RepoDir: repo, Harness: "codex", Tier: "primary", MaxTemplateVersion: testMaxTemplateVersion}
	divergent := writeHostConfig(t, `{
  "schema_version": 1, "host_id": "test-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"repository-safe": {"efforts": ["high"]}, "host-safe": {"efforts": ["high"]}}}
  }, "pins": {"codex.primary": {"model": "host-safe", "effort": "high"}}
}`)
	_, err := ValidateLaunchForHost(request, divergent, func(string) (string, error) { return "/bin/codex", nil })
	if err == nil || !strings.Contains(err.Error(), "not auditable until I074") {
		t.Fatalf("divergent host pin error = %v", err)
	}
	identical := writeHostConfig(t, strings.Replace(string(mustReadFile(t, divergent)), `"model": "host-safe"`, `"model": "repository-safe"`, 1))
	resolution, err := ValidateLaunchForHost(request, identical, func(string) (string, error) { return "/bin/codex", nil })
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Entry.ID != "repository-safe" || resolution.Host.Status != HostPinned {
		t.Fatalf("identical pin resolution = %#v", resolution)
	}
}

func writeHostConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routing-host.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestParseLaunchRoutingRejectsGlobalAmbiguity(t *testing.T) {
	cases := []struct {
		name, content string
	}{
		{"empty template version", "template_version:\n"},
		{"duplicate template version", "template_version: 11\ntemplate_version: 12\n"},
		{"malformed template version", "template_version: twelve\n"},
		{"non decimal template version", "template_version: +12\n"},
		{"newer template version", "template_version: 14\n"},
		{"duplicate routing blocks", "model_routing:\n  codex.primary: first\nmodel_routing:\n  codex.primary: second\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseLaunchRouting(tc.content, testMaxTemplateVersion)
			requireConfigurationError(t, err)
		})
	}
}

func TestValidateLaunchRejectsStrictRequestedInput(t *testing.T) {
	cases := []struct {
		name, content, harness, tier string
	}{
		{"duplicate requested dotted key", "model_routing:\n  codex.primary: one\n  codex.primary: two\n", "codex", "primary"},
		{"duplicate selected bare key", "model_routing:\n  primary: one\n  primary: two\n", "claude", "primary"},
		{"missing colon", "model_routing:\n  codex.primary gpt-5.6-sol\n", "codex", "primary"},
		{"empty id", "model_routing:\n  codex.primary:\n", "codex", "primary"},
		{"multiple model ids", "model_routing:\n  codex.primary: one two\n", "codex", "primary"},
		{"repeated effort separator", "model_routing:\n  codex.primary: one @ high @ low\n", "codex", "primary"},
		{"malformed alternate", "model_routing:\n  codex.primary: one alt:\n", "codex", "primary"},
		{"invalid pi effort vocabulary", "model_routing:\n  pi.routine: qwen3.8-27b-q8_0 @ high\n", "pi", "routine"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateWorkflow(t, tc.content, tc.harness, tc.tier, "")
			requireConfigurationError(t, err)
		})
	}

	t.Run("unreadable present input", func(t *testing.T) {
		repoFile := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(repoFile, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := ValidateLaunch(LaunchRequest{RepoDir: repoFile, Harness: "codex", Tier: "primary", MaxTemplateVersion: testMaxTemplateVersion})
		requireConfigurationError(t, err)
	})
}

func TestValidateLaunchRequestedKeyIsolationAndClaudePrecedence(t *testing.T) {
	content := "model_routing:\n  claude.primary: claude-dotted-safe\n  primary: claude-bare-safe\n  codex.routine broken unrelated row\n"
	entry, err := validateWorkflow(t, content, "claude", "primary", "")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "claude-dotted-safe" {
		t.Fatalf("entry.ID = %q, want dotted row to win", entry.ID)
	}
}

func TestValidateLaunchPreservesStrictRoutingSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, content, harness, tier string
	}{
		{
			name:    "malformed routing header",
			content: "model_routing: bogus\n  codex.primary: bespoke-safe\n",
			harness: "codex",
			tier:    "primary",
		},
		{
			name:    "whitespace before requested key colon",
			content: "model_routing:\n  codex.primary : bespoke-safe\n",
			harness: "codex",
			tier:    "primary",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateWorkflow(t, tc.content, tc.harness, tc.tier, "")
			requireConfigurationError(t, err)
		})
	}

	for _, tc := range []struct {
		name, content, wantID string
	}{
		{
			name:    "commented header and row",
			content: "model_routing: # routes\n  codex.primary: comment-safe # selected\n",
			wantID:  "comment-safe",
		},
		{
			name:    "dotted wins duplicate shadowed bare",
			content: "model_routing:\n  claude.primary: dotted-safe\n  primary: first-bare\n  primary: second-bare\n",
			wantID:  "dotted-safe",
		},
		{
			name:    "dotted wins malformed shadowed bare",
			content: "model_routing:\n  claude.primary: dotted-safe\n  primary malformed-bare\n",
			wantID:  "dotted-safe",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			harness := "codex"
			if strings.HasPrefix(tc.wantID, "dotted") {
				harness = "claude"
			}
			entry, err := validateWorkflow(t, tc.content, harness, "primary", "")
			if err != nil {
				t.Fatal(err)
			}
			if entry.ID != tc.wantID {
				t.Fatalf("entry.ID = %q, want %q", entry.ID, tc.wantID)
			}
		})
	}
}

func TestValidateLaunchUsesOnlyExactAlternateDelimiter(t *testing.T) {
	for _, id := range []string{"salt:model", "vault:autoencoder"} {
		t.Run(id, func(t *testing.T) {
			entry, err := validateWorkflow(t, "model_routing:\n  codex.primary: "+id+"\n", "codex", "primary", "")
			if err != nil {
				t.Fatal(err)
			}
			if entry.ID != id {
				t.Fatalf("entry.ID = %q, want %q", entry.ID, id)
			}
		})
	}

	for _, tc := range []struct {
		name, value string
	}{
		{"repeated delimiter", "one alt: two alt: three"},
		{"empty alternate", "one alt:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateWorkflow(t, "model_routing:\n  codex.primary: "+tc.value+"\n", "codex", "primary", "")
			requireConfigurationError(t, err)
		})
	}
}

func TestValidateLaunchClassifiesHarnessHistoryCurrentFirst(t *testing.T) {
	t.Run("cross-cell history is retired", func(t *testing.T) {
		_, err := validateWorkflow(t, "model_routing:\n  claude.primary: claude-sonnet-5\n", "claude", "primary", "")
		requireLaunchRefusal(t, err, ReasonRetiredModel, "claude.primary", "claude-sonnet-5", "")
	})

	t.Run("current ID wins over history elsewhere", func(t *testing.T) {
		tbl := cloneDefaultTable(t)
		primary := tbl.Harnesses["claude"]["primary"]
		primary.History = append(primary.History, historyEntry{ID: "claude-opus-5"})
		tbl.Harnesses["claude"]["primary"] = primary
		validateTable(tbl)
		snap, err := parseLaunchRouting("model_routing:\n  claude.primary: claude-opus-5\n", testMaxTemplateVersion)
		if err != nil {
			t.Fatal(err)
		}
		entry, err := validateLaunchFrom(tbl, snap, "claude", "primary", "")
		if err != nil {
			t.Fatal(err)
		}
		if entry.ID != "claude-opus-5" {
			t.Fatalf("entry.ID = %q, want current harness ID", entry.ID)
		}
	})
}

func TestValidateLaunchRefusesSelectedPolicyViolations(t *testing.T) {
	for _, tc := range []struct {
		name, content, harness, tier, value string
		reason                              LaunchReason
		rule                                string
	}{
		{"historical dotted", "model_routing:\n  claude.routine: claude-sonnet-5\n", "claude", "routine", "claude-sonnet-5", ReasonRetiredModel, ""},
		{"historical changed effort", "model_routing:\n  claude.routine: claude-sonnet-5 @ xhigh\n", "claude", "routine", "claude-sonnet-5", ReasonRetiredModel, ""},
		{"historical bare", "model_routing:\n  fallback: claude-opus-4-8\n", "claude", "fallback", "claude-opus-4-8", ReasonRetiredModel, ""},
		{"unsafe custom", "model_routing:\n  codex.primary: bad;id\n", "codex", "primary", "bad;id", ReasonInvalidModelID, ""},
		{"generic selector pattern", "model_routing:\n  codex.primary: AUTO\n", "codex", "primary", "AUTO", ReasonForbiddenModel, "generic-selector"},
		{"bare family pattern", "model_routing:\n  codex.primary: OPUS\n", "codex", "primary", "OPUS", ReasonForbiddenModel, "bare-family"},
		{"vendor auto pattern", "model_routing:\n  codex.primary: vendor-AUTO-model\n", "codex", "primary", "vendor-AUTO-model", ReasonForbiddenModel, "vendor-auto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateWorkflow(t, tc.content, tc.harness, tc.tier, "")
			requireLaunchRefusal(t, err, tc.reason, tc.harness+"."+tc.tier, tc.value, tc.rule)
		})
	}

	for _, token := range defaults.ModelValidation.ForbiddenTokens {
		t.Run("exact token "+token, func(t *testing.T) {
			_, err := validateWorkflow(t, "model_routing:\n  codex.primary: "+token+"\n", "codex", "primary", "")
			requireLaunchRefusal(t, err, ReasonForbiddenModel, "codex.primary", token, "token:"+token)
		})
	}
}

func TestValidateLaunchExpectedCandidateClassification(t *testing.T) {
	cases := []struct {
		name, harness, tier, candidate string
		reason                         LaunchReason
		rule, detail                   string
	}{
		{"syntax before deny", "codex", "primary", "auto ", ReasonInvalidModelID, "", ""},
		{"shorthand alias", "claude", "routine", "opus", ReasonForbiddenModel, "token:opus", ""},
		{"case difference", "codex", "primary", "GPT-5.6-SOL", ReasonUnmappedDispatch, "", ""},
		{"other active tier", "codex", "primary", "gpt-5.6-terra", ReasonRouteMismatch, "", "codex.routine"},
		{"historical", "claude", "primary", "claude-sonnet-5", ReasonRetiredModel, "", ""},
		{"safe unknown", "codex", "primary", "bespoke-safe", ReasonUnmappedDispatch, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateLaunch(LaunchRequest{Harness: tc.harness, Tier: tc.tier, Expected: tc.candidate, MaxTemplateVersion: testMaxTemplateVersion})
			refusal := requireLaunchRefusal(t, err, tc.reason, tc.harness+"."+tc.tier, tc.candidate, tc.rule)
			if refusal.Detail != tc.detail {
				t.Fatalf("Detail = %q, want %q", refusal.Detail, tc.detail)
			}
		})
	}

	entry, err := ValidateLaunch(LaunchRequest{Harness: "codex", Tier: "primary", Expected: "gpt-5.6-sol", MaxTemplateVersion: testMaxTemplateVersion})
	if err != nil || entry.ID != "gpt-5.6-sol" {
		t.Fatalf("exact expected entry=%#v err=%v", entry, err)
	}

	for _, tier := range Tiers {
		entry, err := ValidateLaunch(LaunchRequest{Harness: "pi", Tier: tier, Expected: "qwen3.8-27b-q8_0", MaxTemplateVersion: testMaxTemplateVersion})
		if err != nil || entry.ID != "qwen3.8-27b-q8_0" {
			t.Errorf("shared pi id for tier %s: entry=%#v err=%v", tier, entry, err)
		}
	}
}

func TestValidateLaunchExpectedRejectsUnsafeIDSyntax(t *testing.T) {
	for _, candidate := range []string{
		strings.Repeat("a", 129),
		" leading", "trailing ", "two words", "line\nbreak", "tab\tbyte",
		`quoted"id`, "back`tick", "$dollar", "$(subshell)", "semi;colon",
		`back\slash`, "pipe|id", "amp&id", "left<id", "right>id", "(group)",
	} {
		t.Run(strconv.Quote(candidate), func(t *testing.T) {
			_, err := ValidateLaunch(LaunchRequest{Harness: "codex", Tier: "primary", Expected: candidate, MaxTemplateVersion: testMaxTemplateVersion})
			requireLaunchRefusal(t, err, ReasonInvalidModelID, "codex.primary", candidate, "")
		})
	}
}

func TestValidateLaunchExpectedUsesSameSnapshotForOtherTiers(t *testing.T) {
	content := "model_routing:\n  codex.primary: requested-active\n  codex.routine: other-active\n"
	_, err := validateWorkflow(t, content, "codex", "primary", "other-active")
	refusal := requireLaunchRefusal(t, err, ReasonRouteMismatch, "codex.primary", "other-active", "")
	if refusal.Detail != "codex.routine" {
		t.Fatalf("Detail = %q, want codex.routine", refusal.Detail)
	}

	content = "model_routing:\n  codex.primary: requested-active\n  codex.routine: malformed @ high @ low\n"
	_, err = validateWorkflow(t, content, "codex", "primary", "malformed")
	requireLaunchRefusal(t, err, ReasonUnmappedDispatch, "codex.primary", "malformed", "")
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

// I036 (D8): MirrorRows renders one row per (harness, tier) of the embedded
// table — harnesses sorted, tiers in Tiers order — and every row round-trips
// through Resolve: written to a WORKFLOW.md, each row resolves back to the
// table's own (id, effort), so the rendered mirror and the reader can never
// disagree on the emitted format.
func TestMirrorRows_CoverEveryHarnessTierAndRoundTrip(t *testing.T) {
	rows := MirrorRows()
	if want := len(Harnesses()) * len(Tiers); len(rows) != want {
		t.Fatalf("MirrorRows() = %d rows, want %d (every harness x tier)", len(rows), want)
	}
	// Assert the row's content, not its column alignment: the key column is
	// padded to the longest harness.tier key, so adding a harness with a longer
	// name reflows every row (I110's "openweights" did exactly that). The
	// alignment itself is covered by the round-trip below, which is what
	// actually has to hold.
	var foundClaudeRoutine bool
	for _, row := range rows {
		if strings.Join(strings.Fields(row), " ") == "claude.routine: claude-opus-5 @ low" {
			foundClaudeRoutine = true
			break
		}
	}
	if !foundClaudeRoutine {
		t.Errorf("MirrorRows() = %q, want explicit low-effort Claude routine row", rows)
	}
	content := "model_routing:\n"
	for _, row := range rows {
		content += row + "\n"
	}
	dir := writeWorkflow(t, content)
	i := 0
	for _, harness := range Harnesses() {
		for _, tier := range Tiers {
			key := harness + "." + tier + ":"
			if !strings.Contains(rows[i], key) {
				t.Errorf("rows[%d] = %q, want key %q (harness-sorted, tier-fixed order)", i, rows[i], key)
			}
			i++
			def, err := Resolve("", harness, tier)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Resolve(dir, harness, tier)
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != def.ID || got.Effort != def.Effort {
				t.Errorf("round-trip %s.%s = {%s, %s}, want {%s, %s}", harness, tier, got.ID, got.Effort, def.ID, def.Effort)
			}
			// A rendered default read back from disk reports Inherited, the
			// provenance the refresh rule keys on.
			if got.Provenance != Inherited {
				t.Errorf("round-trip %s.%s provenance = %s, want %s", harness, tier, got.Provenance, Inherited)
			}
		}
	}
}
