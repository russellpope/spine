package update

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/gate"
)

// futureClass is the class a later pack ships and go@1 does not.
const futureClass = "future-class"

// shipsGo2 stands in a spine binary that ships both go@1 and a later pack:
// go@2's class list is go@1's plus one new class. It is the only thing a
// second pack version needs for the pin to be testable — the registry and
// the real packClasses map stay untouched.
func shipsGo2(t *testing.T) {
	t.Helper()
	real := packClassesFor
	go1, ok := real(gate.PackID())
	if !ok {
		t.Fatalf("this binary does not ship %s", gate.PackID())
	}
	go2 := append(append([]string(nil), go1...), futureClass)
	packClassesFor = func(id string) ([]string, bool) {
		if id == "go@2" {
			return append([]string(nil), go2...), true
		}
		return real(id)
	}
	t.Cleanup(func() { packClassesFor = real })
}

// sorted returns a sorted copy, so a region's render order (mutate last, in
// its own pipeline) can be compared against a class list.
func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// AC (I098): the pin is a frozen list. A binary that also ships a later pack
// renders `gate_pack: go@1` as exactly the go@1 classes — the new class is
// reachable only by moving the pin, which is what ADR 0015 item 2 and spec
// story 23 promise.
func TestPinnedPackRendersItsOwnFrozenClassList(t *testing.T) {
	shipsGo2(t)
	go1, _ := gate.PackClassesFor(gate.PackID())

	pinned := renderGateRegion(gatePackSettings{
		pack: gate.PackID(), disabled: map[string]bool{}, config: map[string]string{}})
	// mutate renders last, in its own advisory pipeline, so the region's
	// stage order is not the class list's — compare the sets.
	got := sorted(regionStageNames(splitLines(pinned)))
	if !reflect.DeepEqual(got, go1) {
		t.Errorf("go@1 rendered from a binary shipping go@2:\n got %v\nwant %v", got, go1)
	}
	if strings.Contains(pinned, futureClass) {
		t.Errorf("a class this binary added reached a repo pinned at %s:\n%s", gate.PackID(), pinned)
	}

	// The other direction: the later pack does render its own list, so the
	// test above is about the pin and not about the class being unrenderable.
	moved := renderGateRegion(gatePackSettings{
		pack: "go@2", disabled: map[string]bool{}, config: map[string]string{}})
	wantGo2 := sorted(append(append([]string(nil), go1...), futureClass))
	if !reflect.DeepEqual(sorted(regionStageNames(splitLines(moved))), wantGo2) {
		t.Errorf("go@2 did not render its own class list:\n%s", moved)
	}
}

// AC (I098), end to end: a full update of a repo pinned at go@1 on a binary
// that also ships go@2 writes a maipipe.toml with no trace of the later
// pack's class.
func TestPinnedRepoUpdateIgnoresLaterPackClasses(t *testing.T) {
	shipsGo2(t)
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	if strings.Contains(got, futureClass) {
		t.Fatalf("go@1 repo got a go@2 class after update:\n%s", got)
	}
	go1, _ := gate.PackClassesFor(gate.PackID())
	if names := sorted(regionStageNames(splitLines(got))); !reflect.DeepEqual(names, go1) {
		t.Errorf("rendered stages:\n got %v\nwant %v", names, go1)
	}
}

// AC (I098): the plan names the stages a render adds to, and drops from, the
// region already on disk, so the new blocking step and the definition_hash
// re-approval are both visible before --write.
func TestPlanReportsAddedAndRemovedStages(t *testing.T) {
	dir := gateRepo(t, "[tskip, n-plus-one]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}

	// Re-enabling two classes adds their two stages, and drops none.
	optIn(t, dir, gate.PackID(), "[]", nil)
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if want := []string{"n-plus-one", "tskip"}; !reflect.DeepEqual(mp.StagesAdded, want) {
		t.Errorf("StagesAdded = %v, want %v", mp.StagesAdded, want)
	}
	if len(mp.StagesRemoved) != 0 {
		t.Errorf("StagesRemoved = %v, want none", mp.StagesRemoved)
	}
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}

	// And the reverse: disabling one drops its stage and adds nothing.
	optIn(t, dir, gate.PackID(), "[tskip]", nil)
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp = report(t, reports, MaipipeFile)
	if want := []string{"tskip"}; !reflect.DeepEqual(mp.StagesRemoved, want) {
		t.Errorf("StagesRemoved = %v, want %v", mp.StagesRemoved, want)
	}
	if len(mp.StagesAdded) != 0 {
		t.Errorf("StagesAdded = %v, want none", mp.StagesAdded)
	}
}

// Negative control: a render that changes no stage reports no stage churn,
// so the plan's added/removed lines mean what they say. A gate_pack_config
// value change rewrites the region's bytes without touching the stage list.
func TestUnchangedStageListReportsNoChurn(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	optIn(t, dir, gate.PackID(), "[]", map[string]string{"build_outputs": "bin/demo"})
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	mp := report(t, reports, MaipipeFile)
	if mp.State != Pending {
		t.Fatalf("a changed gate_pack_config did not re-render the region: state=%v", mp.State)
	}
	if len(mp.StagesAdded) != 0 || len(mp.StagesRemoved) != 0 {
		t.Errorf("stage churn reported for a config-only change: added=%v removed=%v",
			mp.StagesAdded, mp.StagesRemoved)
	}
}
