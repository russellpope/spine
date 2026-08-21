package update

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/gate"
)

// futureClass is the class a later pack ships and the pinned pack does not.
const futureClass = "future-class"

// narrowGo1 is a go@1 class list deliberately *narrower* than the live
// registry: three classes, one of them the advisory mutate. A stub go@1 that
// merely equalled CheckNames() would let a render that ignored the pin pass
// this file's assertions, since today the two happen to coincide — the exact
// "passes for the wrong reason" shape this ticket exists to eliminate.
var narrowGo1 = []string{"binary-hygiene", "mutate", "tskip"}

// narrowGo2 is the later pack: narrowGo1 plus one class go@1 never had.
var narrowGo2 = append(append([]string(nil), narrowGo1...), futureClass)

// shipsTwoPacks stands in a spine binary shipping both go@1 and a later
// go@2, with go@1's list narrower than the registry. Nothing but
// packClassesFor is replaced: the registry and gate.packClasses stay as they
// are, so what the render follows is observable.
func shipsTwoPacks(t *testing.T) {
	t.Helper()
	real := packClassesFor
	packClassesFor = func(id string) ([]string, bool) {
		switch id {
		case gate.PackID():
			return append([]string(nil), narrowGo1...), true
		case "go@2":
			return append([]string(nil), narrowGo2...), true
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
//
// go@1 is stubbed narrower than the live registry, so a render that went back
// to enumerating gate.CheckNames() fails here — on the go@1 leg, not only via
// the go@2 one.
func TestPinnedPackRendersItsOwnFrozenClassList(t *testing.T) {
	shipsTwoPacks(t)

	pinned := renderGateRegion(gatePackSettings{
		pack: gate.PackID(), disabled: map[string]bool{}, config: map[string]string{}})
	// mutate renders last, in its own advisory pipeline, so the region's
	// stage order is not the class list's — compare the sets.
	if got := sorted(regionStageNames(splitLines(pinned))); !reflect.DeepEqual(got, sorted(narrowGo1)) {
		t.Errorf("%s did not render its frozen class list:\n got %v\nwant %v",
			gate.PackID(), got, sorted(narrowGo1))
	}
	// Named individually: a registered class outside the pin must not appear,
	// whether it comes from the registry or from the later pack.
	for _, c := range append(gate.CheckNames(), futureClass) {
		if slices.Contains(narrowGo1, c) {
			continue
		}
		if strings.Contains(pinned, "\""+c+"\"") {
			t.Errorf("class %q reached a repo pinned at %s:\n%s", c, gate.PackID(), pinned)
		}
	}

	// The other direction: the later pack does render its own list, so the
	// test above is about the pin and not about a class being unrenderable.
	moved := renderGateRegion(gatePackSettings{
		pack: "go@2", disabled: map[string]bool{}, config: map[string]string{}})
	if got := sorted(regionStageNames(splitLines(moved))); !reflect.DeepEqual(got, sorted(narrowGo2)) {
		t.Errorf("go@2 did not render its own class list:\n got %v\nwant %v", got, sorted(narrowGo2))
	}
}

// AC (I098), end to end: a full update of a repo pinned at go@1 on a binary
// whose registry has grown past go@1 writes a maipipe.toml carrying exactly
// the go@1 stages.
func TestPinnedRepoUpdateIgnoresLaterPackClasses(t *testing.T) {
	shipsTwoPacks(t)
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(dir, MaipipeFile))
	if names := sorted(regionStageNames(splitLines(got))); !reflect.DeepEqual(names, sorted(narrowGo1)) {
		t.Fatalf("rendered stages:\n got %v\nwant %v\n\n%s", names, sorted(narrowGo1), got)
	}
	if strings.Contains(got, futureClass) {
		t.Errorf("a go@1 repo got a go@2 class after update:\n%s", got)
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
	if len(mp.StagesChanged) != 0 {
		t.Errorf("StagesChanged = %v, want none for an add-only render", mp.StagesChanged)
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
	if len(mp.StagesChanged) != 0 {
		t.Errorf("StagesChanged = %v, want none for a remove-only render", mp.StagesChanged)
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
		t.Errorf("stage set churn reported for a config-only change: added=%v removed=%v",
			mp.StagesAdded, mp.StagesRemoved)
	}
	if want := []string{"gitignore-control"}; !reflect.DeepEqual(mp.StagesChanged, want) {
		t.Errorf("config-only definition delta = %v, want %v", mp.StagesChanged, want)
	}
}

// TestPinnedRunLinesAreRenderedAndReadAtTheRepoBoundary covers the managed
// region as an owner sees it: every stage receives the repo's pack pin, an
// old bare region is a safe stale migration, and another pin is a local edit
// rather than a projection of this repo's WORKFLOW.md.
func TestPinnedRunLinesAreRenderedAndReadAtTheRepoBoundary(t *testing.T) {
	dir := gateRepo(t, "[]", nil)
	if _, err := Run(Options{Dir: dir, Write: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, MaipipeFile)
	pinned := readFile(t, path)
	for _, check := range regionStageNames(splitLines(pinned)) {
		want := `run = "spine gate go@1 ` + check + `"`
		if !strings.Contains(pinned, want) {
			t.Errorf("stage %q does not carry the repo pin %q:\n%s", check, want, pinned)
		}
	}
	if !strings.Contains(pinned, `run = "spine gate go@1 mutate"`) {
		t.Errorf("mutation stage does not carry the repo pin:\n%s", pinned)
	}

	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := report(t, reports, MaipipeFile).Unrecognized; len(got) != 0 {
		t.Errorf("canonical pinned region is unrecognized: %v", got)
	}

	bare := strings.ReplaceAll(pinned, "spine gate go@1 ", "spine gate go ")
	if err := os.WriteFile(path, []byte(bare), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	legacy := report(t, reports, MaipipeFile)
	if legacy.State != Pending || len(legacy.Unrecognized) != 0 {
		t.Errorf("bare legacy region = state %v unrecognized %v, want safe stale migration", legacy.State, legacy.Unrecognized)
	}

	foreign := strings.ReplaceAll(pinned, "spine gate go@1 ", "spine gate go@2 ")
	if err := os.WriteFile(path, []byte(foreign), 0o644); err != nil {
		t.Fatal(err)
	}
	reports, err = Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := report(t, reports, MaipipeFile).Unrecognized; len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), "go@2") {
		t.Errorf("foreign pin was accepted as region content: %v", got)
	}
}
