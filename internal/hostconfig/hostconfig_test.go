package hostconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPathUsesInjectedUserConfigDirectory(t *testing.T) {
	got, err := defaultPath(func() (string, error) { return "/config", nil })
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/config", "spine", "routing-host.json"); got != want {
		t.Fatalf("defaultPath() = %q, want %q", got, want)
	}
}

func TestLoadAbsentFileReturnsErrNotConfiguredWithoutCreatingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spine", "routing-host.json")
	_, err := Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/true", nil })
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Load() error = %v, want ErrNotConfigured", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("Load() created %q: stat error = %v", path, statErr)
	}
}

func TestLoadValidatesExactRouteAndPin(t *testing.T) {
	path := writeConfig(t, `{
  "schema_version": 1,
  "host_id": "work-mac",
  "harnesses": {
    "claude": {
      "available": true,
      "executable": "claude-auto",
      "launch_contract_ref": "owner-verified:gateway",
      "models": {
        "gpt-5.6-sol": {"efforts": ["high"], "observed_ids": ["gateway/gpt-5.6-sol"]}
      }
    }
  },
  "pins": {"claude.primary": {"model": "gpt-5.6-sol", "effort": "high"}}
}`)
	config, err := Load(path, []string{"claude"}, func(name string) (string, error) {
		if name != "claude-auto" {
			t.Fatalf("lookup name = %q", name)
		}
		return "/private/bin/claude-auto", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Pins["claude.primary"]; got.Model != "gpt-5.6-sol" || got.Effort != "high" {
		t.Fatalf("pin = %#v", got)
	}
}

func TestLoadRejectsClosedSchemaAndDuplicateRules(t *testing.T) {
	valid := `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:1", "models": {"m": {"efforts": ["high"], "observed_ids": ["seen"]}}}
  }, "pins": {"claude.primary": {"model": "m", "effort": "high"}}
}`
	for _, tc := range []struct {
		name, content string
	}{
		{"unknown root", strings.Replace(valid, `"host_id": "host",`, `"host_id": "host", "token": "secret",`, 1)},
		{"forbidden base URL", strings.Replace(valid, `"host_id": "host",`, `"host_id": "host", "base_url": "https://example.test",`, 1)},
		{"forbidden auth header", strings.Replace(valid, `"host_id": "host",`, `"host_id": "host", "auth_header": "Bearer secret",`, 1)},
		{"forbidden model overrides", strings.Replace(valid, `"host_id": "host",`, `"host_id": "host", "modelOverrides": {},`, 1)},
		{"unknown harness", strings.Replace(valid, `"available": true,`, `"available": true, "env": {},`, 1)},
		{"unknown route", strings.Replace(valid, `"efforts": ["high"]`, `"efforts": ["high"], "args": []`, 1)},
		{"unknown pin", strings.Replace(valid, `"model": "m",`, `"model": "m", "credentials": "x",`, 1)},
		{"duplicate JSON key", strings.Replace(valid, `"host_id": "host"`, `"host_id": "host", "host_id": "other"`, 1)},
		{"duplicate effort", strings.Replace(valid, `["high"]`, `["high", "high"]`, 1)},
		{"duplicate observed ID", strings.Replace(valid, `"seen"]`, `"seen", "seen"]`, 1)},
		{"empty identifier", strings.Replace(valid, `"host"`, `""`, 1)},
		{"control identifier", strings.Replace(valid, `"host"`, `"host\u000a"`, 1)},
		{"other control identifier", strings.Replace(valid, `"host"`, `"host\u001f"`, 1)},
		{"unknown flavor pin", strings.Replace(valid, `claude.primary`, `codex.primary`, 1)},
		{"unknown tier pin", strings.Replace(valid, `claude.primary`, `claude.expert`, 1)},
		{"unavailable pin harness", strings.Replace(valid, `"available": true`, `"available": false`, 1)},
		{"pin missing route", strings.Replace(valid, `"model": "m"`, `"model": "other"`, 1)},
		{"pin unsupported effort", strings.Replace(valid, `"effort": "high"`, `"effort": "low"`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, tc.content)
			_, err := Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/claude", nil })
			if err == nil {
				t.Fatal("Load() error = nil")
			}
			if !strings.Contains(err.Error(), path) {
				t.Fatalf("error %q does not name config path %q", err, path)
			}
		})
	}
}

func TestLoadRejectsCaseMismatchedSchemaMembersAtEveryObjectDepth(t *testing.T) {
	valid := `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:1", "models": {"m": {"efforts": ["high"]}}}
  }, "pins": {"claude.primary": {"model": "m", "effort": "high"}}
}`
	for _, tc := range []struct {
		name, old, replacement string
	}{
		{"root", `"schema_version"`, `"SCHEMA_VERSION"`},
		{"harness", `"available"`, `"AVAILABLE"`},
		{"route", `"efforts"`, `"EFFORTS"`},
		{"pin", `"model"`, `"MODEL"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, strings.Replace(valid, tc.old, tc.replacement, 1))
			if _, err := Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/claude", nil }); err == nil {
				t.Fatal("Load() accepted case-mismatched schema member")
			}
		})
	}
}

func TestLoadRejectsExplicitNullForTypedOptionalMembers(t *testing.T) {
	valid := `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:1", "models": {"m": {"efforts": ["high"], "observed_ids": ["seen"], "gateway_ref": "gateway"}}}
  }, "pins": {"claude.primary": {"model": "m", "effort": "high", "evidence_refs": ["evidence"]}}
}`
	for _, tc := range []struct{ name, old, replacement string }{
		{"observed IDs", `"observed_ids": ["seen"]`, `"observed_ids": null`},
		{"gateway reference", `"gateway_ref": "gateway"`, `"gateway_ref": null`},
		{"evidence references", `"evidence_refs": ["evidence"]`, `"evidence_refs": null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, strings.Replace(valid, tc.old, tc.replacement, 1))
			if _, err := Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/claude", nil }); err == nil {
				t.Fatal("Load() accepted explicit null for typed optional member")
			}
		})
	}
}

func TestLoadRejectsMalformedUTF8BeforeJSONDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routing-host.json")
	content := []byte(`{"schema_version":1,"host_id":"host`)
	content = append(content, 0xff)
	content = append(content, []byte(`","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:1","models":{"m":{"efforts":["high"]}}}},"pins":{}}`)...)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/claude", nil }); err == nil {
		t.Fatal("Load() accepted malformed UTF-8")
	}
}

func TestLoadDiagnosticsDoNotExposeRawConfigurationValues(t *testing.T) {
	const secret = "raw-observed-id-must-not-escape"
	path := writeConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:1", "models": {
      "m": {"efforts": ["high"], "observed_ids": ["`+secret+`"]},
      "n": {"efforts": ["high"], "observed_ids": ["`+secret+`"]}
    }}
  }, "pins": {}}
`)
	_, err := Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/claude", nil })
	if err == nil {
		t.Fatal("Load() accepted duplicate observed ID")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("Load() diagnostic leaked raw configuration value: %q", err)
	}
}

func TestLoadAllowsEqualRoutesInDifferentHarnessesAndOnlyLooksUpExecutables(t *testing.T) {
	path := writeConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:1", "models": {"m": {"efforts": ["high"]}}},
    "codex": {"available": true, "executable": "codex", "launch_contract_ref": "fleet:2", "models": {"m": {"efforts": ["high"]}}}
  }, "pins": {}}
`)
	var calls []string
	_, err := Load(path, []string{"claude", "codex"}, func(name string) (string, error) {
		calls = append(calls, name)
		return "/bin/" + name, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "claude,codex" {
		t.Fatalf("lookup calls = %v", calls)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routing-host.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
