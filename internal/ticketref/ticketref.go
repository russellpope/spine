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
var rangeInTextRe = regexp.MustCompile(`(^|[^A-Za-z0-9])(I\d+-I\d+)([^A-Za-z0-9]|$)`)

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
	for _, match := range rangeInTextRe.FindAllStringSubmatch(text, -1) {
		start, end, width, ok := Range(match[2])
		if ok && len(id)-1 == width && want >= start && want <= end {
			return true
		}
	}
	return false
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
