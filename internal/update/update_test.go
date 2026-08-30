package update

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
	"github.com/russellpope/spine/internal/tmpl"
)

func forceFiles(t *testing.T, opts *Options, paths ...string) {
	t.Helper()
	opts.ForceFiles = paths
}

func TestUpdateForceFileRejectsInvalidAuthorityBeforePolicyOrWrite(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "empty", paths: []string{""}, want: `update: --force-file "" must be repository-relative and must not contain ".."`},
		{name: "absolute", paths: []string{"/WORKFLOW.md"}, want: `update: --force-file "/WORKFLOW.md" must be repository-relative and must not contain ".."`},
		{name: "raw traversal", paths: []string{"docs/../WORKFLOW.md"}, want: `update: --force-file "docs/../WORKFLOW.md" must be repository-relative and must not contain ".."`},
		{name: "backslash traversal", paths: []string{"docs\\..\\WORKFLOW.md"}, want: `update: --force-file "docs\\..\\WORKFLOW.md" must be repository-relative and must not contain ".."`},
		{name: "drive absolute", paths: []string{"C:\\WORKFLOW.md"}, want: `update: --force-file "C:\\WORKFLOW.md" must be repository-relative and must not contain ".."`},
		{name: "drive relative", paths: []string{`C:WORKFLOW.md`}, want: `update: --force-file "C:WORKFLOW.md" must be repository-relative and must not contain ".."`},
		{name: "UNC", paths: []string{"\\\\server\\share\\WORKFLOW.md"}, want: `update: --force-file "\\\\server\\share\\WORKFLOW.md" must be repository-relative and must not contain ".."`},
		{name: "normalized duplicate", paths: []string{"./WORKFLOW.md", "WORKFLOW.md"}, want: `update: duplicate --force-file "WORKFLOW.md"`},
		{name: "unknown unmanaged", paths: []string{"README.md"}, want: `update: --force-file "README.md" must name a managed file in this update plan`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
				t.Fatal(err)
			}
			before := readFile(t, filepath.Join(dir, "WORKFLOW.md"))
			callback := false
			writerCalled := false
			previousWriter := writeFileAtomic
			writeFileAtomic = func(string, []byte) error {
				writerCalled = true
				return nil
			}
			t.Cleanup(func() { writeFileAtomic = previousWriter })
			opts := Options{Dir: dir, Write: true, BeforeWrite: func([]GateConfigAdvisory) { callback = true }}
			forceFiles(t, &opts, tt.paths...)
			reports, err := Run(opts)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if reports != nil {
				t.Fatalf("rejected authority returned reports: %#v", reports)
			}
			if callback {
				t.Fatal("rejected authority reached the write callback")
			}
			if writerCalled {
				t.Fatal("rejected authority reached the atomic writer")
			}
			if got := readFile(t, filepath.Join(dir, "WORKFLOW.md")); got != before {
				t.Fatal("rejected authority changed WORKFLOW.md")
			}
		})
	}
}

// I124: repository path spelling is an input protocol, not a property of the
// host running the test. A Windows-style separator must identify the same
// nested planned report on every host, while both raw separator forms expose
// traversal before normalization can conceal it.
func TestNormalizeForceFilesUsesCanonicalRepositoryPathsOnEveryHost(t *testing.T) {
	paths, selected, err := normalizeForceFiles([]string{`docs\issues\README.md`, "WORKFLOW.md"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(paths, ","), "docs/issues/README.md,WORKFLOW.md"; got != want {
		t.Fatalf("normalized paths = %q, want %q", got, want)
	}
	if !selected["docs/issues/README.md"] || !selected["WORKFLOW.md"] {
		t.Fatalf("selected = %#v, want canonical repository paths", selected)
	}
	for _, raw := range []string{"docs/../WORKFLOW.md", `docs\..\WORKFLOW.md`} {
		if _, _, err := normalizeForceFiles([]string{raw}); err == nil {
			t.Fatalf("normalizeForceFiles(%q) accepted raw traversal", raw)
		}
	}
	if _, _, err := normalizeForceFiles([]string{"./docs/issues/README.md", `docs\issues\README.md`}); err == nil {
		t.Fatal("normalized duplicate accepted mixed separators")
	}
}

func TestUpdateForceFileRejectsPathsOutsideThisPlan(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		paths   []string
	}{
		{name: "profile not owned", profile: "knowledge", paths: []string{"docs/issues/README.md"}},
		{name: "maipipe not planned", profile: "go-service", paths: []string{MaipipeFile}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if _, err := scaffold.Init(dir, tt.profile, "demo"); err != nil {
				t.Fatal(err)
			}
			writerCalled := false
			previousWriter := writeFileAtomic
			writeFileAtomic = func(string, []byte) error {
				writerCalled = true
				return nil
			}
			t.Cleanup(func() { writeFileAtomic = previousWriter })
			opts := Options{Dir: dir, Write: true}
			forceFiles(t, &opts, tt.paths...)
			reports, err := Run(opts)
			want := `update: --force-file "` + tt.paths[0] + `" must name a managed file in this update plan`
			if err == nil || err.Error() != want {
				t.Fatalf("error = %v, want %q", err, want)
			}
			if reports != nil {
				t.Fatalf("rejected plan membership returned reports: %#v", reports)
			}
			if writerCalled {
				t.Fatal("rejected plan membership reached the atomic writer")
			}
		})
	}
}

func TestUpdateForceFileAuthorizesOnlyItsExactManagedReport(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	siblingPath := filepath.Join(dir, "docs", "issues", "README.md")
	if err := os.WriteFile(wfPath, append([]byte(readFile(t, wfPath)), []byte("custom_rule: keep\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeWorkflow := readFile(t, wfPath)
	beforeSibling := readFile(t, siblingPath)
	if err := os.WriteFile(siblingPath, append([]byte(beforeSibling), []byte("local issue convention\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	// A clean member is valid and produces no artificial pending report.
	clean := Options{Dir: t.TempDir()}
	if _, err := scaffold.Init(clean.Dir, "go-service", "clean"); err != nil {
		t.Fatal(err)
	}
	forceFiles(t, &clean, "./WORKFLOW.md")
	cleanReports, err := Run(clean)
	if err != nil {
		t.Fatal(err)
	}
	if got := report(t, cleanReports, "WORKFLOW.md"); got.State != UpToDate {
		t.Fatalf("selected clean member state = %v, want UpToDate", got.State)
	}

	opts := Options{Dir: dir, Write: true}
	forceFiles(t, &opts, `docs\issues\README.md`)
	reports, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := report(t, reports, "WORKFLOW.md"); got.State != SkippedUnrecognized {
		t.Fatalf("unselected WORKFLOW.md state = %v, want SkippedUnrecognized", got.State)
	}
	if got := report(t, reports, "docs/issues/README.md"); got.State != Pending || !got.SelectedByForceFile {
		t.Fatalf("selected nested sibling = %#v, want pending scoped selection", got)
	}
	if got := readFile(t, siblingPath); strings.Contains(got, "local issue convention") {
		t.Fatal("selected nested sibling was not regenerated")
	}
	if got := readFile(t, wfPath); got != beforeWorkflow {
		t.Fatal("unselected WORKFLOW.md changed under scoped authority")
	}
}

func TestUpdateRejectsMixedGlobalAndScopedForceBeforePlanning(t *testing.T) {
	opts := Options{Dir: filepath.Join(t.TempDir(), "missing"), Force: true}
	forceFiles(t, &opts, "WORKFLOW.md")
	writerCalled := false
	previousWriter := writeFileAtomic
	writeFileAtomic = func(string, []byte) error {
		writerCalled = true
		return nil
	}
	t.Cleanup(func() { writeFileAtomic = previousWriter })
	reports, err := Run(opts)
	want := "update: --force cannot be combined with --force-file; choose one overwrite authority"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if reports != nil {
		t.Fatalf("mixed authority returned reports: %#v", reports)
	}
	if writerCalled {
		t.Fatal("mixed authority reached the atomic writer")
	}
}

// I123: reports are a real pre-write seam. The callback must see the fully
// preflighted plan before its first atomic write; moving it after the write
// would leave a caller unable to present advance configuration advice.
func TestUpdatePreWriteGateAdvisoriesFollowPreflightAndPrecedeWrites(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	marker := filepath.Join(dir, "preflight-ran")
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "maipipe")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n: > "+marker+"\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	want := []GateConfigAdvisory{
		{Class: "fixture-manifest", Key: "fixture_manifest"},
		{Class: "gitignore-control", Key: "build_outputs"},
		{Class: "n-plus-one", Key: "n_plus_one_clients"},
		{Class: "test-enum-vs-spec", Key: "test_enum_spec"},
	}
	calls := 0
	_, err := Run(Options{
		Dir:   dir,
		Write: true,
		BeforeWrite: func(got []GateConfigAdvisory) {
			calls++
			if !reflect.DeepEqual(got, want) {
				t.Errorf("advisories = %#v, want %#v", got, want)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Errorf("callback ran before candidate preflight: %v", err)
			}
			if _, err := os.Stat(filepath.Join(dir, MaipipeFile)); !os.IsNotExist(err) {
				t.Errorf("callback ran after a write: maipipe stat err=%v", err)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("pre-write advisory callback calls = %d, want 1", calls)
	}
	if _, err := os.Stat(filepath.Join(dir, MaipipeFile)); err != nil {
		t.Fatalf("write did not follow the callback: %v", err)
	}
}

// I123's advisory is still useful when a candidate preflight refuses the
// whole plan. It must be delivered before the refusal, while every planned
// file remains byte-identical.
func TestUpdatePreWriteGateAdvisoriesPrecedeRefusalWithoutWrites(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	beforeWorkflow := setKey(readFile(t, wfPath), "template_version", "10")
	if err := os.WriteFile(wfPath, []byte(beforeWorkflow), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	bin := filepath.Join(binDir, "maipipe")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho refused >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	called := false
	_, err := Run(Options{
		Dir:   dir,
		Write: true,
		BeforeWrite: func(got []GateConfigAdvisory) {
			called = true
			if len(got) != 4 {
				t.Errorf("advisories = %#v, want four missing required inputs", got)
			}
			if _, err := os.Stat(filepath.Join(dir, MaipipeFile)); !os.IsNotExist(err) {
				t.Errorf("callback ran after a write: maipipe stat err=%v", err)
			}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "no files were written") {
		t.Fatalf("refused update error = %v, want whole-plan no-write refusal", err)
	}
	if !called {
		t.Fatal("pre-write advisory callback was not called before refusal")
	}
	if got := readFile(t, wfPath); got != beforeWorkflow {
		t.Errorf("WORKFLOW.md changed after refusal:\nwant %q\n got %q", beforeWorkflow, got)
	}
	if _, err := os.Stat(filepath.Join(dir, MaipipeFile)); !os.IsNotExist(err) {
		t.Errorf("maipipe.toml created despite refusal: %v", err)
	}
}

const gen0HbmviewClaude = `# hbmview

Uses the **unified workflow** — see ` + "`WORKFLOW.md`" + ` for the active profile (` + "`rust`" + `) and stages.

- Specs / PRDs -> ` + "`docs/specs/`" + `
- Decisions (ADRs) -> ` + "`docs/adr/`" + `
- Issue / bug ledger -> ` + "`docs/issues/`" + ` (dependency convention in ` + "`docs/issues/README.md`" + `)
- Handoffs -> ` + "`docs/handoffs/`" + `

**Mandatory gates:** a PRD up front (grill-with-docs -> to-prd) and verification before completion.
**Model:** see ` + "`WORKFLOW.md`" + ` ` + "`model_default`" + ` (swappable).
`

func writeRepo(t *testing.T, workflow, claude string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if workflow != "" {
		if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if claude != "" {
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claude), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func report(t *testing.T, reports []FileReport, path string) FileReport {
	t.Helper()
	for _, r := range reports {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("no report for %s in %#v", path, reports)
	return FileReport{}
}

func TestDiffEmptyWhenEqual(t *testing.T) {
	if d := Diff("x", "a\nb", "a\nb"); d != "" {
		t.Errorf("got %q", d)
	}
	d := Diff("x", "a\nb\nc", "a\nB\nc")
	if !strings.Contains(d, "- b") || !strings.Contains(d, "+ B") || !strings.Contains(d, "  a") {
		t.Errorf("diff:\n%s", d)
	}
}

// Diff(path, "", content) is a brand-new file: every line is an addition,
// never a spurious "- " row for the phantom empty "old" line.
func TestDiffEmptyOldNoSpuriousMinusRow(t *testing.T) {
	d := Diff("x", "", "a\nb\n")
	for _, line := range strings.Split(d, "\n") {
		if strings.HasPrefix(line, "- ") {
			t.Errorf("unexpected '- ' row in empty-old diff:\n%s", d)
		}
	}
	if !strings.Contains(d, "+ a") || !strings.Contains(d, "+ b") {
		t.Errorf("missing expected '+' rows:\n%s", d)
	}
}

func TestFreshInitIsUpToDate(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("%s: state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestGen0CleanClaim(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("WORKFLOW state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	for _, want := range []string{"template_version: 13", "claude.primary:", "claude-fable-5",
		"profile: rust", "functional_harness: cli"} {
		if !strings.Contains(wf.newContent, want) {
			t.Errorf("regenerated WORKFLOW missing %q", want)
		}
	}
	// I036: the gen0 model_default (claude-opus-4-8) is an inherited default
	// of a retired key — it retires quietly rather than surviving or being
	// flagged as a divergence.
	if strings.Contains(wf.newContent, "model_default") {
		t.Error("retired model_default key survived regeneration")
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != Pending {
		t.Fatalf("CLAUDE state=%v", cl.State)
	}
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin v13 -->") {
		t.Error("claimed CLAUDE.md lacks markers")
	}
	if got := strings.Count(cl.newContent, "# hbmview"); got != 1 {
		t.Errorf("clean claim duplicated content, %d title lines", got)
	}
}

func TestGen0WriteThenUpToDate(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.State != UpToDate {
			t.Errorf("after write, %s state=%v diff:\n%s", r.Path, r.State, r.Diff)
		}
	}
}

func TestUnrecognizedEditsSkipUnlessForce(t *testing.T) {
	custom := gen0Hbmview + "custom_rule: never deploy on fridays\n"
	dir := writeRepo(t, custom, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != SkippedUnrecognized || len(wf.Unrecognized) != 1 ||
		!strings.Contains(wf.Unrecognized[0], "custom_rule") {
		t.Fatalf("state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	// force regenerates (dropping the line) and write applies it
	if _, err := Run(Options{Dir: dir, Write: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "WORKFLOW.md"))
	if strings.Contains(string(got), "custom_rule") {
		t.Error("force did not drop unrecognized line")
	}
	if !strings.Contains(string(got), "template_version: 13") {
		t.Error("force did not regenerate")
	}
}

func TestCustomChoiceSurvivesUpdate(t *testing.T) {
	custom := strings.Replace(gen0Hbmview, "functional_harness: cli", "functional_harness: rest", 1)
	dir := writeRepo(t, custom, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	wf := report(t, reports, "WORKFLOW.md")
	if wf.State != Pending {
		t.Fatalf("state=%v unrec=%v", wf.State, wf.Unrecognized)
	}
	if !strings.Contains(wf.newContent, "functional_harness: rest") {
		t.Error("user harness choice lost")
	}
}

func TestLegacyClaudeWithUserContentPreserved(t *testing.T) {
	userClaude := gen0HbmviewClaude + "\n## Local invariants\n\n- verify with `lms ps --json`\n"
	dir := writeRepo(t, gen0Hbmview, userClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != Pending {
		t.Fatalf("state=%v", cl.State)
	}
	if !strings.Contains(cl.newContent, "lms ps --json") {
		t.Error("user content dropped")
	}
	if !strings.HasPrefix(cl.newContent, "<!-- spine:begin") {
		t.Error("markers not inserted at top")
	}
}

func TestMarkerBlockReplacedUserTailKept(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(raw, []byte("\n## Notes\n\n- remote is github\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != UpToDate {
		t.Fatalf("same-version block should be up-to-date, state=%v diff:\n%s", cl.State, cl.Diff)
	}
}

func TestUnbalancedMarkersSkipped(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw, _ := os.ReadFile(path)
	broken := strings.Replace(string(raw), "<!-- spine:end -->", "", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	cl := report(t, reports, "CLAUDE.md")
	if cl.State != SkippedUnrecognized {
		t.Fatalf("unbalanced markers must skip even with force, state=%v", cl.State)
	}
}

// Scoped overwrite authority does not weaken the damaged-marker guard: a
// selected file lacking a complete machine-owned region has no regenerable
// content and must remain untouched.
func TestForceFileDoesNotOverwriteBrokenClaudeMarkers(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CLAUDE.md")
	raw := readFile(t, path)
	broken := strings.Replace(raw, "<!-- spine:end -->", "", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	reports, err := Run(Options{Dir: dir, Write: true, ForceFiles: []string{"CLAUDE.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := report(t, reports, "CLAUDE.md"); got.State != SkippedUnrecognized || !got.SelectedByForceFile {
		t.Fatalf("selected broken CLAUDE.md = %#v, want selected skipped report", got)
	}
	if got := readFile(t, path); got != broken {
		t.Fatal("selected broken CLAUDE.md changed under --force-file")
	}
}

func TestMissingWorkflowIsHardError(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want error when WORKFLOW.md missing")
	}
}

func TestUpdateKnowledgeManifest(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "knowledge", "vault"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "docs/harness-interface.md" || r.Path == "docs/issues/README.md" || r.Path == "docs/issues/_template.md" {
			t.Errorf("knowledge update must not manage %s", r.Path)
		}
		if r.State != UpToDate {
			t.Errorf("%s not up-to-date after fresh init", r.Path)
		}
	}
}

func TestUpdateManagesEvalsReadmeOnlyWhenPresent(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Path == "docs/evals/README.md" {
			t.Fatal("evals README managed without docs/evals/")
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "evals"), 0o755); err != nil {
		t.Fatal(err)
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range reports {
		if r.Path == "docs/evals/README.md" {
			found = true
			if r.State != Pending || !r.Created {
				t.Errorf("want Pending+Created, got state=%v created=%v", r.State, r.Created)
			}
		}
	}
	if !found {
		t.Fatal("evals README not planned despite docs/evals/ existing")
	}
}

func TestAdoptModeSynthesizesWorkflow(t *testing.T) {
	dir := t.TempDir()
	// pre-existing hand-authored CLAUDE.md, praxis-style
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("## Repo invariants\n\n- push with git push github main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, AdoptProfile: "go-service", AdoptName: "praxis"})
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]FileReport{}
	for _, r := range reports {
		byPath[r.Path] = r
	}
	wf := byPath["WORKFLOW.md"]
	if wf.State != Pending || !wf.Created {
		t.Fatalf("WORKFLOW.md state=%v created=%v", wf.State, wf.Created)
	}
	if !strings.Contains(wf.Diff, "profile: go-service") || !strings.Contains(wf.Diff, "template_version: 13") || !strings.Contains(wf.Diff, "# Workflow — praxis") {
		t.Errorf("diff=%q", wf.Diff)
	}
	cl := byPath["CLAUDE.md"]
	if cl.State != Pending || cl.Created {
		t.Fatalf("CLAUDE.md state=%v created=%v (want claim of existing file)", cl.State, cl.Created)
	}
	if !strings.Contains(cl.Diff, "spine:begin") || !strings.Contains(cl.Diff, "Repo invariants") {
		t.Errorf("claim must insert markers and keep hand content; diff=%q", cl.Diff)
	}
}

func TestMissingWorkflowStillErrorsWithoutAdoptMode(t *testing.T) {
	if _, err := Run(Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("want error")
	}
}

// C1: docs/adr/README.md is the one machine-owned file where unrecognized
// hand-authored content is preserved as-is, not skipped/warned. --force
// remains the explicit opt-in to regenerate it from the template.
func TestLegacyADRReadmePreserved(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	handAuthored := "# Architecture Decision Records\n\nSee the index below.\n\n| # | Decision |\n|---|---|\n| 0001 | Something |\n"
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "README.md"), []byte(handAuthored), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, reports, "docs/adr/README.md")
	if r.State != UpToDate || !r.Preserved || r.Diff != "" {
		t.Fatalf("state=%v preserved=%v diff=%q", r.State, r.Preserved, r.Diff)
	}
	// preserved files must not count as outstanding work.
	reports2, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, rr := range reports2 {
		if rr.Path == "docs/adr/README.md" && rr.State != UpToDate {
			t.Errorf("preserved file counted as outstanding: state=%v", rr.State)
		}
	}
	// --force is the explicit opt-in to regenerate from the template.
	forced, err := Run(Options{Dir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	fr := report(t, forced, "docs/adr/README.md")
	if fr.State != Pending || fr.Preserved {
		t.Fatalf("force: state=%v preserved=%v", fr.State, fr.Preserved)
	}
}

// Absent or template-matched docs/adr/README.md keeps existing behavior:
// create when missing, up-to-date (not preserved) when it already matches
// the template.
func TestFreshInitADRReadmeNotPreserved(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, reports, "docs/adr/README.md")
	if r.State != UpToDate || r.Preserved {
		t.Fatalf("state=%v preserved=%v (template-matched file must not be marked preserved)", r.State, r.Preserved)
	}
}

func TestVersionDowngradeGuard(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bumped := strings.Replace(string(raw), "template_version: 13", "template_version: 14", 1)
	if bumped == string(raw) {
		t.Fatal("template_version: 13 not found in scaffolded WORKFLOW.md to bump")
	}
	if err := os.WriteFile(path, []byte(bumped), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Run(Options{Dir: dir})
	if err == nil || !strings.Contains(err.Error(), "generation") {
		t.Fatalf("want error mentioning generation, got %v", err)
	}
}

func TestAgentsMdCreatedWhenMissing(t *testing.T) {
	// A repo with WORKFLOW.md + CLAUDE.md but no AGENTS.md (the state every
	// existing gen-6 repo is in) gains AGENTS.md on update.
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a := report(t, reports, "AGENTS.md")
	if a.State != Pending || !a.Created {
		t.Fatalf("AGENTS.md state=%v created=%v", a.State, a.Created)
	}
	if !strings.Contains(a.Diff, "read by **Codex**") {
		t.Errorf("AGENTS.md diff missing Codex-tuned body:\n%s", a.Diff)
	}
}

func TestAgentsMdMarkerReplacePreservesHandContent(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	// pre-place an AGENTS.md with a stale block + hand-authored tail.
	stale := "<!-- spine:begin v5 -->\nold codex brief\n<!-- spine:end -->\n\n## Local notes\nkeep me\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = reports
	got, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if strings.Contains(s, "old codex brief") {
		t.Error("stale block survived marker replacement")
	}
	if !strings.Contains(s, "## Local notes") || !strings.Contains(s, "keep me") {
		t.Error("hand-authored content outside markers was lost")
	}
	if strings.Count(s, "<!-- spine:begin") != 1 {
		t.Error("marker replacement duplicated the block")
	}
}

func TestAgentsMdUnbalancedMarkersFlagged(t *testing.T) {
	dir := writeRepo(t, gen0Hbmview, gen0HbmviewClaude)
	bad := "<!-- spine:begin v6 -->\nbody\n<!-- spine:begin v6 -->\n<!-- spine:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	a := report(t, reports, "AGENTS.md")
	if len(a.Unrecognized) == 0 {
		t.Fatal("unbalanced markers should be flagged, not clobbered")
	}
	if !strings.Contains(a.Unrecognized[0], "AGENTS.md") {
		t.Errorf("unbalanced-marker message should name AGENTS.md, got %q", a.Unrecognized[0])
	}
}

func TestClaudeAndAgentsBlocksShareVocabulary(t *testing.T) {
	vals := tmpl.Values{Project: "demo", Profile: "go-service", Version: tmpl.Version()}
	claude, err := tmpl.Render("current", "CLAUDE.md.tmpl", vals)
	if err != nil {
		t.Fatal(err)
	}
	agents, err := tmpl.Render("current", "AGENTS.md.tmpl", vals)
	if err != nil {
		t.Fatal(err)
	}
	// The two harness briefs must not drift on the load-bearing vocabulary.
	for _, term := range []string{
		"WORKFLOW.md", "docs/specs/", "docs/adr/", "docs/issues/", "docs/handoffs",
		"primary", "routine", "mechanical", "fallback",
		"verification before completion",
	} {
		if !strings.Contains(claude, term) {
			t.Errorf("CLAUDE.md.tmpl missing shared term %q", term)
		}
		if !strings.Contains(agents, term) {
			t.Errorf("AGENTS.md.tmpl missing shared term %q", term)
		}
	}
}

func TestRunSurfacesEvalsDirStatError(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	// docs/evals as a symlink loop: Stat fails with ELOOP, not ENOENT.
	if err := os.Symlink("evals", filepath.Join(dir, "docs", "evals")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir}); err == nil {
		t.Fatal("want Stat error surfaced, got nil (silently skipped evals-README before v3)")
	}
}

// A repo scaffolded at the *previous* generation must, on a plain update,
// gain AGENTS.md and advance its stamp — with no hand-written migration code.
func TestGen6RepoGainsAgentsMdOnUpdate(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "go-service", "demo"); err != nil {
		t.Fatal(err)
	}
	// Simulate a repo stamped at the prior generation.
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	downgraded := strings.Replace(string(raw), "template_version: 13", "template_version: 9", 1)
	if downgraded == string(raw) {
		t.Fatal("could not stage a prior-generation fixture")
	}
	if err := os.WriteFile(wfPath, []byte(downgraded), 0o644); err != nil {
		t.Fatal(err)
	}
	// Remove AGENTS.md so we're truly in the pre-A state.
	if err := os.Remove(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("update did not create AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), "<!-- spine:begin v13 -->") {
		t.Error("AGENTS.md not stamped at v13")
	}
	wfAfter, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wfAfter), "template_version: 13") {
		t.Error("WORKFLOW.md did not advance to gen 13")
	}
}
