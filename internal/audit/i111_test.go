package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeI111Dispatches(t *testing.T, path, cwd, ticketID string, models ...string) {
	t.Helper()
	var b strings.Builder
	for i, modelID := range models {
		fmt.Fprintf(&b,
			`{"type":"assistant","cwd":%q,"message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_%d","name":"Task","input":{"description":"%s: fixture work","model":%q,"prompt":"You are implementing ticket %s."}}]}}`+"\n",
			cwd, i+1, ticketID, modelID, ticketID)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeLayoutOpenweightsDispatchUsesOpenweightsMapping(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I111": "primary"})
	transcripts := t.TempDir()
	writeDispatchTranscript(t, dir, transcripts, map[string]string{"I111": "FW-Kimi-K3"})

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I111"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I111 verdict = %s (%s), want match against the openweights table", r.Verdict, r.Detail)
	}
}

func TestD28StillRejectsUnqualifiedOpenweightsDispatch(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I112": "primary"})
	transcripts := t.TempDir()
	line := `{"type":"assistant","message":{"model":"claude-fable-5","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Task","input":{"description":"I112: unrelated repo work","model":"FW-Kimi-K3","prompt":"You are implementing ticket I112."}}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(transcripts, "s1.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I112"]
	if r.Verdict != VerdictNoTranscript {
		t.Fatalf("I112 verdict = %s (%s), want no-transcript because D28 applies to the claude-layout source", r.Verdict, r.Detail)
	}
	if len(rep.Unmatched) != 1 || !strings.Contains(rep.Unmatched[0].Description, "I112") {
		t.Fatalf("want disqualified openweights dispatch in Unmatched, got %+v", rep.Unmatched)
	}
}

func TestClaudeLayoutMixedModelsAreJudgedPerToken(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I113": "primary"})
	transcripts := t.TempDir()
	writeI111Dispatches(t, filepath.Join(transcripts, "s1.jsonl"), dir, "I113",
		"FW-Kimi-K3", "claude-fable-5")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I113"]
	if got := strings.Join(r.Actuals, ","); got != "FW-Kimi-K3,claude-fable-5" {
		t.Fatalf("I113 actuals = %q, want both tokens", got)
	}
	if r.Verdict != VerdictMatch {
		t.Fatalf("I113 verdict = %s (%s), want both tokens matched within their own harness", r.Verdict, r.Detail)
	}
}

func TestClaudeLayoutSubagentActualUsesOpenweightsMapping(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I114": "mechanical"})
	transcripts := t.TempDir()
	writeOrphanSubagent(t, transcripts, "s1", "openweights", "unlinked", dir,
		"I114: openweights worker", "FW-GLM-5.2")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I114"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I114 verdict = %s (%s), want subagent actual matched against openweights.mechanical", r.Verdict, r.Detail)
	}
}

func TestAmbiguousModelIDFallsBackToTranscriptSource(t *testing.T) {
	dir := t.TempDir()
	workflow := `profile: go-service
template_version: 11
model_routing:
  claude.routine: shared-model
  openweights.mechanical: shared-model
`
	writeAuditRepo(t, dir, workflow, map[string]string{"I115": "routine"})
	transcripts := t.TempDir()
	writeDispatchTranscript(t, dir, transcripts, map[string]string{"I115": "shared-model"})

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I115"]
	if r.Verdict != VerdictMatch {
		t.Fatalf("I115 verdict = %s (%s), want ambiguous id resolved within the claude source table", r.Verdict, r.Detail)
	}
}

func TestUnknownModelIDPreservesTranscriptSourceBehavior(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I116": "primary"})
	transcripts := t.TempDir()
	writeDispatchTranscript(t, dir, transcripts, map[string]string{"I116": "unknown-model"})

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I116"]
	if r.Verdict != VerdictUnmappedDispatch || !strings.Contains(r.Detail, "no claude entry") {
		t.Fatalf("I116 = %s (%s), want the prior claude-source unmapped verdict", r.Verdict, r.Detail)
	}
}

func TestD28StillRejectsUnqualifiedOpenweightsSubagent(t *testing.T) {
	dir := t.TempDir()
	writeAuditRepo(t, dir, gen9DefaultWorkflow, map[string]string{"I117": "mechanical"})
	transcripts := t.TempDir()
	writeOrphanSubagent(t, transcripts, "s1", "openweights", "unlinked", "",
		"I117: unrelated repo worker", "FW-GLM-5.2")

	rep, err := Run(Options{RepoDir: dir, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	r := rowsByID(t, rep)["I117"]
	if r.Verdict != VerdictNoTranscript {
		t.Fatalf("I117 verdict = %s (%s), want no-transcript because D28 applies to the claude-layout subagent", r.Verdict, r.Detail)
	}
}
