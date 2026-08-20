package scaffold_test

import (
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// TestValidSlugGrammar pins maikanban ADR 0007: two components, each 1–100
// ASCII bytes, alphanumeric at both ends, `._-` permitted inside.
func TestValidSlugGrammar(t *testing.T) {
	long := strings.Repeat("a", 100)
	cases := []struct {
		slug string
		want bool
	}{
		{"acme/x", true},
		{"a/b", true}, // single-character components
		{"a-c.m_e/some.repo-1", true},
		{long + "/" + long, true},
		{long + "a/x", false},      // 101-byte owner
		{"x/" + long + "a", false}, // 101-byte repo
		{"acme", false},            // no repo component
		{"acme/", false},
		{"/x", false},
		{"-acme/x", false},  // leading punctuation
		{"acme/x-", false},  // trailing punctuation
		{"acme/x/y", false}, // three components
		{"-/x", false},      // lone punctuation component
		{"ac me/x", false},  // space
		{"acmé/x", false},   // non-ASCII
	}
	for _, c := range cases {
		if got := scaffold.ValidSlug(c.slug); got != c.want {
			t.Errorf("ValidSlug(%q) = %v, want %v", c.slug, got, c.want)
		}
	}
}
