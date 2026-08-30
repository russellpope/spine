// Package ticketref owns the ticket-id and inclusive-range token grammar used
// by cursor resolution and transcript attribution.
package ticketref

import (
	"regexp"
	"strconv"
	"strings"
)

var idRe = regexp.MustCompile(`^I\d+$`)
var rangeRe = regexp.MustCompile(`^I(\d+)-I(\d+)$`)
var referenceInTextRe = regexp.MustCompile(`(^|[^A-Za-z0-9_-])(I\d+(?:-I\d+)?)`)

// IsID reports whether raw is one canonical I-prefixed ticket id.
func IsID(raw string) bool { return idRe.MatchString(raw) }

// Range parses an inclusive range with equal-width numeric endpoints.
func Range(raw string) (start, end, width int, ok bool) {
	m := rangeRe.FindStringSubmatch(raw)
	if m == nil || len(m[1]) != len(m[2]) {
		return 0, 0, 0, false
	}
	start, err1 := strconv.Atoi(m[1])
	end, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || start > end {
		return 0, 0, 0, false
	}
	return start, end, len(m[1]), true
}

// Contains reports whether text names id either literally or through a valid
// inclusive range token. It does not expand malformed or descending ranges.
func Contains(text, id string) bool {
	return ContainsStandalone(text, id)
}

// ContainsStandalone applies the strict standalone reference boundary to both
// literal IDs and ranges. It is used for D21 opening-line attribution, where
// a ticket-looking substring inside a larger hyphenated token is not a claim.
func ContainsStandalone(text, id string) bool {
	for _, reference := range References(text) {
		if reference == id {
			return true
		}
		start, end, width, ok := Range(reference)
		if !ok || len(id)-1 != width || !IsID(id) {
			continue
		}
		want, err := strconv.Atoi(strings.TrimPrefix(id, "I"))
		if err == nil && want >= start && want <= end {
			return true
		}
	}
	return false
}

// References returns distinct standalone ticket IDs and valid inclusive range
// tokens in first-occurrence order.
func References(text string) []string {
	seen := map[string]bool{}
	var references []string
	for _, match := range referenceInTextRe.FindAllStringSubmatchIndex(text, -1) {
		referenceStart, referenceEnd := match[4], match[5]
		if referenceEnd < len(text) && isReferenceWordChar(text[referenceEnd]) {
			continue
		}
		reference := text[referenceStart:referenceEnd]
		_, _, _, isRange := Range(reference)
		if !IsID(reference) && !isRange || seen[reference] {
			continue
		}
		seen[reference] = true
		references = append(references, reference)
	}
	return references
}

// ReferenceCount returns the number of distinct standalone explicit reference
// groups. One range is one group even when it covers several IDs. auditedIDs
// is retained for call-site compatibility; an explicit non-audited reference
// still makes an opening ambiguous.
func ReferenceCount(text string, auditedIDs []string) int {
	return len(References(text))
}

func isAlnum(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func isReferenceWordChar(value byte) bool {
	return isAlnum(value) || value == '_' || value == '-'
}
