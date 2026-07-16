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
