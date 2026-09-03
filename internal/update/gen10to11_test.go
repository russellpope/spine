package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/gate"
)

// gen11ContentLines are the emitted-content changes gen 11 ships (I085): the
// three gate-pack keys and the five gate_pack_config sub-key rows. Gen 11
// removes nothing, so every entry is an added ("+") line. Values are empty by
// default — the pack is opt-in (ADR 0015): a repo sets gate_pack: go@1.
var gen11ContentLines = map[string]bool{
	"gate_pack:                         # gate pack rendered into maipipe.toml (go@1); empty = no pack":              true,
	"gate_pack_disabled: []             # check classes dropped from the rendered pipeline":                          true,
	"gate_pack_config:                  # per-check inputs; a non-empty value reaches its stage as SPINE_GATE_<KEY>": true,
	"test_enum_spec:                  # spec file test-enum-vs-spec reads the enumerated values from":                true,
	"fixture_manifest:                # manifest path fixture-manifest requires":                                     true,
	"build_outputs:                   # build output paths gitignore-control requires to be ignored":                 true,
	"n_plus_one_clients:              # client method names n-plus-one looks for in loops":                           true,
	"tskip_allow:                     # test files tskip tolerates a skip in":                                        true,
}

// isGen11ContentDiffLine reports whether a unified-diff line carries the
// gen-11 content change above, or is a bare added/removed blank line.
func isGen11ContentDiffLine(line string) bool {
	if len(line) == 0 || (line[0] != '+' && line[0] != '-') {
		return false
	}
	body := strings.TrimSpace(line[1:])
	return body == "" || gen11ContentLines[body]
}

// stageGen10Repo copies the spine-gen10 capture — spine's own real
// WORKFLOW.md and CLAUDE.md at generation 10, verbatim (the same real-repo
// fixture style as spine-gen9) — into a temp dir, with mutate applied to
// WORKFLOW.md first (nil = pristine).
func stageGen10Repo(t *testing.T, mutate func(string) string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "spine-gen10", name))
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

// AC (I085): a captured real generation-10 repo upgrades with only sanctioned
// content-diff lines — the stamp and the declared gen-11 gate-pack keys — and
// zero unrecognized lines.
func TestGen10To11PristineUpdatesCleanly(t *testing.T) {
	dir := stageGen10Repo(t, nil)
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
				t.Errorf("%s: pristine gen-10 lines misread as local edits: %v", r.Path, r.Unrecognized)
			}
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			isRenderOnly := mirrorRenderDiff(r.Diff)
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
				if isGen11ContentDiffLine(line) {
					continue
				}
				if isGen12ContentDiffLine(line) {
					continue
				}
				if isI114ContentDiffLine(line) { // I114's conscious current-template edit; see cursor_hygiene_test.go
					continue
				}
				if sanctionedRefreshLine(line, r.ModelRefreshes) { // a mirror-row change this report itemizes (I128); see modelrouting_test.go
					continue
				}
				if isRenderOnly(line) { // mirror reflow or a newly shipped (harness, tier); see mirrorRenderDiff
					continue
				}
				t.Errorf("%s: unexpected changed line %q — 10→11 must be stamp plus declared gen-11 content only", r.Path, line)
			}
		}
	}
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		if !seen[name] {
			t.Errorf("%s: never reported by Run — the lock did not exercise it", name)
		}
	}
}

// AC (I085): the written migration stamps generation 11, renders the three
// gate-pack keys with empty (opt-in) defaults, scaffolds the remediation
// README, creates no maipipe.toml — the fleet negative control: a repo
// without gate_pack gets no gate file — and is idempotent.
func TestGen10To11MigrationAddsGatePackKeys(t *testing.T) {
	dir := stageGen10Repo(t, nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, "template_version: 14") {
		t.Error("migrated WORKFLOW.md missing template_version: 14")
	}
	keys := ExtractKeys(got)
	if keys["gate_pack"] != "" {
		t.Errorf("gate_pack = %q, want empty — the pack is opt-in", keys["gate_pack"])
	}
	if keys["gate_pack_disabled"] != "[]" {
		t.Errorf("gate_pack_disabled = %q, want []", keys["gate_pack_disabled"])
	}
	for _, k := range gatePackConfigKeys {
		if v, ok := keys["gate_pack_config."+k]; !ok || v != "" {
			t.Errorf("gate_pack_config.%s = %q ok=%v, want an empty rendered row", k, v, ok)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, MaipipeFile)); !os.IsNotExist(err) {
		t.Errorf("%s exists after an update with no gate_pack (err=%v)", MaipipeFile, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "remediation", "README.md")); err != nil {
		t.Errorf("docs/remediation/README.md not scaffolded by update: %v", err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("second pass %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

// AC (I085): the gate-pack keys are preserved choices — an opted-in repo's
// pack, disabled list and per-check config survive the regeneration that
// refreshes the prose around them (ADR 0002).
func TestGatePackKeysSurviveUpdateAsChoices(t *testing.T) {
	dir := stageGen10Repo(t, nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	optIn(t, dir, "go@1", "[tskip]", map[string]string{"fixture_manifest": "docs/fixtures.md"})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if len(wf.Unrecognized) > 0 {
		t.Errorf("gate-pack choices misread as local edits: %v", wf.Unrecognized)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if err != nil {
		t.Fatal(err)
	}
	keys := ExtractKeys(string(raw))
	if keys["gate_pack"] != "go@1" || keys["gate_pack_disabled"] != "[tskip]" ||
		keys["gate_pack_config.fixture_manifest"] != "docs/fixtures.md" {
		t.Errorf("gate-pack choices lost: %q %q %q", keys["gate_pack"],
			keys["gate_pack_disabled"], keys["gate_pack_config.fixture_manifest"])
	}
}

// optIn rewrites the gate-pack keys of a repo's WORKFLOW.md the way an owner
// would: set gate_pack, drop classes, point checks at files.
func optIn(t *testing.T, dir, pack, disabled string, config map[string]string) {
	t.Helper()
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := setKey(string(raw), "gate_pack", pack)
	content = setKey(content, "gate_pack_disabled", disabled)
	for k, v := range config {
		content = setKey(content, "gate_pack_config."+k, v)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ExtractKeys(content)["gate_pack"]; got != pack {
		t.Fatalf("optIn did not take: gate_pack = %q", got)
	}
}

// AC (I085): an unknown pack version is reported by name and nothing is
// written — a repo pinned to go@2 never silently gets a go@1 region.
func TestUnknownGatePackIsReported(t *testing.T) {
	dir := stageGen10Repo(t, nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	optIn(t, dir, "go@2", "[]", nil)
	for _, force := range []bool{false, true} {
		reports, err := Run(Options{Dir: dir, Write: true, Force: force})
		if err != nil {
			t.Fatal(err)
		}
		mp := report(t, reports, MaipipeFile)
		if mp.State != SkippedUnrecognized || len(mp.Unrecognized) != 1 ||
			!strings.Contains(mp.Unrecognized[0], "go@2") {
			t.Fatalf("force=%v: want the unknown pack reported, got state=%v unrec=%v",
				force, mp.State, mp.Unrecognized)
		}
		if _, err := os.Stat(filepath.Join(dir, MaipipeFile)); !os.IsNotExist(err) {
			t.Errorf("force=%v: %s written for an unknown pack", force, MaipipeFile)
		}
	}
}

// Canonical-form byte determinism of the rendered region: a pure function of
// (pack, disabled, config), stable across calls.
func TestGateRegionRenderIsDeterministic(t *testing.T) {
	s := gatePackSettings{
		pack:     gate.PackID(),
		disabled: map[string]bool{"tskip": true},
		config:   map[string]string{"build_outputs": "bin/spine"},
	}
	first := renderGateRegion(s)
	if second := renderGateRegion(s); first != second {
		t.Fatalf("render not byte-stable:\n%s\n---\n%s", first, second)
	}
	if !strings.HasPrefix(first, "# spine:begin gate-pack go@1\n") ||
		!strings.HasSuffix(first, "# spine:end\n") {
		t.Errorf("region markers wrong:\n%s", first)
	}
	if !strings.Contains(first, "[pipelines.mutation-go]\nprofile = \"audit\"\n") {
		t.Errorf("mutation-go pipeline missing from the canonical render:\n%s", first)
	}
}
