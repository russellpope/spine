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
		// Negative control for the stray-module rule: a testdata module
		// tree must not be a finding.
		"testdata/mod/go.mod": []byte("module fixture/testdata/mod\n\ngo 1.26\n"),
	}
	seeded = map[string][]byte{
		"go.mod":   []byte("module fixture\n\ngo 1.26\n"),
		"main.go":  []byte("package main\n\nfunc main() {}\n"),
		"bin/tool": elfBytes(),
		// A go.mod under testdata is not a subtree `go build ./...`
		// silently skipped: the toolchain excludes testdata by rule.
		"testdata/mod/go.mod": []byte("module fixture/testdata/mod\n\ngo 1.26\n"),
		"data/x.tar":          tarBytes(),
		"tools/go.mod":        []byte("module fixture/tools\n\ngo 1.26\n"),
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

	"example.com/fixture/internal/priv"
	"example.com/fixture/lib"
)

type printed int

func (p printed) String() string { return "printed" }

func main() { fmt.Println(lib.Exported(), priv.Used(), printed(1)) }
`),
		// priv is under internal/: no other module can import it, so its
		// exported API is not a contract the gate must keep live. Used is
		// live because main calls it.
		"internal/priv/priv.go": []byte(`package priv

// Used is exported but internal, and reached from main.
func Used() string { return "used" }
`),
	}
	seeded = map[string][]byte{}
	for k, v := range good {
		seeded[k] = v
	}
	seeded["lib/dead.go"] = []byte(`package lib

func unreached() string { return "nobody calls me" }
`)
	// The internal counterpart of lib.Exported: exported, unreached, and
	// under internal/, so it must be reported while lib.Exported is not.
	seeded["internal/priv/dead.go"] = []byte(`package priv

func Unreached() string { return "no module can import me" }
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
			"dead-code-callgraph", deadGood, deadSeeded, nil, 2,
			[]string{
				"unreachable function", "lib.unreached", "lib/dead.go",
				"priv.Unreached", "internal/priv/dead.go",
			},
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

// TestGateVersionedPinIsAuthoritative exercises the public CLI contract: a
// versioned pin controls attribution, while an unshipped pin fails before it
// can emit a maipipe findings document. The bare form remains the stateless
// hand-run form and attributes as the binary's own pack.
func TestGateVersionedPinIsAuthoritative(t *testing.T) {
	_, seeded := tskipFixtures()
	dir := gateRepo(t, seeded)

	code, out, errs := runCmd(t, "gate", "go@1", "tskip", "--dir", dir)
	if code != 1 {
		t.Fatalf("pinned gate exit = %d, want findings exit 1; stdout=%q stderr=%q", code, out, errs)
	}
	if !strings.Contains(out, "go@1/tskip") {
		t.Errorf("pinned finding code missing from output: %q", out)
	}

	code, out, errs = runCmd(t, "gate", "go", "tskip", "--dir", dir)
	if code != 1 || !strings.Contains(out, gate.PackID()+"/tskip") {
		t.Errorf("bare go behavior changed: exit=%d stdout=%q stderr=%q", code, out, errs)
	}

	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs = runCmd(t, "gate", "go@9", "tskip", "--dir", dir)
	if code != 2 {
		t.Errorf("unshipped pin exit = %d, want 2; stdout=%q stderr=%q", code, out, errs)
	}
	if !strings.Contains(errs, "go@9") {
		t.Errorf("unshipped-pin refusal does not name the pin: %q", errs)
	}
	if _, err := os.Stat(results); !os.IsNotExist(err) {
		t.Errorf("unshipped pin wrote a results document: err=%v", err)
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
		{"unknown pack", []string{"gate", "rust", "tskip", "--dir", dir}, "unshipped pack"},
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
	wants := []string{"<pack>[@<v>]", "go@1", "SPINE_GATE_TSKIP_ALLOW", "SPINE_GATE_BUILD_OUTPUTS", "SPINE_GATE_FIXTURE_MANIFEST", "SPINE_GATE_TEST_ENUM_SPEC", "SPINE_GATE_N_PLUS_ONE_CLIENTS", "SPINE_GATE_CLEANUP_FUNCS", "SPINE_GATE_MUTATE_SPEC", "SPINE_GATE_MUTATE_VERIFY", "SPINE_GATE_MUTATE_TIMEOUT", "MAIPIPE_RESULTS", "0 pass, 1 findings, 2 misconfiguration"}
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

// rawFindingKeys decodes the results file's findings as generic maps so a
// test can assert which keys are present, not just their decoded values
// (a missing int decodes as 0 and hides an omitted key).
func rawFindingKeys(t *testing.T, path string) []map[string]any {
	t.Helper()
	var doc struct {
		Findings []map[string]any `json:"findings"`
	}
	if err := json.Unmarshal([]byte(readFile(t, path)), &doc); err != nil {
		t.Fatalf("results JSON: %v", err)
	}
	if len(doc.Findings) == 0 {
		t.Fatal("no findings in results file")
	}
	return doc.Findings
}

// TestGateResultsOmitLineZero (I092): a finding without a line — arm 1 of
// gitignore-control names a path, not a line — must omit the key rather
// than emit 0, which maipipe rejects as "finding line must be a positive
// 64-bit integer" and fails the whole stage as results_invalid.
func TestGateResultsOmitLineZero(t *testing.T) {
	_, seeded := gitignoreControlFixtures()
	dir := gateRepo(t, seeded)
	t.Setenv("SPINE_GATE_BUILD_OUTPUTS", "bin/spine")
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs := runCmd(t, "gate", "go", "gitignore-control", "--dir", dir)
	if code != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	var sawLineless bool
	for _, f := range rawFindingKeys(t, results) {
		line, present := f["line"]
		if !present {
			sawLineless = true
			continue
		}
		if n, ok := line.(float64); !ok || n < 1 {
			t.Errorf("finding emits a non-positive line %v: %+v", line, f)
		}
	}
	if !sawLineless {
		t.Errorf("expected the declared-output finding to carry no line key: %s", readFile(t, results))
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

// TestGateDeadCodeRootRule is the AC2 rule fixture, plus the importable-API
// boundary: exactly the unreachable unexported function and the unreachable
// exported function under internal/ are flagged. A function reached only
// from a test is live; an exported function of an importable library package
// is live because a library's callers are outside the module; an exported
// function under internal/ has no such callers and is a candidate.
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
	want := map[string]string{
		"lib.unreached":  "lib/dead.go",
		"priv.Unreached": "internal/priv/dead.go",
	}
	if len(r.Findings) != len(want) {
		t.Fatalf("want exactly %d unreachable functions, got %+v", len(want), r.Findings)
	}
	for _, f := range r.Findings {
		matched := false
		for name, file := range want {
			if strings.Contains(f.Message, name) {
				matched = true
				if f.File != file || f.Line != 3 {
					t.Errorf("finding=%+v, want file %s line 3", f, file)
				}
				delete(want, name)
				break
			}
		}
		if !matched {
			t.Errorf("unexpected finding %+v", f)
		}
	}
	if len(want) != 0 {
		t.Errorf("missing findings for %v", want)
	}
	// The importable exported API must never be reported.
	if strings.Contains(out, "lib.Exported") {
		t.Errorf("exported API of an importable package reported dead:\n%s", out)
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
			// The importer sorts before the package that is actually broken
			// (cmd < internal): the refusal must name the first compile
			// error, not the downstream "could not import … (no export
			// data)" symptom (I093.2).
			"does not type-check, importer sorts first",
			map[string][]byte{
				"go.mod":              []byte("module example.com/fixture\n\ngo 1.22\n"),
				"cmd/main.go":         []byte("package main\n\nimport \"example.com/fixture/internal/inv\"\n\nfunc main() { _ = inv.Name() }\n"),
				"internal/inv/inv.go": []byte("package inv\n\nfunc Name() string { return undefinedThing }\n"),
			},
			[]string{"does not type-check", "internal/inv/inv.go:3:", "undefined: undefinedThing"},
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
				if strings.Contains(errs, "no export data") {
					t.Errorf("stderr names the downstream symptom, not the cause: %q", errs)
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

// mutateModule is the mutation battery's fixture tree: one tested function
// (Double, whose behaviour the suite can see) and one untested one (Label,
// whose behaviour it cannot). That asymmetry is what makes a KILLED and a
// SURVIVED row happen for real rather than by assertion.
const mutateModule = `package fixture

func Double(n int) int { return n * 2 }

func Label() string { return "count" }
`

const mutateModuleTest = `package fixture

import "testing"

func TestDouble(t *testing.T) {
	if Double(3) != 6 {
		t.Fatal("double")
	}
}
`

// mutateSpec exercises every outcome the checklist names: a killed probe, a
// survived one (the blind spot), a site the literal no longer has, a
// mutation that breaks the build, and a report-only probe excluded from the
// scorable denominator.
const mutateSpec = `[
  {"id": "M1-invocation", "file": "calc.go", "find": "return n * 2", "replace": "return n * 3", "desc": "Double returns the wrong multiple"},
  {"id": "M2-units", "file": "calc.go", "find": "return \"count\"", "replace": "return \"total\"", "desc": "Label reports a different unit"},
  {"id": "M3-drift", "file": "calc.go", "find": "return n * 9", "replace": "return n * 8", "desc": "a site the tree no longer has"},
  {"id": "M4-build", "file": "calc.go", "find": "func Label() string", "replace": "func Label() string int", "desc": "mutation that does not compile"},
  {"id": "M5-lifecycle", "file": "calc.go", "find": "package fixture", "replace": "package fixture\n\nvar guard = 1", "report_only": true, "desc": "near-untestable class"}
]`

func mutateFixture(t *testing.T, spec, testFile string) string {
	t.Helper()
	return gateRepo(t, map[string][]byte{
		"go.mod":                  []byte("module fixture\n\ngo 1.26\n"),
		"calc.go":                 []byte(mutateModule),
		"calc_test.go":            []byte(testFile),
		"docs/mutation-spec.json": []byte(spec),
	})
}

// TestGateMutatePositiveControl is the mutate class's positive-control pair
// at the CLI seam: a suite that catches one behaviour change (KILLED) and
// misses another (SURVIVED), plus the three invalid-probe shapes. The class
// is advisory, so the pair is in the rows and the two kill rates rather than
// in the exit code, which is 0 either way.
func TestGateMutatePositiveControl(t *testing.T) {
	dir := mutateFixture(t, mutateSpec, mutateModuleTest)
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)

	code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", dir)
	if code != 0 {
		t.Fatalf("survivors must not fail the advisory lane: code=%d stdout=%q stderr=%q", code, out, errs)
	}
	if errs != "" {
		t.Errorf("stderr not pristine: %q", errs)
	}
	r := readResults(t, results)
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (advisory)", r.Status)
	}
	type row struct {
		result, severity string
		line             int
	}
	want := map[string]row{
		"M1-invocation": {"KILLED", "info", 3},
		"M2-units":      {"SURVIVED", "warning", 5},
		"M3-drift":      {"NO-SITE", "info", 0},
		"M4-build":      {"BUILD-ERR", "info", 5},
		"M5-lifecycle":  {"SURVIVED", "warning", 1},
	}
	seen := map[string]bool{}
	for _, f := range r.Findings {
		id, _, _ := strings.Cut(f.Message, " ")
		w, ok := want[id]
		if !ok {
			t.Errorf("unexpected row %+v", f)
			continue
		}
		seen[id] = true
		if !strings.HasPrefix(f.Message, id+" "+w.result+" ") {
			t.Errorf("%s: message = %q, want result %s", id, f.Message, w.result)
		}
		if f.Severity != w.severity {
			t.Errorf("%s: severity = %q, want %q", id, f.Severity, w.severity)
		}
		if f.Line != w.line {
			t.Errorf("%s: line = %d, want %d", id, f.Line, w.line)
		}
		if f.Code != "go@1/mutate" {
			t.Errorf("%s: code = %q", id, f.Code)
		}
		if f.File != "calc.go" {
			t.Errorf("%s: file = %q, want the probe's file", id, f.File)
		}
		if id == "M5-lifecycle" && !strings.HasSuffix(f.Message, "[report-only]") {
			t.Errorf("%s: report-only probe not marked: %q", id, f.Message)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("rows = %d, want %d: %+v", len(seen), len(want), r.Findings)
	}
	// Raw counts every valid probe; scorable drops the report-only one.
	for _, w := range []string{
		"kill rate (raw): 1/3 = 33%   (excluded: 1 no-site, 1 build-err)",
		"kill rate (scorable): 1/2 = 50%   (excluded: 1 report-only, 1 no-site, 1 build-err)",
	} {
		if !strings.Contains(r.Summary, w) {
			t.Errorf("summary missing %q: %q", w, r.Summary)
		}
	}
	// The tree under --dir is never mutated.
	if got := readFile(t, filepath.Join(dir, "calc.go")); got != mutateModule {
		t.Errorf("--dir was mutated:\n%s", got)
	}
}

// TestGateMutateHumanTable is the no-MAIPIPE_RESULTS path: the table, the
// two kill rates, and the survivor list the checklist's record format asks
// for.
func TestGateMutateHumanTable(t *testing.T) {
	dir := mutateFixture(t, mutateSpec, mutateModuleTest)
	code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", dir)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	for _, want := range []string{
		"severity", "go@1/mutate", "M1-invocation KILLED",
		"kill rate (raw): 1/3 = 33%", "kill rate (scorable): 1/2 = 50%",
		"surviving mutations (behaviour the suite cannot see):",
		"  - M2-units: Label reports a different unit",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}
}

// TestGateMutateControlFails is the battery's own negative control: a tree
// whose suite is already red makes every probe meaningless, so no probe
// runs, one error finding is reported, and the stage fails (exit 1) — the
// one condition under which the advisory lane fails at all.
func TestGateMutateControlFails(t *testing.T) {
	red := `package fixture

import "testing"

func TestDouble(t *testing.T) {
	t.Fatal("this tree is red before any mutation")
}
`
	dir := mutateFixture(t, mutateSpec, red)
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", dir)
	if code != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	r := readResults(t, results)
	if r.Status != "fail" || len(r.Findings) != 1 {
		t.Fatalf("status=%q findings=%+v", r.Status, r.Findings)
	}
	f := r.Findings[0]
	if f.Severity != "error" || f.Code != "go@1/mutate" ||
		!strings.Contains(f.Message, "control failed: unmutated tree is not green") {
		t.Errorf("control finding = %+v", f)
	}
	if !strings.Contains(r.Summary, "no probes run") {
		t.Errorf("summary = %q", r.Summary)
	}
	// The control finding has no site (I092): maipipe rejects line 0
	// outright and a placeholder file path, so neither key may appear.
	for _, key := range []string{"file", "line"} {
		if _, present := rawFindingKeys(t, results)[0][key]; present {
			t.Errorf("control finding carries a %q key maipipe would reject: %s", key, readFile(t, results))
		}
	}
	// The control failed inside the working copy, which is a tracked-files-
	// only tree: the operator needs the output and a tree still on disk to
	// look at, because the failure need not reproduce under --dir.
	if !strings.Contains(f.Message, "this tree is red before any mutation") {
		t.Errorf("control finding carries no verify output: %q", f.Message)
	}
	kept := mutateWorkingCopy(t, f.Message)
	info, err := os.Stat(kept)
	if err != nil || !info.IsDir() {
		t.Fatalf("working copy %q not kept for the operator: %v", kept, err)
	}
	if _, err := os.Stat(filepath.Join(kept, "calc.go")); err != nil {
		t.Errorf("kept working copy is not the tree: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(kept) })
}

// mutateWorkingCopy pulls the kept working-copy path out of a control-failure
// message.
func mutateWorkingCopy(t *testing.T, message string) string {
	t.Helper()
	_, rest, ok := strings.Cut(message, "Working copy kept at ")
	if !ok {
		t.Fatalf("control finding does not name the working copy: %q", message)
	}
	path, _, _ := strings.Cut(rest, "\n")
	return strings.TrimSpace(path)
}

// TestGateMutateVerifyEnvScrubbed (I092): the tree's own suite runs without
// the stage's MAIPIPE_RESULTS and SPINE_GATE_* — a test in the tree that
// exercises a results-emitting tool would otherwise write to the stage's
// results path and fail the control. The fixture suite is red exactly when
// either leaks through.
func TestGateMutateVerifyEnvScrubbed(t *testing.T) {
	envSensitive := `package fixture

import (
	"os"
	"testing"
)

func TestDouble(t *testing.T) {
	if os.Getenv("MAIPIPE_RESULTS") != "" {
		t.Fatal("stage results path leaked into the tree's suite")
	}
	if os.Getenv("SPINE_GATE_TSKIP_ALLOW") != "" {
		t.Fatal("gate config leaked into the tree's suite")
	}
	if Double(3) != 6 {
		t.Fatal("double")
	}
}
`
	dir := mutateFixture(t, mutateSpec, envSensitive)
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	t.Setenv("SPINE_GATE_TSKIP_ALLOW", "x_test.go")
	code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", dir)
	if code != 0 {
		t.Fatalf("control must pass with the stage env scrubbed: code=%d stdout=%q stderr=%q", code, out, errs)
	}
	r := readResults(t, results)
	if r.Status != "pass" || strings.Contains(r.Summary, "control failed") {
		t.Fatalf("status=%q summary=%q", r.Status, r.Summary)
	}
}

// TestGateMutateRemovesWorkingCopyOnSuccess is the other half of the
// keep-on-failure rule: a run whose control passes leaves nothing behind.
func TestGateMutateRemovesWorkingCopyOnSuccess(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	dir := mutateFixture(t, mutateSpec, mutateModuleTest)
	if code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", dir); code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "spine-mutate-") {
			t.Errorf("working copy %s left behind after a successful run", e.Name())
		}
	}
}

// TestGateMutateSpecMisconfiguration: no spec at the default path and no
// override is misconfiguration (exit 2) naming both the variable and the
// path it looked at; a custom path is honoured.
func TestGateMutateSpecMisconfiguration(t *testing.T) {
	bare := gateRepo(t, map[string][]byte{
		"go.mod":       []byte("module fixture\n\ngo 1.26\n"),
		"calc.go":      []byte(mutateModule),
		"calc_test.go": []byte(mutateModuleTest),
	})
	code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", bare)
	if code != 2 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	for _, want := range []string{"SPINE_GATE_MUTATE_SPEC", "docs/mutation-spec.json"} {
		if !strings.Contains(errs, want) {
			t.Errorf("stderr missing %q: %q", want, errs)
		}
	}

	// Unparseable spec at an overridden path: still misconfiguration.
	broken := gateRepo(t, map[string][]byte{
		"go.mod":       []byte("module fixture\n\ngo 1.26\n"),
		"calc.go":      []byte(mutateModule),
		"calc_test.go": []byte(mutateModuleTest),
		"probes.json":  []byte("{not json"),
	})
	t.Setenv("SPINE_GATE_MUTATE_SPEC", "probes.json")
	code, _, errs = runCmd(t, "gate", "go", "mutate", "--dir", broken)
	if code != 2 || !strings.Contains(errs, "probes.json") {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
}

// TestGateMutateCustomVerify: the verify command is overridable, runs with
// sh -c in the copy, and — being one phase — reports no BUILD-ERR, so a
// mutation that would not compile reads as SURVIVED under it.
func TestGateMutateCustomVerify(t *testing.T) {
	dir := mutateFixture(t, mutateSpec, mutateModuleTest)
	t.Setenv("SPINE_GATE_MUTATE_VERIFY", `grep -q "n \* 2" calc.go`)
	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv("MAIPIPE_RESULTS", results)
	code, out, errs := runCmd(t, "gate", "go", "mutate", "--dir", dir)
	if code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	r := readResults(t, results)
	want := map[string]string{
		"M1-invocation": "KILLED", "M2-units": "SURVIVED", "M3-drift": "NO-SITE",
		"M4-build": "SURVIVED", "M5-lifecycle": "SURVIVED",
	}
	for _, f := range r.Findings {
		id, _, _ := strings.Cut(f.Message, " ")
		if w := want[id]; w == "" || !strings.HasPrefix(f.Message, id+" "+w+" ") {
			t.Errorf("row %q, want result %q", f.Message, want[id])
		}
	}
	if strings.Contains(string(mustJSON(t, r)), "BUILD-ERR") {
		t.Errorf("a one-phase verify command cannot tell BUILD-ERR from KILLED: %+v", r.Findings)
	}
}

// readFile is a test-local reader for asserting a fixture file's bytes.
func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// unparseableTestdata is what Go repos routinely keep under testdata: a
// template source, and a deliberately broken one. Neither is part of any
// build — the toolchain excludes testdata by rule — so no syntactic check
// class may exit 2 on them.
func unparseableTestdata() map[string][]byte {
	return map[string][]byte{
		"pkg/testdata/tmpl_test.go": []byte("package {{.Package}}\n\nfunc Test{{.Name}}(t *testing.T) { t.Skip(\"tmpl\") }\n"),
		"pkg/testdata/bad.go":       []byte("this is not go at all {\n"),
	}
}

// withTestdata returns files plus the unparseable testdata sources.
func withTestdata(files map[string][]byte) map[string][]byte {
	out := map[string][]byte{}
	for k, v := range files {
		out[k] = v
	}
	for k, v := range unparseableTestdata() {
		out[k] = v
	}
	return out
}

// TestGateSyntacticClassesTolerateTestdata is the positive-control pair for
// the testdata rule: with an unparseable template and a broken source under
// testdata, every tree-walking class still passes its known-good fixture
// (exit 0, not the exit 2 a parse error would have produced) and still fails
// on the seeded content (exit 1). gitignore-control walks testdata on
// purpose, so its tolerance is per-file, not per-directory.
func TestGateSyntacticClassesTolerateTestdata(t *testing.T) {
	tskipGood, tskipSeeded := tskipFixtures()
	ignGood, ignSeeded := gitignoreControlFixtures()
	enumGood, enumSeeded := testEnumVsSpecFixtures()
	nplusGood, nplusSeeded := nPlusOneFixtures()
	cases := []struct {
		check        string
		good, seeded map[string][]byte
		env          map[string]string
	}{
		{"tskip", tskipGood, tskipSeeded, nil},
		{"gitignore-control", ignGood, ignSeeded, map[string]string{"SPINE_GATE_BUILD_OUTPUTS": "bin/spine"}},
		{"test-enum-vs-spec", enumGood, enumSeeded, map[string]string{"SPINE_GATE_TEST_ENUM_SPEC": "docs/spec.md"}},
		{"n-plus-one", nplusGood, nplusSeeded, map[string]string{"SPINE_GATE_N_PLUS_ONE_CLIENTS": "Query,Fetch"}},
	}
	for _, tc := range cases {
		for _, arm := range []struct {
			name  string
			files map[string][]byte
			want  int
		}{
			{"good", tc.good, 0},
			{"seeded", tc.seeded, 1},
		} {
			t.Run(tc.check+"/"+arm.name, func(t *testing.T) {
				for k, v := range tc.env {
					t.Setenv(k, v)
				}
				dir := gateRepo(t, withTestdata(arm.files))
				code, out, errs := runCmd(t, "gate", "go", tc.check, "--dir", dir)
				if code != arm.want {
					t.Fatalf("code=%d want %d stdout=%q stderr=%q", code, arm.want, out, errs)
				}
				if strings.Contains(out, "testdata") || strings.Contains(errs, "testdata") {
					t.Errorf("testdata named in output: stdout=%q stderr=%q", out, errs)
				}
			})
		}
	}
}
