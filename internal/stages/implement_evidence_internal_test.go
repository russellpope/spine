package stages

import "testing"

// TestImplementEvidenceWordBoundary is a white-box (package stages) test for
// implementEvidence's "done|complete" ledger scan. It exists because a
// substring match manufactures evidence from negations: "abandoned"
// contains "done", and "incomplete"/"not complete" contain "complete". A
// pending implement stage ([ ]) with a ledger line like
// "I003: incomplete, blocked on review" must NOT be treated as evidence —
// doing so would fire VerdictPresentUnticked and false-block
// `spine audit stages`, violating "under-detection acceptable, false
// blocking never".
func TestImplementEvidenceWordBoundary(t *testing.T) {
	cases := []struct {
		name string
		line string
		id   string
		want bool
	}{
		{"incomplete is not complete", "I003: incomplete, blocked on review", "I003", false},
		{"abandoned is not done", "I004: abandoned — descoped", "I004", false},
		{"plain done matches", "I003: done", "I003", true},
		{"marked complete matches", "I003: marked complete", "I003", true},
		{"completed with trailing detail matches", "I003: completed (review clean)", "I003", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := implementEvidence(tc.line, []string{tc.id})
			if got[tc.id] != tc.want {
				t.Errorf("implementEvidence(%q, [%q]) = %v, want %v", tc.line, tc.id, got[tc.id], tc.want)
			}
		})
	}
}

// TestClosureRecord is the white-box table for the closure-record predicate
// (I125): a ticket file evidences implement only when its status is fixed
// AND its commits: list names at least one SHA-shaped token. Every other
// status, an empty or absent list, and a placeholder token are non-evidence
// so the new path is load-bearing rather than a blanket pass.
func TestClosureRecord(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		commits string
		want    bool
	}{
		{"fixed with one sha", "fixed", "[a9ddea5]", true},
		{"fixed with several shas", "fixed", "[61d4c40, 5372110, 9ca0dd8]", true},
		{"fixed with a full-length sha", "fixed", "[68aa28ffb51eac886863a0cfbc0d4c266e0426df]", true},
		{"fixed with unbracketed sha", "fixed", "a9ddea5", true},
		{"fixed with empty list", "fixed", "[]", false},
		{"fixed with absent commits", "fixed", "", false},
		{"fixed with placeholder token", "fixed", "[pending]", false},
		{"fixed with too-short token", "fixed", "[abc123]", false},
		{"fixed with non-hex token", "fixed", "[zzzzzzz]", false},
		{"open with sha", "open", "[a9ddea5]", false},
		{"in-progress with sha", "in-progress", "[a9ddea5]", false},
		{"wontfix with sha", "wontfix", "[a9ddea5]", false},
		{"superseded with sha", "superseded", "[a9ddea5]", false},
		{"status case is exact", "Fixed", "[a9ddea5]", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fm := map[string]string{"status": tc.status}
			if tc.commits != "" {
				fm["commits"] = tc.commits
			}
			if got := closureRecord(fm); got != tc.want {
				t.Errorf("closureRecord(status=%q commits=%q) = %v, want %v", tc.status, tc.commits, got, tc.want)
			}
		})
	}
}
