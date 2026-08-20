package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/gate"
)

// maipipeOnPATH reports whether the grammar check can run at all. Tests that
// are meaningless without the binary carry the condition in their name; the
// rest say in their log which half of the check they exercised.
func maipipeOnPATH(t *testing.T) bool {
	t.Helper()
	_, err := exec.LookPath("maipipe")
	return err == nil
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
