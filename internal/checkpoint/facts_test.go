package checkpoint

import "testing"

// The facts region obeys canonical form: its bytes are a pure function of
// its values. Rendering the same values twice — and rendering values parsed
// back out of a block — must produce identical bytes.
func TestFactsBlockIsByteDeterministic(t *testing.T) {
	f := Facts{
		Touched:           []string{"internal/checkpoint/new.go", "cmd/spine/main.go"},
		Gate:              GatePass,
		SHA:               "0123456789abcdef0123456789abcdef01234567",
		EffortRecommended: "high",
		Written:           "2026-08-18T17:04:05Z",
	}
	want := `<!-- spine:checkpoint:facts -->
touched:
- internal/checkpoint/new.go
- cmd/spine/main.go
gate: pass
sha: 0123456789abcdef0123456789abcdef01234567
effort_recommended: high
written: 2026-08-18T17:04:05Z
<!-- /spine:checkpoint:facts -->`
	if got := f.Block(); got != want {
		t.Fatalf("block =\n%s\nwant\n%s", got, want)
	}
	if f.Block() != f.Block() {
		t.Fatal("two renderings of the same values differ")
	}
	round, err := ParseFacts(Split("x\n" + f.Block() + "\n").Facts)
	if err != nil {
		t.Fatalf("ParseFacts: %v", err)
	}
	if round.Block() != want {
		t.Fatalf("round-trip block =\n%s", round.Block())
	}
}

func TestFactsBlockWithEmptyTouched(t *testing.T) {
	f := Facts{Gate: GateNone, SHA: "abc", EffortRecommended: "low", Written: "2026-08-18T00:00:00Z"}
	want := `<!-- spine:checkpoint:facts -->
touched:
gate: none
sha: abc
effort_recommended: low
written: 2026-08-18T00:00:00Z
<!-- /spine:checkpoint:facts -->`
	if got := f.Block(); got != want {
		t.Fatalf("block =\n%s\nwant\n%s", got, want)
	}
}

func TestCanonicalDetectsDrift(t *testing.T) {
	canonical := "touched:\n- a.go\ngate: pass\nsha: abc\neffort_recommended: high\nwritten: 2026-08-18T00:00:00Z"
	if !Canonical(canonical) {
		t.Error("canonical body reported as drifted (negative control)")
	}
	for name, body := range map[string]string{
		"reordered keys":   "touched:\n- a.go\nsha: abc\ngate: pass\neffort_recommended: high\nwritten: 2026-08-18T00:00:00Z",
		"extra whitespace": "touched:\n- a.go\ngate:  pass\nsha: abc\neffort_recommended: high\nwritten: 2026-08-18T00:00:00Z",
		"unknown line":     canonical + "\nnote: hand edited",
		"missing key":      "touched:\n- a.go\ngate: pass\nsha: abc\nwritten: 2026-08-18T00:00:00Z",
		"bad gate":         "touched:\ngate: maybe\nsha: abc\neffort_recommended: high\nwritten: 2026-08-18T00:00:00Z",
	} {
		if Canonical(body) {
			t.Errorf("%s: reported canonical", name)
		}
	}
}
