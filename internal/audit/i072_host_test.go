package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditHostConfigPreflightRejectsInvalidConfigBeforeTranscriptTraversal(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I072": "primary"})
	transcripts := t.TempDir()
	path := writeAuditHostConfig(t, `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:x","models":{"m":{"efforts":["high"],"token":"secret"}}}},"pins":{}}`)
	report, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts}, path, func(string) (string, error) { return "/bin/tool", nil })
	if err == nil || len(report.Tickets) != 0 || len(report.Unmatched) != 0 {
		t.Fatalf("report=%#v err=%v, want configuration failure before traversal", report, err)
	}
}

func TestAuditHostPreflightLeavesPreferenceVerdictsUntouched(t *testing.T) {
	repo := filepath.Join("testdata", "clean", "repo")
	transcripts := filepath.Join("testdata", "clean", "transcripts")
	baseline, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	path := writeAuditHostConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {"host-safe": {"efforts": ["high"], "observed_ids": ["gateway/host-safe"]}}}
  }, "pins": {"claude.primary": {"model": "host-safe", "effort": "high"}}
}`)
	got, err := runWithHostPath(
		Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts},
		path,
		func(string) (string, error) { return "/bin/tool", nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if !sameAuditReport(baseline, got) {
		t.Fatalf("host preflight changed audit report\nbaseline=%#v\ngot=%#v", baseline, got)
	}
}

func TestAuditHostPreflightRejectsEveryInvalidBoundaryBeforeTraversal(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I072": "primary"})
	for _, tc := range []struct{ name, config string }{
		{"malformed", `{"schema_version":`},
		{"path-bearing executable", `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"./claude","launch_contract_ref":"fleet:x","models":{"m":{"efforts":["high"]}}}},"pins":{}}`},
		{"unreachable pin", `{"schema_version":1,"host_id":"host","harnesses":{"claude":{"available":true,"executable":"claude","launch_contract_ref":"fleet:x","models":{"m":{"efforts":["high"]}}}},"pins":{"claude.primary":{"model":"other","effort":"high"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeAuditHostConfig(t, tc.config)
			report, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: t.TempDir()}, path, func(string) (string, error) { return "/bin/tool", nil })
			if err == nil || len(report.Tickets) != 0 || len(report.Unmatched) != 0 {
				t.Fatalf("report=%#v err=%v, want preflight configuration failure", report, err)
			}
		})
	}
}

func TestAuditHostPreflightAllowsUnpinnedUnreachablePreference(t *testing.T) {
	repo := filepath.Join("testdata", "clean", "repo")
	transcripts := filepath.Join("testdata", "clean", "transcripts")
	path := writeAuditHostConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {"not-the-preference": {"efforts": ["high"]}}}
  }, "pins": {}}
`)
	if _, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts}, path, func(string) (string, error) { return "/bin/tool", nil }); err != nil {
		t.Fatalf("unpinned unreachable preference rejected: %v", err)
	}
}

func writeAuditHostConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routing-host.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func sameAuditReport(left, right Report) bool {
	return string(mustMarshalAuditReport(left)) == string(mustMarshalAuditReport(right))
}

func mustMarshalAuditReport(report Report) []byte {
	data, err := json.Marshal(report)
	if err != nil {
		panic(err)
	}
	return data
}
