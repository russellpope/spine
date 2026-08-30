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
var referenceInTextRe = regexp.MustCompile(`(^|[^A-Za-z0-9_-])(I\d+(?:-I\d+)?)([^A-Za-z0-9_-]|$)`)

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
	if !IsID(id) {
		return false
	}
	if containsToken(text, id) {
		return true
	}
	want, err := strconv.Atoi(strings.TrimPrefix(id, "I"))
	if err != nil {
		return false
	}
	for _, reference := range References(text) {
		start, end, width, ok := Range(reference)
		if ok && len(id)-1 == width && want >= start && want <= end {
			return true
		}
	}
	return false
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
	for _, match := range referenceInTextRe.FindAllStringSubmatch(text, -1) {
		reference := match[2]
		_, _, _, isRange := Range(reference)
		if !IsID(reference) && !isRange || seen[reference] {
			continue
		}
		seen[reference] = true
		references = append(references, reference)
	}
	return references
}

// ReferenceCount returns the number of distinct standalone reference groups
// that claim at least one audited ticket ID. One range is one group even when
// it expands to several IDs.
func ReferenceCount(text string, auditedIDs []string) int {
	count := 0
	for _, reference := range References(text) {
		for _, id := range auditedIDs {
			if reference == id {
				count++
				break
			}
			start, end, width, ok := Range(reference)
			if !ok || !IsID(id) || len(id)-1 != width {
				continue
			}
			want, err := strconv.Atoi(strings.TrimPrefix(id, "I"))
			if err == nil && want >= start && want <= end {
				count++
				break
			}
		}
	}
	return count
}

func containsToken(text, token string) bool {
	for start := 0; ; {
		i := strings.Index(text[start:], token)
		if i < 0 {
			return false
		}
		i += start
		before := i == 0 || !isAlnum(text[i-1])
		afterIndex := i + len(token)
		after := afterIndex >= len(text) || !isAlnum(text[afterIndex])
		if before && after {
			return true
		}
		start = i + 1
	}
}

func isAlnum(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}
