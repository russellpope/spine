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

// ValidOwner reports whether s is usable as the owner half of a slug.
func ValidOwner(s string) bool { return validSlugComponent(s) }

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
// unset. The value comes from the `origin` remote (both halves), else from
// the flag value ownerFlag, else from the global maikanban.defaultOwner — the
// latter two paired with the directory basename, and see resolveSlug for when
// each applies. It returns the value it wrote, or noted=true when nothing
// resolved and the caller should print SlugNote. An existing value is never
// overwritten, and every failure is silent — init does not fail over fleet
// identity.
//
// Known property: `dir` is used as-is, so scaffolding into a subdirectory of a
// repo (`spine init --dir sub`) writes the *enclosing* repo's config, and
// maikanban would then identify `sub/` by the parent's slug. That is
// deliberate — scaffolding docs into a subdirectory of a repo you own is a
// legitimate use, and gating on `rev-parse --show-toplevel` would break it.
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
	slug = resolveSlug(dir, ownerFlag)
	if slug == "" || !ValidSlug(slug) {
		return "", true
	}
	if _, ok := gitConfig(dir, SlugKey, slug); !ok {
		return "", true
	}
	return slug, false
}

// resolveSlug returns the whole `owner/repo` to stamp, or "" when spine must
// refuse and leave the note instead. A GitHub `origin` names both halves and
// is the authority on the repo name: a clone into a differently named
// directory (a worktree, a second checkout) must still carry the remote's
// identity, since that is what maikanban resolves against.
//
// The presence of *any* origin is evidence that the directory basename is not
// authoritative, so an origin spine cannot parse (a non-GitHub host, an
// unusual form) refuses rather than falling through to the silent
// maikanban.defaultOwner path: refusing is visible and fixable, a wrong stamp
// is silent and permanent. An explicit --owner is deliberately still honoured
// there — the operator typing the flag has said what they mean.
//
// The basename fallback therefore applies only when there is no origin at all
// (both the --owner and the maikanban.defaultOwner paths), plus the explicit
// --owner case above.
func resolveSlug(dir, ownerFlag string) string {
	if url, ok := gitConfig(dir, "--get", "remote.origin.url"); ok && url != "" {
		if slug := slugFromRemote(url); slug != "" {
			return slug
		}
		if !ValidOwner(ownerFlag) {
			return ""
		}
		return basenameSlug(dir, ownerFlag)
	}
	owner := ownerFlag
	if !ValidOwner(owner) {
		v, ok := gitConfig(dir, "--global", "--get", "maikanban.defaultOwner")
		if !ok || !ValidOwner(v) {
			return ""
		}
		owner = v
	}
	return basenameSlug(dir, owner)
}

func basenameSlug(dir, owner string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	return owner + "/" + filepath.Base(abs)
}

// slugFromRemote pulls `<owner>/<repo>` out of a github.com[:/]<owner>/<repo>
// remote URL — ssh, https and scp-style forms alike, with any trailing `.git`
// and `/` stripped. The host is matched as a substring anywhere in the URL,
// which is what lets one pass cover the scp, ssh:// and https forms.
//
// GitHub only, by design: maikanban slugs are GitHub `owner/repo` identities,
// and every other host yields "" so that resolveSlug refuses and init leaves
// the note. Do not add a branch for another host here — that reintroduces
// exactly the silent wrong-stamp path this branch removed twice (a host whose
// path layout differs would produce a grammar-valid, permanent, wrong slug).
func slugFromRemote(url string) string {
	// `/` first: `ssh://git@github.com:22/acme/x.git` matches both separators,
	// and only the `/` reading puts the owner (not the port) first.
	for _, sep := range []string{"github.com/", "github.com:"} {
		_, rest, found := strings.Cut(url, sep)
		if !found {
			continue
		}
		segs := strings.Split(strings.TrimSuffix(strings.TrimSuffix(rest, "/"), ".git"), "/")
		if len(segs) > 2 && allDigits(segs[0]) {
			segs = segs[1:] // scp-style host:port left of the path
		}
		if len(segs) != 2 {
			continue
		}
		if slug := segs[0] + "/" + segs[1]; ValidSlug(slug) {
			return slug
		}
	}
	return ""
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
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
