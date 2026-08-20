package gate

import (
	"reflect"
	"strings"
	"testing"
)

// go1Classes is the golden list: the check classes `gate_pack: go@1` renders,
// frozen. It is a literal here on purpose — the point is that it is written
// down twice, so no edit to the registry can move it silently.
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

// TestGo1FrozenClassList holds the live registries and the go@1 frozen class
// list to each other, which is what makes `gate_pack: go@1` a pin (I098).
// Without it, registering a check inserts a blocking stage into the rendered
// region of every repo pinned at go@1 on their next `spine update` — the
// outcome ADR 0015 item 2 and spec story 23 forbid.
func TestGo1FrozenClassList(t *testing.T) {
	const contract = "`gate_pack: go@1` pins a frozen class list (ADR 0015 item 2, " +
		"spec story 23): a repo pinned at go@1 gets these classes and only these, " +
		"so a check may not reach it without the owner moving the pin.\n" +
		"To add or rename a class: ship it under a new pack version — bump " +
		"gate.PackVersion, add the version's class list to gate.packClasses, and " +
		"fork a golden list for it here, leaving go@1's alone. Editing the go@1 " +
		"list itself changes what already-pinned repos enforce, and needs a " +
		"recorded reason in the commit."

	if got := CheckNames(); !reflect.DeepEqual(got, go1Classes) {
		t.Errorf("the check registry and the go@1 class list disagree:\n"+
			" registry %v\n    go@1 %v\n\n%s", got, go1Classes, contract)
	}
	got, ok := PackClassesFor(PackID())
	if !ok {
		t.Fatalf("this binary does not ship %s, the pack version it renders as\n\n%s", PackID(), contract)
	}
	if !reflect.DeepEqual(got, go1Classes) {
		t.Errorf("gate.packClasses[%d] and the go@1 golden list disagree:\n"+
			" packClasses %v\n        go@1 %v\n\n%s", PackVersion, got, go1Classes, contract)
	}
}

// A pack identifier this binary does not ship is not a class list: it is
// refused, never guessed at (the other half of the pin — an old binary
// declines a newer pack rather than approximating it).
func TestPackClassesForRejectsPacksNotShipped(t *testing.T) {
	for _, id := range []string{"go@2", "go@0", "go", "rust@1", "go@x", "go@1 "} {
		if classes, ok := PackClassesFor(id); ok {
			t.Errorf("PackClassesFor(%q) = %v, true; want refused", id, classes)
		}
	}
	if ids := PackIDs(); !reflect.DeepEqual(ids, []string{"go@1"}) {
		t.Errorf("PackIDs() = %v, want [go@1]", ids)
	}
	if !strings.Contains(strings.Join(PackIDs(), ", "), PackID()) {
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
