package gate

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func tskipViolationDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "seeded_test.go"), []byte("package seeded\n\nimport \"testing\"\n\nfunc TestSeeded(t *testing.T) { t.Skip(\"seeded violation\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestResolvedPinsDriveGateAttributionAndClassMembership names the two
// production breaks I103 prevents: attributing a go@1 run as the binary's
// own pack, and allowing a class introduced by a later pack through go@1.
// The later pack and its only-new class exist only for this test; production
// continues to ship just go@1.
func TestResolvedPinsDriveGateAttributionAndClassMembership(t *testing.T) {
	const laterOnly = "later-only"
	oldClasses := packClasses
	oldCheck, hadCheck := checks[laterOnly]
	packClasses = map[int][]string{
		1: append([]string(nil), oldClasses[1]...),
		2: append(append([]string(nil), oldClasses[1]...), laterOnly),
	}
	checks[laterOnly] = func(string, Config) ([]Finding, error) { return nil, nil }
	t.Cleanup(func() {
		packClasses = oldClasses
		if hadCheck {
			checks[laterOnly] = oldCheck
		} else {
			delete(checks, laterOnly)
		}
	})

	pinned, ok := ResolvePack("go@1")
	if !ok {
		t.Fatal("ResolvePack(go@1) refused a shipped pin")
	}
	if got, want := pinned.Code("tskip"), "go@1/tskip"; got != want {
		t.Errorf("go@1 code = %q, want %q", got, want)
	}
	bare, ok := ResolvePack("go")
	if !ok || bare.ID != PackID() {
		t.Errorf("bare go resolves to %#v, %v; want binary pack %s", bare, ok, PackID())
	}
	if _, ok := ResolvePack("go@9"); ok {
		t.Error("ResolvePack(go@9) accepted an unshipped pin")
	}

	dir := tskipViolationDir(t)
	var out, errs bytes.Buffer
	if got := Run("go@1", "tskip", dir, &out, &errs, EnvConfig()); got != 1 {
		t.Fatalf("pinned tskip exit = %d, want findings exit 1; stderr=%q", got, errs.String())
	}
	if !strings.Contains(out.String(), "go@1/tskip") {
		t.Errorf("pinned finding not attributed to go@1: %q", out.String())
	}

	out.Reset()
	errs.Reset()
	if got := Run("go@1", laterOnly, dir, &out, &errs, EnvConfig()); got != 2 {
		t.Errorf("out-of-pin class exit = %d, want 2; stdout=%q stderr=%q", got, out.String(), errs.String())
	}
	for _, want := range []string{"go@1", laterOnly} {
		if !strings.Contains(errs.String(), want) {
			t.Errorf("out-of-pin refusal does not name %q: %q", want, errs.String())
		}
	}

	results := filepath.Join(t.TempDir(), "results.json")
	t.Setenv(ResultsEnvVar, results)
	out.Reset()
	errs.Reset()
	if got := Run("go@9", "tskip", dir, &out, &errs, EnvConfig()); got != 2 {
		t.Errorf("unshipped pin exit = %d, want 2; stdout=%q stderr=%q", got, out.String(), errs.String())
	}
	if _, err := os.Stat(results); !os.IsNotExist(err) {
		t.Errorf("unshipped pin wrote findings document: err=%v", err)
	}
}

// goldenClasses is the golden list per pack version: the check classes
// `gate_pack: go@<version>` renders, frozen. Literals on purpose — the point
// is that each version's list is written down twice, so no edit to the
// registry or to gate.packClasses can move one silently. A new pack version
// gets its own entry here; the entries already present are never edited to
// accommodate it.
var goldenClasses = map[int][]string{
	1: go1Classes,
}

// go1Classes is what `gate_pack: go@1` renders, and what every repo pinned at
// go@1 enforces, on any spine binary.
var go1Classes = []string{
	"binary-hygiene",
	"dead-code-callgraph",
	"deferred-cleanup-errcheck",
	"fixture-manifest",
	"gitignore-control",
	"mutate",
	"n-plus-one",
	"test-enum-vs-spec",
	"tskip",
}

// TestFrozenClassLists holds each shipped pack version's class list to a
// golden literal, and the live registries to the union of all of them. That
// pair is what makes `gate_pack: go@N` a pin (I098): without it, registering
// a check inserts a blocking stage into the rendered region of every pinned
// repo on their next `spine update` — the outcome ADR 0015 item 2 and spec
// story 23 forbid.
//
// The two assertions fail for different reasons and say so separately: a
// version's list moving is a change to what already-pinned repos enforce; the
// union disagreeing with the registry is a class registered under no pack (it
// can never run) or named by a pack but unregistered (its stage fails as an
// unknown check). The union, not go@1 specifically, is what the registry is
// held to — so shipping go@2 is a normal edit here rather than a false alarm.
func TestFrozenClassLists(t *testing.T) {
	const frozen = "A pack version pins a frozen class list (ADR 0015 item 2, spec story 23): " +
		"a repo pinned at go@N gets that version's classes and only those, so a check " +
		"may not reach it without the owner moving the pin.\n" +
		"To add or rename a class: ship it under a NEW pack version — add its list to " +
		"gate.packClasses and its golden literal to goldenClasses here, leaving the " +
		"existing versions' lists untouched. Editing a shipped version's list changes " +
		"what already-pinned repos enforce, and needs a recorded reason in the commit."
	const closure = "Every registered check class must be reachable through some pack " +
		"version, and every class a pack version names must be registered: an " +
		"unreachable class can never run, and an unregistered one renders a stage that " +
		"exits 2 as an unknown check.\n" +
		"Registering a check is therefore two edits, not one: the registry map, and the " +
		"class list of the pack version that ships it (with its golden literal here)."

	for v, want := range goldenClasses {
		got, ok := PackClassesFor(PackName + "@" + strconv.Itoa(v))
		if !ok {
			t.Errorf("goldenClasses has %s@%d but gate.packClasses does not ship it — a pinned "+
				"repo would be refused rather than rendered\n\n%s", PackName, v, frozen)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("gate.packClasses[%d] and its golden list disagree:\n"+
				" packClasses %v\n      golden %v\n\n%s", v, got, want, frozen)
		}
	}
	for _, id := range PackIDs() {
		v, err := strconv.Atoi(strings.TrimPrefix(id, PackName+"@"))
		if err != nil {
			t.Fatalf("PackIDs() returned an unparseable id %q", id)
		}
		if _, ok := goldenClasses[v]; !ok {
			t.Errorf("this binary ships %s with no golden list here — the version's class "+
				"list is written down once and so is unpinned\n\n%s", id, frozen)
		}
	}

	union := map[string]bool{}
	for _, classes := range goldenClasses {
		for _, c := range classes {
			union[c] = true
		}
	}
	want := make([]string, 0, len(union))
	for c := range union {
		want = append(want, c)
	}
	sort.Strings(want)
	if got := CheckNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("the check registry and the classes the shipped packs name disagree:\n"+
			" registry %v\n   packs %v\n\n%s\n\n%s", got, want, closure, frozen)
	}
	// The pack this binary renders as must itself be one it ships: PackID()
	// is what every finding's code field carries.
	if _, ok := PackClassesFor(PackID()); !ok {
		t.Errorf("this binary renders findings as %s, a pack it does not ship\n\n%s", PackID(), frozen)
	}
}

// A pack identifier this binary does not ship is not a class list: it is
// refused, never guessed at (the other half of the pin — an old binary
// declines a newer pack rather than approximating it). The unshipped version
// is derived, not hard-coded, so this stays true when a later pack ships.
func TestPackClassesForRejectsPacksNotShipped(t *testing.T) {
	next := 0
	for v := range packClasses {
		if v > next {
			next = v
		}
	}
	unshipped := PackName + "@" + strconv.Itoa(next+1)
	for _, id := range []string{unshipped, "go@0", "go", "rust@1", "go@x", "go@1 ", ""} {
		if classes, ok := PackClassesFor(id); ok {
			t.Errorf("PackClassesFor(%q) = %v, true; want refused", id, classes)
		}
	}
	for _, id := range PackIDs() {
		if _, ok := PackClassesFor(id); !ok {
			t.Errorf("PackIDs() names %s, which PackClassesFor refuses", id)
		}
	}
	if !slices.Contains(PackIDs(), PackID()) {
		t.Errorf("PackIDs() %v omits the pack this binary renders as (%s)", PackIDs(), PackID())
	}
}

// The frozen list is the pin's contract, not a scratch buffer: a caller that
// mutates what it gets back must not move the next caller's list.
func TestPackClassesForReturnsACopy(t *testing.T) {
	first, _ := PackClassesFor(PackID())
	first[0] = "clobbered"
	second, _ := PackClassesFor(PackID())
	if second[0] == "clobbered" {
		t.Errorf("PackClassesFor aliases the frozen list: %v", second)
	}
}

// TestSortFindingsOrder pins the documented results-contract order — file
// asc, then line asc, then message asc — on a fixture that ties at each
// level. The byte-equality test in cmd/spine only proves stability; it
// survived a reversed file comparator (I093.1, mutation probe
// gate-sort-findings-reversed).
func TestSortFindingsOrder(t *testing.T) {
	in := []Finding{
		{File: "b.go", Line: 1, Message: "m"},
		{File: "a.go", Line: 9, Message: "m"},
		{File: "a.go", Line: 2, Message: "z"},
		{File: "a.go", Line: 2, Message: "a"},
	}
	want := []Finding{
		{File: "a.go", Line: 2, Message: "a"},
		{File: "a.go", Line: 2, Message: "z"},
		{File: "a.go", Line: 9, Message: "m"},
		{File: "b.go", Line: 1, Message: "m"},
	}
	sortFindings(in)
	if !reflect.DeepEqual(in, want) {
		t.Errorf("sortFindings order:\n got %+v\nwant %+v", in, want)
	}
}
