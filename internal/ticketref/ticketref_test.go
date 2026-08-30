package ticketref

import "testing"

func TestRangeMatchesCursorGrammar(t *testing.T) {
	for _, tc := range []struct {
		raw               string
		start, end, width int
		ok                bool
	}{
		{raw: "I051-I056", start: 51, end: 56, width: 3, ok: true},
		{raw: "I051-I051", start: 51, end: 51, width: 3, ok: true},
		{raw: "I051-I52", ok: false},
		{raw: "I056-I051", ok: false},
		{raw: "I051-I05X", ok: false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			start, end, width, ok := Range(tc.raw)
			if start != tc.start || end != tc.end || width != tc.width || ok != tc.ok {
				t.Fatalf("Range(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)", tc.raw, start, end, width, ok, tc.start, tc.end, tc.width, tc.ok)
			}
		})
	}
}

func TestContainsRecognizesLiteralAndInteriorReferencesOnly(t *testing.T) {
	for _, tc := range []struct {
		text, id string
		want     bool
	}{
		{text: "tickets I051-I056", id: "I053", want: true},
		{text: "tickets (I051-I056).", id: "I056", want: true},
		{text: "tickets I051-I05X", id: "I053", want: false},
		{text: "tickets I056-I051", id: "I053", want: false},
		{text: "ticket I0510", id: "I051", want: false},
	} {
		if got := Contains(tc.text, tc.id); got != tc.want {
			t.Errorf("Contains(%q, %q) = %v, want %v", tc.text, tc.id, got, tc.want)
		}
	}
}
