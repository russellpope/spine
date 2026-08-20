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

// The misattribution guards (ticket I090), end to end. A claude-team lead
// runs several workers at once, so a spawn borrowing the WRONG worker's
// prompt is this parser's one dangerous failure mode. The teampair fixture
// starts two workers back-to-back and briefs them in the same order, which
// is the arrangement that discriminates: pairing by worker handle gives each
// ticket its own spawn's model, while ignoring the handle (or attributing to
// the most recent spawn regardless) hands I601 the second worker's model.
func TestTeamSpawnPairsByWorkerHandle(t *testing.T) {
	rows := rowsByID(t, runFixture(t, "teampair"))
	for _, tc := range []struct {
		id      string
		actual  string
		verdict Verdict
		why     string
	}{
		{"I601", "claude-sonnet-5", VerdictMatch, "impl-a's own spawn, not impl-b's"},
		{"I602", "claude-fable-5", VerdictMatch, "impl-b's own spawn"},
		// A worker restarted before it was ever briefed: the live process is
		// the second spawn, so the most recent unattributed spawn for that
		// handle wins — scanning forward would credit the dead one.
		{"I604", "claude-fable-5", VerdictMatch, "impl-c's restart, not its first start"},
	} {
		r := rows[tc.id]
		if r.Verdict != tc.verdict {
			t.Errorf("%s verdict = %s (%s), want %s — %s", tc.id, r.Verdict, r.Detail, tc.verdict, tc.why)
		}
		if got := strings.Join(r.Actuals, ","); got != tc.actual {
			t.Errorf("%s actuals = %q, want %q — %s", tc.id, got, tc.actual, tc.why)
		}
	}
	// A second prompt to an already-briefed worker is follow-up
	// conversation, not a new assignment: the ticket it names stays
	// visibly unjudged rather than inheriting a spawn it never had.
	if r := rows["I603"]; r.Verdict != VerdictNoTranscript {
		t.Errorf("I603 verdict = %s (%s), want no-transcript", r.Verdict, r.Detail)
	}
}

// Unit coverage for the same guards, plus the spawn-token-wins rule: when a
// spawn command names its own ticket, a following prompt naming a different
// one must not overwrite it — otherwise a worker's second assignment would
// drag its first ticket's evidence along.
func TestAttributeTeamPrompt(t *testing.T) {
	spawn := func(target, desc string) dispatch {
		return dispatch{description: desc, teamTarget: target}
	}
	for _, tc := range []struct {
		name    string
		spawns  []dispatch
		prompt  teamPrompt
		wantIdx int // index of the spawn expected to take the prompt; -1 for none
	}{
		{
			name:    "pairs by worker handle, not recency",
			spawns:  []dispatch{spawn("impl-a", "start a"), spawn("impl-b", "start b")},
			prompt:  teamPrompt{target: "impl-a", text: "work I601"},
			wantIdx: 0,
		},
		{
			name:    "most recent spawn for the handle wins",
			spawns:  []dispatch{spawn("impl-c", "start c"), spawn("impl-c", "restart c")},
			prompt:  teamPrompt{target: "impl-c", text: "work I604"},
			wantIdx: 1,
		},
		{
			name:    "no spawn for the handle: nothing borrows",
			spawns:  []dispatch{spawn("impl-a", "start a")},
			prompt:  teamPrompt{target: "impl-z", text: "work I605"},
			wantIdx: -1,
		},
		{
			name:    "prompt with no handle attributes nothing",
			spawns:  []dispatch{spawn("impl-a", "start a")},
			prompt:  teamPrompt{text: "work I605"},
			wantIdx: -1,
		},
		{
			name:    "spawn naming its own ticket keeps it",
			spawns:  []dispatch{spawn("impl-b", "start b # I402")},
			prompt:  teamPrompt{target: "impl-b", text: "now do I407"},
			wantIdx: -1,
		},
		{
			name: "already-attributed spawn ignores a second prompt",
			spawns: []dispatch{
				{description: "start a", teamTarget: "impl-a", prompt: "work I601"},
			},
			prompt:  teamPrompt{target: "impl-a", text: "also I603"},
			wantIdx: -1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := append([]dispatch(nil), tc.spawns...)
			before := make([]string, len(ds))
			for i, d := range ds {
				before[i] = d.prompt
			}
			attributeTeamPrompt(ds, tc.prompt)
			for i, d := range ds {
				want := before[i]
				if i == tc.wantIdx {
					want = tc.prompt.text
				}
				if d.prompt != want {
					t.Errorf("spawn %d (%s) prompt = %q, want %q", i, d.teamTarget, d.prompt, want)
				}
			}
		})
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
			name:    "spawn parenthesized inside a trailing comment",
			command: "cat notes # (herdr agent start x --kind claude -- claude --model claude-opus-5)",
		},
		{
			name:    "trailing comment still carries the ticket for a real spawn",
			command: "herdr agent start impl-2 --kind claude -- claude --model claude-opus-5 # I402",
			want:    teamSpawn{model: "claude-opus-5", target: "impl-2"},
		},
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
