package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
		if finding.Severity != "warn" || finding.Path != path || !strings.Contains(finding.Message, "claude.") || !strings.Contains(finding.Message, filepath.Clean(repo)) || !strings.Contains(finding.Message, "requested ") || !strings.Contains(finding.Message, "@") {
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

func TestHostRoutingCheckReportsInvalidConfigMatrixAsDeterministicD16Errors(t *testing.T) {
	validHarness := `"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:codex","models":{"gpt-5.6-sol":{"efforts":["xhigh"]}}}`
	for _, tc := range []struct {
		name, config, wantDetail, redacted string
		lookup                             func(string) (string, error)
	}{
		{
			name:       "malformed JSON",
			config:     `{"schema_version":`,
			wantDetail: "host routing configuration",
			redacted:   `{"schema_version":`,
		},
		{
			name:       "nested unknown member",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{` + validHarness[:len(validHarness)-1] + `,"nested_secret":"do-not-report"}},"pins":{}}`,
			wantDetail: "harness has an unsupported member",
			redacted:   "do-not-report",
		},
		{
			name:       "unsupported schema version",
			config:     `{"schema_version":2,"host_id":"host","harnesses":{` + validHarness + `},"pins":{}}`,
			wantDetail: "schema_version must be integer 1",
			redacted:   `"schema_version":2`,
		},
		{
			name:       "prohibited security member",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:claude","models":{"model":{"efforts":["high"],"auth_header":"Bearer do-not-report"}}}},"pins":{}}`,
			wantDetail: "model route has an unsupported member",
			redacted:   "Bearer do-not-report",
		},
		{
			name:       "unavailable pinned harness",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":false,"executable":"codex","launch_contract_ref":"fleet:codex"}},"pins":{"codex.primary":{"model":"gpt-5.6-sol","effort":"xhigh"}}}`,
			wantDetail: "pin names an unavailable harness",
			redacted:   "gpt-5.6-sol",
		},
		{
			name:       "path-bearing executable",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"./do-not-report","launch_contract_ref":"fleet:codex","models":{"gpt-5.6-sol":{"efforts":["xhigh"]}}}},"pins":{}}`,
			wantDetail: "harness executable must be a bare name",
			redacted:   "./do-not-report",
		},
		{
			name:       "unresolvable executable",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{` + validHarness + `},"pins":{}}`,
			wantDetail: "available harness executable is not resolvable",
			redacted:   "lookup detail",
			lookup: func(string) (string, error) {
				return "", errors.New("lookup detail must not be reported")
			},
		},
		{
			name:       "absent pinned model",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{` + validHarness + `},"pins":{"codex.primary":{"model":"do-not-report","effort":"xhigh"}}}`,
			wantDetail: "pin model@effort is not declared by its harness",
			redacted:   "do-not-report",
		},
		{
			name:       "unsupported pin effort",
			config:     `{"schema_version":1,"host_id":"host","harnesses":{` + validHarness + `},"pins":{"codex.primary":{"model":"gpt-5.6-sol","effort":"do-not-report"}}}`,
			wantDetail: "pin model@effort is not declared by its harness",
			redacted:   "do-not-report",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeDoctorHostConfig(t, tc.config)
			lookup := tc.lookup
			if lookup == nil {
				lookup = func(string) (string, error) { return "/bin/tool", nil }
			}

			first := hostRoutingCheck(t.TempDir(), path, lookup)
			second := hostRoutingCheck(t.TempDir(), path, lookup)
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("findings are not deterministic: first=%#v second=%#v", first, second)
			}
			if len(first) != 1 {
				t.Fatalf("findings = %#v, want one D16 error", first)
			}
			finding := first[0]
			if finding.ID != "D16" || finding.Severity != "error" || finding.Path != path {
				t.Fatalf("finding = %#v, want D16 error on %q", finding, path)
			}
			if !strings.Contains(finding.Message, tc.wantDetail) {
				t.Errorf("message = %q, want safe detail %q", finding.Message, tc.wantDetail)
			}
			if strings.Contains(finding.Message, tc.redacted) {
				t.Errorf("message leaked raw configuration detail %q: %q", tc.redacted, finding.Message)
			}
		})
	}
}

func TestHostRoutingCheckSkipsAbsentAndUnavailableHarnesses(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "routing-host.json")
	if findings := hostRoutingCheck(t.TempDir(), absent, func(string) (string, error) {
		t.Fatal("lookup called for absent config")
		return "", nil
	}); len(findings) != 0 {
		t.Fatalf("absent findings = %#v", findings)
	}

	path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "doctor-host", "harnesses": {
    "codex": {"available": false, "executable": "codex", "launch_contract_ref": "fleet:codex"}
  }, "pins": {}}
`)
	if findings := hostRoutingCheck(t.TempDir(), path, func(string) (string, error) {
		t.Fatal("lookup called for unavailable harness")
		return "", nil
	}); len(findings) != 0 {
		t.Fatalf("unavailable findings = %#v", findings)
	}
}

func TestHostRoutingCheckIdenticalPinSuppressesOnlyMatchingPreferenceWarning(t *testing.T) {
	path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "doctor-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:codex", "models": {"gpt-5.6-sol": {"efforts": ["xhigh"]}}}
  }, "pins": {"codex.primary": {"model": "gpt-5.6-sol", "effort": "xhigh"}}}
`)
	findings := hostRoutingCheck(t.TempDir(), path, func(string) (string, error) { return "/bin/codex", nil })
	for _, finding := range findings {
		if strings.Contains(finding.Message, "codex.primary") {
			t.Fatalf("identical pin finding = %#v", finding)
		}
	}
}

func TestHostRoutingCheckUsesLexicalAvailableHarnessOrderWithParallelExplicitPaths(t *testing.T) {
	for _, hostID := range []string{"alpha", "beta"} {
		hostID := hostID
		t.Run(hostID, func(t *testing.T) {
			t.Parallel()
			repo := t.TempDir()
			if err := os.WriteFile(filepath.Join(repo, "WORKFLOW.md"), []byte("model_routing:\n  claude.primary: unavailable-claude\n  codex.primary: unavailable-codex\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			path := writeDoctorHostConfig(t, `{
  "schema_version": 1, "host_id": "`+hostID+`", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:codex", "models": {"known-codex": {"efforts": ["high"]}}},
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:claude", "models": {"known-claude": {"efforts": ["high"]}}}
  }, "pins": {}}
`)
			var lookedUp []string
			findings := hostRoutingCheck(repo, path, func(name string) (string, error) {
				lookedUp = append(lookedUp, name)
				return "/bin/" + name, nil
			})
			if got, want := strings.Join(lookedUp, ","), "claude,codex"; got != want {
				t.Fatalf("executable lookup order = %q, want %q", got, want)
			}
			if len(findings) != 8 {
				t.Fatalf("D16 findings = %#v, want eight unreachable preferences", findings)
			}
			wantKeys := []string{
				"claude.primary", "claude.routine", "claude.mechanical", "claude.fallback",
				"codex.primary", "codex.routine", "codex.mechanical", "codex.fallback",
			}
			for i, finding := range findings {
				if finding.Path != path {
					t.Fatalf("finding %d path = %q, want explicit path %q", i, finding.Path, path)
				}
				if !strings.Contains(finding.Message, wantKeys[i]) {
					t.Fatalf("finding %d = %#v, want lexical key %q", i, finding, wantKeys[i])
				}
			}
		})
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
