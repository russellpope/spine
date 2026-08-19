package main

import (
	"encoding/json"
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

// TestGatePositiveControls is the positive control pair for each check
// class: a known-good repo the class passes (exit 0) and a seeded violation
// it fails (exit 1), with each finding attributable to go@1/<check>.
func TestGatePositiveControls(t *testing.T) {
	tskipGood, tskipSeeded := tskipFixtures()
	binGood, binSeeded := binaryHygieneFixtures()
	cases := []struct {
		check  string
		good   map[string][]byte
		seeded map[string][]byte
		want   []string // substrings expected in the seeded run's findings
	}{
		{"tskip", tskipGood, tskipSeeded, []string{"t.Skip call", "tb.SkipNow call", "s.T().Skip call"}},
		{"binary-hygiene", binGood, binSeeded, []string{"bin/tool", "data/x.tar", "tools/go.mod"}},
	}
	for _, tc := range cases {
		t.Run(tc.check+"/good", func(t *testing.T) {
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
			dir := gateRepo(t, tc.seeded)
			results := filepath.Join(t.TempDir(), "results.json")
			t.Setenv("MAIPIPE_RESULTS", results)
			code, out, errs := runCmd(t, "gate", "go", tc.check, "--dir", dir)
			if code != 1 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
			}
			r := readResults(t, results)
			if r.Status != "fail" || len(r.Findings) != len(tc.want) {
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
	for _, want := range []string{"go@1", "tskip", "binary-hygiene", "SPINE_GATE_TSKIP_ALLOW", "MAIPIPE_RESULTS", "0 pass, 1 findings, 2 misconfiguration"} {
		if !strings.Contains(errs, want) {
			t.Errorf("gate usage missing %q:\n%s", want, errs)
		}
	}
}
