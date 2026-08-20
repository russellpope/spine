package audit

import (
	"strings"
	"testing"
)

// Positive control (ticket I090): claude-team worker spawns are Bash
// commands, not Task/Agent tool calls. Every ticket in the team fixture was
// built that way and every one must be JUDGED — the pre-I090 audit reported
// all three as no-transcript.
func TestTeamSpawnsAreJudged(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "team"))
	for _, tc := range []struct {
		id      string
		actual  string
		verdict Verdict
	}{
		// Start command named no ticket; the following `herdr agent prompt`
		// did, and the routine model matches the routine annotation.
		{"I401", "claude-sonnet-5", VerdictMatch},
		// Start command named the ticket itself; routine model on a primary
		// ticket with no recorded reason.
		{"I402", "claude-sonnet-5", VerdictSilentDescent},
		// The cmux shape: a send whose payload invokes claude.
		{"I403", "claude-fable-5", VerdictMatch},
	} {
		r := rows[tc.id]
		if r.Verdict != tc.verdict {
			t.Errorf("%s verdict = %s (%s), want %s", tc.id, r.Verdict, r.Detail, tc.verdict)
		}
		if got := strings.Join(r.Actuals, ","); got != tc.actual {
			t.Errorf("%s actuals = %q, want %q", tc.id, got, tc.actual)
		}
	}
}

// A spawn matching no ticket is listed informationally like any other
// unmatched dispatch, carrying the effort the command declared — reported,
// never judged.
func TestUnmatchedTeamSpawnCarriesEffort(t *testing.T) {
	rep := runFixture(t, "team")
	got := map[string]string{}
	for _, d := range rep.Unmatched {
		got[d.Model] = d.Effort
	}
	if len(got) != 2 {
		t.Fatalf("want the 2 scratch spawns unmatched, got %+v", rep.Unmatched)
	}
	if got["claude-fable-5"] != "high" {
		t.Errorf("scratch spawn effort = %q, want high", got["claude-fable-5"])
	}
	if got["claude-opus-4-8"] != "" {
		t.Errorf("spawn declaring no effort must report none, got %q", got["claude-opus-4-8"])
	}
}

// The load-bearing check for the positive control above: with I090's
// recognition switched off, every team-built ticket falls back to the
// no-transcript verdict the ticket was filed about. If this fails, the
// fixture is being judged for some other reason and proves nothing.
func TestTeamSpawnRecognitionIsLoadBearing(t *testing.T) {
	recognizeTeamSpawns = false
	t.Cleanup(func() { recognizeTeamSpawns = true })
	for id, r := range rowsByID(t, runFixture(t, "team")) {
		if r.Verdict != VerdictNoTranscript {
			t.Errorf("%s verdict = %s (%s), want no-transcript without the recognition", id, r.Verdict, r.Detail)
		}
	}
}

// Negative controls (ticket I090): each ticket in the teamnoise fixture is
// named by exactly one Bash command that looks spawn-ish but is not a
// dispatch. Recognizing any of them would judge a ticket nobody dispatched.
func TestTeamSpawnLookalikesAreNotDispatches(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "teamnoise"))
	for _, tc := range []struct{ id, why string }{
		{"I501", "a herdr command that is not `agent start`"},
		{"I502", "a cmux send whose payload is not a claude invocation"},
		{"I503", "a spawn command quoted inside a heredoc, not run"},
	} {
		if r := rows[tc.id]; r.Verdict != VerdictNoTranscript {
			t.Errorf("%s verdict = %s (%s), want no-transcript: %s", tc.id, r.Verdict, r.Detail, tc.why)
		}
	}
}

// Command-shape unit coverage for the recognizer itself, including the
// effort extraction the report surfaces on unmatched dispatches.
func TestParseTeamSpawn(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command string
		want    teamSpawn
	}{
		{
			name:    "herdr agent start",
			command: "herdr agent start impl-1 --kind claude --pane %3 -- claude --model claude-opus-5 --effort low",
			want:    teamSpawn{model: "claude-opus-5", effort: "low", target: "impl-1"},
		},
		{
			name:    "herdr agent start without effort",
			command: "herdr agent start rev-1 --kind claude --pane %5 -- claude --model claude-fable-5",
			want:    teamSpawn{model: "claude-fable-5", target: "rev-1"},
		},
		{
			name:    "cmux send quoted claude payload",
			command: "cmux send --pane %7 'claude --model claude-opus-5 --effort high'",
			want:    teamSpawn{model: "claude-opus-5", effort: "high", target: "%7"},
		},
		{
			name:    "cmux send after a payload separator",
			command: "cmux send --pane %7 -- claude --model=claude-opus-5 --effort=high",
			want:    teamSpawn{model: "claude-opus-5", effort: "high", target: "%7"},
		},
		{
			name:    "spawn after a shell separator",
			command: "cd /tmp && herdr agent start impl-2 --kind claude -- claude --model claude-opus-5",
			want:    teamSpawn{model: "claude-opus-5", target: "impl-2"},
		},
		{name: "herdr query", command: "herdr agent get impl-1 --json"},
		{name: "herdr pane list", command: "herdr pane list"},
		{name: "herdr workspace list", command: "herdr workspace list"},
		{name: "start without a model", command: "herdr agent start impl-1 --kind claude --pane %3"},
		{name: "cmux send running something else", command: "cmux send --pane %2 'make test'"},
		{name: "spawn named mid-sentence", command: "echo 'we use herdr agent start x -- claude --model claude-opus-5'"},
		{
			name:    "spawn quoted in a heredoc body",
			command: "cat > doc.md <<'EOF'\nherdr agent start x --kind claude -- claude --model claude-opus-5\nEOF",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseTeamSpawn(tc.command)
			if want := tc.want != (teamSpawn{}); ok != want {
				t.Fatalf("recognized = %v, want %v", ok, want)
			}
			if ok && got != tc.want {
				t.Errorf("spawn = %+v, want %+v", got, tc.want)
			}
		})
	}
}
