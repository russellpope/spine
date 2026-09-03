package model

import "testing"

// I128: a historical id's successor is the current id of the row whose
// history lists it, preferring the requested tier. Current ids and unknown
// ids have no successor.
func TestSuccessorID(t *testing.T) {
	for _, tc := range []struct {
		name          string
		harness, tier string
		id            string
		want          string
		ok            bool
	}{
		{"same tier", "claude", "primary", "claude-fable-5", "claude-fable-5-1", true},
		{"cross tier falls back to the row that lists it", "claude", "routine", "claude-fable-5", "claude-fable-5-1", true},
		{"routine history", "claude", "routine", "claude-sonnet-5", "claude-opus-5", true},
		{"current id is not historical", "claude", "primary", "claude-fable-5-1", "", false},
		{"unknown id", "claude", "primary", "local-llama-70b", "", false},
		{"unknown harness", "nope", "primary", "claude-fable-5", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SuccessorID(tc.harness, tc.tier, tc.id)
			if got != tc.want || ok != tc.ok {
				t.Errorf("SuccessorID(%q, %q, %q) = %q, %v; want %q, %v", tc.harness, tc.tier, tc.id, got, ok, tc.want, tc.ok)
			}
		})
	}
}
