package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/scaffold"
)

// slugRepo makes a git repo named `name` under a temp dir, with the global
// git config redirected to an empty file so the developer's own
// maikanban.defaultOwner cannot leak into these tests.
func slugRepo(t *testing.T, name string, remote string) string {
	t.Helper()
	globalCfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(globalCfg, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", globalCfg)
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q")
	if remote != "" {
		git(t, dir, "remote", "add", "origin", remote)
	}
	return dir
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func slugOf(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "config", "--get", scaffold.SlugKey).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// TestInitStampsSlugFromOrigin is the primary path: an origin remote names
// BOTH halves of the slug, and the stamp is reported as a created item.
// Re-running init leaves the value alone. The working directory is
// deliberately named something other than the remote's repo (the case of a
// clone or worktree under a different name) so the test fails if the repo half
// ever falls back to the directory basename.
func TestInitStampsSlugFromOrigin(t *testing.T) {
	dir := slugRepo(t, "x-checkout", "git@github.com:acme/x.git")
	code, out, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "x")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "create: git config maikanban.repositorySlug acme/x") {
		t.Errorf("stamp not reported under create: out=%q", out)
	}
	if strings.Contains(out, scaffold.SlugNote) {
		t.Errorf("note printed despite a resolved owner: out=%q", out)
	}
	if got := slugOf(t, dir); got != "acme/x" {
		t.Fatalf("slug = %q, want acme/x", got)
	}
	// re-run: value unchanged and no longer reported as created
	code, out, errs = runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "x")
	if code != 0 {
		t.Fatalf("re-run code=%d stderr=%q", code, errs)
	}
	if strings.Contains(out, "create: git config") {
		t.Errorf("re-run re-stamped the slug: out=%q", out)
	}
	if got := slugOf(t, dir); got != "acme/x" {
		t.Fatalf("re-run slug = %q, want acme/x", got)
	}
}

// TestInitNoOwnerPrintsNote is the negative control for the stamp: with no
// origin, no --owner and no maikanban.defaultOwner, init still succeeds and
// writes no config — it only tells the owner what to run.
func TestInitNoOwnerPrintsNote(t *testing.T) {
	dir := slugRepo(t, "x", "")
	code, out, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "x")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, scaffold.SlugNote) {
		t.Errorf("note line missing: out=%q", out)
	}
	if got := slugOf(t, dir); got != "" {
		t.Fatalf("slug written without an owner: %q", got)
	}
}

// TestInitOwnerFlagStampsBasename covers the --owner fallback: no origin, so
// the flag names the owner and the directory names the repo.
func TestInitOwnerFlagStampsBasename(t *testing.T) {
	dir := slugRepo(t, "widget", "")
	code, out, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "widget", "--owner", "acme")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "create: git config maikanban.repositorySlug acme/widget") {
		t.Errorf("stamp not reported: out=%q", out)
	}
	if got := slugOf(t, dir); got != "acme/widget" {
		t.Fatalf("slug = %q, want acme/widget", got)
	}
}

// TestInitDefaultOwnerConfigStamps covers the last fallback in the resolution
// order: the global maikanban.defaultOwner.
func TestInitDefaultOwnerConfigStamps(t *testing.T) {
	dir := slugRepo(t, "widget", "")
	git(t, dir, "config", "--global", "maikanban.defaultOwner", "fleetco")
	code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "widget")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if got := slugOf(t, dir); got != "fleetco/widget" {
		t.Fatalf("slug = %q, want fleetco/widget", got)
	}
}

// TestInitLeavesPreExistingSlugUntouched: a repo whose owner already chose a
// slug keeps it, even when origin would resolve to something else.
func TestInitLeavesPreExistingSlugUntouched(t *testing.T) {
	dir := slugRepo(t, "x-checkout", "git@github.com:acme/x.git")
	git(t, dir, "config", scaffold.SlugKey, "chosen/name")
	code, out, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "x")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if strings.Contains(out, "create: git config") || strings.Contains(out, scaffold.SlugNote) {
		t.Errorf("init spoke about a slug it must not touch: out=%q", out)
	}
	if got := slugOf(t, dir); got != "chosen/name" {
		t.Fatalf("slug = %q, want chosen/name", got)
	}
}

// TestInitRejectsInvalidOwnerFlag: an unusable --owner is reported on stderr
// rather than silently falling through to the note line. Init still exits 0.
func TestInitRejectsInvalidOwnerFlag(t *testing.T) {
	dir := slugRepo(t, "widget", "")
	code, out, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "widget", "--owner", "bad owner")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(errs, `--owner "bad owner" is not a valid slug component`) {
		t.Errorf("invalid --owner not reported: stderr=%q", errs)
	}
	if !strings.Contains(out, scaffold.SlugNote) {
		t.Errorf("note line missing: out=%q", out)
	}
	if got := slugOf(t, dir); got != "" {
		t.Fatalf("slug written from an invalid owner: %q", got)
	}
}

// TestInitOriginBeatsOwnerFlag pins the resolution order the ticket
// specifies and the controller ruled stays: a GitHub origin outranks an
// explicit --owner. A "helpful" reorder must fail here.
func TestInitOriginBeatsOwnerFlag(t *testing.T) {
	dir := slugRepo(t, "x-checkout", "git@github.com:acme/x.git")
	code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "x", "--owner", "other")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if got := slugOf(t, dir); got != "acme/x" {
		t.Fatalf("slug = %q, want acme/x (origin outranks --owner)", got)
	}
}

// TestInitUnparseableOriginRefusesToStamp is the Important-1 control: an
// origin on another host is evidence that the directory basename is NOT the
// repository name, so the silent maikanban.defaultOwner path must not stamp.
// Before this rule the fixture below was stamped `fleetco/x-checkout` — both
// halves wrong, permanently, with nothing printed.
func TestInitUnparseableOriginRefusesToStamp(t *testing.T) {
	dir := slugRepo(t, "x-checkout", "git@gitlab.com:realowner/x.git")
	git(t, dir, "config", "--global", "maikanban.defaultOwner", "fleetco")
	code, out, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "x")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, scaffold.SlugNote) {
		t.Errorf("note line missing: out=%q", out)
	}
	if got := slugOf(t, dir); got != "" {
		t.Fatalf("stamped %q from an origin spine cannot parse", got)
	}
}

// TestInitUnparseableOriginHonoursOwnerFlag: the refusal above is about the
// silent path. An operator who types --owner has said what they mean, and
// that behaviour is deliberately unchanged.
func TestInitUnparseableOriginHonoursOwnerFlag(t *testing.T) {
	dir := slugRepo(t, "widget", "git@gitlab.com:realowner/x.git")
	code, _, errs := runCmd(t, "init", "--dir", dir, "--profile", "library-cli", "--name", "widget", "--owner", "acme")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if got := slugOf(t, dir); got != "acme/widget" {
		t.Fatalf("slug = %q, want acme/widget", got)
	}
}
