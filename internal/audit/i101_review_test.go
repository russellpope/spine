package audit

import (
	"strings"
	"testing"
)

// These are the public Run-level regressions from I101's final review. They
// catch a post-spawn rewrite borrowing a later brief and a later shell segment
// supplying evidence to an otherwise raw prompt. The same real fixture covers
// the documented fail-closed degradation, isolation, and precedence cases.
func TestReviewBriefAttributionGuards(t *testing.T) {
	rep := runFixture(t, "briefreview")
	rows := rowsByID(t, rep)
	for _, tc := range []struct {
		id   string
		want Verdict
	}{
		{"I701", VerdictMatch},        // rewrite after spawn must not replace it with I706
		{"I706", VerdictNoTranscript}, // unrelated later brief must not replace raw I711
		{"I711", VerdictMatch},        // raw prompt fallback survives a later echo $(cat ...)
		{"I712", VerdictNoTranscript}, // brief first line has no ticket
		{"I713", VerdictNoTranscript}, // --brief has no recorded write
		{"I714", VerdictNoTranscript}, // another session's write cannot be borrowed
		{"I715", VerdictMatch},        // resolved spawn brief wins over prompt brief
		{"I716", VerdictNoTranscript},
		{"I717", VerdictMatch}, // same Bash block: later rewrite cannot replace old brief
		{"I718", VerdictNoTranscript},
		{"I719", VerdictMatch},        // pre-spawn /exit must not consume the assignment prompt
		{"I720", VerdictMatch},        // nested executed start pairs a later prompt
		{"I721", VerdictNoTranscript}, // quoted nested prose is not a dispatch
		{"I722", VerdictMatch},        // double-quoted nested command still executes
		{"I723", VerdictMatch},        // nested body start and prompt pair by absolute offset
	} {
		if got := rows[tc.id].Verdict; got != tc.want {
			t.Errorf("%s verdict = %s (%s), want %s", tc.id, got, rows[tc.id].Detail, tc.want)
		}
	}
	for _, d := range rep.Unmatched {
		if strings.Contains(d.Description, "impl-prose") {
			t.Errorf("single-quoted nested prose created a dispatch: %+v", d)
		}
	}
}
