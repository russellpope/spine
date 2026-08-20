package scaffold

import "testing"

// TestSlugFromRemoteForms pins remote parsing: the remote names both halves,
// `.git` is stripped, and an ssh URL carrying a port does not hand the port to
// the owner slot.
func TestSlugFromRemoteForms(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"git@github.com:acme/x.git", "acme/x"},
		{"https://github.com/acme/x.git", "acme/x"},
		{"https://github.com/acme/x", "acme/x"},
		{"https://github.com/acme/x/", "acme/x"},
		{"ssh://git@github.com:22/acme/x.git", "acme/x"},
		{"ssh://git@github.com/acme/x.git", "acme/x"},
		{"git@gitlab.com:acme/x.git", ""},
		{"https://github.com/acme", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := slugFromRemote(c.url); got != c.want {
			t.Errorf("slug from %q = %q, want %q", c.url, got, c.want)
		}
	}
}
