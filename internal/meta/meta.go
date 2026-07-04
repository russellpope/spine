// Package meta holds the shared artifact-file helpers: front-matter parsing
// (the first "---" ... "---" fence block) and title slugification. adr,
// handoff, eval, and doctor all consume it.
package meta

import (
	"strconv"
	"strings"
)

// Bounds returns the line indices of the first "---" ... "---" block: start
// is the opening fence, end the closing fence. -1, -1 if no block exists.
func Bounds(lines []string) (start, end int) {
	start, end = -1, -1
	for i, line := range lines {
		if line != "---" {
			continue
		}
		if start == -1 {
			start = i
			continue
		}
		end = i
		break
	}
	return start, end
}

// Parse returns every "key: value" (and bare "key:") pair inside the front-
// matter block, first occurrence winning. The colon must end the line (bare
// "key:") or be followed by a space — "key:value" with no space is not a
// pair, preserving parity with the "key: " prefix rule flippedContent-style
// consumers match on. has is false when no block exists; body content
// outside the block can never contribute keys.
func Parse(content string) (kv map[string]string, has bool) {
	lines := strings.Split(content, "\n")
	start, end := Bounds(lines)
	if start == -1 || end == -1 {
		return nil, false
	}
	kv = map[string]string{}
	for _, line := range lines[start+1 : end] {
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) == "" || strings.ContainsAny(k, " \t") {
			continue
		}
		if v != "" && !strings.HasPrefix(v, " ") {
			continue
		}
		if _, seen := kv[k]; !seen {
			kv[k] = strings.TrimSpace(v)
		}
	}
	return kv, true
}

// Slugify lowercases s and keeps only ASCII letters/digits, collapsing every
// other rune into a single '-' separator (trimmed at both ends). Lossy for
// non-ASCII by design; may return "" — callers must reject that.
func Slugify(s string) string {
	s = strings.ToLower(s)
	var b []rune
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b = append(b, r)
		default:
			if len(b) > 0 && b[len(b)-1] != '-' {
				b = append(b, '-')
			}
		}
	}
	return strings.Trim(string(b), "-")
}

// UnquoteScalar returns the Go-unquoted form of s when s parses as a
// double-quoted string — the front-matter form New writes since gen 3 (adr)
// and gen 4 (handoff, eval). Anything that does not parse — including every
// pre-quoting title — passes through verbatim (ratified display contract).
func UnquoteScalar(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
}
