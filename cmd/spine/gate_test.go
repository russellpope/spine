package main

import (
	"encoding/json"

	"github.com/russellpope/spine/internal/gate"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gateRepo builds a real git repo in a tempdir with the given files
// (slash-separated relative paths -> content) and commits them. The
// binary-hygiene check class reads the tracked set from git, so its
// fixtures have to be real repos, and binary fixture bytes are generated
// here rather than committed into spine's own testdata.
func gateRepo(t *testing.T, files map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "spine@example.test"},
		{"config", "user.name", "spine test"},
		{"add", "-A"},
		{"-c", "commit.gpgsign=false", "commit", "-q", "-m", "fixture"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// goodTestFile is the tskip negative control: it contains a Skip token, but
// on an unrelated receiver, so a class that matched any ident receiver would
// fire here.
const goodTestFile = `package fixture

import "testing"

type queue struct{}

func (queue) Skip() {}

func TestGood(t *testing.T) {
	var q queue
	q.Skip()
	if 1+1 != 2 {
		t.Fatal("arithmetic")
	}
}
`

// skippedTestFile seeds all three receiver shapes the class must catch: the
// conventional t, a testing.TB helper parameter, and the suite-style T()
// accessor.
const skippedTestFile = `package fixture

import "testing"

type suite struct{ t *testing.T }

func (s *suite) T() *testing.T { return s.t }

func TestSkipped(t *testing.T) {
	t.Skip("seeded violation")
}

func skipHelper(tb testing.TB) {
	tb.SkipNow()
}

func TestSuiteSkipped(t *testing.T) {
	s := &suite{t: t}
	s.T().Skip("seeded violation")
}
`

func tskipFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":           []byte("module fixture\n\ngo 1.26\n"),
		"pkg/good_test.go": []byte(goodTestFile),
		"README.md":        []byte("fixture\n"),
	}
	seeded = map[string][]byte{
		"go.mod":              []byte("module fixture\n\ngo 1.26\n"),
		"pkg/good_test.go":    []byte(goodTestFile),
		"pkg/skipped_test.go": []byte(skippedTestFile),
	}
	return good, seeded
}

// elfBytes is a minimal file whose header matches the ELF signature —
// generated here so no executable-looking fixture is committed to spine.
func elfBytes() []byte {
	b := make([]byte, 64)
	copy(b, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	return b
}

// tarBytes is a file whose header carries the tar signature at offset 257 —
// the one magic in the list that is not at offset 0, and so the one a short
// read would silently skip.
func tarBytes() []byte {
	b := make([]byte, 512)
	copy(b, []byte("fixture.txt"))
	copy(b[257:], []byte("ustar\x0000"))
	return b
}

func binaryHygieneFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":    []byte("module fixture\n\ngo 1.26\n"),
		"main.go":   []byte("package main\n\nfunc main() {}\n"),
		"docs/x.md": []byte("notes\n"),
	}
	seeded = map[string][]byte{
		"go.mod":       []byte("module fixture\n\ngo 1.26\n"),
		"main.go":      []byte("package main\n\nfunc main() {}\n"),
		"bin/tool":     elfBytes(),
		"data/x.tar":   tarBytes(),
		"tools/go.mod": []byte("module fixture/tools\n\ngo 1.26\n"),
	}
	return good, seeded
}

// gitignoreControlFixtures: the good repo ignores its declared build
// output and nothing else; the seeded one inverts both arms — the build
// output is tracked-able and an entry point is hidden by an ignore rule.
func gitignoreControlFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":     []byte("module fixture\n\ngo 1.26\n"),
		".gitignore": []byte("/bin/\n"),
		"main.go":    []byte("package main\n\nfunc main() {}\n"),
		"pkg/lib.go": []byte("package pkg\n"),
	}
	seeded = map[string][]byte{
		"go.mod":             []byte("module fixture\n\ngo 1.26\n"),
		".gitignore":         []byte("/cmd/hidden/\n"),
		"main.go":            []byte("package main\n\nfunc main() {}\n"),
		"cmd/hidden/main.go": []byte("package main\n\nfunc main() {}\n"),
	}
	return good, seeded
}

func fixtureManifestFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":               []byte("module fixture\n\ngo 1.26\n"),
		"testdata/manifest.md": []byte("- case: empty input\n- case: one row\n"),
	}
	seeded = map[string][]byte{
		"go.mod":               []byte("module fixture\n\ngo 1.26\n"),
		"testdata/manifest.md": []byte("\n   \n"),
	}
	return good, seeded
}

// enumSpec is the spec side of test-enum-vs-spec: one marked block whose
// backticked tokens are the documented values of one type.
const enumSpec = `# Fixture spec

Severity values:

<!-- spine:enum Severity -->
` + "`low`, `med`, `high`" + `
<!-- /spine:enum -->
`

const enumCode = `package fixture

type Severity string

const (
	Low  Severity = "low"
	Med  Severity = "med"
	High Severity = "high"
)
`

func testEnumVsSpecFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":       []byte("module fixture\n\ngo 1.26\n"),
		"docs/spec.md": []byte(enumSpec),
		"severity.go":  []byte(enumCode),
	}
	seeded = map[string][]byte{
		"go.mod": []byte("module fixture\n\ngo 1.26\n"),
		// The spec documents a value no const declares, and the code
		// declares one the spec never mentions: one finding per side.
		"docs/spec.md": []byte(strings.Replace(enumSpec, "`high`", "`high`, `retired`", 1)),
		"severity.go":  []byte(strings.Replace(enumCode, ")\n", "\tCrit Severity = \"crit\"\n)\n", 1)),
	}
	return good, seeded
}

// deferredCleanupGood is the negative control for the cleanup class: the
// two shapes that are not findings. A deferred func literal inspects the
// error, and a Close that returns nothing has no error to discard — a class
// matching on the name alone would fire on both.
const deferredCleanupGood = `package fixture

import "os"

type quietCloser struct{}

func (quietCloser) Close() {}

func report(err error) { _ = err }

func Copy(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			report(cerr)
		}
	}()
	var q quietCloser
	defer q.Close()
	return nil
}
`

const deferredCleanupSeeded = `package fixture

import "os"

func Read(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func Scratch(path string) error {
	defer os.RemoveAll(path)
	return nil
}
`

func deferredCleanupFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":  []byte("module example.com/fixture\n\ngo 1.22\n"),
		"copy.go": []byte(deferredCleanupGood),
	}
	seeded = map[string][]byte{
		"go.mod":  []byte("module example.com/fixture\n\ngo 1.22\n"),
		"copy.go": []byte(deferredCleanupGood),
		"read.go": []byte(deferredCleanupSeeded),
	}
	return good, seeded
}

// deadCodeFixtures: a library module with a main package. The good repo
// holds exactly the three shapes the root rule must keep live — an exported
// function of a library package, a function reached only from a test, and
// main itself; the seeded repo adds one unexported function nothing reaches.
func deadCodeFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod": []byte("module example.com/fixture\n\ngo 1.22\n"),
		"lib/lib.go": []byte(`package lib

// Exported is the library's API: no caller in this module, live by the
// library rule.
func Exported() string { return helper() }

// testOnly is reached only from lib_test.go.
func testOnly() string { return "test-only" }

func helper() string { return "helper" }
`),
		"lib/lib_test.go": []byte(`package lib

import "testing"

func TestTestOnly(t *testing.T) {
	if testOnly() == "" {
		t.Fatal("empty")
	}
}
`),
		// printed is the interface-satisfaction control, and it lives in the
		// main package on purpose: there the exported-library-API rule does
		// not apply, so its String is live only because fmt calls it through
		// fmt.Stringer. Nothing in the module names the method.
		"cmd/app/main.go": []byte(`package main

import (
	"fmt"

	"example.com/fixture/lib"
)

type printed int

func (p printed) String() string { return "printed" }

func main() { fmt.Println(lib.Exported(), printed(1)) }
`),
	}
	seeded = map[string][]byte{}
	for k, v := range good {
		seeded[k] = v
	}
	seeded["lib/dead.go"] = []byte(`package lib

func unreached() string { return "nobody calls me" }
`)
	return good, seeded
}

const nPlusOneGood = `package fixture

type client struct{}

func (client) Query(id int) int { return id }

// Batch makes one call and loops over the result: the shape the class must
// not flag.
func Batch(c client, ids []int) int {
	total := c.Query(len(ids))
	for range ids {
		total++
	}
	return total
}
`

const nPlusOneSeeded = `package fixture

// PerRow is one round trip per iteration, directly and through a func
// literal declared in the loop body.
func PerRow(c client, ids []int) int {
	total := 0
	for _, id := range ids {
		total += c.Query(id)
	}
	for i := 0; i < len(ids); i++ {
		func() { total += c.Query(i) }()
	}
	return total
}
`

func nPlusOneFixtures() (good, seeded map[string][]byte) {
	good = map[string][]byte{
		"go.mod":    []byte("module example.com/fixture\n\ngo 1.22\n"),
		"client.go": []byte(nPlusOneGood),
	}
	seeded = map[string][]byte{
		"go.mod":    []byte("module example.com/fixture\n\ngo 1.22\n"),
		"client.go": []byte(nPlusOneGood),
		"perrow.go": []byte(nPlusOneSeeded),
	}
	return good, seeded
}

// TestGatePositiveControls is the positive control pair for each check
// class: a known-good repo the class passes (exit 0) and a seeded violation
// it fails (exit 1), with each finding attributable to go@1/<check>.
func TestGatePositiveControls(t *testing.T) {
	tskipGood, tskipSeeded := tskipFixtures()
	binGood, binSeeded := binaryHygieneFixtures()
	ignGood, ignSeeded := gitignoreControlFixtures()
	manGood, manSeeded := fixtureManifestFixtures()
	enumGood, enumSeeded := testEnumVsSpecFixtures()
	deferGood, deferSeeded := deferredCleanupFixtures()
	deadGood, deadSeeded := deadCodeFixtures()
	nplusGood, nplusSeeded := nPlusOneFixtures()
	cases := []struct {
		check  string
		good   map[string][]byte
		seeded map[string][]byte
		env    map[string]string // gate_pack_config for this class
		wantN  int               // findings expected from the seeded run
		want   []string          // substrings expected in the seeded run's findings
	}{
		{"tskip", tskipGood, tskipSeeded, nil, 3, []string{"t.Skip call", "tb.SkipNow call", "s.T().Skip call"}},
		{"binary-hygiene", binGood, binSeeded, nil, 3, []string{"bin/tool", "data/x.tar", "tools/go.mod"}},
		{
			"gitignore-control", ignGood, ignSeeded,
			map[string]string{"SPINE_GATE_BUILD_OUTPUTS": "bin/spine"}, 2,
			[]string{"declared build output not ignored", "ignored entry point", "cmd/hidden/main.go"},
		},
		{
			"fixture-manifest", manGood, manSeeded,
			map[string]string{"SPINE_GATE_FIXTURE_MANIFEST": "testdata/manifest.md"}, 1,
			[]string{"fixture manifest empty", "testdata/manifest.md"},
		},
		{
			"test-enum-vs-spec", enumGood, enumSeeded,
			map[string]string{"SPINE_GATE_TEST_ENUM_SPEC": "docs/spec.md"}, 2,
			[]string{"is declared in code but not enumerated", "no const declares it", "severity.go", "docs/spec.md"},
		},
		{
			"deferred-cleanup-errcheck", deferGood, deferSeeded, nil, 2,
			[]string{"deferred cleanup call discards its error", "f.Close", "os.RemoveAll", "read.go"},
		},
		{
			"dead-code-callgraph", deadGood, deadSeeded, nil, 1,
			[]string{"unreachable function", "lib.unreached", "lib/dead.go"},
		},
		{
			"n-plus-one", nplusGood, nplusSeeded,
			map[string]string{"SPINE_GATE_N_PLUS_ONE_CLIENTS": "Query,Fetch"}, 2,
			[]string{"call in loop", "Query", "perrow.go"},
		},
	}
	for _, tc := range cases {
		setGateEnv := func(t *testing.T) {
			t.Helper()
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
		}
		t.Run(tc.check+"/good", func(t *testing.T) {
			setGateEnv(t)
			dir := gateRepo(t, tc.good)
			code, out, errs := runCmd(t, "gate", "go", tc.check, "--dir", dir)
			if code != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			if !strings.Contains(out, "no findings") {
				t.Errorf("stdout=%q", out)
			}
			if errs != "" {
				t.Errorf("stderr not pristine: %q", errs)
			}
		})
		t.Run(tc.check+"/seeded", func(t *testing.T) {
			setGateEnv(t)
			dir := gateRepo(t, tc.seeded)
			results := filepath.Join(t.TempDir(), "results.json")
			t.Setenv("MAIPIPE_RESULTS", results)
			code, out, errs := runCmd(t, "gate", "go", tc.check, "--dir", dir)
			if code != 1 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			r := readResults(t, results)
			if r.Status != "fail" || len(r.Findings) != tc.wantN {
				t.Fatalf("status=%q findings=%+v", r.Status, r.Findings)
			}
			for i, f := range r.Findings {
				if f.Code != "go@1/"+tc.check {
					t.Errorf("finding %d code=%q", i, f.Code)
				}
				if f.Severity == "" || f.Message == "" || f.File == "" {
					t.Errorf("finding %d incomplete: %+v", i, f)
				}
			}
			for _, want := range tc.want {
				if !strings.Contains(string(mustJSON(t, r)), want) {
					t.Errorf("results missing %q: %+v", want, r.Findings)
				}
			}
		})
	}
}

type gateResults struct {
	MaipipeResults *int   `json:"maipipe_results"`
	Status         string `json:"status"`
	Summary        string `json:"summary"`
	Findings       []struct {
		Severity string `json:"severity"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Code     string `json:"code"`
	} `json:"findings"`
}

func readResults(t *testing.T, path string) gateResults {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("results file: %v", err)
	}
	var r gateResults
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("results JSON: %v\n%s", err, b)
	}
	if r.MaipipeResults == nil || *r.MaipipeResults != 0 {
		t.Fatalf("maipipe_results key missing or non-zero: %s", b)
	}
	if r.Status == "" || r.Summary == "" {
		t.Fatalf("required keys missing: %s", b)
	}
	return r
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestGateResultsFileOnlyWhenEnvSet asserts the JSON-vs-table split: with
// MAIPIPE_RESULTS set a file is written; without it, stdout carries the
// human table and no file appears.
func TestGateResultsFileOnlyWhenEnvSet(t *testing.T) {
	_, seeded := tskipFixtures()
	dir := gateRepo(t, seeded)
	results := filepath.Join(t.TempDir(), "results.json")

	code, out, _ := runCmd(t, "gate", "go", "tskip", "--dir", dir)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(out, "severity") || !strings.Contains(out, "go@1/tskip") {
		t.Errorf("expected a human table on stdout, got %q", out)
	}
	if _, err := os.Stat(results); !os.IsNotExist(err) {
		t.Errorf("results file written without MAIPIPE_RESULTS: %v", err)
	}

	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, _ = runCmd(t, "gate", "go", "tskip", "--dir", dir)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if out != "" {
		t.Errorf("stdout should be empty when results go to a file: %q", out)
	}
	if _, err := os.Stat(results); err != nil {
		t.Errorf("results file not written: %v", err)
	}
}

// TestGateResultsDeterministic asserts byte-identical results across two
// runs, the premise that lets pipelines diff findings.
func TestGateResultsDeterministic(t *testing.T) {
	_, seeded := binaryHygieneFixtures()
	dir := gateRepo(t, seeded)
	var prev []byte
	for i := 0; i < 2; i++ {
		results := filepath.Join(t.TempDir(), "results.json")
		t.Setenv("MAIPIPE_RESULTS", results)
		if code, _, errs := runCmd(t, "gate", "go", "binary-hygiene", "--dir", dir); code != 1 {
			t.Fatalf("code=%d stderr=%q", code, errs)
		}
		b, err := os.ReadFile(results)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 && string(b) != string(prev) {
			t.Errorf("results not deterministic:\n%s\n---\n%s", prev, b)
		}
		prev = b
	}
}

// TestGateTskipAllowlist: an allowlisted entry is not a finding, by file and
// by path:line. The unset case is covered by the positive control above —
// no allowlist means zero tolerance, not misconfiguration.
func TestGateTskipAllowlist(t *testing.T) {
	_, seeded := tskipFixtures()
	dir := gateRepo(t, seeded)
	for _, allow := range []string{"pkg/skipped_test.go", "pkg/skipped_test.go:10,pkg/skipped_test.go:14,pkg/skipped_test.go:19", " , pkg/skipped_test.go "} {
		t.Setenv("SPINE_GATE_TSKIP_ALLOW", allow)
		code, out, errs := runCmd(t, "gate", "go", "tskip", "--dir", dir)
		if code != 0 {
			t.Errorf("allow=%q: code=%d stdout=%q stderr=%q", allow, code, out, errs)
		}
	}
	// A non-matching line number still fails: the allowlist is per call.
	t.Setenv("SPINE_GATE_TSKIP_ALLOW", "pkg/skipped_test.go:999")
	if code, _, _ := runCmd(t, "gate", "go", "tskip", "--dir", dir); code != 1 {
		t.Errorf("stale line allowlist should not suppress: code=%d", code)
	}
	// A malformed entry is misconfiguration and names the variable.
	t.Setenv("SPINE_GATE_TSKIP_ALLOW", "pkg/skipped_test.go:notaline")
	code, _, errs := runCmd(t, "gate", "go", "tskip", "--dir", dir)
	if code != 2 || !strings.Contains(errs, "SPINE_GATE_TSKIP_ALLOW") {
		t.Errorf("code=%d stderr=%q", code, errs)
	}
}

func TestGateMisconfiguration(t *testing.T) {
	good, _ := tskipFixtures()
	dir := gateRepo(t, good)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown pack", []string{"gate", "rust", "tskip", "--dir", dir}, "unknown pack"},
		{"unknown check", []string{"gate", "go", "bogus", "--dir", dir}, "unknown check"},
		{"missing args", []string{"gate", "go"}, "usage: spine gate"},
		{"dir not a directory", []string{"gate", "go", "tskip", "--dir", filepath.Join(dir, "go.mod")}, "not a directory"},
		{"dir missing", []string{"gate", "go", "tskip", "--dir", filepath.Join(dir, "nope")}, "--dir"},
		{"not a git repo", []string{"gate", "go", "binary-hygiene", "--dir", t.TempDir()}, "git"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errs := runCmd(t, tc.args...)
			if code != 2 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			if !strings.Contains(errs, tc.want) {
				t.Errorf("stderr=%q, want %q", errs, tc.want)
			}
		})
	}
}

func TestGateUsageDocumentsPack(t *testing.T) {
	code, out, _ := runCmd(t, "help")
	if code != 0 || !strings.Contains(out, "gate") {
		t.Fatalf("spine help does not mention gate: %q", out)
	}
	_, _, errs := runCmd(t, "gate", "go")
	wants := []string{"go@1", "SPINE_GATE_TSKIP_ALLOW", "SPINE_GATE_BUILD_OUTPUTS", "SPINE_GATE_FIXTURE_MANIFEST", "SPINE_GATE_TEST_ENUM_SPEC", "SPINE_GATE_N_PLUS_ONE_CLIENTS", "SPINE_GATE_CLEANUP_FUNCS", "MAIPIPE_RESULTS", "0 pass, 1 findings, 2 misconfiguration"}
	// The check list is derived from the registry, so a class that ships
	// without a usage entry is a test failure, not a documentation drift.
	wants = append(wants, gate.CheckNames()...)
	for _, want := range wants {
		if !strings.Contains(errs, want) {
			t.Errorf("gate usage missing %q:\n%s", want, errs)
		}
	}
}

// TestGateGitignoreControlArmsIndependent is the acceptance criterion for
// the hidden-entry-point control: a repo that correctly ignores its build
// outputs (arm 1 clean) still fails on an ignored package main file.
func TestGateGitignoreControlArmsIndependent(t *testing.T) {
	dir := gateRepo(t, map[string][]byte{
		"go.mod":             []byte("module fixture\n\ngo 1.26\n"),
		".gitignore":         []byte("/bin/\n/cmd/hidden/\n"),
		"main.go":            []byte("package main\n\nfunc main() {}\n"),
		"cmd/hidden/main.go": []byte("package main\n\nfunc main() {}\n"),
	})
	t.Setenv("SPINE_GATE_BUILD_OUTPUTS", "bin/spine")
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs := runCmd(t, "gate", "go", "gitignore-control", "--dir", dir)
	if code != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	r := readResults(t, results)
	if len(r.Findings) != 1 {
		t.Fatalf("want exactly the arm 2 finding, got %+v", r.Findings)
	}
	f := r.Findings[0]
	if !strings.Contains(f.Message, "ignored entry point") || f.File != "cmd/hidden/main.go" || f.Line != 1 {
		t.Errorf("finding=%+v", f)
	}
}

// TestGateFixtureManifestMissing: an absent manifest is a finding, not
// misconfiguration — the manifest is what the class checks.
func TestGateFixtureManifestMissing(t *testing.T) {
	dir := gateRepo(t, map[string][]byte{"go.mod": []byte("module fixture\n\ngo 1.26\n")})
	t.Setenv("SPINE_GATE_FIXTURE_MANIFEST", "testdata/manifest.md")
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	if code, _, errs := runCmd(t, "gate", "go", "fixture-manifest", "--dir", dir); code != 1 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	r := readResults(t, results)
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "fixture manifest missing") {
		t.Fatalf("findings=%+v", r.Findings)
	}
}

// TestGateDeadCodeRootRule is the AC2 rule fixture: in a library module,
// exactly the unreachable unexported function is flagged — a function
// reached only from a test is live, and an exported function of a library
// package is live because a library's callers are outside the module.
func TestGateDeadCodeRootRule(t *testing.T) {
	_, seeded := deadCodeFixtures()
	dir := gateRepo(t, seeded)
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs := runCmd(t, "gate", "go", "dead-code-callgraph", "--dir", dir)
	if code != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	r := readResults(t, results)
	if len(r.Findings) != 1 {
		t.Fatalf("want exactly the unreachable function, got %+v", r.Findings)
	}
	f := r.Findings[0]
	if !strings.Contains(f.Message, "lib.unreached") || f.File != "lib/dead.go" || f.Line != 3 {
		t.Errorf("finding=%+v", f)
	}
}

// TestGateNPlusOneCallSites pins the call-in-loop rule to the two seeded
// sites: the direct call in a range body and the one in a func literal
// declared inside a for body. The call outside any loop is not a finding.
func TestGateNPlusOneCallSites(t *testing.T) {
	_, seeded := nPlusOneFixtures()
	dir := gateRepo(t, seeded)
	t.Setenv("SPINE_GATE_N_PLUS_ONE_CLIENTS", "Query")
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	if code, _, errs := runCmd(t, "gate", "go", "n-plus-one", "--dir", dir); code != 1 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	r := readResults(t, results)
	if len(r.Findings) != 2 {
		t.Fatalf("findings=%+v", r.Findings)
	}
	for _, f := range r.Findings {
		if f.File != "perrow.go" {
			t.Errorf("finding outside the seeded file: %+v", f)
		}
	}
	if r.Findings[0].Line != 8 || r.Findings[1].Line != 11 {
		t.Errorf("call sites=%d,%d want 8,11", r.Findings[0].Line, r.Findings[1].Line)
	}
}

// TestGateTypeCheckedClassesRejectNonCompilingRepo: a gate cannot judge
// code the compiler has not agreed to, so a --dir that does not build is
// misconfiguration (exit 2) for the classes that type-check, and the
// message names the package. The syntactic classes are unaffected.
func TestGateTypeCheckedClassesRejectNonCompilingRepo(t *testing.T) {
	// The two loader failure shapes an operator has to tell apart: a module
	// that cannot be loaded at all, and one that loads but does not
	// type-check. (A cgo package is the third way to reach the loader's
	// error path; CgoFiles are type-checked with FakeImportC, but a cgo
	// fixture needs a working C toolchain and so is not hermetic here.)
	cases := []struct {
		name  string
		files map[string][]byte
		want  []string
	}{
		{
			"does not type-check",
			map[string][]byte{
				"go.mod":    []byte("module example.com/fixture\n\ngo 1.22\n"),
				"broken.go": []byte("package fixture\n\nfunc Broken() int { return \"not an int\" }\n"),
			},
			[]string{"does not type-check", "example.com/fixture"},
		},
		{
			"cannot load",
			map[string][]byte{
				"go.mod":  []byte("module\n"),
				"lib.go":  []byte("package fixture\n\nfunc Fine() int { return 1 }\n"),
				"note.md": []byte("a go.mod with no module path\n"),
			},
			[]string{"cannot load the module under --dir", "go.mod"},
		},
	}
	for _, tc := range cases {
		dir := gateRepo(t, tc.files)
		for _, check := range []string{"deferred-cleanup-errcheck", "dead-code-callgraph"} {
			t.Run(tc.name+"/"+check, func(t *testing.T) {
				code, out, errs := runCmd(t, "gate", "go", check, "--dir", dir)
				if code != 2 {
					t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
				}
				for _, want := range tc.want {
					if !strings.Contains(errs, want) {
						t.Errorf("stderr=%q, want %q", errs, want)
					}
				}
			})
		}
	}
}

// TestGateDeadCodeInterfaceSatisfaction is the Important 1 control: a
// method reached only through an interface held outside the module — a
// String called by fmt and nothing else — is live. A call graph built from
// the module's own references alone reports it unreachable.
func TestGateDeadCodeInterfaceSatisfaction(t *testing.T) {
	good, _ := deadCodeFixtures()
	dir := gateRepo(t, good)
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs := runCmd(t, "gate", "go", "dead-code-callgraph", "--dir", dir)
	if code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	if _, err := os.Stat(results); err == nil {
		r := readResults(t, results)
		if len(r.Findings) != 0 {
			t.Fatalf("findings=%+v", r.Findings)
		}
	}
}

// TestGateCleanupFuncsEnv: SPINE_GATE_CLEANUP_FUNCS extends the default
// name set. It is env-only — no gate_pack_config key — so it is read
// straight from the environment and unset means the defaults alone.
func TestGateCleanupFuncsEnv(t *testing.T) {
	dir := gateRepo(t, map[string][]byte{
		"go.mod": []byte("module example.com/fixture\n\ngo 1.22\n"),
		"shutdown.go": []byte(`package fixture

type server struct{}

func (server) Shutdown() error { return nil }

func Serve(s server) {
	defer s.Shutdown()
}
`),
	})
	if code, _, errs := runCmd(t, "gate", "go", "deferred-cleanup-errcheck", "--dir", dir); code != 0 {
		t.Fatalf("Shutdown is not a default cleanup name: code=%d stderr=%q", code, errs)
	}
	t.Setenv("SPINE_GATE_CLEANUP_FUNCS", "Shutdown")
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	if code, _, errs := runCmd(t, "gate", "go", "deferred-cleanup-errcheck", "--dir", dir); code != 1 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	r := readResults(t, results)
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Message, "s.Shutdown") {
		t.Fatalf("findings=%+v", r.Findings)
	}
}

// TestGateConfigMisconfiguration: every config-driven class exits 2 naming
// the variable an operator has to set, and names the path when the
// configured input itself is unusable.
func TestGateConfigMisconfiguration(t *testing.T) {
	dir := gateRepo(t, map[string][]byte{
		"go.mod":       []byte("module fixture\n\ngo 1.26\n"),
		"main.go":      []byte("package main\n\nfunc main() {}\n"),
		"docs/bare.md": []byte("no markers here\n"),
	})
	cases := []struct {
		name  string
		check string
		env   map[string]string
		want  []string
	}{
		{"build outputs unset", "gitignore-control", nil, []string{"SPINE_GATE_BUILD_OUTPUTS"}},
		{"build outputs empty", "gitignore-control", map[string]string{"SPINE_GATE_BUILD_OUTPUTS": " , "}, []string{"SPINE_GATE_BUILD_OUTPUTS"}},
		{"manifest unset", "fixture-manifest", nil, []string{"SPINE_GATE_FIXTURE_MANIFEST"}},
		{"enum spec unset", "test-enum-vs-spec", nil, []string{"SPINE_GATE_TEST_ENUM_SPEC"}},
		{"n-plus-one clients unset", "n-plus-one", nil, []string{"SPINE_GATE_N_PLUS_ONE_CLIENTS"}},
		{"n-plus-one clients empty", "n-plus-one", map[string]string{"SPINE_GATE_N_PLUS_ONE_CLIENTS": " , "}, []string{"SPINE_GATE_N_PLUS_ONE_CLIENTS"}},
		{"enum spec missing", "test-enum-vs-spec", map[string]string{"SPINE_GATE_TEST_ENUM_SPEC": "docs/nope.md"}, []string{"SPINE_GATE_TEST_ENUM_SPEC", "docs/nope.md"}},
		{"enum spec without marker", "test-enum-vs-spec", map[string]string{"SPINE_GATE_TEST_ENUM_SPEC": "docs/bare.md"}, []string{"SPINE_GATE_TEST_ENUM_SPEC", "docs/bare.md", "spine:enum"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			code, out, errs := runCmd(t, "gate", "go", tc.check, "--dir", dir)
			if code != 2 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			for _, want := range tc.want {
				if !strings.Contains(errs, want) {
					t.Errorf("stderr=%q, want %q", errs, want)
				}
			}
		})
	}
}
