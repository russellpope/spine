package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWorkflowAgent(t *testing.T, transcripts, session, workflow, agent, repo, prompt, model, agentType string) {
	t.Helper()
	dir := filepath.Join(transcripts, session, "subagents", "workflows", workflow)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "agent-"+agent)
	lines := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":%q}}`+"\n", repo, prompt)
	if model != "" {
		lines += fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":%q,"content":[{"type":"text","text":"done"}]}}`+"\n", repo, model)
	}
	if err := os.WriteFile(base+".jsonl", []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(`{"agentType":%q,"spawnDepth":1}`, agentType)
	if err := os.WriteFile(base+".meta.json", []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeWorkflowRun writes the sibling run metadata observed in the Workflow
// layout. Per-agent sidecars only admit the transcript; agent-correlated
// workflowProgress entries carry fallback model evidence.
func writeWorkflowRun(t *testing.T, transcripts, session, workflow, agent, model string) {
	t.Helper()
	writeWorkflowRunRaw(t, transcripts, session, workflow, fmt.Sprintf(`{"defaultModel":"claude-haiku-4-5","workflowProgress":[{"agentId":%q,"model":%q,"label":"implement"}]}`,
		agent, model))
}

func writeWorkflowRunRaw(t *testing.T, transcripts, session, workflow, raw string) {
	t.Helper()
	path := filepath.Join(transcripts, session, "workflows", workflow+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowSubagentOpeningLineProvidesTicketEvidence(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I052": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo,
		"Implement ticket I052 (docs/issues/I052-example.md) completely.\nMore context.",
		"claude-sonnet-5", "workflow-subagent")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I052"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I052 verdict = %s (%s), want match from workflow opening line", row.Verdict, row.Detail)
	}
}

func TestWorkflowSubagentUsesAgentCorrelatedRunMetadataModelWhenTranscriptHasNone(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I054": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo,
		"Implement ticket I054", "", "workflow-subagent")
	writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I054"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I054 verdict = %s (%s), want agent-correlated workflow run metadata model match", row.Verdict, row.Detail)
	}
}

func TestWorkflowRunMetadataFallbackFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name        string
		runSession  string
		runWorkflow string
		raw         string
		wantWarn    string
	}{
		{
			name:     "missing expected run metadata",
			wantWarn: `workflow run metadata unavailable for session "session-1" workflow "wf_1" — model fallback skipped`,
		},
		{
			name:     "malformed expected run metadata",
			raw:      `{"workflowProgress":`,
			wantWarn: "malformed workflow run metadata — model fallback skipped",
		},
		{
			name:     "missing workflow progress",
			raw:      `{"defaultModel":"claude-haiku-4-5"}`,
			wantWarn: "workflow run metadata has no workflowProgress — model fallback skipped",
		},
		{
			name:     "mismatched agent metadata",
			raw:      `{"workflowProgress":[{"agentId":"other-worker","model":"claude-sonnet-5"}]}`,
			wantWarn: `workflow run metadata has no exact entry for agent "worker" — model fallback skipped`,
		},
		{
			name:     "ambiguous matching agent metadata",
			raw:      `{"workflowProgress":[{"agentId":"worker","model":"claude-sonnet-5"},{"agentId":"worker","model":"claude-haiku-4-5"}]}`,
			wantWarn: `workflow run metadata has multiple entries for agent "worker" — model fallback skipped`,
		},
		{
			name:     "matching agent with no model",
			raw:      `{"workflowProgress":[{"agentId":"worker","label":"implement"}]}`,
			wantWarn: `workflow run metadata entry for agent "worker" has no model — model fallback skipped`,
		},
		{
			name:        "wrong session metadata is ignored",
			runSession:  "session-2",
			runWorkflow: "wf_1",
			raw:         `{"workflowProgress":[{"agentId":"worker","model":"claude-sonnet-5"}]}`,
			wantWarn:    `workflow run metadata unavailable for session "session-1" workflow "wf_1" — model fallback skipped`,
		},
		{
			name:        "wrong workflow metadata is ignored",
			runSession:  "session-1",
			runWorkflow: "wf_other",
			raw:         `{"workflowProgress":[{"agentId":"worker","model":"claude-sonnet-5"}]}`,
			wantWarn:    `workflow run metadata unavailable for session "session-1" workflow "wf_1" — model fallback skipped`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I054": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I054", "", "workflow-subagent")
			if tc.raw != "" {
				session, workflow := tc.runSession, tc.runWorkflow
				if session == "" {
					session = "session-1"
				}
				if workflow == "" {
					workflow = "wf_1"
				}
				writeWorkflowRunRaw(t, transcripts, session, workflow, tc.raw)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if got := rowsByID(t, rep)["I054"].Verdict; got != VerdictNoTranscript {
				t.Fatalf("I054 verdict = %s, want no-transcript when metadata fallback is unsafe", got)
			}
			if !warningContains(rep.Warnings, tc.wantWarn) {
				t.Fatalf("warnings = %q, want %q", rep.Warnings, tc.wantWarn)
			}
		})
	}
}

func warningContains(warnings []string, want string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return true
		}
	}
	return false
}

func TestWorkflowCodeReviewerRemainsExcluded(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I053": "primary"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "guardian", repo,
		"Review ticket I053", "claude-haiku-4-5", "code-reviewer")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I053"]
	if row.Verdict != VerdictNoTranscript {
		t.Fatalf("I053 verdict = %s (%s), workflow code-reviewer must stay excluded", row.Verdict, row.Detail)
	}
}

func TestWorkflowUsesOnlyFirstUserMessageAndRejectsMultiTicketOpening(t *testing.T) {
	for _, tc := range []struct {
		name, first, later string
		wantWarning        bool
	}{
		{name: "later ticket is not opening evidence", first: "General task with no ticket", later: "Implement I052"},
		{name: "multi-ticket opening is ambiguous", first: "Implement I052 and I053", wantWarning: true},
		{name: "comma-adjacent tickets are ambiguous", first: "Implement I052,I053", wantWarning: true},
		{name: "space-adjacent tickets are ambiguous", first: "Implement I052 I053", wantWarning: true},
		{name: "non-audited explicit ticket is ambiguous", first: "Implement I052 and I999", wantWarning: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I052": "routine", "I053": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, tc.first, "claude-sonnet-5", "workflow-subagent")
			if tc.later != "" {
				path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl")
				raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":%q}}`+"\n", repo, tc.first) +
					fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":%q}}`+"\n", repo, tc.later) +
					fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"done"}]}}`+"\n", repo)
				if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			rows := rowsByID(t, rep)
			for _, id := range []string{"I052", "I053"} {
				if got := rows[id].Verdict; got != VerdictNoTranscript {
					t.Errorf("%s verdict = %s (%s), want no-transcript", id, got, rows[id].Detail)
				}
			}
			warningFound := false
			for _, warning := range rep.Warnings {
				warningFound = warningFound || strings.Contains(warning, "workflow opening line names multiple tickets")
			}
			if warningFound != tc.wantWarning {
				t.Errorf("multi-ticket warning = %v, want %v; warnings=%q", warningFound, tc.wantWarning, rep.Warnings)
			}
		})
	}
}

func TestWorkflowOpeningTextBlockAndTranscriptModelTakePrecedence(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I054": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "unused", "", "workflow-subagent")
	writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-haiku-4-5")
	base := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker")
	raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":[{"type":"text","text":"Implement I054\nDetails"}]}}`+"\n", repo) +
		fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"done"}]}}`+"\n", repo)
	if err := os.WriteFile(base+".jsonl", []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I054"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I054 verdict = %s (%s), transcript model must outrank metadata fallback", row.Verdict, row.Detail)
	}
}

func TestWorkflowMalformedMissingUnknownAndDeeperMetadataStayExcluded(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I055": "routine"})
	transcripts := t.TempDir()
	for _, agent := range []string{"missing", "malformed", "unknown", "deeper"} {
		writeWorkflowAgent(t, transcripts, "session-1", "wf_1", agent, repo,
			"Implement I055", "claude-sonnet-5", "workflow-subagent")
	}
	dir := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1")
	if err := os.Remove(filepath.Join(dir, "agent-missing.meta.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent-malformed.meta.json"), []byte(`{"agentType":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent-unknown.meta.json"), []byte(`{"agentType":"unknown","spawnDepth":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent-deeper.meta.json"), []byte(`{"agentType":"workflow-subagent","spawnDepth":2}`), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I055"]
	if row.Verdict != VerdictNoTranscript {
		t.Fatalf("I055 verdict = %s (%s), invalid workflow metadata must stay excluded", row.Verdict, row.Detail)
	}
	var missing, malformed bool
	for _, warning := range rep.Warnings {
		missing = missing || strings.Contains(warning, "agent-missing.meta.json: workflow metadata unreadable")
		malformed = malformed || strings.Contains(warning, "agent-malformed.meta.json: malformed workflow metadata")
	}
	if !missing || !malformed {
		t.Fatalf("workflow metadata warnings missing=%v malformed=%v: %q", missing, malformed, rep.Warnings)
	}
}
