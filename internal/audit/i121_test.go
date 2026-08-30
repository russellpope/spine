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

func TestHyphenatedNonRangeDoesNotAttributeInteriorIDs(t *testing.T) {
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
	if got := rows["I051"].Verdict; got != VerdictMatch {
		t.Fatalf("literal endpoint I051 verdict = %s (%s), want match", got, rows["I051"].Detail)
	}
	for _, id := range []string{"I052", "I053"} {
		if got := rows[id].Verdict; got != VerdictNoTranscript {
			t.Errorf("%s verdict = %s (%s), malformed range must not expand", id, got, rows[id].Detail)
		}
	}
}
