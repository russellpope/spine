package audit

import (
	"path/filepath"
	"testing"
)

// I121: a dispatch range is one attribution token. Dropping range expansion
// leaves the interior tickets with no evidence even though the dispatch names
// the complete inclusive set.
func TestTicketRangeDispatchAttributesEveryInteriorID(t *testing.T) {
	repo := t.TempDir()
	tickets := map[string]string{}
	for _, id := range []string{"I051", "I052", "I053", "I054", "I055", "I056"} {
		tickets[id] = "routine"
	}
	writeAuditRepo(t, repo, gen9DefaultWorkflow, tickets)

	transcripts := t.TempDir()
	writeSingleDispatch(t, filepath.Join(transcripts, "range.jsonl"), repo,
		"I051-I056", "tickets I051-I056", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I051", "I052", "I053", "I054", "I055", "I056"} {
		if got := rows[id].Verdict; got != VerdictMatch {
			t.Errorf("%s verdict = %s (%s), want match from inclusive range", id, got, rows[id].Detail)
		}
	}
}

func TestMalformedPartialRangeDoesNotAttributeAnyIDs(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{
		"I051": "routine",
		"I052": "routine",
		"I053": "routine",
	})

	transcripts := t.TempDir()
	writeSingleDispatch(t, filepath.Join(transcripts, "not-range.jsonl"), repo,
		"I051-I05X", "work I051-I05X", "claude-sonnet-5")

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I051", "I052", "I053"} {
		if got := rows[id].Verdict; got != VerdictNoTranscript {
			t.Errorf("%s verdict = %s (%s), malformed partial range must not attribute", id, got, rows[id].Detail)
		}
	}
}

// I121: direct Claude dispatch attribution must use the same strict boundary
// grammar as opening-line attribution, so hyphen-embedded endpoint IDs never
// supply routing evidence.
func TestClaudeDispatchRejectsEveryEndpointOfNonStandaloneTicketForms(t *testing.T) {
	for _, tc := range []struct {
		name, description string
		ids               []string
	}{
		{name: "surrounding hyphens", description: "slug-I051-I056-tail", ids: []string{"I051", "I056"}},
		{name: "chained range", description: "I051-I056-I060", ids: []string{"I051", "I056", "I060"}},
		{name: "leading hyphen", description: "-I051", ids: []string{"I051"}},
		{name: "trailing hyphen", description: "I051-", ids: []string{"I051"}},
		{name: "malformed partial range", description: "I051-I05X", ids: []string{"I051"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			tickets := make(map[string]string, len(tc.ids))
			for _, id := range tc.ids {
				tickets[id] = "routine"
			}
			writeAuditRepo(t, repo, gen9DefaultWorkflow, tickets)

			transcripts := t.TempDir()
			writeSingleDispatch(t, filepath.Join(transcripts, "malformed.jsonl"), repo,
				"I000", tc.description, "claude-sonnet-5")

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: transcripts})
			if err != nil {
				t.Fatal(err)
			}
			rows := rowsByID(t, rep)
			for _, id := range tc.ids {
				if got := rows[id].Verdict; got != VerdictNoTranscript {
					t.Errorf("%s verdict = %s (%s), want no-transcript from non-standalone dispatch form", id, got, rows[id].Detail)
				}
			}
		})
	}
}

func TestCodexDispatchTaskReferenceRequiresAFullPathComponent(t *testing.T) {
	for _, tc := range []struct {
		text string
		want bool
	}{
		{text: "/tmp/dispatch-task-I051.md", want: true},
		{text: "slug-dispatch-task-I051.md", want: false},
		{text: "/tmp/dispatch-task-I051.md-tail", want: false},
	} {
		if got := containsCodexDispatchTaskReference(tc.text, "I051"); got != tc.want {
			t.Errorf("containsCodexDispatchTaskReference(%q, I051) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestCodexWorkerRangeIsOneOpeningLineReference(t *testing.T) {
	repo := t.TempDir()
	tickets := map[string]string{}
	for _, id := range []string{"I051", "I052", "I053", "I054", "I055", "I056"} {
		tickets[id] = "routine"
	}
	writeAuditRepo(t, repo, gen9DefaultWorkflow, tickets)
	claudeDir := t.TempDir()
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker", "worker", "", repo, "user", topLevelSource),
		codexUserMessageLine("tickets I051-I056"),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: claudeDir, CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	rows := rowsByID(t, rep)
	for _, id := range []string{"I051", "I052", "I053", "I054", "I055", "I056"} {
		if got := rows[id].Verdict; got != VerdictMatch {
			t.Errorf("%s verdict = %s (%s), want range-attributed match", id, got, rows[id].Detail)
		}
	}
}

func TestCodexWorkerRangeWithInteriorOnlyAuditedTicketContributes(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I500": "routine"})
	claudeDir := t.TempDir()
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker", "worker", "", repo, "user", topLevelSource),
		codexUserMessageLine("Implement tickets I001-I999"),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: claudeDir, CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I500"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I500 verdict = %s (%s), want match from interior-only range", row.Verdict, row.Detail)
	}
}

func TestCodexWorkerHugeRangeWithInteriorOnlyAuditedTicketStaysBounded(t *testing.T) {
	repo := t.TempDir()
	writeAuditRepo(t, repo, gen9DefaultWorkflow, map[string]string{"I500000000": "routine"})
	codexDir := t.TempDir()
	writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
		codexSessionMetaLine("worker", "worker", "", repo, "user", topLevelSource),
		codexUserMessageLine("Implement tickets I000000000-I999999999"),
		codexTurnContextLine("gpt-5.6-terra"),
	)

	rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
	if err != nil {
		t.Fatal(err)
	}
	row := rowsByID(t, rep)["I500000000"]
	if row.Verdict != VerdictMatch {
		t.Fatalf("I500000000 verdict = %s (%s), want match from huge interior range", row.Verdict, row.Detail)
	}
}
