package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeWorkflowAgent(t *testing.T, transcripts, session, workflow, agent, repo, prompt, model, agentType string, metaModel bool) {
	t.Helper()
	dir := filepath.Join(transcripts, session, "subagents", "workflows", workflow)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "agent-"+agent)
	lines := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":%q}}`+"\n", repo, prompt)
	if model != "" && !metaModel {
		lines += fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":%q,"content":[{"type":"text","text":"done"}]}}`+"\n", repo, model)
	}
	if err := os.WriteFile(base+".jsonl", []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(`{"agentType":%q,"spawnDepth":1}`, agentType)
	if metaModel {
		meta = fmt.Sprintf(`{"agentType":%q,"spawnDepth":1,"model":%q}`, agentType, model)
	}
	if err := os.WriteFile(base+".meta.json", []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowSubagentOpeningLineProvidesTicketEvidence(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I052": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo,
		"Implement ticket I052 (docs/issues/I052-example.md) completely.\nMore context.",
		"claude-sonnet-5", "workflow-subagent", false)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I052"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I052 verdict = %s (%s), want match from workflow opening line", row.Verdict, row.Detail)
	}
}

func TestWorkflowSubagentUsesMetadataModelWhenTranscriptHasNone(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I054": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo,
		"Implement ticket I054", "claude-sonnet-5", "workflow-subagent", true)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I054"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I054 verdict = %s (%s), want metadata-model match", row.Verdict, row.Detail)
	}
}

func TestWorkflowCodeReviewerRemainsExcluded(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I053": "primary"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "guardian", repo,
		"Review ticket I053", "claude-haiku-4-5", "code-reviewer", false)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I053"]
	if row.Verdict != VerdictNoTranscript {
		t.Fatalf("I053 verdict = %s (%s), workflow code-reviewer must stay excluded", row.Verdict, row.Detail)
	}
}
