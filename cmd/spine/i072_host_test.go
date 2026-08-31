package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelHostOutputUsesFinalPairAndExposesTrail(t *testing.T) {
	repo := writeModelHostWorkflow(t, "model_routing:\n  codex.primary: repository-safe @ xhigh\n")
	path := writeModelHostConfig(t, `{
  "schema_version": 1, "host_id": "cli-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"repository-safe": {"efforts": ["xhigh"]}, "host-safe": {"efforts": ["high"]}}}
  }, "pins": {"codex.primary": {"model": "host-safe", "effort": "high", "evidence_refs": ["owner:I068"]}}
}`)
	code, out, errs := runModelWithHostPath(t, path, "--dir", repo, "codex", "primary")
	if code != 0 || out != "host-safe\n" || errs != "" {
		t.Fatalf("text: code=%d stdout=%q stderr=%q", code, out, errs)
	}
	code, out, errs = runModelWithHostPath(t, path, "--dir", repo, "--effort", "codex", "primary")
	if code != 0 || out != "high\n" || errs != "" {
		t.Fatalf("effort: code=%d stdout=%q stderr=%q", code, out, errs)
	}
	code, out, errs = runModelWithHostPath(t, path, "--dir", repo, "--json", "codex", "primary")
	if code != 0 || errs != "" {
		t.Fatalf("json: code=%d stdout=%q stderr=%q", code, out, errs)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["id"] != "host-safe" || decoded["effort"] != "high" || decoded["provenance"] != "host" {
		t.Fatalf("old JSON fields = %#v", decoded)
	}
	requested, _ := decoded["requested"].(map[string]any)
	host, _ := decoded["host"].(map[string]any)
	pin, _ := decoded["pin"].(map[string]any)
	if requested["id"] != "repository-safe" || host["id"] != "cli-host" || host["status"] != "pinned" || pin["model"] != "host-safe" {
		t.Fatalf("trail requested=%#v host=%#v pin=%#v", requested, host, pin)
	}
}

func TestModelHostIdenticalPinPreservesFinalEntryMetadata(t *testing.T) {
	path := writeModelHostConfig(t, `{
  "schema_version": 1, "host_id": "cli-host", "harnesses": {
    "pi": {"available": true, "executable": "pi", "launch_contract_ref": "fleet:test", "models": {"qwen3.8-27b-q8_0": {"efforts": ["medium"]}}}
  }, "pins": {"pi.routine": {"model": "qwen3.8-27b-q8_0", "effort": "medium"}}
}`)
	code, out, errs := runModelWithHostPath(t, path, "--json", "pi", "routine")
	if code != 0 || errs != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	aliases, _ := decoded["aliases"].([]any)
	alternate, _ := decoded["alternate"].(map[string]any)
	if len(aliases) != 2 || alternate == nil || decoded["provenance"] != "default" {
		t.Fatalf("identical pinned final metadata = %#v", decoded)
	}
	if strings.Join([]string{aliases[0].(string), aliases[1].(string)}, ",") != "qwen3.8,qwen" || alternate["id"] != "qwen3.8-27b-q8_0" || alternate["effort"] != "xhigh" {
		t.Fatalf("identical pinned final metadata = %#v", decoded)
	}
}

func TestModelHostDivergentPinDoesNotCarryRequestedMetadata(t *testing.T) {
	path := writeModelHostConfig(t, `{
  "schema_version": 1, "host_id": "cli-host", "harnesses": {
    "pi": {"available": true, "executable": "pi", "launch_contract_ref": "fleet:test", "models": {"qwen3.8-27b-q8_0": {"efforts": ["medium"]}, "host-safe": {"efforts": ["high"]}}}
  }, "pins": {"pi.routine": {"model": "host-safe", "effort": "high"}}
}`)
	code, out, errs := runModelWithHostPath(t, path, "--json", "pi", "routine")
	if code != 0 || errs != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	aliases, _ := decoded["aliases"].([]any)
	requested, _ := decoded["requested"].(map[string]any)
	if decoded["id"] != "host-safe" || decoded["effort"] != "high" || decoded["provenance"] != "host" || len(aliases) != 0 || decoded["alternate"] != nil || requested["id"] != "qwen3.8-27b-q8_0" || requested["provenance"] != "default" {
		t.Fatalf("divergent pinned metadata = %#v", decoded)
	}
}

func TestModelHostNoPinUsesReachableStatus(t *testing.T) {
	path := writeModelHostConfig(t, `{
  "schema_version": 1, "host_id": "cli-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"gpt-5.6-sol": {"efforts": ["xhigh"]}}}
  }, "pins": {}}
`)
	code, out, errs := runModelWithHostPath(t, path, "--json", "codex", "primary")
	if code != 0 || errs != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errs)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatal(err)
	}
	host, _ := decoded["host"].(map[string]any)
	if host["status"] != "reachable" {
		t.Fatalf("host status = %#v, want reachable", host)
	}
}

func TestModelHostConfigFailureWritesNoStandardOutput(t *testing.T) {
	path := writeModelHostConfig(t, `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:test","models":{"m":{"efforts":["high"],"token":"secret"}}}},"pins":{}}`)
	for _, args := range [][]string{{"codex", "primary"}, {"--effort", "codex", "primary"}, {"--json", "codex", "primary"}} {
		code, out, errs := runModelWithHostPath(t, path, args...)
		if code != 2 || out != "" || !strings.Contains(errs, "host routing configuration") || strings.Count(errs, "\n") != 1 {
			t.Fatalf("%v: code=%d stdout=%q stderr=%q", args, code, out, errs)
		}
	}
}

func TestModelHostFailureClassesRejectEveryOutputMode(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		lookup func(string) (string, error)
	}{
		{"malformed config", `{"schema_version":`, func(string) (string, error) { return "/bin/tool", nil }},
		{"unavailable harness", `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":false,"executable":"codex","launch_contract_ref":"fleet:test"}},"pins":{}}`, func(string) (string, error) { return "/bin/tool", nil }},
		{"missing executable", `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:test","models":{"gpt-5.6-sol":{"efforts":["xhigh"]}}}},"pins":{}}`, func(string) (string, error) { return "", errors.New("not found") }},
		{"unreachable preference", `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:test","models":{"other":{"efforts":["high"]}}}},"pins":{}}`, func(string) (string, error) { return "/bin/tool", nil }},
		{"unreachable pin", `{"schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:test","models":{"gpt-5.6-sol":{"efforts":["xhigh"]}}}},"pins":{"codex.primary":{"model":"other","effort":"high"}}}`, func(string) (string, error) { return "/bin/tool", nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeModelHostConfig(t, tc.config)
			for _, args := range [][]string{{"codex", "primary"}, {"--effort", "codex", "primary"}, {"--json", "codex", "primary"}} {
				code, out, errs := runModelWithHostPathAndLookup(t, path, tc.lookup, args...)
				if code != 2 || out != "" || !strings.Contains(errs, "host routing configuration") || strings.Count(errs, "\n") != 1 {
					t.Fatalf("%v: code=%d stdout=%q stderr=%q", args, code, out, errs)
				}
			}
		})
	}
}

func TestModelAlternateJSONKeepsLegacyBytesWithoutHostTrail(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "routing-host.json")
	validPath := writeModelHostConfig(t, `{
  "schema_version": 1, "host_id": "alternate-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"gpt-5.6-sol": {"efforts": ["xhigh"]}}}
  }, "pins": {}}
`)
	args := []string{"--alternate", "--json", "pi", "routine"}
	code, legacy, errs := runModelWithHostPath(t, missingPath, args...)
	if code != 0 || errs != "" {
		t.Fatalf("missing config: code=%d stdout=%q stderr=%q", code, legacy, errs)
	}
	wantLegacy := "{\"harness\":\"pi\",\"flavor\":\"pi\",\"tier\":\"routine\",\"id\":\"qwen3.8-27b-q8_0\",\"effort\":\"medium\",\"aliases\":[\"qwen3.8\",\"qwen\"],\"alternate\":{\"id\":\"qwen3.8-27b-q8_0\",\"effort\":\"xhigh\"},\"provenance\":\"default\"}\n"
	if legacy != wantLegacy {
		t.Fatalf("missing config alternate JSON = %q, want legacy bytes %q", legacy, wantLegacy)
	}
	for _, field := range []string{`"requested"`, `"host"`, `"pin"`} {
		if strings.Contains(legacy, field) {
			t.Fatalf("missing config JSON contains host trail %s: %q", field, legacy)
		}
	}
	code, got, errs := runModelWithHostPath(t, validPath, args...)
	if code != 0 || errs != "" {
		t.Fatalf("valid config: code=%d stdout=%q stderr=%q", code, got, errs)
	}
	if got != legacy {
		t.Fatalf("valid config alternate JSON = %q, want legacy bytes %q", got, legacy)
	}
	for _, args := range [][]string{
		{"--alternate", "pi", "routine"},
		{"--alternate", "--effort", "pi", "routine"},
		{"--alternate", "--json", "pi", "routine"},
	} {
		malformedPath := writeModelHostConfig(t, `{"schema_version":`)
		code, out, errs := runModelWithHostPath(t, malformedPath, args...)
		if code != 2 || out != "" || !strings.Contains(errs, "host routing configuration") {
			t.Fatalf("malformed %v: code=%d stdout=%q stderr=%q", args, code, out, errs)
		}
	}
}

func TestModelValidateHostDivergenceRefusesBeforeExpect(t *testing.T) {
	repo := writeModelHostWorkflow(t, "template_version: 13\nmodel_routing:\n  codex.primary: repository-safe\n")
	path := writeModelHostConfig(t, `{
  "schema_version": 1, "host_id": "cli-host", "harnesses": {
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:test", "models": {"repository-safe": {"efforts": ["high"]}, "host-safe": {"efforts": ["high"]}}}
  }, "pins": {"codex.primary": {"model": "host-safe", "effort": "high"}}
}`)
	for _, expect := range []string{"repository-safe", "host-safe"} {
		code, out, errs := runModelWithHostPath(t, path, "--dir", repo, "validate", "--expect", expect, "codex", "primary")
		if code != 2 || out != "" || !strings.Contains(errs, "not auditable until I074") {
			t.Fatalf("expect %q: code=%d stdout=%q stderr=%q", expect, code, out, errs)
		}
	}
}

func TestAuditRoutingHostPreflightReturnsUsageErrorBeforeOutput(t *testing.T) {
	repo := t.TempDir()
	writeAuditFixtureRepo(t, repo, map[string]string{"I072": "primary"})
	path := writeModelHostConfig(t, `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:x","models":{"m":{"efforts":["high"],"token":"secret"}}}},"pins":{}}`)
	var out, errs bytes.Buffer
	code := cmdAuditRoutingWithHostPath([]string{"--dir", repo, "--transcripts", t.TempDir()}, &out, &errs, path, func(string) (string, error) { return "/bin/tool", nil })
	if code != 2 || out.Len() != 0 || !strings.Contains(errs.String(), "host routing configuration") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), errs.String())
	}
}

func TestAuditRoutingPreflightsHostConfigBeforeDefaultDiscovery(t *testing.T) {
	// This is deliberately a non-git fixture: default Claude discovery would
	// fail on git worktree inspection if it ran before the host preflight.
	repo := t.TempDir()
	writeAuditFixtureRepo(t, repo, map[string]string{"I072": "routine"})
	path := writeModelHostConfig(t, `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:x","models":{"m":{"efforts":["high"],"token":"secret"}}}},"pins":{}}`)
	var out, errs bytes.Buffer
	code := cmdAuditRoutingWithHostPathAndDefaults(
		[]string{"--dir", repo}, &out, &errs, path, func(string) (string, error) { return "/bin/tool", nil },
		func(string) ([]string, error) {
			t.Fatal("default Claude transcript discovery ran before host preflight")
			return nil, nil
		},
		func() (string, error) {
			t.Fatal("default Codex session discovery ran before host preflight")
			return "", nil
		},
	)
	if code != 2 || out.Len() != 0 || !strings.Contains(errs.String(), "host routing configuration") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), errs.String())
	}
}

func TestModelValidateHostCommandMatrix(t *testing.T) {
	repo := writeModelHostWorkflow(t, "template_version: 13\nmodel_routing:\n  codex.primary: repository-safe\n")
	noHost := filepath.Join(t.TempDir(), "routing-host.json")
	code, out, errs := runModelWithHostPath(t, noHost, "--dir", repo, "validate", "codex", "primary")
	if code != 0 || out != "repository-safe\n" || errs != "" {
		t.Fatalf("no host: code=%d stdout=%q stderr=%q", code, out, errs)
	}

	identical := writeModelHostConfig(t, `{
  "schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:x","models":{"repository-safe":{"efforts":["high"]}}}},
  "pins":{"codex.primary":{"model":"repository-safe","effort":"high"}}}`)
	for _, args := range [][]string{
		{"--dir", repo, "validate", "--expect", "repository-safe", "codex", "primary"},
		{"--dir", repo, "validate", "--expect", "wrong-safe", "codex", "primary"},
	} {
		code, out, errs = runModelWithHostPath(t, identical, args...)
		if args[4] == "repository-safe" {
			if code != 0 || out != "repository-safe\n" || errs != "" {
				t.Fatalf("identical pin %v: code=%d stdout=%q stderr=%q", args, code, out, errs)
			}
		} else if code != 1 || out != "" || !strings.Contains(errs, "unmapped-dispatch") {
			t.Fatalf("post-identity expect %v: code=%d stdout=%q stderr=%q", args, code, out, errs)
		}
	}

	forbidden := writeModelHostConfig(t, `{
  "schema_version":1,"host_id":"host","harnesses":{"codex":{"available":true,"executable":"codex","launch_contract_ref":"fleet:x","models":{"repository-safe":{"efforts":["high"]},"unsafe pin":{"efforts":["high"]}}}},
  "pins":{"codex.primary":{"model":"unsafe pin","effort":"high"}}}`)
	code, out, errs = runModelWithHostPath(t, forbidden, "--dir", repo, "validate", "--expect", "repository-safe", "codex", "primary")
	if code != 2 || out != "" || !strings.Contains(errs, "host pin failed launch policy") || strings.Contains(errs, "unsafe pin") {
		t.Fatalf("forbidden pin: code=%d stdout=%q stderr=%q", code, out, errs)
	}
}

func runModelWithHostPath(t *testing.T, hostPath string, args ...string) (int, string, string) {
	return runModelWithHostPathAndLookup(t, hostPath, func(string) (string, error) { return "/bin/tool", nil }, args...)
}

func runModelWithHostPathAndLookup(t *testing.T, hostPath string, lookup func(string) (string, error), args ...string) (int, string, string) {
	t.Helper()
	var out, errs bytes.Buffer
	code := cmdModelWithHostPath(args, &out, &errs, hostPath, lookup)
	return code, out.String(), errs.String()
}

func writeModelHostWorkflow(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "WORKFLOW.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeModelHostConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routing-host.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
