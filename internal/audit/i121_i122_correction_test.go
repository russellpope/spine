package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexWorkerMultipleExplicitReferenceGroupsAreAmbiguous(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opening string
		tickets map[string]string
	}{
		{name: "comma adjacent", opening: "Implement I051,I052", tickets: map[string]string{"I051": "routine", "I052": "routine"}},
		{name: "space adjacent", opening: "Implement I051 I052", tickets: map[string]string{"I051": "routine", "I052": "routine"}},
		{name: "non audited explicit", opening: "Implement I052 and I999", tickets: map[string]string{"I052": "routine"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeAuditRepo(t, repo, gen9DefaultWorkflow, tc.tickets)
			codexDir := t.TempDir()
			writeCodexFile(t, filepath.Join(codexDir, "worker.jsonl"),
				codexSessionMetaLine("worker", "worker", "", repo, "user", topLevelSource),
				codexUserMessageLine(tc.opening),
				codexTurnContextLine("gpt-5.6-terra"),
			)

			rep, err := Run(Options{RepoDir: repo, ClaudeTranscriptsDir: t.TempDir(), CodexSessionsDir: codexDir})
			if err != nil {
				t.Fatal(err)
			}
			for id := range tc.tickets {
				row := rowsByID(t, rep)[id]
				if row.Verdict != VerdictUnattributedTranscript {
					t.Errorf("%s verdict = %s (%s), want ambiguous unattributed transcript", id, row.Verdict, row.Detail)
				}
				if !strings.Contains(row.Detail, "opening line names multiple tickets") {
					t.Errorf("%s detail = %q, want ambiguous opening detail", id, row.Detail)
				}
			}
		})
	}
}
