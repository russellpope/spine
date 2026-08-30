package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/gate"
)

// gateRepo is a gen-11 repo that has opted into the pack: WORKFLOW.md is
// migrated first, then the gate-pack keys are set the way an owner would.
func gateRepo(t *testing.T, disabled string, config map[string]string) string {
	t.Helper()
	dir := stageGen10Repo(t, nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	optIn(t, dir, gate.PackID(), disabled, config)
	return dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func clearGatePack(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte(setKey(readFile(t, path), "gate_pack", "")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validateMaipipe(t *testing.T, path string) {
	t.Helper()
	output, err := exec.Command("maipipe", "validate", path).CombinedOutput()
	if err != nil {
		t.Fatalf("maipipe validate %s: %v\n%s", path, err, output)
	}
}

// I097: clearing a pack must stop before planning removal when repo-owned
// stages outside the managed region compose either of its pipelines. Removing
// the guard makes the fixture unloadable, so this is a load-bearing control.
func TestGatePackOptOutRefusesExternalCompositions(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	lanes := `
[pipelines.full]

[[pipelines.full.stage]]
name = "gates"
pipeline = "gate-go"

[pipelines.audit]

[[pipelines.audit.stage]]
name = "mutation"
pipeline = "mutation-go"
`
	if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	clearGatePack(t, dir)
	before := readFile(t, path)

	plan, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatalf("dry-run opt-out returned an early error instead of a reviewable report: %v", err)
	}
	mp := report(t, plan, MaipipeFile)
	if mp.State != Pending ||
		!strings.Contains(mp.Refusal, "gate_pack cleared but 2 stage(s) still compose the pack — remove them, then re-run") ||
		!strings.Contains(mp.Refusal, `pipeline "full" stage "gates"`) ||
		!strings.Contains(mp.Refusal, `pipeline "audit" stage "mutation"`) {
		t.Fatalf("opt-out report = %#v, want a refusal naming every repo-owned composition", mp)
	}
	if got := readFile(t, path); got != before {
		t.Fatal("refused opt-out changed maipipe.toml")
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err == nil {
		t.Fatal("--write accepted an opt-out with repo-owned compositions")
	}
	if got := readFile(t, path); got != before {
		t.Fatal("refused --write changed maipipe.toml")
	}

	begin, end, err := gateRegionBounds(before)
	if err != nil {
		t.Fatal(err)
	}
	unguarded := strings.Join(append(append([]string{}, splitLines(before)[:begin]...), splitLines(before)[end+1:]...), "\n")
	if err := os.WriteFile(path, []byte(unguarded), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("maipipe", "validate", path).CombinedOutput()
	if err == nil || !strings.Contains(string(output), `composes unknown pipeline "gate-go"`) {
		t.Fatalf("unguarded removal validate = %v\n%s\nwant unknown gate-go composition", err, output)
	}
}

// I097 review control: TOML permits a comment after an array-table header and
// omits optional whitespace around assignments. Those spellings must still
// produce the pre-deletion reviewable refusal, not defer discovery to
// maipipe's generic validation error.
func TestGatePackOptOutRefusalRecognizesCompactCommentedStage(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	const lanes = `
[pipelines.full]

[[pipelines.full.stage]] # owner comment
name="gates"
pipeline="gate-go"

[pipelines.audit]

[[pipelines.audit.stage]] # another owner comment
name="mutation # owner note"
pipeline="mutation-go"
`
	if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	clearGatePack(t, dir)

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending || !strings.Contains(mp.Refusal, "gate_pack cleared but") ||
		!strings.Contains(mp.Refusal, `pipeline "full" stage "gates"`) ||
		!strings.Contains(mp.Refusal, `pipeline "audit" stage "mutation # owner note"`) {
		t.Fatalf("compact/commented composition report = %#v, want both named refusals", mp)
	}
}

// I097 review control: TOML literal strings use single quotes and retain # as
// data. The targeted reader must name the owner before maipipe reaches its
// generic unknown-pipeline validation error.
func TestGatePackOptOutRefusalRecognizesLiteralStringStage(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	const lanes = `
[pipelines.full]

[[pipelines.full.stage]]
name='gates # owner'
pipeline='gate-go'
`
	if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	clearGatePack(t, dir)

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending || !strings.Contains(mp.Refusal, "gate_pack cleared but") ||
		!strings.Contains(mp.Refusal, `pipeline "full" stage "gates # owner"`) {
		t.Fatalf("literal-string composition report = %#v, want full/gates owner refusal", mp)
	}
}

// Primary-review control: maipipe accepts whitespace inside an array-table
// header and quoted assignment keys. spine must still aggregate every owner
// before a candidate deletion reaches maipipe validation.
func TestGatePackOptOutRefusalRecognizesSpacedQuotedStage(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	const lanes = `
[pipelines.full]

[[ pipelines . full . stage ]]
"name" = "gates"
'pipeline' = 'gate-go'

[pipelines.audit]

[[ pipelines . audit . stage ]]
'name' = 'mutation # owner'
"pipeline" = "mutation-go"
`
	if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	clearGatePack(t, dir)

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending || !strings.Contains(mp.Refusal, "gate_pack cleared but 2 stage(s)") ||
		!strings.Contains(mp.Refusal, `pipeline "full" stage "gates"`) ||
		!strings.Contains(mp.Refusal, `pipeline "audit" stage "mutation # owner"`) {
		t.Fatalf("spaced/quoted composition report = %#v, want both owner refusals", mp)
	}
}

// Final re-verification control: quote-aware table paths retain their decoded
// owner names, including dots, before the opt-out planner aggregates refs.
func TestGatePackOptOutRefusalRecognizesQuotedTablePaths(t *testing.T) {
	cases := []struct {
		name, lanes, owner, stage string
	}{
		{
			name: "quoted-owner-with-dot",
			lanes: `
[[pipelines."full.lane".stage]]
name = "gates"
pipeline = "gate-go"
`,
			owner: "full.lane",
			stage: "gates",
		},
		{
			name: "quoted-key-segments",
			lanes: `
[[ "pipelines" . full . 'stage' ]]
name = "gates two"
pipeline = "gate-go"
`,
			owner: "full",
			stage: "gates two",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := gateRepo(t, "[]", nil)
			path := filepath.Join(dir, MaipipeFile)
			if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(readFile(t, path)+tc.lanes), 0o644); err != nil {
				t.Fatal(err)
			}
			clearGatePack(t, dir)

			reports, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			mp := report(t, reports, MaipipeFile)
			want := `pipeline "` + tc.owner + `" stage "` + tc.stage + `"`
			if mp.State != Pending || !strings.Contains(mp.Refusal, "gate_pack cleared but") || !strings.Contains(mp.Refusal, want) {
				t.Fatalf("quoted-path report = %#v, want %s", mp, want)
			}
		})
	}
}

// Final re-verification control: malformed bare key segments are not owners.
// They must bypass the targeted composition reader and remain maipipe grammar
// errors rather than an I097 composition refusal.
func TestGatePackOptOutIgnoresMalformedBareTablePath(t *testing.T) {
	cases := []struct {
		name, header string
	}{
		{"space", "[[pipelines.bad owner.stage]]"},
		{"punctuation", "[[pipelines.bad!owner.stage]]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := gateRepo(t, "[]", nil)
			path := filepath.Join(dir, MaipipeFile)
			if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
				t.Fatal(err)
			}
			lanes := "\n" + tc.header + "\nname = \"gates\"\npipeline = \"gate-go\"\n"
			if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
				t.Fatal(err)
			}
			clearGatePack(t, dir)

			reports, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			mp := report(t, reports, MaipipeFile)
			if mp.Refusal == "" || strings.HasPrefix(mp.Refusal, "gate_pack cleared but") {
				t.Fatalf("malformed %q report = %#v, want maipipe grammar refusal", tc.header, mp)
			}
		})
	}
}

// Final re-verification control: check the installed maipipe grammar before
// asserting spine's boundary. TOML basic-string escapes aggregate normally;
// Go-only escapes must defer to maipipe instead of manufacturing an I097
// owner composition.
func TestGatePackOptOutHeaderBasicStringEscapeBoundary(t *testing.T) {
	cases := []struct {
		name, escape string
		valid        bool
	}{
		{"hex", `\x2e`, true},
		{"unicode-short", `\u002e`, true},
		{"unicode-long", `\U0000002e`, true},
		{"quote", `\"`, true},
		{"backslash", `\\`, true},
		{"backspace", `\b`, true},
		{"tab", `\t`, true},
		{"newline", `\n`, true},
		{"form-feed", `\f`, true},
		{"carriage-return", `\r`, true},
		{"escape", `\e`, true},
		{"octal", `\101`, false},
		{"bell", `\a`, false},
		{"vertical-tab", `\v`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			grammarDir := t.TempDir()
			grammarPath := filepath.Join(grammarDir, MaipipeFile)
			grammar := "schema = 0\n\n[[pipelines.\"owner" + tc.escape + "\".stage]]\nname = \"check\"\nrun = \"true\"\n"
			if err := os.WriteFile(grammarPath, []byte(grammar), 0o644); err != nil {
				t.Fatal(err)
			}
			grammarOut, grammarErr := exec.Command("maipipe", "validate", grammarPath).CombinedOutput()
			missingEscape := strings.Contains(string(grammarOut), "missing escaped value")
			if tc.valid && missingEscape {
				t.Fatalf("maipipe rejected accepted escape %q as syntax: %v\n%s", tc.escape, grammarErr, grammarOut)
			}
			if !tc.valid && (grammarErr == nil || !missingEscape) {
				t.Fatalf("maipipe grammar escape %q err=%v out=%s, want syntax rejection", tc.escape, grammarErr, grammarOut)
			}

			dir := gateRepo(t, "[]", nil)
			path := filepath.Join(dir, MaipipeFile)
			if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
				t.Fatal(err)
			}
			lanes := "\n[[pipelines.\"owner" + tc.escape + "\".stage]]\nname = \"gates\"\npipeline = \"gate-go\"\n"
			if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
				t.Fatal(err)
			}
			clearGatePack(t, dir)
			reports, err := Run(Options{Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			mp := report(t, reports, MaipipeFile)
			if tc.valid && !strings.HasPrefix(mp.Refusal, "gate_pack cleared but") {
				t.Fatalf("valid escape %q report = %#v, want I097 refusal", tc.escape, mp)
			}
			if !tc.valid && (mp.Refusal == "" || strings.HasPrefix(mp.Refusal, "gate_pack cleared but")) {
				t.Fatalf("invalid escape %q report = %#v, want maipipe refusal", tc.escape, mp)
			}
		})
	}
}

// maipipe's \e escape is distinct from an escaped backslash followed by e.
// The latter must retain both data characters in the owner name.
func TestGatePackOptOutHeaderEscapedBackslashE(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	const lanes = `
[[pipelines."owner\\e".stage]]
name = "gates"
pipeline = "gate-go"
`
	if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, _ := exec.Command("maipipe", "validate", path).CombinedOutput(); strings.Contains(string(output), "missing escaped value") {
		t.Fatalf("maipipe rejected escaped-backslash control as syntax:\n%s", output)
	}
	clearGatePack(t, dir)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if !strings.HasPrefix(mp.Refusal, "gate_pack cleared but") ||
		!strings.Contains(mp.Refusal, `pipeline "owner\\e" stage "gates"`) {
		t.Fatalf("escaped-backslash-e report = %#v, want literal backslash owner", mp)
	}
}

// I097: without an outside consumer, opt-out is an ordinary marker-inclusive
// deletion. It stays behind I104's real maipipe validation preflight and a
// second run is a clean no-op.
func TestGatePackOptOutDeletesUnreferencedManagedRegion(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	lanes := `
[pipelines.full]

[[pipelines.full.stage]]
name = "build"
run = "true"
`
	if err := os.WriteFile(path, []byte(readFile(t, path)+lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	clearGatePack(t, dir)

	plan, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, plan, MaipipeFile)
	if mp.State != Pending || !strings.Contains(mp.Diff, "- # spine:begin gate-pack go@1") ||
		!strings.Contains(mp.Diff, "- # spine:end") {
		t.Fatalf("opt-out plan = %#v, want marker-inclusive deletion diff", mp)
	}
	if mp.Preflight != maipipeValidatePreflight {
		t.Fatalf("opt-out preflight = %q, want %q", mp.Preflight, maipipeValidatePreflight)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); strings.Contains(got, gateRegionBegin) || strings.Contains(got, gateRegionEnd) {
		t.Fatalf("opt-out left managed markers behind:\n%s", got)
	}
	validateMaipipe(t, path)

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == MaipipeFile {
			t.Fatalf("second opt-out run reported %s: %#v", MaipipeFile, r)
		}
	}
}

// I104 option B: a repository that enables a gate pack cannot safely refresh
// maipipe.toml without maipipe itself. That one file is skipped, but a normal
// update still applies every other pending file.
func TestNoMaipipeSkipsMaipipeAndWritesOtherPendingFiles(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	mpPath := filepath.Join(dir, MaipipeFile)
	const sentinel = "schema = 0\n# preserve this exact file when maipipe is unavailable\n"
	if err := os.WriteFile(mpPath, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}

	wfPath := filepath.Join(dir, "WORKFLOW.md")
	beforeWorkflow := setKey(readFile(t, wfPath), "template_version", "10")
	if err := os.WriteFile(wfPath, []byte(beforeWorkflow), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())

	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatalf("missing maipipe should skip only %s, not refuse the update: %v", MaipipeFile, err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != SkippedPreflight {
		t.Fatalf("%s state = %v, want a preflight skip", MaipipeFile, mp.State)
	}
	if mp.Preflight != noMaipipePreflight {
		t.Errorf("preflight = %q, want %q", mp.Preflight, noMaipipePreflight)
	}
	if mp.Diff != "" {
		t.Errorf("a skipped %s has a writable diff:\n%s", MaipipeFile, mp.Diff)
	}
	if got := readFile(t, mpPath); got != sentinel {
		t.Errorf("%s changed without maipipe:\nwant %q\n got %q", MaipipeFile, sentinel, got)
	}
	if got := readFile(t, wfPath); got == beforeWorkflow || !strings.Contains(got, "template_version: 12") {
		t.Errorf("another pending file was not applied:\n%s", got)
	}
}

// AC (I085, amended I091): absent maipipe.toml + gate_pack set → the file
// is created as maipipe's required top-level `schema = 0` followed by the
// region and nothing else, with one stage per enabled check class.
func TestGatePackCreatesMaipipeWithSchemaAndRegionOnly(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending || !mp.Created {
		t.Fatalf("want a created maipipe.toml, got state=%v created=%v", mp.State, mp.Created)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	if !strings.HasPrefix(got, "schema = 0\n\n# spine:begin gate-pack "+gate.PackID()+"\n") ||
		!strings.HasSuffix(got, "# spine:end\n") {
		t.Fatalf("created file is not schema line + region:\n%s", got)
	}
	// maipipe's stage array is the singular `stage` (I091): the plural is a
	// parse error ("unknown field `stages`"), so it must never render.
	if strings.Contains(got, ".stages]]") {
		t.Fatalf("region renders the plural [[…stages]] maipipe rejects:\n%s", got)
	}
	if !strings.Contains(got, "[pipelines.gate-go]\nprofile = \"full\"\n") {
		t.Errorf("missing gate-go pipeline header:\n%s", got)
	}
	for _, check := range gate.CheckNames() {
		want := "run = \"spine gate go@1 " + check + "\""
		if !strings.Contains(got, want) {
			t.Errorf("missing stage for check class %q:\n%s", check, got)
		}
	}
	if n := strings.Count(got, "[[pipelines.gate-go.stage]]"); n != len(gate.CheckNames())-1 {
		t.Errorf("stage count = %d, want %d (mutate is the advisory lane, not a gate-go stage)",
			n, len(gate.CheckNames())-1)
	}
	if strings.Contains(got, "env = {") {
		t.Errorf("env rendered with no gate_pack_config set:\n%s", got)
	}
}

// AC (I086): the advisory battery is its own audit-profile pipeline with one
// stage, rendered after gate-go inside the same region — never a stage in
// the enforcement lane.
func TestGatePackRendersMutationPipeline(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	want := "[pipelines.mutation-go]\nprofile = \"audit\"\n\n" +
		"[[pipelines.mutation-go.stage]]\nname = \"mutate\"\nrun = \"spine gate go@1 mutate\"\n"
	if !strings.Contains(got, want) {
		t.Fatalf("mutation-go pipeline missing or not canonical:\n%s", got)
	}
	if n := strings.Count(got, "[[pipelines.mutation-go.stage]]"); n != 1 {
		t.Errorf("mutation-go stage count = %d, want 1", n)
	}
	if strings.Index(got, "[pipelines.mutation-go]") < strings.Index(got, "[pipelines.gate-go]") {
		t.Errorf("mutation-go rendered before gate-go:\n%s", got)
	}
	if strings.Contains(got, "[[pipelines.gate-go.stage]]\nname = \"mutate\"") {
		t.Errorf("mutate rendered as a gate-go stage:\n%s", got)
	}
}

// AC (I086): gate_pack_disabled: [mutate] omits the whole mutation-go
// pipeline — a disabled class has no lane, not an empty one.
func TestGatePackDisabledMutateOmitsPipeline(t *testing.T) {
	dir := gateRepo(t, "[mutate]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	if strings.Contains(got, "[pipelines.mutation-go]") {
		t.Errorf("mutation-go pipeline rendered for a disabled class:\n%s", got)
	}
	if strings.Contains(got, "run = \"spine gate go@1 mutate\"") {
		t.Errorf("disabled mutate still rendered:\n%s", got)
	}
	if !strings.Contains(got, "run = \"spine gate go@1 tskip\"") {
		t.Error("disabling mutate dropped the gate-go stages too")
	}
}

// AC (I086): a repo whose region was rendered before this ticket (gate-go
// only) refreshes to include mutation-go as an inherited change — its own
// lanes untouched, and nothing reported as a local edit.
func TestGatePackPreMutationRegionRefreshes(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	// maipipe requires the top-level schema key (I091), and update now
	// refuses to write a maipipe.toml maipipe could not load (I096), so the
	// user-lane fixture carries it the way a real repo does.
	lanes := "schema = 0\n\n[pipelines.full]\n\n[[pipelines.full.stage]]\nname = \"build\"\nrun = \"make build\"\n"
	path := filepath.Join(dir, MaipipeFile)
	if err := os.WriteFile(path, []byte(lanes+"\n"+preMutationRegion), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if len(mp.Unrecognized) > 0 {
		t.Fatalf("a pre-mutation region read as local edits: %v", mp.Unrecognized)
	}
	if mp.State != Pending {
		t.Fatalf("state = %v, want the region refreshed", mp.State)
	}
	got := readFile(t, path)
	if !strings.HasPrefix(got, lanes) {
		t.Errorf("the refresh disturbed the user lanes:\n%s", got)
	}
	if !strings.Contains(got, "[pipelines.mutation-go]") {
		t.Errorf("refresh did not add mutation-go:\n%s", got)
	}
	if n := strings.Count(got, "# spine:begin gate-pack "); n != 1 {
		t.Errorf("region count = %d, want 1", n)
	}
}

// preMutationRegion is the region exactly as I085 rendered it — gate-go
// only, with the three-line header comment of that generation. It is fixture
// text on purpose: the point of the test is that a repo pinned before this
// ticket refreshes rather than reporting drift.
const preMutationRegion = `# spine:begin gate-pack go@1
# spine manages this region. Change it through the gate_pack keys in
# WORKFLOW.md and re-run ` + "`spine update`" + `, never by hand.
# Compose the pack into your own lane with a stage: pipeline = "gate-go"

[pipelines.gate-go]
profile = "full"

[[pipelines.gate-go.stage]]
name = "binary-hygiene"
run = "spine gate go binary-hygiene"

[[pipelines.gate-go.stage]]
name = "tskip"
run = "spine gate go tskip"
# spine:end
`

// AC (I085): gate_pack_disabled drops exactly the named class.
func TestGatePackDisabledOmitsStage(t *testing.T) {
	dir := gateRepo(t, "[tskip]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	if strings.Contains(got, "spine gate go@1 tskip") {
		t.Errorf("disabled check class still rendered:\n%s", got)
	}
	if n := strings.Count(got, "[[pipelines.gate-go.stage]]"); n != len(gate.CheckNames())-2 {
		t.Errorf("stage count = %d, want %d", n, len(gate.CheckNames())-2)
	}
	if !strings.Contains(got, "spine gate go@1 binary-hygiene") {
		t.Error("dropping one class dropped others too")
	}
}

// AC (I085): a non-empty gate_pack_config value reaches its class's stage as
// SPINE_GATE_<KEY>; classes that take no configuration get no env.
func TestGatePackConfigRendersEnv(t *testing.T) {
	dir := gateRepo(t, "[]", map[string]string{
		"fixture_manifest": "docs/fixtures.md",
		"build_outputs":    "bin/spine",
	})
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	for _, want := range []string{
		"run = \"spine gate go@1 fixture-manifest\"\nenv = { SPINE_GATE_FIXTURE_MANIFEST = \"docs/fixtures.md\" }",
		"run = \"spine gate go@1 gitignore-control\"\nenv = { SPINE_GATE_BUILD_OUTPUTS = \"bin/spine\" }",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "env = {"); n != 2 {
		t.Errorf("env tables = %d, want 2 (only configured classes carry env):\n%s", n, got)
	}
	if strings.Contains(got, "SPINE_GATE_CLEANUP_FUNCS") {
		t.Error("env-only knob rendered into the region")
	}
}

// AC (I095, reading A): the region is a pure projection of WORKFLOW.md, so
// a hand-edited env value inside it is not a preserved choice. The dry-run
// plan shows the revert as an ordinary diff — not an unrecognized edit —
// and --write refreshes it with no --force.
func TestGatePackEditedEnvValueRefreshesWithoutForce(t *testing.T) {
	dir := gateRepo(t, "[]", map[string]string{"fixture_manifest": "docs/fixtures.md"})
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, MaipipeFile)
	const rendered = `SPINE_GATE_FIXTURE_MANIFEST = "docs/fixtures.md"`
	const edited = `SPINE_GATE_FIXTURE_MANIFEST = "docs/elsewhere.md"`
	if err := os.WriteFile(path, []byte(strings.Replace(readFile(t, path), rendered, edited, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, plan, MaipipeFile)
	if mp.State != Pending || len(mp.Unrecognized) != 0 {
		t.Fatalf("want a plain Pending refresh, got state=%v unrec=%v", mp.State, mp.Unrecognized)
	}
	if !strings.Contains(mp.Diff, "- env = { "+edited+" }") || !strings.Contains(mp.Diff, "+ env = { "+rendered+" }") {
		t.Errorf("plan diff is the review surface; it does not show the drop:\n%s", mp.Diff)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); strings.Contains(got, edited) || !strings.Contains(got, rendered) {
		t.Errorf("edited env value survived a refresh without --force:\n%s", got)
	}
}

// AC (I085): an existing maipipe.toml keeps the owner's own lanes
// byte-for-byte; the region is appended, then refreshed in place.
func TestGatePackPreservesUserLanes(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	// maipipe requires the top-level schema key (I091), and update now
	// refuses to write a maipipe.toml maipipe could not load (I096), so the
	// user-lane fixture carries it the way a real repo does.
	lanes := "schema = 0\n\n[pipelines.full]\n\n[[pipelines.full.stage]]\nname = \"build\"\nrun = \"make build\"\n"
	path := filepath.Join(dir, MaipipeFile)
	if err := os.WriteFile(path, []byte(lanes), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, path)
	if !strings.HasPrefix(got, lanes) {
		t.Fatalf("user lanes not preserved byte-for-byte:\n%s", got)
	}
	if !strings.Contains(got, "# spine:begin gate-pack ") {
		t.Fatalf("region not appended:\n%s", got)
	}
	// A config change refreshes the region in place, lanes still untouched.
	optIn(t, dir, gate.PackID(), "[]", map[string]string{"tskip_allow": "internal/gate/testdata"})
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if len(mp.Unrecognized) > 0 {
		t.Errorf("spine's own region read as a local edit: %v", mp.Unrecognized)
	}
	got = readFile(t, path)
	if !strings.HasPrefix(got, lanes) {
		t.Errorf("refresh disturbed the user lanes:\n%s", got)
	}
	if !strings.Contains(got, "env = { SPINE_GATE_TSKIP_ALLOW = \"internal/gate/testdata\" }") {
		t.Errorf("region not refreshed with the new config:\n%s", got)
	}
	if n := strings.Count(got, "# spine:begin gate-pack "); n != 1 {
		t.Errorf("region count = %d, want 1", n)
	}
	// Idempotent: a second run has nothing to do.
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if mp := report(t, reports, MaipipeFile); mp.State != UpToDate {
		t.Errorf("second pass state=%v diff:\n%s", mp.State, mp.Diff)
	}
}

// AC (I085): an edit inside the region is reported as unrecognized and the
// file is skipped — never silently kept, never silently overwritten. --force
// is the explicit opt-in that regenerates it.
func TestGatePackRegionEditIsUnrecognized(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, MaipipeFile)
	edited := strings.Replace(readFile(t, path),
		"run = \"spine gate go@1 tskip\"", "run = \"echo tskip\"", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != SkippedUnrecognized || len(mp.Unrecognized) != 1 ||
		!strings.Contains(mp.Unrecognized[0], "echo tskip") {
		t.Fatalf("want the edited line reported, got state=%v unrec=%v", mp.State, mp.Unrecognized)
	}
	if got := readFile(t, path); !strings.Contains(got, "echo tskip") {
		t.Error("edited region overwritten without --force")
	}
	if _, err := Run(Options{Dir: dir, Write: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); strings.Contains(got, "echo tskip") {
		t.Error("--force did not regenerate the region")
	}
}

// AC (I085): damaged markers — a begin without an end — are hand-repair
// work: reported, file skipped, and --force cannot paper over them.
func TestGatePackBrokenMarkerIsReported(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, MaipipeFile)
	broken := strings.Replace(readFile(t, path), "# spine:end\n", "", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, force := range []bool{false, true} {
		reports, err := Run(Options{Dir: dir, Write: true, Force: force})
		if err != nil {
			t.Fatal(err)
		}
		mp := report(t, reports, MaipipeFile)
		if mp.State != SkippedUnrecognized || len(mp.Unrecognized) != 1 ||
			!strings.Contains(mp.Unrecognized[0], "unbalanced") {
			t.Fatalf("force=%v: want unbalanced markers reported, got state=%v unrec=%v",
				force, mp.State, mp.Unrecognized)
		}
		if got := readFile(t, path); got != broken {
			t.Errorf("force=%v: file with damaged markers was rewritten", force)
		}
	}
}

// I097/ADR 0018: a safe-looking deletion is still refused when its resulting
// maipipe.toml is invalid. The existing region stays byte-for-byte intact.
func TestGatePackOptOutUsesMaipipePreflight(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, filepath.Join(dir, MaipipeFile))
	clearGatePack(t, dir)
	if _, err := Run(Options{Dir: dir, Write: true}); err == nil || !strings.Contains(err.Error(), "maipipe validate rejected") {
		t.Fatalf("opt-out without a remaining user pipeline error = %v, want maipipe preflight refusal", err)
	}
	if after := readFile(t, filepath.Join(dir, MaipipeFile)); after != before {
		t.Error("preflight-refused opt-out changed maipipe.toml")
	}

	// Fleet negative control: a repo that never opted in gets no maipipe.toml.
	fresh := stageGen10Repo(t, nil)
	if _, err := Run(Options{Dir: fresh, Write: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fresh, MaipipeFile)); !os.IsNotExist(err) {
		t.Errorf("maipipe.toml created for a repo without gate_pack (err=%v)", err)
	}
}
