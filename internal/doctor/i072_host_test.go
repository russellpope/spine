package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestHostRoutingCheckD17MapsSanitizedPinEvidenceAfterD16(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	evalDir := today + "-routing-check"
	ref := "eval:" + evalDir + "/runs/gpt-5-6-sol.md"
	runPath := "docs/evals/" + evalDir + "/runs/gpt-5-6-sol.md"
	evalPath := "docs/evals/" + evalDir + "/eval.md"
	for _, tc := range []struct {
		name, ref, run, wantPath, wantMessage string
		write                                 bool
	}{
		{"no reference", "owner:I068", "", "routing-host.json", "pin codex.primary has no eligible eval reference", false},
		{"bad reference", "eval:not-a-reference", "", "routing-host.json", "pin codex.primary has a malformed eval reference", false},
		{"missing", ref, "", runPath, "pin codex.primary references missing eval evidence", false},
		{"malformed", ref, "not front matter\n", runPath, "pin codex.primary references malformed eval evidence", true},
		{"stale", ref, d17Run("2020-01-01", "gpt-5.6-sol", d17PassingBattery()), runPath, "pin codex.primary references stale eval evidence", true},
		{"mismatch", ref, d17Run(today, "gpt-5.6-sol-preview", d17PassingBattery()), runPath, "pin codex.primary eval model does not exactly match pinned model", true},
		{"no battery", ref, d17Run(today, "gpt-5.6-sol", ""), runPath, "pin codex.primary eval evidence has no battery record", true},
		{"failed battery", ref, d17Run(today, "gpt-5.6-sol", d17FailedBattery()), runPath, "pin codex.primary eval battery verdict is fail", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			if tc.write {
				d17Write(t, filepath.Join(repo, evalPath), "---\ntitle: demo\ncreated: "+today+"\nprompt: prompt.md\nrubric: rubric.md\n---\n")
				d17Write(t, filepath.Join(repo, runPath), tc.run)
			}
			path := d17HostConfig(t, tc.ref)
			findings := hostRoutingCheck(repo, path, func(string) (string, error) { return "/bin/codex", nil })
			var d16, d17 []Finding
			for _, finding := range findings {
				if finding.ID == "D16" {
					d16 = append(d16, finding)
				}
				if finding.ID == "D17" {
					d17 = append(d17, finding)
				}
			}
			if len(d17) != 1 || !reflect.DeepEqual(d17[0], Finding{"D17", "warn", tc.wantPath, tc.wantMessage}) {
				t.Fatalf("D17 findings = %#v", d17)
			}
			for i, finding := range findings {
				if finding.ID == "D17" && i < len(d16) {
					t.Fatalf("D17 before D16: %#v", findings)
				}
				if strings.Contains(finding.Message, tc.ref) || strings.Contains(finding.Message, "gpt-5.6-sol-preview") {
					t.Fatalf("finding leaked evidence value: %#v", finding)
				}
			}
		})
	}
}

func TestHostRoutingCheckD17LeavesHealthyEvidenceQuiet(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	repo := t.TempDir()
	evalDir := today + "-routing-check"
	d17Write(t, filepath.Join(repo, "docs", "evals", evalDir, "eval.md"), "---\ntitle: demo\ncreated: "+today+"\nprompt: prompt.md\nrubric: rubric.md\n---\n")
	d17Write(t, filepath.Join(repo, "docs", "evals", evalDir, "runs", "gpt-5-6-sol.md"), d17Run(today, "gpt-5.6-sol", d17PassingBattery()))
	findings := hostRoutingCheck(repo, d17HostConfig(t, "eval:"+evalDir+"/runs/gpt-5-6-sol.md"), func(string) (string, error) { return "/bin/codex", nil })
	for _, finding := range findings {
		if finding.ID == "D17" {
			t.Fatalf("healthy evidence produced D17: %#v", findings)
		}
	}
}

func d17HostConfig(t *testing.T, ref string) string {
	t.Helper()
	return writeDoctorHostConfig(t, `{"schema_version":1,"host_id":"doctor-host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:codex","models":{"gpt-5.6-sol":{"efforts":["xhigh"]}}}},"pins":{"codex.primary":{"model":"gpt-5.6-sol","effort":"xhigh","evidence_refs":[`+strconv.Quote(ref)+`]}}}`)
}

func d17Run(created, model, battery string) string {
	return "---\nname: gpt-5-6-sol\ncreated: " + created + "\nmodel: " + model + "\nstage: raw\nscore: 1\n" + battery + "---\n"
}

func d17PassingBattery() string {
	return "battery_version: 1\nbattery_verdict: pass\nbattery_results: invocation=KILLED,wiring=KILLED,flag-honoured=KILLED,column-presence=KILLED,column-order=KILLED,ordering=KILLED,units-labels=KILLED,security-default=REPORT-ONLY,lifecycle=REPORT-ONLY,error-path-behaviour=KILLED\n"
}

func d17FailedBattery() string {
	return strings.Replace(strings.Replace(d17PassingBattery(), "battery_verdict: pass", "battery_verdict: fail", 1), "invocation=KILLED", "invocation=SURVIVED", 1)
}

func d17Write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
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
