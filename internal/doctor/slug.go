package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/scaffold"
)

// slugCheck is D12 (I094): a repo that carries docs/issues/ is part of
// maikanban's fleet, and maikanban requires `maikanban.repositorySlug` on it
// (maikanban ADR 0007). Missing or malformed is a warn carrying the remedy —
// never an error: an unstamped repo is unreachable for maikanban but perfectly
// healthy for spine. A non-git directory has nowhere to hold the key and is
// silent, as is a well-formed slug.
func slugCheck(dir string) []Finding {
	if fi, err := os.Stat(filepath.Join(dir, "docs", "issues")); err != nil || !fi.IsDir() {
		return nil // D1 covers structural absence
	}
	if exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run() != nil {
		return nil
	}
	out, err := exec.Command("git", "-C", dir, "config", "--get", scaffold.SlugKey).Output()
	got := strings.TrimSpace(string(out))
	if err != nil || got == "" {
		return []Finding{{"D12", "warn", "docs/issues", scaffold.SlugKey +
			" is unset (maikanban fleet identity) — run: " + scaffold.SlugRemedy}}
	}
	if !scaffold.ValidSlug(got) {
		return []Finding{{"D12", "warn", "docs/issues", fmt.Sprintf("%s is malformed (%q) — run: %s",
			scaffold.SlugKey, got, scaffold.SlugRemedy)}}
	}
	return nil
}
