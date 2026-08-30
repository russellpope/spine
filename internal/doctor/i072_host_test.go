package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostRoutingCheckCoversEveryTierOfEveryAvailableHarness(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "WORKFLOW.md"), []byte("model_routing:\n  claude.primary: unreachable-primary\n  codex.primary: unreachable-codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "doctor-host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:claude", "models": {"claude-fable-5": {"efforts": ["high"]}}},
    "codex": {"available": false, "executable": "codex", "launch_contract_ref": "fleet:codex", "models": {}}
  }, "pins": {}}
`)
	findings := hostRoutingCheck(repo, path, func(string) (string, error) { return "/bin/tool", nil })
	var hostFindings []Finding
	for _, finding := range findings {
		if finding.ID == "D16" {
			hostFindings = append(hostFindings, finding)
		}
	}
	if len(hostFindings) != 4 {
		t.Fatalf("D16 findings = %#v, want one unreachable warning for each Claude tier", hostFindings)
	}
	for _, finding := range hostFindings {
		if finding.Severity != "warn" || finding.Path != path || !strings.Contains(finding.Message, "claude.") {
			t.Fatalf("finding = %#v", finding)
		}
	}
}

func TestHostRoutingCheckTreatsPinsAsAuditableOnlyWhenIDsMatch(t *testing.T) {
	repo := t.TempDir()
	path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "doctor-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:codex", "models": {"gpt-5.6-sol": {"efforts": ["xhigh"]}, "host-safe": {"efforts": ["high"]}}}
  }, "pins": {"codex.primary": {"model": "host-safe", "effort": "high"}}
}`)
	findings := hostRoutingCheck(repo, path, func(string) (string, error) { return "/bin/tool", nil })
	var divergent int
	for _, finding := range findings {
		if finding.ID == "D16" && strings.Contains(finding.Message, "not auditable until I074") {
			divergent++
		}
	}
	if divergent != 1 {
		t.Fatalf("findings = %#v, want one divergent-pin D16 warning", findings)
	}
}

func TestHostRoutingCheckReportsInvalidConfigAsD16Error(t *testing.T) {
	path := writeDoctorHostConfig(t, `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:x","models":{"m":{"efforts":["high"],"auth_header":"secret"}}}},"pins":{}}`)
	findings := hostRoutingCheck(t.TempDir(), path, func(string) (string, error) { return "/bin/tool", nil })
	if len(findings) != 1 || findings[0].ID != "D16" || findings[0].Severity != "error" || findings[0].Path != path {
		t.Fatalf("findings = %#v", findings)
	}
}

func writeDoctorHostConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routing-host.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
