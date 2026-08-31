package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/russellpope/spine/internal/hostconfig"
)

func TestDeclarationEvidenceRequiresExactHostRouteAndIdentity(t *testing.T) {
	path := writeAuditHostConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {
      "gateway-pinned": {"efforts": ["high"], "observed_ids": ["gateway/pinned"]},
      "gateway-other": {"efforts": ["high"], "observed_ids": ["gateway/other"]}
    }}
  }, "pins": {"claude.routine": {"model": "gateway-pinned", "effort": "high"}}
}`)
	config, err := hostconfig.Load(path, []string{"claude"}, func(string) (string, error) { return "/bin/claude", nil })
	if err != nil {
		t.Fatal(err)
	}
	routes := newObservedRouteIndex(config)
	declared := DeclarationEvidence{
		Identity:       evidenceIdentity{source: "claude", session: "s1", dispatch: "d1"},
		Harness:        "claude",
		Model:          "gateway-pinned",
		ExpectedModel:  "gateway-pinned",
		ExpectedEffort: "high",
		DeclaredEffort: "high",
	}
	for _, tc := range []struct {
		name     string
		observed declarationObservation
		want     DeclarationModelState
	}{
		{"exact mapped worker", declarationObservation{identity: declared.Identity, model: "gateway/pinned", linkedWorker: true}, DeclarationModelConfirmed},
		{"different mapped route", declarationObservation{identity: declared.Identity, model: "gateway/other", linkedWorker: true}, DeclarationModelMismatch},
		{"unmapped raw id", declarationObservation{identity: declared.Identity, model: "gateway/unknown", linkedWorker: true}, DeclarationModelUnconfirmable},
		{"different dispatch", declarationObservation{identity: evidenceIdentity{source: "claude", session: "s1", dispatch: "d2"}, model: "gateway/pinned", linkedWorker: true}, DeclarationModelUnconfirmable},
		{"root only linkage", declarationObservation{identity: evidenceIdentity{source: "codex", session: "s1"}, model: "gateway/pinned", linkedWorker: true}, DeclarationModelUnconfirmable},
		{"unlinked worker", declarationObservation{identity: declared.Identity, model: "gateway/pinned"}, DeclarationModelUnconfirmable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := judgeDeclarationModel(declared, tc.observed, routes)
			if got != tc.want {
				t.Fatalf("model state = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAuditHostDeclarationUsesFinalPinAndLeavesAbsentObservedEffortUnconfirmable(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I074": "routine"})
	transcripts := t.TempDir()
	writeAuditHostDispatch(t, transcripts, repo, "gateway/pinned")
	hostPath := writeAuditHostConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {
      "gateway-pinned": {"efforts": ["high"], "observed_ids": ["gateway/pinned"]}
    }}
  }, "pins": {"claude.routine": {"model": "gateway-pinned", "effort": "high"}}
}`)
	rep, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts}, hostPath, func(string) (string, error) { return "/bin/claude", nil })
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I074"]
	if row.Verdict != VerdictDeclarationUnconfirmable {
		t.Fatalf("verdict = %q (%s), want %q", row.Verdict, row.Detail, VerdictDeclarationUnconfirmable)
	}
	if row.ExpectedEffort != "high" || row.DeclaredEffort != "high" || row.ObservedEffort != "-" {
		t.Fatalf("effort detail = expected=%q declared=%q observed=%q, want high/high/-", row.ExpectedEffort, row.DeclaredEffort, row.ObservedEffort)
	}
}

func TestAuditHostDeclarationResolutionFailuresRemainExplainableUnconfirmable(t *testing.T) {
	for _, tc := range []struct {
		name string
		host string
	}{
		{
			name: "configured unavailable harness",
			host: `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": false, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {}}
  }, "pins": {}}
`,
		},
		{
			name: "unreachable unpinned final route",
			host: `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": true, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {
      "other": {"efforts": ["high"]}
    }}
  }, "pins": {}}
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I074": "routine"})
			transcripts := t.TempDir()
			writeAuditHostDispatch(t, transcripts, repo, "gateway/pinned")

			rep, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts}, writeAuditHostConfig(t, tc.host), func(string) (string, error) { return "/bin/claude", nil })
			if err != nil {
				t.Fatal(err)
			}
			row := rowsByID(t, rep)["I074"]
			if row.Verdict != VerdictDeclarationUnconfirmable || rep.Blocking() {
				t.Fatalf("verdict = %q (%s), blocking=%v; want explainable nonblocking %q", row.Verdict, row.Detail, rep.Blocking(), VerdictDeclarationUnconfirmable)
			}
			if len(row.DeclarationEvents) != 1 {
				t.Fatalf("declaration events = %#v, want one retained completed declaration", row.DeclarationEvents)
			}
			event := row.DeclarationEvents[0]
			if event.Harness != "claude" || event.Model != "gateway-pinned" || event.DeclaredEffort != "high" || event.Correlation != "source:claude session:s1 dispatch:toolu_1" {
				t.Fatalf("raw declaration = %#v, want retained triple and exact identity", event)
			}
			if event.ExpectedModel != "" || event.ExpectedEffort != "" || event.ModelStatus != DeclarationModelUnconfirmable || event.Verdict != VerdictDeclarationUnconfirmable {
				t.Fatalf("resolution failure event = %#v, want missing expected route and unconfirmable proof", event)
			}
		})
	}
}

func TestAuditNoHostKeepsDeclaredAndLegacyModelOnlyPathsByteCompatible(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I074": "routine"})
	transcripts := t.TempDir()
	writeAuditHostDispatch(t, transcripts, repo, "gateway/pinned")
	missingHost := filepath.Join(t.TempDir(), "missing-routing-host.json")

	rep, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts}, missingHost, func(string) (string, error) { return "/bin/claude", nil })
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I074"]
	if len(row.DeclarationEvents) != 0 {
		t.Fatalf("no-host declaration events = %#v, want the existing declared-only path", row.DeclarationEvents)
	}
	if row.ExpectedEffort != "medium" || row.DeclaredEffort != "high" || row.DeclarationStatus != "unauthorized-declaration" || row.ObservedEffort != "-" {
		t.Fatalf("no-host declaration = %#v, want existing declared-only effort fields", row)
	}

	legacyRepo := filepath.Join("testdata", "clean", "repo")
	legacyTranscripts := filepath.Join("testdata", "clean", "transcripts")
	first, err := runWithHostPath(Options{RepoDir: legacyRepo, ClaudeTranscriptsDir: legacyTranscripts}, filepath.Join(t.TempDir(), "missing-one.json"), func(string) (string, error) { return "/bin/claude", nil })
	if err != nil {
		t.Fatal(err)
	}
	second, err := runWithHostPath(Options{RepoDir: legacyRepo, ClaudeTranscriptsDir: legacyTranscripts}, filepath.Join(t.TempDir(), "missing-two.json"), func(string) (string, error) { return "/bin/claude", nil })
	if err != nil {
		t.Fatal(err)
	}
	if !sameAuditReport(first, second) {
		t.Fatalf("legacy model-only no-host report changed\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestAuditHostDeclarationFiltersOnlyItsOwnLegacyIdentity(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I074": "routine"})
	transcripts := t.TempDir()
	writeAuditHostDispatch(t, transcripts, repo, "gateway/pinned")
	writeOrphanSubagent(t, transcripts, "s1", "independent", "other-dispatch", repo, "I074 independent legacy evidence", "claude-haiku-4-5")
	host := writeAuditHostConfig(t, `{
  "schema_version": 1, "host_id": "host", "harnesses": {
    "claude": {"available": false, "executable": "claude", "launch_contract_ref": "fleet:x", "models": {}}
  }, "pins": {}}
`)

	rep, err := runWithHostPath(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts}, host, func(string) (string, error) { return "/bin/claude", nil })
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I074"]
	if len(row.DeclarationEvents) != 1 || row.DeclarationEvents[0].Verdict != VerdictDeclarationUnconfirmable {
		t.Fatalf("declaration events = %#v, want one unconfirmable completed declaration", row.DeclarationEvents)
	}
	if row.Verdict != VerdictSilentDescent || !rep.Blocking() {
		t.Fatalf("combined verdict = %q (%s), blocking=%v; want independent legacy silent descent", row.Verdict, row.Detail, rep.Blocking())
	}
}

func writeAuditHostDispatch(t *testing.T, transcripts, repo, observedModel string) {
	t.Helper()
	line := `{"type":"assistant","cwd":` + mustJSON(t, repo) + `,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"herdr agent start worker --kind claude --pane 1 -- claude --model gateway-pinned --effort high I074"}}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(transcripts, "s1.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(transcripts, "s1", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "agent-worker.meta.json"), []byte(`{"toolUseId":"toolu_1","description":"I074 worker"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "agent-worker.jsonl"), []byte(`{"type":"assistant","cwd":`+mustJSON(t, repo)+`,"message":{"model":`+mustJSON(t, observedModel)+`,"role":"assistant","content":[]}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value string) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

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
