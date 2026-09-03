package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// stageRepoWithPrimary scaffolds a current-generation repo and pins the
// claude.primary mirror row to value (value "" keeps the rendered default).
func stageRepoWithPrimary(t *testing.T, value string) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Init(dir, "rust", "demo"); err != nil {
		t.Fatal(err)
	}
	if value == "" {
		return dir
	}
	path := filepath.Join(dir, "WORKFLOW.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	replaced := false
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "claude.primary:" {
			lines[i] = "  claude.primary: " + value
			replaced = true
			break
		}
	}
	if !replaced {
		t.Fatalf("no claude.primary row in scaffolded WORKFLOW.md:\n%s", raw)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func findingsWithID(findings []Finding, id string) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.ID == id {
			out = append(out, f)
		}
	}
	return out
}

// I128 item 1: a mirror carrying a retired id is named distinctly from
// generation lag. D18 names the row, the retired id, its successor, and the
// update remedy, on WORKFLOW.md, as a warning (launch validation refuses
// the row, so dispatch is blocked until it is fixed).
func TestRetiredMirrorD18NamesIDSuccessorAndRemedy(t *testing.T) {
	dir := stageRepoWithPrimary(t, "claude-fable-5")
	findings, err := Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsWithID(findings, "D18")
	if len(got) != 1 {
		t.Fatalf("D18 findings = %#v, want exactly one for the retired primary row (all: %#v)", got, findings)
	}
	f := got[0]
	if f.Severity != "warn" || f.Path != "WORKFLOW.md" {
		t.Errorf("finding = %#v, want a warn on WORKFLOW.md", f)
	}
	for _, want := range []string{"claude.primary", `"claude-fable-5"`, "claude-fable-5-1", "retired", "spine update --write"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("message %q lacks %q", f.Message, want)
		}
	}
}

// The stuck override (retired id at a foreign effort) is retired all the
// same and gets the same remedy, which I128 item 2 makes correct.
func TestRetiredMirrorD18FiresOnRetiredOverride(t *testing.T) {
	dir := stageRepoWithPrimary(t, "claude-fable-5 @ xhigh")
	findings, err := Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsWithID(findings, "D18")
	if len(got) != 1 || !strings.Contains(got[0].Message, `"claude-fable-5"`) {
		t.Fatalf("D18 findings = %#v, want one naming the retired id", got)
	}
}

// Negative controls: a refreshed mirror, a current-id override, and an
// unrelated override are not retired and stay silent.
func TestRetiredMirrorD18SilentOnCurrentAndForeignIDs(t *testing.T) {
	for _, value := range []string{"", "claude-fable-5-1 @ xhigh", "local-llama-70b"} {
		dir := stageRepoWithPrimary(t, value)
		findings, err := Run(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got := findingsWithID(findings, "D18"); len(got) != 0 {
			t.Errorf("claude.primary %q: D18 findings = %#v, want none", value, got)
		}
	}
}

// I128 item 3: host configs match byte-exactly by design (I051/I072), so a
// host file listing only the retired id makes the refreshed default
// unreachable. D16 now names the host file as the remedy and says the
// listed id is retired.
func TestHostRoutingCheckUnreachableNamesHostFileAndRetiredID(t *testing.T) {
	repo := stageRepoWithPrimary(t, "")
	path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "doctor-host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:claude", "models": {"claude-fable-5": {"efforts": ["high"]}, "claude-opus-5": {"efforts": ["low", "high"]}, "claude-haiku-4-5": {"efforts": ["low"]}}}
  }, "pins": {}}
`)
	findings := hostRoutingCheck(repo, path, func(string) (string, error) { return "/bin/tool", nil })
	var primary []Finding
	for _, f := range findingsWithID(findings, "D16") {
		if strings.Contains(f.Message, "claude.primary") {
			primary = append(primary, f)
		}
	}
	if len(primary) != 1 {
		t.Fatalf("D16 primary findings = %#v, want exactly one unreachable warning (all: %#v)", primary, findings)
	}
	msg := primary[0].Message
	for _, want := range []string{"claude-fable-5-1@high", "not reachable", "add it to", path, `"claude-fable-5"`, "retired"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q lacks %q", msg, want)
		}
	}
}

// Negative control: a host file that lists no historical id of the
// requested lineage gets the host-file remedy but no retired hint.
func TestHostRoutingCheckUnreachableWithoutRetiredHint(t *testing.T) {
	repo := stageRepoWithPrimary(t, "")
	path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "doctor-host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:claude", "models": {"host-only-model": {"efforts": ["high"]}}}
  }, "pins": {}}
`)
	findings := hostRoutingCheck(repo, path, func(string) (string, error) { return "/bin/tool", nil })
	for _, f := range findingsWithID(findings, "D16") {
		if !strings.Contains(f.Message, "claude.primary") {
			continue
		}
		if !strings.Contains(f.Message, "add it to") || !strings.Contains(f.Message, path) {
			t.Errorf("message %q lacks the host-file remedy", f.Message)
		}
		if strings.Contains(f.Message, "retired") {
			t.Errorf("message %q carries a retired hint with no historical id listed", f.Message)
		}
		return
	}
	t.Fatalf("no D16 finding for claude.primary in %#v", findings)
}
