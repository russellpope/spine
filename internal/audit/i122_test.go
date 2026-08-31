package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestWorkflowRunMetadataIgnoresPhaseRowsBeforeExactAgentEntry(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I060": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_real", "worker", repo, "Implement I060", "", "workflow-subagent")
	writeWorkflowRunRaw(t, transcripts, "session-1", "wf_real", `{"defaultModel":"claude-haiku-4-5","workflowProgress":[{"index":0,"title":"Implement","type":"workflow_phase"},{"agentId":"worker","model":"claude-sonnet-5","type":"workflow_agent"}]}`)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	if row := rowsByID(t, rep)["I060"]; row.Verdict != VerdictMatch {
		t.Fatalf("I060 verdict = %s (%s), want match from the exact workflow agent entry after a phase row", row.Verdict, row.Detail)
	}
}

func TestWorkflowTranscriptRejectsUnsafeFiles(t *testing.T) {
	for _, tc := range []struct {
		name   string
		linkTo func(t *testing.T, transcripts, base string) string
	}{
		{
			name: "outside root symlink",
			linkTo: func(t *testing.T, transcripts, base string) string {
				t.Helper()
				outside := filepath.Join(t.TempDir(), "outside-agent.jsonl")
				if err := os.WriteFile(outside, []byte(`{"type":"user","cwd":"outside","message":{"role":"user","content":"Implement I061"}}`+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return outside
			},
		},
		{
			name: "same object symlink",
			linkTo: func(t *testing.T, transcripts, base string) string {
				t.Helper()
				target := filepath.Join(filepath.Dir(base), "worker-source.jsonl")
				if err := os.WriteFile(target, []byte(`{"type":"user","cwd":"same-object","message":{"role":"user","content":"Implement I061"}}`+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return target
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I061": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_link", "worker", repo, "Implement I061", "claude-sonnet-5", "workflow-subagent")
			base := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_link", "agent-worker")
			if err := os.Remove(base + ".jsonl"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(tc.linkTo(t, transcripts, base), base+".jsonl"); err != nil {
				t.Fatal(err)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I061"]; row.Verdict != VerdictNoTranscript {
				t.Fatalf("I061 verdict = %s (%s), want no-transcript after unsafe workflow JSONL", row.Verdict, row.Detail)
			}
			if !warningContains(rep.Warnings, "workflow transcript unsafe — transcript skipped") {
				t.Fatalf("warnings = %q, want unsafe workflow transcript warning", rep.Warnings)
			}
		})
	}
}

func TestWorkflowTranscriptRejectsAtomicReplacement(t *testing.T) {
	for attempt := 0; attempt < 16; attempt++ {
		repo := t.TempDir()
		writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I062": "routine"})
		transcripts := t.TempDir()
		writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I062", "claude-sonnet-5", "workflow-subagent")
		base := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker")
		replacement := base + ".replacement"
		if err := os.WriteFile(replacement, []byte(`{"type":"user","cwd":`+mustJSON(t, repo)+`,"message":{"role":"user","content":"Implement I062"}}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		setWorkflowTranscriptBeforeOpen(t, func(path string) {
			if path == base+".jsonl" {
				if err := os.Rename(replacement, base+".jsonl"); err != nil {
					t.Fatal(err)
				}
			}
		})

		rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
		if err != nil {
			t.Fatal(err)
		}
		if row := rowsByID(t, rep)["I062"]; row.Verdict != VerdictNoTranscript {
			t.Fatalf("attempt %d: I062 verdict = %s (%s), want no-transcript after atomic workflow JSONL replacement", attempt, row.Verdict, row.Detail)
		}
		if !warningContains(rep.Warnings, "workflow transcript unsafe — transcript skipped") {
			t.Fatalf("attempt %d: warnings = %q, want unsafe workflow transcript warning", attempt, rep.Warnings)
		}
	}
}

// Replacing any named path component after the pre-open hook must invalidate
// workflow evidence. A retained descriptor into the renamed-away tree is safe
// to read, but it is no longer the evidence named by the transcript root.
func TestWorkflowEvidenceRejectsAncestorReplacement(t *testing.T) {
	for _, tc := range []struct {
		name       string
		evidence   string
		components []string
		warning    string
	}{
		{name: "transcript session", evidence: "transcript", components: []string{"session-1"}, warning: "workflow transcript unsafe — transcript skipped"},
		{name: "transcript subagents", evidence: "transcript", components: []string{"session-1", "subagents"}, warning: "workflow transcript unsafe — transcript skipped"},
		{name: "transcript workflows", evidence: "transcript", components: []string{"session-1", "subagents", "workflows"}, warning: "workflow transcript unsafe — transcript skipped"},
		{name: "transcript workflow directory", evidence: "transcript", components: []string{"session-1", "subagents", "workflows", "wf_1"}, warning: "workflow transcript unsafe — transcript skipped"},
		{name: "transcript file", evidence: "transcript", components: []string{"session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl"}, warning: "workflow transcript unsafe — transcript skipped"},
		{name: "sidecar session", evidence: "sidecar", components: []string{"session-1"}, warning: "workflow metadata unsafe — transcript skipped"},
		{name: "sidecar subagents", evidence: "sidecar", components: []string{"session-1", "subagents"}, warning: "workflow metadata unsafe — transcript skipped"},
		{name: "sidecar workflows", evidence: "sidecar", components: []string{"session-1", "subagents", "workflows"}, warning: "workflow metadata unsafe — transcript skipped"},
		{name: "sidecar workflow directory", evidence: "sidecar", components: []string{"session-1", "subagents", "workflows", "wf_1"}, warning: "workflow metadata unsafe — transcript skipped"},
		{name: "sidecar file", evidence: "sidecar", components: []string{"session-1", "subagents", "workflows", "wf_1", "agent-worker.meta.json"}, warning: "workflow metadata unsafe — transcript skipped"},
		{name: "run session", evidence: "run", components: []string{"session-1"}, warning: "workflow run metadata unsafe — model fallback skipped"},
		{name: "run workflows", evidence: "run", components: []string{"session-1", "workflows"}, warning: "workflow run metadata unsafe — model fallback skipped"},
		{name: "run file", evidence: "run", components: []string{"session-1", "workflows", "wf_1.json"}, warning: "workflow run metadata unsafe — model fallback skipped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for attempt := 0; attempt < 3; attempt++ {
				repo := t.TempDir()
				writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I915": "routine"})
				transcripts := t.TempDir()
				model := "claude-sonnet-5"
				if tc.evidence == "run" {
					model = ""
				}
				writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I915", model, "workflow-subagent")
				if tc.evidence == "run" {
					writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")
				}

				target := filepath.Join(append([]string{transcripts}, tc.components...)...)
				replaceWorkflowPathDuringOpen(t, tc.evidence, transcripts, repo, workflowEvidencePath(transcripts, tc.evidence), target)

				rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
				if err != nil {
					t.Fatal(err)
				}
				if row := rowsByID(t, rep)["I915"]; row.Verdict != VerdictNoTranscript {
					t.Fatalf("attempt %d: I915 = %+v, want no-transcript after replacing %s", attempt, row, tc.name)
				}
				if !warningContains(rep.Warnings, tc.warning) {
					t.Fatalf("attempt %d: warnings = %q, want %q", attempt, rep.Warnings, tc.warning)
				}
			}
		})
	}
}

// Replacing the configured transcript root after it is opened must invalidate
// every workflow evidence carrier. Revalidating only names beneath the retained
// root would otherwise accept the renamed-away tree as current evidence.
func TestWorkflowEvidenceRejectsTranscriptRootReplacement(t *testing.T) {
	for _, tc := range []struct {
		name     string
		evidence string
		warning  string
	}{
		{name: "transcript", evidence: "transcript", warning: "workflow transcript unsafe — transcript skipped"},
		{name: "sidecar", evidence: "sidecar", warning: "workflow metadata unsafe — transcript skipped"},
		{name: "run", evidence: "run", warning: "workflow run metadata unsafe — model fallback skipped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for attempt := 0; attempt < 3; attempt++ {
				repo := t.TempDir()
				writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I917": "routine"})
				transcripts := t.TempDir()
				model := "claude-sonnet-5"
				if tc.evidence == "run" {
					model = ""
				}
				writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I917", model, "workflow-subagent")
				if tc.evidence == "run" {
					writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")
				}

				replaceWorkflowPathDuringOpen(t, tc.evidence, transcripts, repo, workflowEvidencePath(transcripts, tc.evidence), transcripts)

				rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
				if err != nil {
					t.Fatal(err)
				}
				if row := rowsByID(t, rep)["I917"]; row.Verdict != VerdictNoTranscript {
					t.Fatalf("attempt %d: I917 = %+v, want no-transcript after replacing transcript root for %s evidence", attempt, row, tc.name)
				}
				if !warningContains(rep.Warnings, tc.warning) {
					t.Fatalf("attempt %d: warnings = %q, want %q", attempt, rep.Warnings, tc.warning)
				}
			}
		})
	}
}

func replaceWorkflowPathDuringOpen(t *testing.T, evidence, transcripts, repo, evidencePath, target string) {
	t.Helper()
	replaced := false
	hook := func(path string) {
		if replaced || path != evidencePath {
			return
		}
		replaced = true
		stale := target + ".stale"
		if err := os.Rename(target, stale); err != nil {
			t.Fatal(err)
		}
		switch evidence {
		case "transcript":
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I916", "claude-sonnet-5", "workflow-subagent")
		case "sidecar":
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I915", "claude-sonnet-5", "code-reviewer")
		case "run":
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I915", "", "workflow-subagent")
			writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-haiku-4-5")
		default:
			t.Fatalf("unknown workflow evidence %q", evidence)
		}
	}
	switch evidence {
	case "transcript":
		setWorkflowTranscriptBeforeOpen(t, hook)
	case "sidecar", "run":
		setWorkflowMetadataBeforeOpen(t, hook)
	default:
		t.Fatalf("unknown workflow evidence %q", evidence)
	}
}

func workflowEvidencePath(transcripts, evidence string) string {
	switch evidence {
	case "transcript", "sidecar":
		return filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker."+map[string]string{"transcript": "jsonl", "sidecar": "meta.json"}[evidence])
	case "run":
		return filepath.Join(transcripts, "session-1", "workflows", "wf_1.json")
	default:
		return ""
	}
}

// A temporary rename that restores the same object at the same named path is
// legitimate. The revalidation must reject changed identities, not directory
// activity by itself.
func TestWorkflowEvidenceAcceptsRestoredSameObject(t *testing.T) {
	for _, evidence := range []string{"transcript", "sidecar", "run"} {
		t.Run(evidence, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I916": "routine"})
			transcripts := t.TempDir()
			model := "claude-sonnet-5"
			if evidence == "run" {
				model = ""
			}
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I916", model, "workflow-subagent")
			if evidence == "run" {
				writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")
			}
			target := workflowEvidencePath(transcripts, evidence)
			hook := func(path string) {
				if path != target {
					return
				}
				staged := target + ".staged"
				if err := os.Rename(target, staged); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(staged, target); err != nil {
					t.Fatal(err)
				}
			}
			switch evidence {
			case "transcript":
				setWorkflowTranscriptBeforeOpen(t, hook)
			case "sidecar", "run":
				setWorkflowMetadataBeforeOpen(t, hook)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I916"]; row.Verdict != VerdictMatch {
				t.Fatalf("I916 = %+v, want match when the named path restores the same object", row)
			}
		})
	}
}

func TestWorkflowEvidenceAcceptsRestoredTranscriptRootObject(t *testing.T) {
	for _, evidence := range []string{"transcript", "sidecar", "run"} {
		t.Run(evidence, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I918": "routine"})
			transcriptsParent := t.TempDir()
			transcripts := filepath.Join(transcriptsParent, "transcripts")
			if err := os.Mkdir(transcripts, 0o755); err != nil {
				t.Fatal(err)
			}
			model := "claude-sonnet-5"
			if evidence == "run" {
				model = ""
			}
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I918", model, "workflow-subagent")
			if evidence == "run" {
				writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")
			}

			target := workflowEvidencePath(transcripts, evidence)
			hook := func(path string) {
				if path != target {
					return
				}
				staged := transcripts + ".staged"
				if err := os.Rename(transcripts, staged); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(staged, transcripts); err != nil {
					t.Fatal(err)
				}
			}
			switch evidence {
			case "transcript":
				setWorkflowTranscriptBeforeOpen(t, hook)
			case "sidecar", "run":
				setWorkflowMetadataBeforeOpen(t, hook)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I918"]; row.Verdict != VerdictMatch {
				t.Fatalf("I918 = %+v, want match when the transcript root path restores the same object", row)
			}
		})
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
			wantWarn: "malformed workflow run metadata — model fallback skipped",
		},
		{
			name:     "phase row carrying agent evidence",
			raw:      `{"workflowProgress":[{"type":"workflow_phase","agentId":"worker"}]}`,
			wantWarn: "malformed workflow run metadata — model fallback skipped",
		},
		{
			name:     "unknown typed workflow entry",
			raw:      `{"workflowProgress":[{"type":"workflow_worker","agentId":"worker","model":"claude-sonnet-5"}]}`,
			wantWarn: "malformed workflow run metadata — model fallback skipped",
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

// A workflow transcript may not mix one event's user-shaped message with an
// assistant-shaped envelope.  The event itself is the evidence carrier: an
// invalid carrier cannot claim either a ticket or its model fallback.
func TestWorkflowInvalidEventShapesNeverSupplyRoutingEvidence(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "assistant envelope user message", raw: `{"type":"assistant","message":{"role":"user","model":"claude-sonnet-5","content":"Implement I060"}}`},
		{name: "user envelope assistant message", raw: `{"type":"user","message":{"role":"assistant","content":"Implement I060"}}`},
		{name: "missing type", raw: `{"message":{"role":"user","content":"Implement I060"}}`},
		{name: "empty type", raw: `{"type":"","message":{"role":"user","content":"Implement I060"}}`},
		{name: "missing role", raw: `{"type":"user","message":{"content":"Implement I060"}}`},
		{name: "empty role", raw: `{"type":"user","message":{"role":"","content":"Implement I060"}}`},
		{name: "case variant type value", raw: `{"type":"User","message":{"role":"user","content":"Implement I060"}}`},
		{name: "case variant role value", raw: `{"type":"user","message":{"role":"USER","content":"Implement I060"}}`},
		{name: "case variant envelope key", raw: `{"TYPE":"user","message":{"role":"user","content":"Implement I060"}}`},
		{name: "case variant nested key", raw: `{"type":"user","message":{"ROLE":"user","content":"Implement I060"}}`},
		{name: "duplicate envelope key", raw: `{"type":"assistant","type":"user","message":{"role":"user","content":"Implement I060"}}`},
		{name: "case equivalent envelope keys", raw: `{"type":"assistant","TYPE":"user","message":{"role":"user","content":"Implement I060"}}`},
		{name: "duplicate nested key", raw: `{"type":"user","message":{"role":"assistant","role":"user","content":"Implement I060"}}`},
		{name: "case equivalent nested keys", raw: `{"type":"user","message":{"role":"assistant","ROLE":"user","content":"Implement I060"}}`},
		{name: "missing content", raw: `{"type":"user","message":{"role":"user"}}`},
		{name: "null content", raw: `{"type":"user","message":{"role":"user","content":null}}`},
		{name: "non message object", raw: `{"type":"user","message":[]}`},
		{name: "non text content", raw: `{"type":"user","message":{"role":"user","content":1}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I060": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "unused", "", "workflow-subagent")
			base := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker")
			raw := strings.Replace(tc.raw, "{", fmt.Sprintf(`{"cwd":%q,`, repo), 1)
			if err := os.WriteFile(base+".jsonl", []byte(raw+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")

			var firstWarnings []string
			for attempt := 0; attempt < 3; attempt++ {
				rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
				if err != nil {
					t.Fatal(err)
				}
				row := rowsByID(t, rep)["I060"]
				if row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
					t.Fatalf("attempt %d: I060 = %+v, want no transcript evidence from an invalid workflow event", attempt, row)
				}
				if !warningContains(rep.Warnings, "1 malformed line(s) skipped") {
					t.Fatalf("attempt %d: warnings = %q, want one malformed workflow line warning", attempt, rep.Warnings)
				}
				if attempt == 0 {
					firstWarnings = rep.Warnings
				} else if !reflect.DeepEqual(rep.Warnings, firstWarnings) {
					t.Fatalf("attempt %d: warnings = %q, want stable %q", attempt, rep.Warnings, firstWarnings)
				}
			}
		})
	}
}

// Nested tool-use fields cross the workflow validator into parseLine, where
// encoding/json would otherwise case-fold aliases and apply last-member-wins.
// Each of these mutations must instead discard the whole carrier, independent
// of member order, so no nested dispatch or assistant model becomes evidence.
func TestWorkflowNestedSemanticKeysFailClosedBeforeGenericDispatchParsing(t *testing.T) {
	validInput := `{"description":"I121 nested dispatch","prompt":"Implement I121","model":"claude-haiku-4-5","command":"herdr agent start worker --kind claude -- claude --model claude-haiku-4-5 I121"}`
	tests := []struct {
		name  string
		block string
	}{
		{name: "exact duplicate content type", block: `{"type":"tool_use","type":"text","id":"tool-1","name":"Agent","input":` + validInput + `}`},
		{name: "content type alias first", block: `{"TYPE":"text","type":"tool_use","id":"tool-1","name":"Agent","input":` + validInput + `}`},
		{name: "content type alias last", block: `{"type":"tool_use","TYPE":"text","id":"tool-1","name":"Agent","input":` + validInput + `}`},
		{name: "exact duplicate tool id", block: `{"type":"tool_use","id":"tool-1","id":"tool-2","name":"Agent","input":` + validInput + `}`},
		{name: "tool id alias first", block: `{"type":"tool_use","ID":"tool-2","id":"tool-1","name":"Agent","input":` + validInput + `}`},
		{name: "tool id alias last", block: `{"type":"tool_use","id":"tool-1","ID":"tool-2","name":"Agent","input":` + validInput + `}`},
		{name: "exact duplicate tool name", block: `{"type":"tool_use","id":"tool-1","name":"noop","name":"Agent","input":` + validInput + `}`},
		{name: "tool name alias first", block: `{"type":"tool_use","id":"tool-1","NAME":"Agent","name":"noop","input":` + validInput + `}`},
		{name: "tool name alias last", block: `{"type":"tool_use","id":"tool-1","name":"noop","NAME":"Agent","input":` + validInput + `}`},
		{name: "exact duplicate tool input", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"ignore"},"input":` + validInput + `}`},
		{name: "tool input alias first", block: `{"type":"tool_use","id":"tool-1","name":"Agent","INPUT":` + validInput + `,"input":{"description":"ignore"}}`},
		{name: "tool input alias last", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"ignore"},"INPUT":` + validInput + `}`},
		{name: "exact duplicate description", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"ignore","description":"I121 nested dispatch","prompt":"Implement I121","model":"claude-haiku-4-5"}}`},
		{name: "description alias first", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"DESCRIPTION":"I121 nested dispatch","description":"ignore","prompt":"Implement I121","model":"claude-haiku-4-5"}}`},
		{name: "description alias last", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"ignore","DESCRIPTION":"I121 nested dispatch","prompt":"Implement I121","model":"claude-haiku-4-5"}}`},
		{name: "exact duplicate prompt", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"ignore","prompt":"Implement I121","model":"claude-haiku-4-5"}}`},
		{name: "prompt alias first", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"I121 nested dispatch","PROMPT":"Implement I121","prompt":"ignore","model":"claude-haiku-4-5"}}`},
		{name: "prompt alias last", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"ignore","PROMPT":"Implement I121","model":"claude-haiku-4-5"}}`},
		{name: "exact duplicate model", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"Implement I121","model":"claude-sonnet-5","model":"claude-haiku-4-5"}}`},
		{name: "model alias first", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"Implement I121","MODEL":"claude-haiku-4-5","model":"claude-sonnet-5"}}`},
		{name: "model alias last", block: `{"type":"tool_use","id":"tool-1","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"Implement I121","model":"claude-sonnet-5","MODEL":"claude-haiku-4-5"}}`},
		{name: "exact duplicate command", block: `{"type":"tool_use","id":"tool-1","name":"Bash","input":{"command":"echo ignore","command":"herdr agent start worker --kind claude -- claude --model claude-haiku-4-5 I121"}}`},
		{name: "command alias first", block: `{"type":"tool_use","id":"tool-1","name":"Bash","input":{"COMMAND":"herdr agent start worker --kind claude -- claude --model claude-haiku-4-5 I121","command":"echo ignore"}}`},
		{name: "command alias last", block: `{"type":"tool_use","id":"tool-1","name":"Bash","input":{"command":"echo ignore","COMMAND":"herdr agent start worker --kind claude -- claude --model claude-haiku-4-5 I121"}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I121": "routine", "I122": "primary"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I122", "", "workflow-subagent")
			path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl")
			raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Implement I122"}}`+"\n"+
				`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-fable-5","content":[%s]}}`+"\n", repo, repo, tc.block)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}

			var firstWarnings []string
			for attempt := 0; attempt < 3; attempt++ {
				rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
				if err != nil {
					t.Fatal(err)
				}
				rows := rowsByID(t, rep)
				for _, id := range []string{"I121", "I122"} {
					if row := rows[id]; row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
						t.Fatalf("attempt %d: %s = %+v, want no routing evidence from an ambiguous nested member", attempt, id, row)
					}
				}
				if !warningContains(rep.Warnings, "1 malformed line(s) skipped") {
					t.Fatalf("attempt %d: warnings = %q, want one stable malformed workflow warning", attempt, rep.Warnings)
				}
				if attempt == 0 {
					firstWarnings = rep.Warnings
				} else if !reflect.DeepEqual(rep.Warnings, firstWarnings) {
					t.Fatalf("attempt %d: warnings = %q, want stable %q", attempt, rep.Warnings, firstWarnings)
				}
			}
		})
	}
}

func TestWorkflowNestedSemanticKeyValidationPreservesUnrelatedToolFields(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I121": "routine", "I122": "primary"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I122", "", "workflow-subagent")
	path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl")
	raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Implement I122"}}`+"\n"+
		`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-fable-5","content":[{"type":"tool_use","id":"tool-1","name":"Agent","cache_control":{"type":"ephemeral"},"input":{"description":"I121 nested dispatch","prompt":"Implement I121","model":"claude-haiku-4-5","opaque_input":{"trace":"retain"}}}]}}`+"\n", repo, repo)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if row := rows["I121"]; row.Verdict != VerdictSilentDescent || !reflect.DeepEqual(row.Actuals, []string{"claude-haiku-4-5"}) {
		t.Fatalf("I121 = %+v, want the valid nested Agent dispatch despite unrelated fields", row)
	}
	if row := rows["I122"]; row.Verdict != VerdictMatch {
		t.Fatalf("I122 = %+v, want the transcript model from the valid assistant carrier", row)
	}
}

// A nested dispatch is its own carrier: a workflow worker whose opening
// message cannot be directly attributed must not erase the nested dispatch's
// explicit prompt, model, and tool-use identity.
func TestWorkflowNestedDispatchSurvivesUnattributableParent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opening string
	}{
		{name: "ticketless parent", opening: "Coordinate the work"},
		{name: "multi-ticket parent", opening: "Coordinate I122 and I123"},
		{name: "malformed parent", opening: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{
				"I121": "routine", "I122": "primary", "I123": "primary",
			})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "unused", "", "workflow-subagent")
			path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl")
			parent := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":%q}}`+"\n", repo, tc.opening)
			if tc.name == "malformed parent" {
				parent = fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":1}}`+"\n", repo)
			}
			raw := parent + fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-fable-5","content":[{"type":"tool_use","id":"tool-121","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"Implement I121 exactly","model":"claude-haiku-4-5"}}]}}`+"\n", repo)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			rows := rowsByID(t, rep)
			if row := rows["I121"]; row.Verdict != VerdictSilentDescent || !reflect.DeepEqual(row.Actuals, []string{"claude-haiku-4-5"}) {
				t.Fatalf("I121 = %+v, want its own nested dispatch evidence despite the parent opening", row)
			}
			for _, id := range []string{"I122", "I123"} {
				if row := rows[id]; row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
					t.Fatalf("%s = %+v, want no parent model evidence from an unattributable opening", id, row)
				}
			}
		})
	}
}

// The exact workflow-scoped identity still governs I078 DISCARDED matching
// when the parent worker is ticketless. A parent-level shortcut must not
// change the nested event's source/session/workflow/agent/tool-use identity.
func TestWorkflowTicketlessParentNestedDispatchKeepsDiscardedIdentity(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I078": "primary"})
	if err := os.MkdirAll(filepath.Join(repo, ".superpowers", "sdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".superpowers", "sdd", "progress.md"), []byte(
		"DISCARDED I078 source:claude session:session-1/wf_1/agent-worker dispatch:tool-078 tier:mechanical reason: nested prototype was discarded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Coordinate the work", "", "workflow-subagent")
	path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl")
	raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Coordinate the work"}}`+"\n"+
		`{"type":"assistant","cwd":%q,"message":{"role":"assistant","content":[{"type":"tool_use","id":"tool-078","name":"Agent","input":{"description":"I078 nested prototype","prompt":"Implement I078 exactly","model":"claude-haiku-4-5"}}]}}`+"\n", repo, repo)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I078"]
	if row.Verdict != VerdictDiscardedWithReason || rep.Blocking() {
		t.Fatalf("I078 = %+v, blocking=%v; want exact nested identity to retain its discarded record", row, rep.Blocking())
	}
	if !strings.Contains(row.Detail, "nested prototype was discarded") {
		t.Fatalf("I078 detail = %q, want the exact discarded-record reason", row.Detail)
	}
}

// Admission occurs before any workflow parsing, so a ticketless parent cannot
// turn an excluded workflow file into nested routing evidence.
func TestWorkflowTicketlessParentNestedDispatchStillRequiresAdmission(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I121": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Coordinate the work", "", "code-reviewer")
	path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.jsonl")
	raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Coordinate the work"}}`+"\n"+
		`{"type":"assistant","cwd":%q,"message":{"role":"assistant","content":[{"type":"tool_use","id":"tool-121","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"Implement I121 exactly","model":"claude-haiku-4-5"}}]}}`+"\n", repo, repo)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	if row := rowsByID(t, rep)["I121"]; row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
		t.Fatalf("I121 = %+v, want excluded workflow metadata to suppress nested evidence", row)
	}
}

func TestWorkflowFirstValidUserEventLatchesAfterMalformedCarrier(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "missing content", raw: `{"type":"user","message":{"role":"user"}}`},
		{name: "null content", raw: `{"type":"user","message":{"role":"user","content":null}}`},
		{name: "non text content", raw: `{"type":"user","message":{"role":"user","content":1}}`},
		{name: "malformed text block", raw: `{"type":"user","message":{"role":"user","content":[{"type":"text","text":1}]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I061": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "unused", "", "workflow-subagent")
			base := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker")
			raw := tc.raw + "\n" +
				fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Implement I061"}}`+"\n", repo) +
				fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"done"}]}}`+"\n", repo)
			if err := os.WriteFile(base+".jsonl", []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I061"]; row.Verdict != VerdictMatch {
				t.Fatalf("I061 = %+v, want the first later structurally valid user event to supply evidence", row)
			}
			if !warningContains(rep.Warnings, "1 malformed line(s) skipped") {
				t.Fatalf("warnings = %q, want malformed opening warning", rep.Warnings)
			}
		})
	}
}

// A coherent assistant event before any coherent user cannot belong to a
// parent brief that appears later. Its nested dispatch remains independent
// evidence, but its model must never be retroactively attached to I913.
func TestWorkflowAssistantBeforeOpeningMakesParentUnavailable(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{
		"I121": "routine",
		"I913": "primary",
	})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_latch", "worker", repo, "unused", "", "workflow-subagent")
	path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_latch", "agent-worker.jsonl")
	raw := `{"type":"user","message":{"role":"user","content":1}}` + "\n" +
		fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-fable-5","content":[{"type":"tool_use","id":"tool-121","name":"Agent","input":{"description":"I121 nested dispatch","prompt":"Implement I121 exactly","model":"claude-haiku-4-5"}}]}}`+"\n", repo) +
		fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Implement I913"}}`+"\n", repo) +
		fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","content":[{"type":"text","text":"done"}]}}`+"\n", repo)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	if row := rows["I913"]; row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
		t.Fatalf("I913 = %+v, want no parent evidence after a coherent pre-opening assistant", row)
	}
	if row := rows["I121"]; row.Verdict != VerdictSilentDescent || !reflect.DeepEqual(row.Actuals, []string{"claude-haiku-4-5"}) {
		t.Fatalf("I121 = %+v, want the pre-opening nested dispatch to retain its own evidence", row)
	}
}

// A pre-opening assistant without a transcript model also closes parent
// attribution. Later workflow-run metadata cannot turn a later user message
// into parent evidence.
func TestWorkflowAssistantBeforeOpeningBlocksParentModelFallback(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I914": "primary"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_latch", "worker", repo, "unused", "", "workflow-subagent")
	path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_latch", "agent-worker.jsonl")
	raw := fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","content":[{"type":"text","text":"before opening"}]}}`+"\n", repo) +
		fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Implement I914"}}`+"\n", repo)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	writeWorkflowRun(t, transcripts, "session-1", "wf_latch", "worker", "claude-fable-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	if row := rowsByID(t, rep)["I914"]; row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
		t.Fatalf("I914 = %+v, want no parent fallback after a coherent pre-opening assistant", row)
	}
}

// The first coherent user opens the parent latch even when it names no single
// audited ticket. Later users cannot replace that opening.
func TestWorkflowFirstCoherentUserCannotBeReplaced(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opening string
	}{
		{name: "empty", opening: ""},
		{name: "ticketless", opening: "Coordinate the work"},
		{name: "multi-ticket", opening: "Coordinate I912 and I913"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{
				"I912": "primary", "I913": "primary", "I914": "primary",
			})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_latch", "worker", repo, "unused", "", "workflow-subagent")
			path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_latch", "agent-worker.jsonl")
			raw := fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":%q}}`+"\n", repo, tc.opening) +
				fmt.Sprintf(`{"type":"user","cwd":%q,"message":{"role":"user","content":"Implement I914"}}`+"\n", repo) +
				fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","model":"claude-fable-5","content":[{"type":"text","text":"done"}]}}`+"\n", repo)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if row := rowsByID(t, rep)["I914"]; row.Verdict != VerdictNoTranscript || len(row.Actuals) != 0 {
				t.Fatalf("I914 = %+v, want no parent evidence from a later user", row)
			}
		})
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

func TestWorkflowDuplicateMetadataMembersFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name            string
		sidecar         string
		run             string
		transcriptModel string
		wantWarning     string
	}{
		{
			name:        "sidecar admission fields",
			sidecar:     `{"agentType":"code-reviewer","agentType":"workflow-subagent","spawnDepth":2,"spawnDepth":1}`,
			wantWarning: "ambiguous workflow metadata — transcript skipped",
		},
		{
			name:        "run workflow progress",
			run:         `{"workflowProgress":[],"workflowProgress":[{"agentId":"worker","model":"claude-sonnet-5"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
		{
			name:        "run agent entry model",
			run:         `{"workflowProgress":[{"agentId":"worker","model":"claude-haiku-4-5","model":"claude-sonnet-5"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
		{
			name:        "run agent entry identity",
			run:         `{"workflowProgress":[{"agentId":"other-worker","agentId":"worker","model":"claude-sonnet-5"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
		{
			name:            "sidecar case-equivalent admission fields",
			sidecar:         `{"agentType":"code-reviewer","AGENTTYPE":"workflow-subagent","spawnDepth":2,"SPAWNDEPTH":1}`,
			transcriptModel: "claude-sonnet-5",
			wantWarning:     "ambiguous workflow metadata — transcript skipped",
		},
		{
			name:            "sidecar case-variant agent type",
			sidecar:         `{"AGENTTYPE":"workflow-subagent","spawnDepth":1}`,
			transcriptModel: "claude-sonnet-5",
			wantWarning:     "ambiguous workflow metadata — transcript skipped",
		},
		{
			name:            "sidecar case-variant spawn depth",
			sidecar:         `{"agentType":"workflow-subagent","SPAWNDEPTH":1}`,
			transcriptModel: "claude-sonnet-5",
			wantWarning:     "ambiguous workflow metadata — transcript skipped",
		},
		{
			name:        "run case-equivalent workflow progress",
			run:         `{"workflowProgress":[],"WORKFLOWPROGRESS":[{"agentId":"worker","model":"claude-sonnet-5"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
		{
			name:        "run case-equivalent agent identity",
			run:         `{"workflowProgress":[{"agentId":"other-worker","AGENTID":"worker","model":"claude-sonnet-5"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
		{
			name:        "run case-equivalent agent model",
			run:         `{"workflowProgress":[{"agentId":"worker","model":"claude-haiku-4-5","MODEL":"claude-sonnet-5"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
		{
			name:        "run case-equivalent agent type",
			run:         `{"workflowProgress":[{"agentId":"worker","model":"claude-sonnet-5","type":"workflow_agent","TYPE":"workflow_agent"}]}`,
			wantWarning: "ambiguous workflow run metadata — model fallback skipped",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I056": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I056", tc.transcriptModel, "workflow-subagent")
			base := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker")
			if tc.sidecar != "" {
				if err := os.WriteFile(base+".meta.json", []byte(tc.sidecar), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.run != "" {
				writeWorkflowRunRaw(t, transcripts, "session-1", "wf_1", tc.run)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			if got := rowsByID(t, rep)["I056"].Verdict; got != VerdictNoTranscript {
				t.Fatalf("I056 verdict = %s, want no-transcript for ambiguous metadata", got)
			}
			if !warningContains(rep.Warnings, tc.wantWarning) {
				t.Fatalf("warnings = %q, want %q", rep.Warnings, tc.wantWarning)
			}
		})
	}
}

func TestWorkflowMetadataRejectsSymlinkScopeEscape(t *testing.T) {
	for _, tc := range []struct {
		name        string
		linkPath    func(string) string
		targetPath  func(string) string
		wantWarning string
	}{
		{
			name: "sidecar",
			linkPath: func(transcripts string) string {
				return filepath.Join(transcripts, "session-a", "subagents", "workflows", "wf_link", "agent-worker.meta.json")
			},
			targetPath: func(transcripts string) string {
				return filepath.Join(transcripts, "session-b", "subagents", "workflows", "wf_link", "agent-worker.meta.json")
			},
			wantWarning: "workflow metadata unsafe — transcript skipped",
		},
		{
			name: "workflow run",
			linkPath: func(transcripts string) string {
				return filepath.Join(transcripts, "session-a", "workflows", "wf_link.json")
			},
			targetPath: func(transcripts string) string {
				return filepath.Join(transcripts, "session-b", "workflows", "wf_link.json")
			},
			wantWarning: "workflow run metadata unsafe — model fallback skipped",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I057": "routine"})
			transcripts := t.TempDir()
			writeWorkflowAgent(t, transcripts, "session-a", "wf_link", "worker", repo, "Implement I057", "", "workflow-subagent")
			writeWorkflowAgent(t, transcripts, "session-b", "wf_link", "worker", repo, "Ignore I057", "", "workflow-subagent")
			writeWorkflowRun(t, transcripts, "session-a", "wf_link", "worker", "claude-haiku-4-5")
			writeWorkflowRun(t, transcripts, "session-b", "wf_link", "worker", "claude-sonnet-5")

			link, target := tc.linkPath(transcripts), tc.targetPath(transcripts)
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts, Session: "session-a"})
			if err != nil {
				t.Fatal(err)
			}
			if got := rowsByID(t, rep)["I057"].Verdict; got != VerdictNoTranscript {
				t.Fatalf("I057 verdict = %s, want no-transcript after a symlink scope escape", got)
			}
			if !warningContains(rep.Warnings, tc.wantWarning) {
				t.Fatalf("warnings = %q, want %q", rep.Warnings, tc.wantWarning)
			}
		})
	}
}

func TestWorkflowSidecarOverOneMiBFailsClosed(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I058": "routine"})
	transcripts := t.TempDir()
	writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I058", "claude-sonnet-5", "workflow-subagent")
	path := filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.meta.json")
	tooLarge := append([]byte(`{"agentType":"workflow-subagent","spawnDepth":1,"padding":"`), make([]byte, (1<<20)+1)...)
	tooLarge = append(tooLarge, []byte(`"}`)...)
	if err := os.WriteFile(path, tooLarge, 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	if got := rowsByID(t, rep)["I058"].Verdict; got != VerdictNoTranscript {
		t.Fatalf("I058 verdict = %s, want no-transcript for oversized sidecar", got)
	}
	if !warningContains(rep.Warnings, "workflow metadata exceeds 1048576 bytes — transcript skipped") {
		t.Fatalf("warnings = %q, want oversized sidecar warning", rep.Warnings)
	}
}

func TestWorkflowMetadataRejectsAtomicReplacement(t *testing.T) {
	for _, tc := range []struct {
		name        string
		targetPath  func(string) string
		replacement []byte
		wantWarning string
	}{
		{
			name: "sidecar",
			targetPath: func(transcripts string) string {
				return filepath.Join(transcripts, "session-1", "subagents", "workflows", "wf_1", "agent-worker.meta.json")
			},
			replacement: []byte(`{"agentType":"code-reviewer","spawnDepth":1}`),
			wantWarning: "workflow metadata unsafe — transcript skipped",
		},
		{
			name: "workflow run",
			targetPath: func(transcripts string) string {
				return filepath.Join(transcripts, "session-1", "workflows", "wf_1.json")
			},
			replacement: []byte(`{"workflowProgress":[{"agentId":"worker","model":"claude-haiku-4-5"}]}`),
			wantWarning: "workflow run metadata unsafe — model fallback skipped",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for attempt := 0; attempt < 16; attempt++ {
				repo := t.TempDir()
				writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I059": "routine"})
				transcripts := t.TempDir()
				writeWorkflowAgent(t, transcripts, "session-1", "wf_1", "worker", repo, "Implement I059", "", "workflow-subagent")
				writeWorkflowRun(t, transcripts, "session-1", "wf_1", "worker", "claude-sonnet-5")
				target := tc.targetPath(transcripts)
				replacement := target + ".replacement"
				if err := os.WriteFile(replacement, tc.replacement, 0o644); err != nil {
					t.Fatal(err)
				}
				setWorkflowMetadataBeforeOpen(t, func(path string) {
					if path == target {
						if err := os.Rename(replacement, target); err != nil {
							t.Fatal(err)
						}
					}
				})

				rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
				if err != nil {
					t.Fatal(err)
				}
				if got := rowsByID(t, rep)["I059"].Verdict; got != VerdictNoTranscript {
					t.Fatalf("attempt %d: I059 verdict = %s, want no-transcript after atomic replacement", attempt, got)
				}
				if !warningContains(rep.Warnings, tc.wantWarning) {
					t.Fatalf("attempt %d: warnings = %q, want %q", attempt, rep.Warnings, tc.wantWarning)
				}
			}
		})
	}
}

func setWorkflowMetadataBeforeOpen(t *testing.T, hook func(string)) {
	t.Helper()
	previous := workflowMetadataBeforeOpen
	workflowMetadataBeforeOpen = hook
	t.Cleanup(func() { workflowMetadataBeforeOpen = previous })
}

func setWorkflowTranscriptBeforeOpen(t *testing.T, hook func(string)) {
	t.Helper()
	previous := workflowTranscriptBeforeOpen
	workflowTranscriptBeforeOpen = hook
	t.Cleanup(func() { workflowTranscriptBeforeOpen = previous })
}
