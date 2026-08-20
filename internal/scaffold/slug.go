package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SlugNote is what init prints when no owner can be determined (I094). Init
// never fails over the slug: maikanban fleet identity is the owner's to set
// when spine cannot infer it.
const SlugNote = "note: set git config maikanban.repositorySlug owner/repo (maikanban fleet identity)"

// SlugRemedy is the command doctor names for a missing or malformed slug.
const SlugRemedy = "git config maikanban.repositorySlug owner/repo"

// SlugKey is the git config key maikanban reads for fleet identity.
const SlugKey = "maikanban.repositorySlug"

// ValidSlug reports whether s is a well-formed `owner/repo` slug under
// maikanban ADR 0007: two components, each 1–100 ASCII bytes, alphanumeric at
// both ends, with `.`, `_` and `-` permitted inside.
func ValidSlug(s string) bool {
	owner, repo, ok := strings.Cut(s, "/")
	return ok && validSlugComponent(owner) && validSlugComponent(repo)
}

func validSlugComponent(c string) bool {
	if len(c) == 0 || len(c) > 100 {
		return false
	}
	for i := 0; i < len(c); i++ {
		b := c[i]
		alnum := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
		if alnum {
			continue
		}
		inner := i > 0 && i < len(c)-1
		if inner && (b == '.' || b == '_' || b == '-') {
			continue
		}
		return false
	}
	return true
}

// StampSlug sets maikanban.repositorySlug on dir when the repo carries
// docs/issues/ (what maikanban's fleet discovery scans for) and the key is
// unset. The owner is taken from the `origin` remote, else from the flag
// value ownerFlag, else from the global maikanban.defaultOwner. It returns
// the value it wrote, or noted=true when no owner resolved and the caller
// should print SlugNote. An existing value is never overwritten, and every
// failure is silent — init does not fail over fleet identity.
func StampSlug(dir, ownerFlag string) (slug string, noted bool) {
	if fi, err := os.Stat(filepath.Join(dir, "docs", "issues")); err != nil || !fi.IsDir() {
		return "", false
	}
	if !isGitRepo(dir) {
		return "", false
	}
	if v, ok := gitConfig(dir, "--get", SlugKey); ok && v != "" {
		return "", false
	}
	owner := resolveOwner(dir, ownerFlag)
	if owner == "" {
		return "", true
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", true
	}
	slug = owner + "/" + filepath.Base(abs)
	if !ValidSlug(slug) {
		return "", true
	}
	if _, ok := gitConfig(dir, SlugKey, slug); !ok {
		return "", true
	}
	return slug, false
}

func resolveOwner(dir, ownerFlag string) string {
	if url, ok := gitConfig(dir, "--get", "remote.origin.url"); ok {
		if owner := ownerFromRemote(url); owner != "" {
			return owner
		}
	}
	if validSlugComponent(ownerFlag) {
		return ownerFlag
	}
	if v, ok := gitConfig(dir, "--global", "--get", "maikanban.defaultOwner"); ok && validSlugComponent(v) {
		return v
	}
	return ""
}

// ownerFromRemote pulls <owner> out of a github.com[:/]<owner>/<repo> remote
// URL — ssh, https and scp-style forms alike. Non-GitHub remotes yield "".
func ownerFromRemote(url string) string {
	for _, sep := range []string{"github.com:", "github.com/"} {
		_, rest, found := strings.Cut(url, sep)
		if !found {
			continue
		}
		if owner, _, ok := strings.Cut(rest, "/"); ok && validSlugComponent(owner) {
			return owner
		}
	}
	return ""
}

func isGitRepo(dir string) bool {
	return exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run() == nil
}

func gitConfig(dir string, args ...string) (string, bool) {
	out, err := exec.Command("git", append([]string{"-C", dir, "config"}, args...)...).Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
