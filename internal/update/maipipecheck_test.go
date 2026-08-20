package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/russellpope/spine/internal/gate"
)

// maipipeOnPATH reports whether the grammar check can run at all. Tests that
// are meaningless without the binary carry the condition in their name; the
// rest say in their log which half of the check they exercised.
//
// A test that returns early asserts nothing while still reporting PASS, which
// is the failure class this repo's own gate pack exists to catch. Setting
// SPINE_REQUIRE_MAIPIPE=1 turns the missing binary into a failure, so CI can
// state that the maipipe-dependent controls really ran.
func maipipeOnPATH(t *testing.T) bool {
	t.Helper()
	if _, err := exec.LookPath("maipipe"); err == nil {
		return true
	}
	if os.Getenv("SPINE_REQUIRE_MAIPIPE") == "1" {
		t.Fatal("SPINE_REQUIRE_MAIPIPE=1 but no maipipe on PATH: the maipipe-dependent controls would assert nothing")
	}
	return false
}

// movedStageFixture is the ticket's first negative control: a
// [[pipelines.gate-go.stage]] block lifted out of the region and dropped a
// few lines past `# spine:end`, the way a hand edit or a merge resolution
// leaves it. Returns the repo dir and the maipipe.toml path.
func movedStageFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, MaipipeFile)
	old := readFile(t, path)
	block := "\n[[pipelines." + gatePipelineName + ".stage]]\nname = \"tskip\"\nrun = \"spine gate " +
		gate.PackName + " tskip\"\n"
	if !strings.Contains(old, block) {
		t.Fatalf("fixture assumption broken, no tskip stage in:\n%s", old)
	}
	moved := strings.Replace(old, block, "", 1) + "\n# moved out here by hand" + block
	if err := os.WriteFile(path, []byte(moved), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

// duplicateTableFixture is the ticket's second negative control: a
// pre-existing [pipelines.gate-go] table outside the region, which spine
// appends the region blind after.
func duplicateTableFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := gateRepo(t, "[]", nil)
	path := filepath.Join(dir, MaipipeFile)
	existing := "schema = 0\n\n[pipelines." + gatePipelineName + "]\nprofile = \"fast\"\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

// AC 2 (I096): a stage moved past `# spine:end` gets re-rendered back inside
// the region, so the file would declare that stage twice. spine refuses the
// write, names the duplicate stage, and leaves the file alone — it does not
// move the stage back.
func TestMovedStageRefusesWrite_requiresMaipipeOnPATH(t *testing.T) {
	if !maipipeOnPATH(t) {
		t.Log("no maipipe on PATH: duplicate stage names are valid TOML, so this control needs the binary")
		return
	}
	dir, path := movedStageFixture(t)
	before := readFile(t, path)
	_, err := Run(Options{Dir: dir, Write: true})
	if err == nil {
		t.Fatal("spine wrote a maipipe.toml with a duplicate stage name")
	}
	msg := err.Error()
	for _, want := range []string{"refusing to write", MaipipeFile, "duplicate stage name", "tskip", gateRegionEnd} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal does not mention %q:\n%s", want, msg)
		}
	}
	if got := readFile(t, path); got != before {
		t.Errorf("file changed on disk despite the refusal:\n%s", got)
	}
}

// AC 3 (I096): a pre-existing [pipelines.gate-go] table outside the region
// makes the spliced result unparseable TOML. The refusal fires on the parse
// alone — no maipipe binary needed.
func TestPreExistingGatePipelineRefusesWrite(t *testing.T) {
	dir, path := duplicateTableFixture(t)
	before := readFile(t, path)
	_, err := Run(Options{Dir: dir, Write: true})
	if err == nil {
		t.Fatal("spine wrote a maipipe.toml that does not parse as TOML")
	}
	msg := err.Error()
	for _, want := range []string{"refusing to write", MaipipeFile, "duplicate table", gatePipelineName} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal does not mention %q:\n%s", want, msg)
		}
	}
	if got := readFile(t, path); got != before {
		t.Errorf("file changed on disk despite the refusal:\n%s", got)
	}
}

// AC 4 (I096), fixture 2: with the check removed — the candidate written
// straight to disk, which is what spine did before this ticket — the file is
// unloadable. Proves the guard is load-bearing rather than decorative.
func TestCheckIsLoadBearingForDuplicateTable(t *testing.T) {
	dir, path := duplicateTableFixture(t)
	candidate := pendingMaipipe(t, dir)
	if err := os.WriteFile(path, []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := parseTOML(candidate); err == nil {
		t.Fatal("check removed: the candidate parses as TOML, so the guard proves nothing")
	} else {
		t.Logf("check removed, file on disk is unloadable: %v", err)
	}
	if !maipipeOnPATH(t) {
		t.Log("no maipipe on PATH: unloadability shown by the TOML parse only")
		return
	}
	out, err := exec.Command("maipipe", "validate", path).CombinedOutput()
	if err == nil {
		t.Fatalf("maipipe loaded the file the guard refuses:\n%s", out)
	}
	t.Logf("maipipe validate on the unguarded write: %s", strings.TrimSpace(string(out)))
}

// AC 4 (I096), fixture 1: same proof for the moved stage — valid TOML, but
// maipipe cannot load it.
func TestCheckIsLoadBearingForMovedStage_requiresMaipipeOnPATH(t *testing.T) {
	if !maipipeOnPATH(t) {
		t.Log("no maipipe on PATH: this proof is about maipipe's grammar, not TOML's")
		return
	}
	dir, path := movedStageFixture(t)
	candidate := pendingMaipipe(t, dir)
	if err := os.WriteFile(path, []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := parseTOML(candidate); err != nil {
		t.Fatalf("fixture assumption broken, the moved stage should still be valid TOML: %v", err)
	}
	out, err := exec.Command("maipipe", "validate", path).CombinedOutput()
	if err == nil {
		t.Fatalf("maipipe loaded the duplicate-stage file:\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate stage name") {
		t.Errorf("want a duplicate stage name from maipipe, got:\n%s", out)
	}
	t.Logf("check removed, file on disk is unloadable: %s", strings.TrimSpace(string(out)))
}

// AC 5 (I096): with no maipipe resolvable, the TOML-parse refusal still
// fires and says the grammar check was skipped.
func TestNoMaipipeOnPATHStillRefusesAndSaysValidateSkipped(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := checkMaipipeContent("/repo/"+MaipipeFile, "schema = 0\n[pipelines.gate-go]\n[pipelines.gate-go]\n")
	if err == nil {
		t.Fatal("no refusal without maipipe on PATH")
	}
	if !strings.Contains(err.Error(), noMaipipeNote) {
		t.Errorf("refusal does not say validation was skipped:\n%s", err)
	}
	// And valid TOML still passes when the grammar check cannot run.
	if err := checkMaipipeContent("/repo/"+MaipipeFile, "schema = 0\n\n[pipelines.fast]\n"); err != nil {
		t.Errorf("valid TOML refused with no maipipe on PATH: %v", err)
	}
}

// AC 1 (I096): spine's own maipipe.toml — the real one at the repo root —
// parses and validates clean, and the normal `spine update --write` path is
// unchanged: the region still renders and the file is still written.
func TestPositiveControlRealRepoFileAndNormalWrite(t *testing.T) {
	own := readFile(t, filepath.Join("..", "..", MaipipeFile))
	if err := checkMaipipeContent(filepath.Join("..", "..", MaipipeFile), own); err != nil {
		t.Fatalf("spine's own maipipe.toml is refused by its own check: %v", err)
	}
	if maipipeOnPATH(t) {
		t.Log("maipipe on PATH: spine's own file checked against maipipe's grammar too")
	} else {
		t.Log("no maipipe on PATH: spine's own file checked against the TOML parse only")
	}
	dir := gateRepo(t, "[]", nil)
	reports, err := Run(Options{Dir: dir, Write: true})
	if err != nil {
		t.Fatalf("normal path refused: %v", err)
	}
	if mp := report(t, reports, MaipipeFile); mp.State != Pending {
		t.Fatalf("maipipe.toml state = %v, want Pending", mp.State)
	}
	written := readFile(t, filepath.Join(dir, MaipipeFile))
	if !strings.Contains(written, gateRegionBegin) {
		t.Fatalf("region not written:\n%s", written)
	}
	if err := checkMaipipeContent(filepath.Join(dir, MaipipeFile), written); err != nil {
		t.Errorf("freshly written maipipe.toml does not load: %v", err)
	}
}

// pendingMaipipe returns the content update would write for maipipe.toml —
// the candidate the check inspects — without writing anything.
func pendingMaipipe(t *testing.T, dir string) string {
	t.Helper()
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending || mp.newContent == "" {
		t.Fatalf("no pending maipipe.toml content: state=%v unrec=%v", mp.State, mp.Unrecognized)
	}
	return mp.newContent
}

// The parse is a real TOML check, not a duplicate-table special case: it
// reads maipipe's grammar-bearing shapes and refuses only what TOML refuses.
func TestParseTOML(t *testing.T) {
	valid := []string{
		"schema = 0\n\n[pipelines.fast]\n\n[[pipelines.fast.stage]]\nname = \"vet\"\nneeds = [\"a\", \"b\"]\n",
		// Quoted keys and quoted table-header segments: legal TOML that an
		// earlier cut of the scanner dropped along with the string, leaving
		// `[]` and ` = 1` behind. See TestScanKeepsQuotedSegments.
		"[pipelines.\"e2e.smoke\"]\nprofile = \"full\"\n",
		"[[pipelines.\"e2e smoke\".stage]]\nname = \"x\"\n",
		"\"my key\" = 1\n\"other key\" = 2\n",
		"a.\"b.c\" = 1\na . b = 2\n",
		"[pipelines.\"a\"]\n[pipelines.\"b\"]\n",
		// A standard table under an array-of-tables entry belongs to that
		// entry, so the pair repeats legally.
		"[[a]]\n[a.b]\nk = 1\n\n[[a]]\n[a.b]\nk = 2\n",
		"x = \"\"\"one line\"\"\"\ny = 1\n",
		"lit = 'C:\\path\\'\nz = 2\n",
		"esc = \"a \\\" b\"\nq = 3\n",
		"[a] # trailing comment\nk = 1\n",
		// The equality rule cuts the other way too: `"a.b"` is one key and
		// `a.b` is two, so these are genuinely different and both stand.
		"\"a.b\" = 1\n\na.b = 2\n",
		"[\"a.b\"]\n\n[a.b]\n",
		// A literal string keeps its backslashes: 'a\b' is a\b, which a
		// tab escape is not.
		"'a\\b' = 1\n\"a\\tb\" = 2\n",
		"schema = 0\r\n\r\n[pipelines.fast]\r\n",
		"# only comments\n\n",
		"a = \"] # [not a comment\"\nb = 1\n",
		"env = { A = \"x\", B = \"y\" }\n",
		"needs = [\n  \"a\",\n  \"b\",\n]\n",
		"text = \"\"\"\n[not.a.table]\n\"\"\"\nk = 1\n",
		"[a.b]\n[a]\n[[a.c]]\nname = \"x\"\n\n[[a.c]]\nname = \"x\"\n",
	}
	for _, in := range valid {
		if err := parseTOML(in); err != nil {
			t.Errorf("valid TOML refused: %v\n%s", err, in)
		}
	}
	invalid := []string{
		"[pipelines.gate-go]\n[pipelines.gate-go]\n",
		"[pipelines.\"a\"]\n[pipelines.\"a\"]\n",
		"\"my key\" = 1\n\"my key\" = 2\n",
		// Fix round 2, regression 2: TOML makes a quoted key equal to the
		// bare key with the same text, so each of these declares one thing
		// twice. Keying the quoted and bare spellings separately would let
		// the guard write a file maipipe cannot load — and on the no-binary
		// path this parse is the only check there is.
		"[pipelines.\"a\"]\n[pipelines.a]\n",
		"[pipelines.a]\n[pipelines.'a']\n",
		"\"a\" = 1\na = 2\n",
		"a.\"b\" = 1\na.b = 2\n",
		"'a' = 1\n\"a\" = 2\n",
		// An escape decodes before the comparison: "ab" is `ab`.
		"\"a\\u0062\" = 1\nab = 2\n",
		"[[a]]\n[a.\"b\"]\n[a.b]\n",
		// 'a\b' (literal, backslash kept) and "a\\b" (escaped backslash)
		// decode to the same key.
		"'a\\b' = 1\n\"a\\\\b\" = 2\n",
		"[[a]]\n[a]\n",
		"[a]\nk = 1\nk = 2\n",
		"[pipelines.fast\nname = \"x\"\n",
		"just some prose\n",
		"needs = [\"a\",\n",
		"text = \"\"\"\nunterminated\n",
		"name = \"unterminated\n",
	}
	for _, in := range invalid {
		if err := parseTOML(in); err == nil {
			t.Errorf("invalid TOML accepted:\n%s", in)
		}
	}
}

// Fix round 1, Important 1 — the negative control for the quoted-key fix.
// The bug was structural: the scanner dropped a string's text instead of
// standing in for it, so `[pipelines."e2e.smoke"]` reached the header parser
// as `[]` and `"my key" = 1` as ` = 1`. This pins the property whose absence
// caused it — a consumed string leaves a placeholder behind and restores to
// exactly what the file said — so a scanner that drops strings again fails
// here, not only in the parse table.
func TestScanKeepsQuotedSegments(t *testing.T) {
	for _, tc := range []struct {
		line, want string
	}{
		{`[pipelines."e2e.smoke"]`, `[pipelines."e2e.smoke"]`},
		{`"my key" = 1`, `"my key" = 1`},
		{`run = "spine gate go tskip" # note`, `run = "spine gate go tskip" `},
	} {
		var st tomlScan
		code, strs, err := st.scan(tc.line)
		if err != nil {
			t.Fatalf("scan(%q): %v", tc.line, err)
		}
		if len(strs) == 0 {
			t.Errorf("scan(%q) consumed no string: dropped, not replaced", tc.line)
		}
		if !strings.Contains(code, strPlaceholder) {
			t.Errorf("scan(%q) = %q, want a placeholder standing in for the string", tc.line, code)
		}
		if got := restoreStrings(code, strs); got != tc.want {
			t.Errorf("restoreStrings(scan(%q)) = %q, want %q", tc.line, got, tc.want)
		}
	}
}

// Fix round 1, Important 1 — cross-check against the grammar authority: a
// quoted pipeline name is a file maipipe really does load, so accepting it
// is not spine's opinion of TOML but maipipe's.
func TestQuotedPipelineNameLoads_requiresMaipipeOnPATH(t *testing.T) {
	if !maipipeOnPATH(t) {
		t.Log("no maipipe on PATH: the quoted-name case is checked by the parse table only")
		return
	}
	content := "schema = 0\n\n[pipelines.\"e2e.smoke\"]\nprofile = \"fast\"\n\n[[pipelines.\"e2e.smoke\".stage]]\nname = \"x\"\nrun = \"true\"\n"
	if err := checkMaipipeContent(filepath.Join(t.TempDir(), MaipipeFile), content); err != nil {
		t.Fatalf("spine refuses a maipipe.toml maipipe loads: %v", err)
	}
}

// Fix round 1, Important 2 (controller ruling) — the refusal is all-or-
// nothing by design, and has to say so: a pending WORKFLOW.md is left
// unwritten too, and the message tells the reader that rather than naming
// maipipe.toml as if it were the only casualty.
func TestRefusalLeavesEveryPendingFileUnwrittenAndSaysSo(t *testing.T) {
	dir, mpPath := duplicateTableFixture(t)
	wfPath := filepath.Join(dir, "WORKFLOW.md")
	// Restamp WORKFLOW.md to an older generation so update has a real
	// pending change for it alongside the doomed maipipe.toml.
	before := setKey(readFile(t, wfPath), "template_version", "10")
	if err := os.WriteFile(wfPath, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if wf := report(t, plan, "WORKFLOW.md"); wf.State != Pending {
		t.Fatalf("fixture assumption broken: WORKFLOW.md state = %v, want Pending", wf.State)
	}
	mpBefore := readFile(t, mpPath)
	_, err = Run(Options{Dir: dir, Write: true})
	if err == nil {
		t.Fatal("no refusal")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no files were written") || !strings.Contains(msg, "WORKFLOW.md") {
		t.Errorf("refusal does not say the whole run was abandoned:\n%s", msg)
	}
	if got := readFile(t, wfPath); got != before {
		t.Error("WORKFLOW.md was written despite the refusal")
	}
	if got := readFile(t, mpPath); got != mpBefore {
		t.Error("maipipe.toml was written despite the refusal")
	}
}

// fakeMaipipe puts a stand-in `maipipe` on PATH for one test and returns its
// directory. body is the shell script the fake runs.
func fakeMaipipe(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "maipipe"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return dir
}

// Fix round 2, regression 1: CommandContext kills the child when the
// deadline fires, so Wait reports an *exec.ExitError and an errors.As test
// alone reads a timeout as maipipe's verdict on the file. A validator that
// never answered says nothing about the content, and telling the reader
// their file was rejected sends them hunting for a defect that is not there.
func TestValidateTimeoutIsNotAVerdict(t *testing.T) {
	// exec, so the killed process *is* the sleep: a shell wrapper would
	// leave the sleep holding the output pipe and CombinedOutput would
	// block for its full duration despite the deadline.
	fakeMaipipe(t, "exec /bin/sleep 5")
	old := maipipeTimeout
	maipipeTimeout = 150 * time.Millisecond
	t.Cleanup(func() { maipipeTimeout = old })

	err := checkMaipipeContent("/repo/"+MaipipeFile, "schema = 0\n\n[pipelines.fast]\n")
	if err == nil {
		t.Fatal("a validator that never answered was treated as approval")
	}
	msg := err.Error()
	if !strings.Contains(msg, "did not finish within") {
		t.Errorf("refusal does not say the validator timed out:\n%s", msg)
	}
	if strings.Contains(msg, "rejected the result") {
		t.Errorf("a timeout is reported as a verdict on the file:\n%s", msg)
	}
}

// The other half of the same distinction: a maipipe that fails immediately
// must not borrow the timeout wording. (A non-zero exit is indistinguishable
// from a real rejection without parsing maipipe's output, and reads as a
// verdict — which is correct for the case this guard exists to catch.)
func TestValidatePromptFailureIsNotReportedAsTimeout(t *testing.T) {
	fakeMaipipe(t, "exit 127")
	err := checkMaipipeContent("/repo/"+MaipipeFile, "schema = 0\n\n[pipelines.fast]\n")
	if err == nil {
		t.Fatal("exit 127 accepted as approval")
	}
	// exit 127 *is* an ExitError, so it reads as a verdict — which is what
	// a real maipipe reporting a bad file also looks like. What must not
	// happen is the timeout wording; the verdict wording is correct here.
	if strings.Contains(err.Error(), "did not finish within") {
		t.Errorf("a prompt failure is reported as a timeout:\n%s", err)
	}
}

// Fix round 2, regression 2 — the equality rule is maipipe's, not spine's
// opinion of TOML: the same mixed quoted/bare pair the parse now refuses is
// a file maipipe refuses too, and refuses for the same reason.
func TestQuotedAndBareKeyAreOneTable_requiresMaipipeOnPATH(t *testing.T) {
	if !maipipeOnPATH(t) {
		t.Log("no maipipe on PATH: the equality rule is checked by the parse table only")
		return
	}
	content := "schema = 0\n\n[pipelines.\"a\"]\nprofile = \"fast\"\n\n[pipelines.a]\nprofile = \"fast\"\n"
	if err := parseTOML(content); err == nil {
		t.Error("spine's parse accepts a quoted/bare duplicate")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, MaipipeFile)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("maipipe", "validate", path).CombinedOutput()
	if err == nil {
		t.Fatalf("fixture assumption broken: maipipe loads the quoted/bare duplicate:\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate key") {
		t.Errorf("want maipipe to call it a duplicate key, got:\n%s", out)
	}
	t.Logf("maipipe agrees: %s", strings.TrimSpace(string(out)))
}
