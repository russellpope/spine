package gate

import (
	"reflect"
	"testing"
)

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
