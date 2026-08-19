package checkpoint

import (
	"fmt"
	"strings"
)

// Facts is the checkpoint's facts region: the harness-written machine facts.
// Sole-writer and canonical-form rules apply exactly as for the cursor —
// only `spine checkpoint new` writes this block, and its bytes are a pure
// function of these values (see Block).
type Facts struct {
	Touched           []string // caller-supplied, caller order preserved
	Gate              string   // pass | fail | none
	SHA               string   // git rev-parse HEAD of the repo
	EffortRecommended string   // recommended per-leg effort
	Written           string   // RFC3339 UTC
}

// Gate values accepted by `spine checkpoint new --gate`.
const (
	GatePass = "pass"
	GateFail = "fail"
	GateNone = "none"
)

// ValidGate reports whether g is one of the three gate statuses.
func ValidGate(g string) bool {
	return g == GatePass || g == GateFail || g == GateNone
}

// Block renders f as the canonical facts region, markers included. The bytes
// are a pure function of f's values: same values in, same bytes out, in this
// fixed key order. No trailing newline — the document writer owns the
// file-level newline convention (mirrors cursor.Cursor.Block).
func (f Facts) Block() string {
	var b strings.Builder
	b.WriteString(FactsOpenTag + "\n")
	b.WriteString("touched:")
	for _, t := range f.Touched {
		b.WriteString("\n- " + strings.TrimSpace(t))
	}
	b.WriteString("\n")
	b.WriteString("gate: " + strings.TrimSpace(f.Gate) + "\n")
	b.WriteString("sha: " + strings.TrimSpace(f.SHA) + "\n")
	b.WriteString("effort_recommended: " + strings.TrimSpace(f.EffortRecommended) + "\n")
	b.WriteString("written: " + strings.TrimSpace(f.Written) + "\n")
	b.WriteString(FactsCloseTag)
	return b.String()
}

// ParseFacts reads a facts region body (the bytes between the markers, as
// Document.Facts holds them) back into Facts. It is deliberately strict: the
// keys must appear exactly once each, in canonical order, with no unknown
// lines — anything else is a hand edit, which the doctor advisory reports.
func ParseFacts(body string) (Facts, error) {
	var f Facts
	lines := strings.Split(strings.Trim(body, "\n"), "\n")
	if len(lines) == 0 || lines[0] != "touched:" {
		return Facts{}, fmt.Errorf("facts region must start with a touched: line")
	}
	i := 1
	for ; i < len(lines) && strings.HasPrefix(lines[i], "- "); i++ {
		f.Touched = append(f.Touched, strings.TrimPrefix(lines[i], "- "))
	}
	for _, key := range []struct {
		name string
		dst  *string
	}{
		{"gate", &f.Gate},
		{"sha", &f.SHA},
		{"effort_recommended", &f.EffortRecommended},
		{"written", &f.Written},
	} {
		if i >= len(lines) {
			return Facts{}, fmt.Errorf("facts region is missing the %s: line", key.name)
		}
		value, ok := strings.CutPrefix(lines[i], key.name+": ")
		if !ok {
			return Facts{}, fmt.Errorf("facts region line %d is not %s: <value>", i+1, key.name)
		}
		*key.dst = value
		i++
	}
	if i != len(lines) {
		return Facts{}, fmt.Errorf("facts region has %d unexpected trailing line(s)", len(lines)-i)
	}
	if !ValidGate(f.Gate) {
		return Facts{}, fmt.Errorf("facts region gate %q is not pass|fail|none", f.Gate)
	}
	return f, nil
}

// Canonical reports whether body is byte-identical to the canonical
// rendering of the values it parses to — the drift check the doctor
// advisory needs.
func Canonical(body string) bool {
	f, err := ParseFacts(body)
	if err != nil {
		return false
	}
	return f.Block() == FactsOpenTag+"\n"+strings.Trim(body, "\n")+"\n"+FactsCloseTag
}
