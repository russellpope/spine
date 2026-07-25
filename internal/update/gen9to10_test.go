package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/templates"
)

// gen10ContentLines are the emitted-content changes gen 10 ships (I036,
// design D8/D16), both sides of the diff. The removed ("-") side is static —
// those spellings are frozen history: the uncommented block header, the bare
// claude tier rows every gen 6–9 render emitted (both fallback values gen 9
// ever carried, pre- and post-I035), and the retired top-level effort: and
// model_default: keys (gen-0 and gen-1+ spellings). The added ("+") side —
// the dotted flavor-axis mirror rows — is rendered from the model table via
// the same code path the template uses, deliberately NOT pinned as literals:
// pinning current table values here would recreate the coupling I036 removes
// (a defaults change must not require touching this lock).
var gen10ContentLines = func() map[string]bool {
	lines := map[string]bool{
		// the block header gains the spine-managed marker comment (D8).
		"model_routing:": true,
		"model_routing:                     # spine-managed defaults; edit a value to override": true,
		// gen 6–9 bare claude tier rows, superseded by the dotted mirror.
		"primary: claude-fable-5          # default thinker: design, judgment, orchestration, final review":  true,
		"routine: claude-sonnet-5         # multi-step mechanical subagent roles":                            true,
		"mechanical: claude-haiku-4-5     # verbatim plan-transcription + single-file mechanical fixes ONLY": true,
		"fallback: claude-opus-4-8        # primary-refused or security-framed work":                         true,
		"fallback: claude-opus-5          # primary-refused or security-framed work":                         true,
		// the retired top-level keys (D16 + controller ruling).
		"effort: high                       # tier default: primary=high, routine=medium, mechanical=low, fallback=high; xhigh reserved for final verification and security-critical passes; per-ticket effort: only on deviation": true,
		"model_default: claude-fable-5      # swappable; re-evaluate on major model/platform releases":                                                                                                                             true,
		"model_default: claude-opus-4-8     # swappable; re-evaluate on major model/platform releases":                                                                                                                             true,
	}
	for _, row := range model.MirrorRows() {
		lines[strings.TrimSpace(row)] = true
	}
	return lines
}()

// isGen10ContentDiffLine reports whether a unified-diff line carries the
// gen-10 content change above, or is a bare added/removed blank line.
func isGen10ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || gen10ContentLines[body]
}

// stageGen9Repo copies the spine-gen9 capture — spine's own real WORKFLOW.md
// and CLAUDE.md at generation 9, verbatim (ultima-style real-repo fixture):
// bare claude tier keys, the stale pre-I035 fallback claude-opus-4-8, and
// the top-level effort: and model_default: keys every fleet repo carries —
// into a temp dir, with mutate applied to WORKFLOW.md first ("" = pristine).
func stageGen9Repo(t *testing.T, mutate func(string) string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "spine-gen9", name))
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

// mustReplace applies a strings.Replace that must change the content —
// guarding the fixture against silent drift under the mutation helpers.
func mustReplace(t *testing.T, content, old, new string) string {
	t.Helper()
	out := strings.Replace(content, old, new, 1)
	if out == content {
		t.Fatalf("fixture line %q not found to replace", old)
	}
	return out
}

// AC (I036): a captured real generation-9 repo upgrades with only sanctioned
// content-diff lines — the stamp, the declared gen-10 mirror/retirement
// content, and the itemized model refresh — and zero unrecognized lines.
func TestGen9To10PristineUpdatesCleanly(t *testing.T) {
	dir := stageGen9Repo(t, nil)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range reports {
		switch r.Path {
		case "WORKFLOW.md", "CLAUDE.md":
			seen[r.Path] = true
			if len(r.Unrecognized) > 0 {
				t.Errorf("%s: pristine gen-9 lines misread as local edits: %v", r.Path, r.Unrecognized)
			}
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			for _, line := range strings.Split(r.Diff, "\n") {
				if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
					continue
				}
				if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
					continue
				}
				if strings.Contains(line, "template_version") || strings.Contains(line, "spine:begin") {
					continue
				}
				if isGen10ContentDiffLine(line) {
					continue
				}
				if isModelRefreshDiffLine(line) { // sanctioned model-table refresh (I035); see modelrouting_test.go
					continue
				}
				t.Errorf("%s: unexpected changed line %q — 9→10 must be stamp plus declared gen-10 content only", r.Path, line)
			}
		}
	}
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		if !seen[name] {
			t.Errorf("%s: never reported by Run — the lock did not exercise it", name)
		}
	}
	// The stale inherited fallback is refreshed AND itemized (D6), now under
	// its flavor-qualified key.
	wf := report(t, reports, "WORKFLOW.md")
	if len(wf.ModelRefreshes) != 1 {
		t.Fatalf("ModelRefreshes = %+v, want exactly the claude fallback refresh", wf.ModelRefreshes)
	}
	m := wf.ModelRefreshes[0]
	if m.Key != "model_routing.claude.fallback" || m.Old != "claude-opus-4-8" || m.New != "claude-opus-5" {
		t.Errorf("refresh item = %+v, want {model_routing.claude.fallback claude-opus-4-8 claude-opus-5}", m)
	}
	if len(wf.ModelOverrides) != 0 {
		t.Errorf("pristine fixture reported overrides: %+v", wf.ModelOverrides)
	}
}

// AC (I036): the written migration stamps generation 10, renders every
// flavor and tier as dotted mirror rows, retires the top-level effort: and
// model_default: keys, leaves the per-ticket/cursor effort grammar untouched
// (D17), and is idempotent.
func TestGen9To10MigrationWritesFlavorMirror(t *testing.T) {
	dir := stageGen9Repo(t, nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, "template_version: 10") {
		t.Error("migrated WORKFLOW.md missing template_version: 10")
	}
	for _, row := range model.MirrorRows() {
		if !containsLine(gotStr, row) {
			t.Errorf("migrated WORKFLOW.md missing mirror row %q", row)
		}
	}
	for _, line := range splitLines(gotStr) {
		if strings.HasPrefix(line, "effort:") || strings.HasPrefix(line, "model_default:") {
			t.Errorf("retired top-level key survived migration: %q", line)
		}
	}
	// D17: the cursor-grammar effort line and the escalation-record grammar
	// are a different concept from the retired repo-level key.
	for _, keep := range []string{
		"    effort: <kebab-name>",
		"    ESCALATION <ticket-id> effort <from>-><to> reason: <one line>",
	} {
		if !containsLine(gotStr, keep) {
			t.Errorf("per-ticket/cursor effort grammar lost in migration: %q", keep)
		}
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "WORKFLOW.md" || r.Path == "CLAUDE.md" {
			if r.State != UpToDate {
				t.Errorf("second pass %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
			}
		}
	}
}

// AC (I036, D16): a customized top-level effort: value survives as
// per-entry effort overrides on the repo's claude entries — the only flavor
// the generations that rendered the key ever dispatched — rather than being
// discarded, and the migrated entries are surfaced in the plan.
func TestGen9To10CustomEffortMigratesToPerEntryOverrides(t *testing.T) {
	dir := stageGen9Repo(t, func(content string) string {
		return mustReplace(t, content,
			"effort: high                       # tier default:",
			"effort: xhigh                       # tier default:")
	})
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) > 0 {
		t.Fatalf("customized effort must migrate, not skip: state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	migrated := map[string]string{}
	for _, o := range wf.ModelOverrides {
		migrated[o.Key] = o.Value
	}
	for _, tier := range model.Tiers {
		key := "model_routing.claude." + tier
		if !strings.HasSuffix(migrated[key], " @ xhigh") {
			t.Errorf("ModelOverrides[%s] = %q, want an ' @ xhigh' per-entry override", key, migrated[key])
		}
	}
	// The stale fallback still refreshes (id) before gaining the effort.
	if len(wf.ModelRefreshes) != 1 || wf.ModelRefreshes[0].Key != "model_routing.claude.fallback" {
		t.Errorf("ModelRefreshes = %+v, want the claude fallback id refresh alongside the effort migration", wf.ModelRefreshes)
	}

	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)
	for _, want := range []string{
		"claude.primary: claude-fable-5 @ xhigh",
		"claude.routine: claude-sonnet-5 @ xhigh",
		"claude.mechanical: claude-haiku-4-5 @ xhigh",
		"claude.fallback: claude-opus-5 @ xhigh",
	} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("migrated WORKFLOW.md missing per-entry effort override %q", want)
		}
	}
	if strings.Contains(gotStr, "codex.routine: gpt-5.6-terra @") {
		t.Error("effort migration leaked onto a codex entry")
	}
	for _, line := range splitLines(gotStr) {
		if strings.HasPrefix(line, "effort:") {
			t.Errorf("retired effort: key survived migration: %q", line)
		}
	}
	// Idempotence: the migrated per-entry overrides now read as deliberate
	// overrides and are preserved.
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf = report(t, reports, "WORKFLOW.md")
	if wf.State != UpToDate {
		t.Errorf("second pass state=%v diff:\n%s", wf.State, wf.Diff)
	}
	if len(wf.ModelOverrides) != 4 {
		t.Errorf("second pass ModelOverrides = %+v, want the four migrated claude entries reported", wf.ModelOverrides)
	}
}

// AC (I036, controller ruling): a customized model_default: whose value
// diverges from the resolved claude primary is surfaced for a human
// decision — the file is skipped, the divergence named — never silently
// dropped and never silently promoted.
func TestGen9To10DivergentModelDefaultSurfaced(t *testing.T) {
	dir := stageGen9Repo(t, func(content string) string {
		return mustReplace(t, content,
			"model_default: claude-fable-5",
			"model_default: my-pinned-model")
	})
	before, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != SkippedUnrecognized {
		t.Fatalf("divergent model_default must skip the file for a human decision, got state=%v", wf.State)
	}
	named := false
	for _, u := range wf.Unrecognized {
		if strings.Contains(u, "my-pinned-model") && strings.Contains(u, "diverges") {
			named = true
		}
	}
	if !named {
		t.Errorf("skip must name the divergent value, got %v", wf.Unrecognized)
	}
	after, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("divergent model_default was written over instead of left for the owner")
	}
}

// A model_default equal to the repo's own deliberate primary override is a
// duplicate, not a divergence: it retires quietly and the override lands in
// the dotted primary row.
func TestGen9To10ModelDefaultMatchingPrimaryOverrideRetiresQuietly(t *testing.T) {
	dir := stageGen9Repo(t, func(content string) string {
		content = mustReplace(t, content, "  primary: claude-fable-5", "  primary: my-local-model")
		return mustReplace(t, content, "model_default: claude-fable-5", "model_default: my-local-model")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) > 0 {
		t.Fatalf("matching model_default must retire quietly: state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "claude.primary: my-local-model") {
		t.Error("primary override did not land in the dotted mirror row")
	}
	for _, line := range splitLines(string(got)) {
		if strings.HasPrefix(line, "model_default:") {
			t.Errorf("model_default survived retirement: %q", line)
		}
	}
}

// AC (I036): a deliberate model override in the bare gen-9 format is kept
// verbatim through the format change into its dotted row, reported as an
// override — and the written no-effort, no-comment dotted value parses and
// is preserved on the next run (D9's "a value with neither still parses").
func TestGen9To10ModelOverrideKeptThroughFormatChange(t *testing.T) {
	dir := stageGen9Repo(t, func(content string) string {
		return mustReplace(t, content, "fallback: claude-opus-4-8", "fallback: claude-opus-3-pinned")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) > 0 {
		t.Fatalf("override misread: state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	if len(wf.ModelRefreshes) != 0 {
		t.Errorf("override wrongly scheduled for refresh: %+v", wf.ModelRefreshes)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Key != "model_routing.claude.fallback" ||
		wf.ModelOverrides[0].Value != "claude-opus-3-pinned" {
		t.Errorf("ModelOverrides = %+v, want the pinned fallback reported under its dotted key", wf.ModelOverrides)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "claude.fallback: claude-opus-3-pinned") {
		t.Error("deliberate override did not survive into the dotted mirror")
	}
	if strings.Contains(string(got), "claude-opus-5") {
		t.Error("override was clobbered by the current default")
	}
	// Second run: the bare "<id>"-only dotted value (no effort suffix, no
	// comment) parses as the same override and the file is stable.
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf = report(t, reports, "WORKFLOW.md")
	if wf.State != UpToDate {
		t.Errorf("second pass state=%v diff:\n%s", wf.State, wf.Diff)
	}
	if len(wf.ModelOverrides) != 1 || wf.ModelOverrides[0].Value != "claude-opus-3-pinned" {
		t.Errorf("second pass ModelOverrides = %+v, want the preserved pinned fallback", wf.ModelOverrides)
	}
}

// Decoupling proof (I036): the WORKFLOW template carries no model id
// literal — the mirror renders from the model table — so a defaults-file
// change needs no matching template edit (the coupling I035 documented).
func TestWorkflowTemplateCarriesNoModelIDs(t *testing.T) {
	raw, err := templates.FS.ReadFile("current/WORKFLOW.md.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	for _, flavor := range model.Flavors() {
		for _, tier := range model.Tiers {
			e, err := model.Resolve("", flavor, tier)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), e.ID) {
				t.Errorf("template pins model id %q — the mirror must render from the table", e.ID)
			}
		}
	}
}
