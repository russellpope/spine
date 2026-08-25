package update

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/model"
	"github.com/russellpope/spine/templates"
)

// gen10ContentLines are the emitted-content changes gen 10 ships (I036/I060,
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
		// I060 replaces the manual handoff-copy instruction with the
		// sole-writer and automatic-embed rules. The cursor grammar itself is
		// deliberately unchanged.
		"**Handoff rule:** `/handoff` and any resume/kickoff prompt MUST embed the verbatim output of `spine cursor` — a prose paraphrase of stage state is incomplete; the reader can't see which upstream stage was skipped from a summary alone. Alongside `spine audit stages` blocking on a missing/stale cursor block in the newest handoff, `spine doctor` advises (warns) on the same condition.": true,
		"**Sole-writer rule:** `spine` is the only legal cursor writer. Mutate the block only with `spine cursor start`, `spine cursor tick <stage>`, `spine cursor here <stage>`, or `spine cursor set`; hand-editing it is a workflow violation.":                                                                                                                                                       true,
		"**Handoff rule:** `spine handoff new` automatically embeds the current cursor block in the handoff it creates; do not copy `spine cursor` output by hand. Alongside `spine audit stages` blocking on a missing/stale cursor block in the newest handoff, `spine doctor` advises (warns) on the same condition.":                                                                                  true,
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

// The model_routing mirror pads its key column to the longest flavor.tier key
// in the table, so adding a flavor with a longer name reflows every row —
// I110's "openweights" did exactly that and broke six tests that had the old
// column widths baked into string literals. The two helpers below match and
// rewrite rows by content instead, so the next flavor cannot repeat it.

// replaceRow rewrites the value of the model_routing row for key, whatever
// padding the fixture happens to carry. The single space it writes is a legal
// override form the reader already accepts.
func replaceRow(t *testing.T, content, key, value string) string {
	t.Helper()
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == key+":" {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			lines[i] = indent + key + ": " + value
			return strings.Join(lines, "\n")
		}
	}
	t.Fatalf("no model_routing row for %q to replace in:\n%s", key, content)
	return ""
}

// hasRow reports whether content carries a model_routing row for key with
// exactly value, ignoring column padding. content may be a file body or a
// unified diff, so a leading +/- marker is stripped before matching.
func hasRow(content, key, value string) bool {
	want := key + ": " + value
	for _, line := range strings.Split(content, "\n") {
		if normalizeRow(line) == want {
			return true
		}
	}
	return false
}

// normalizeRow collapses a line's column padding and drops any unified-diff
// marker, so two renderings of the same row compare equal.
func normalizeRow(line string) string {
	body := strings.TrimLeft(strings.TrimSpace(line), "+-")
	return strings.Join(strings.Fields(body), " ")
}

// mirrorRowKey returns the flavor.tier key of a model_routing mirror row
// carried in a diff line. It insists on a known flavor and a known tier so an
// unrelated dotted key cannot be waved through as mirror rendering.
func mirrorRowKey(line string) (string, bool) {
	fields := strings.Fields(strings.TrimLeft(line, "+-"))
	if len(fields) < 2 || !strings.HasSuffix(fields[0], ":") {
		return "", false
	}
	key := strings.TrimSuffix(fields[0], ":")
	flavor, tier, ok := strings.Cut(key, ".")
	if !ok {
		return "", false
	}
	if !slices.Contains(model.Flavors(), flavor) || !slices.Contains(model.Tiers, tier) {
		return "", false
	}
	return key, true
}

// mirrorRenderDiff classifies the mirror rows in a diff into the two changes
// that are pure rendering rather than content (I110): rows whose body is
// unchanged once padding is collapsed — the reflow every existing row
// undergoes when a longer flavor name widens the key column — and rows for a
// key that the diff only adds, which is a new (flavor, tier) the table now
// ships (design D8). A row whose value genuinely changed appears on both sides
// with different bodies and is deliberately left unsanctioned, so it still has
// to be an itemized model refresh.
func mirrorRenderDiff(diff string) func(string) bool {
	removedBody, addedBody := map[string]bool{}, map[string]bool{}
	removedKey, addedKey := map[string]bool{}, map[string]bool{}
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		key, ok := mirrorRowKey(line)
		if !ok {
			continue
		}
		body := strings.Join(strings.Fields(strings.TrimLeft(line, "+-")), " ")
		switch {
		case strings.HasPrefix(line, "-"):
			removedBody[body], removedKey[key] = true, true
		case strings.HasPrefix(line, "+"):
			addedBody[body], addedKey[key] = true, true
		}
	}
	return func(line string) bool {
		key, ok := mirrorRowKey(line)
		if !ok {
			return false
		}
		body := strings.Join(strings.Fields(strings.TrimLeft(line, "+-")), " ")
		if removedBody[body] && addedBody[body] {
			return true // reflow: same content, new column width
		}
		return strings.HasPrefix(line, "+") && !removedKey[key]
	}
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
				if isGen11ContentDiffLine(line) { // gen 11's conscious content edit; see gen10to11_test.go
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
	// Both stale inherited Claude pairs are refreshed and itemized under their
	// flavor-qualified keys.
	wf := report(t, reports, "WORKFLOW.md")
	if len(wf.ModelRefreshes) != 2 {
		t.Fatalf("ModelRefreshes = %+v, want routine and fallback refreshes", wf.ModelRefreshes)
	}
	routine := wf.ModelRefreshes[0]
	if routine.Key != "model_routing.claude.routine" || routine.Old != "claude-sonnet-5" || routine.New != "claude-opus-5 @ low" {
		t.Errorf("routine refresh = %+v, want {model_routing.claude.routine claude-sonnet-5 claude-opus-5 @ low}", routine)
	}
	m := wf.ModelRefreshes[1]
	if m.Key != "model_routing.claude.fallback" || m.Old != "claude-opus-4-8" || m.New != "claude-opus-5" {
		t.Errorf("refresh item = %+v, want {model_routing.claude.fallback claude-opus-4-8 claude-opus-5}", m)
	}
	if len(wf.ModelOverrides) != 0 {
		t.Errorf("pristine fixture reported overrides: %+v", wf.ModelOverrides)
	}
}

// AC (I036/I060): the written migration stamps generation 10, renders every
// flavor and tier as dotted mirror rows, retires the top-level effort: and
// model_default: keys, leaves the cursor grammar untouched (D17), replaces
// the manual handoff-copy rule with the sole-writer and automatic-embed rules,
// and is idempotent.
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
	if !strings.Contains(gotStr, "template_version: 11") {
		t.Error("migrated WORKFLOW.md missing template_version: 11")
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
	for _, want := range []string{
		"**Sole-writer rule:** `spine` is the only legal cursor writer.",
		"`spine cursor start`",
		"`spine cursor tick <stage>`",
		"`spine cursor here <stage>`",
		"`spine cursor set`",
		"`spine handoff new` automatically embeds the current cursor block",
	} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("migrated WORKFLOW.md missing I060 rule text %q", want)
		}
	}
	const oldHandoffRule = "MUST embed the verbatim output of `spine cursor`"
	if strings.Contains(gotStr, oldHandoffRule) {
		t.Errorf("migrated WORKFLOW.md retained the superseded handoff rule %q", oldHandoffRule)
	}
	if n := strings.Count(gotStr, "**Handoff rule:**"); n != 1 {
		t.Errorf("migrated WORKFLOW.md has %d handoff rules, want exactly one replacement", n)
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

// AC (I060): a gen-9 repo's deliberate surrounding workflow choices remain
// intact through the content-bearing migration. This models the customized
// profile configuration found around the machine-owned stage-cursor section;
// it must update with plain --write, never require --force.
func TestGen9To10PreservesCustomizedSurroundingConfiguration(t *testing.T) {
	dir := stageGen9Repo(t, func(content string) string {
		content = mustReplace(t, content,
			"reviewers: [go-reviewer, python-reviewer]",
			"reviewers: [go-reviewer, security-review]")
		return mustReplace(t, content,
			"functional_harness: cli",
			"functional_harness: rest")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) != 0 {
		t.Fatalf("customized gen-9 repo must update cleanly: state=%v unrecognized=%v", wf.State, wf.Unrecognized)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)
	for _, want := range []string{
		"reviewers: [go-reviewer, security-review]",
		"functional_harness: rest",
		"**Sole-writer rule:** `spine` is the only legal cursor writer.",
		"`spine handoff new` automatically embeds the current cursor block",
	} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("updated customized WORKFLOW.md missing %q", want)
		}
	}
	if strings.Contains(gotStr, "MUST embed the verbatim output of `spine cursor`") {
		t.Error("updated customized WORKFLOW.md retained the superseded handoff rule")
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
		if !o.Migrated {
			t.Errorf("ModelOverrides[%s].Migrated = false on the migration run, want true (review Important 2)", o.Key)
		}
	}
	for _, tier := range model.Tiers {
		key := "model_routing.claude." + tier
		if !strings.HasSuffix(migrated[key], " @ xhigh") {
			t.Errorf("ModelOverrides[%s] = %q, want an ' @ xhigh' per-entry override", key, migrated[key])
		}
	}
	// The stale fallback still refreshes (id) before gaining the effort.
	if len(wf.ModelRefreshes) != 2 || wf.ModelRefreshes[0].Key != "model_routing.claude.routine" || wf.ModelRefreshes[1].Key != "model_routing.claude.fallback" {
		t.Errorf("ModelRefreshes = %+v, want routine and fallback refreshes alongside the effort migration", wf.ModelRefreshes)
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
		"claude.routine: claude-opus-5 @ xhigh",
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
	for _, o := range wf.ModelOverrides {
		if o.Migrated {
			t.Errorf("second pass ModelOverrides[%s].Migrated = true, want false — the override now pre-exists on disk", o.Key)
		}
	}
}

// A customized legacy medium effort remains a real routine override because
// the current routine pair explicitly ships at low. MirrorValue correctly
// canonicalizes that override to the bare id (medium is the tier default),
// but resolver pair comparison still preserves it as a deliberate choice.
func TestGen9To10LegacyMediumEffortOverridesClaudeRoutineDefault(t *testing.T) {
	dir := stageGen9Repo(t, func(content string) string {
		return mustReplace(t, content,
			"effort: high                       # tier default:",
			"effort: medium                       # tier default:")
	})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending || len(wf.Unrecognized) > 0 {
		t.Fatalf("customized effort must migrate, not skip: state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	minted := map[string]string{}
	for _, o := range wf.ModelOverrides {
		minted[o.Key] = o.Value
	}
	if len(minted) != 4 {
		t.Errorf("ModelOverrides = %+v, want every Claude tier migrated", wf.ModelOverrides)
	}
	if got := minted["model_routing.claude.routine"]; got != "claude-opus-5" {
		t.Errorf("claude.routine migration = %q, want bare claude-opus-5 (medium override)", got)
	}
	got, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)
	for _, want := range []string{
		"claude.primary: claude-fable-5 @ medium",
		"claude.routine: claude-opus-5",
		"claude.mechanical: claude-haiku-4-5 @ medium",
		"claude.fallback: claude-opus-5 @ medium",
	} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("migrated WORKFLOW.md missing per-entry effort override %q", want)
		}
	}
	if strings.Contains(gotStr, "claude.routine: claude-opus-5 @ low") {
		t.Error("claude.routine retained the table's low-effort default instead of the migrated medium override")
	}
	// Idempotence: plan again — the second plan must be clean with no diff.
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf = report(t, reports, "WORKFLOW.md")
	if wf.State != UpToDate || wf.Diff != "" {
		t.Errorf("second plan not clean: state=%v diff:\n%s", wf.State, wf.Diff)
	}
	if len(wf.ModelOverrides) != 4 {
		t.Errorf("second pass ModelOverrides = %+v, want the same four entries — no silent strip between runs", wf.ModelOverrides)
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
	if len(wf.ModelRefreshes) != 1 || wf.ModelRefreshes[0].Key != "model_routing.claude.routine" {
		t.Errorf("ModelRefreshes = %+v, want only the inherited routine refresh", wf.ModelRefreshes)
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
	if strings.Contains(string(got), "claude.fallback:   claude-opus-5") {
		t.Error("fallback override was clobbered by the current default")
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
