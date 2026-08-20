package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// maipipeTimeout bounds the validate run. Validation is a parse of one small
// file; anything slower than this is a maipipe that is not going to answer.
const maipipeTimeout = 30 * time.Second

// noMaipipeNote is what a refusal says when no maipipe binary was resolvable:
// the TOML parse still ran, the grammar check maipipe would add did not.
const noMaipipeNote = "maipipe validate skipped: no maipipe binary on PATH"

// duplicateStageHint names the likely cause of the one failure spine can
// diagnose from maipipe's message (I096 Fix 3). A stage name is unique per
// pipeline and the region spine renders declares each check once, so a
// duplicate after a splice means a second copy of that stage sits outside
// the markers. spine refuses; it never moves the copy back.
const duplicateStageHint = "hint: the region spine renders declares each stage once, so a duplicate almost always means a copy of that stage now sits outside the " +
	gateRegionBegin + "… / " + gateRegionEnd + " markers — move or delete it by hand; spine will not rewrite what is outside its region"

// checkMaipipeContent refuses candidate maipipe.toml content that maipipe
// could not load, before anything reaches disk (I096). path is the file the
// content is destined for and is named in every refusal — the maipipe run
// works off a temp copy, so the real file is never touched.
func checkMaipipeContent(path, content string) error {
	bin, lookErr := exec.LookPath("maipipe")
	note := ""
	if lookErr != nil {
		note = " [" + noMaipipeNote + "]"
	}
	if err := parseTOML(content); err != nil {
		return fmt.Errorf("refusing to write %s: %v%s", path, err, note)
	}
	if lookErr != nil {
		return nil
	}
	// The candidate is validated as a temp copy so the real file is never
	// touched by a check that may refuse. This assumes `maipipe validate`
	// resolves nothing relative to the file's own directory: maipipe reads
	// exactly one file with no include mechanism (ADR 0017). If it ever
	// grows includes or repo-relative config, the temp copy's verdict stops
	// matching the real file's and this has to validate in place instead.
	dir, err := os.MkdirTemp("", "spine-maipipe")
	if err != nil {
		return fmt.Errorf("maipipe pre-flight for %s: %w", path, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	candidate := filepath.Join(dir, MaipipeFile)
	if err := os.WriteFile(candidate, []byte(content), 0o644); err != nil {
		return fmt.Errorf("maipipe pre-flight for %s: %w", path, err)
	}
	// A hung or interactive maipipe must not hang spine update.
	ctx, cancel := context.WithTimeout(context.Background(), maipipeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "validate", candidate).CombinedOutput()
	if err == nil {
		return nil
	}
	// Only a non-zero exit is a verdict on the content. A binary that could
	// not run at all — missing library, killed, exit 127, or the timeout
	// above — says nothing about the file, and saying "rejected" would send
	// the reader hunting for a defect that is not there.
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		if ctx.Err() != nil {
			return fmt.Errorf("refusing to write %s: maipipe validate did not finish within %s: %v",
				path, maipipeTimeout, err)
		}
		return fmt.Errorf("refusing to write %s: could not run maipipe validate (%v): %s",
			path, err, strings.TrimSpace(string(out)))
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		msg = err.Error()
	}
	if strings.Contains(msg, "duplicate stage name") {
		msg += "\n" + duplicateStageHint
	}
	return fmt.Errorf("refusing to write %s: maipipe validate rejected the result:\n%s", path, msg)
}

// parseTOML reports whether content parses as TOML, structurally: table
// headers, array-of-tables entries and key lines, with duplicate tables and
// duplicate keys within a table treated as the parse errors they are. It
// does not interpret values — spine only needs to know maipipe can load the
// file, and the values inside the region are its own render.
func parseTOML(content string) error {
	var (
		tables = map[string]bool{}
		arrays = map[string]int{}
		keys   = map[string]bool{}
		cur    string
		st     tomlScan
		// aot is the array-of-tables entry currently open, and aotN which
		// entry of it. A standard table under an entry belongs to that
		// entry, so `[[a]] [a.b] [[a]] [a.b]` is four distinct tables, not
		// a duplicate — qualifying the name is what keeps it that way.
		aot  string
		aotN int
	)
	qualify := func(name string) string {
		if aot != "" && strings.HasPrefix(name, aot+".") {
			return fmt.Sprintf("%s#%d.%s", aot, aotN, strings.TrimPrefix(name, aot+"."))
		}
		return name
	}
	for i, raw := range splitLines(content) {
		n := i + 1
		// A line that continues a multi-line string or array carries no
		// header or key of its own.
		continued := st.inML || st.depth > 0
		code, strs, err := st.scan(raw)
		if err != nil {
			return fmt.Errorf("line %d: %v", n, err)
		}
		code = strings.TrimSpace(code)
		if code == "" || continued {
			continue
		}
		if strings.HasPrefix(code, "[") {
			name, array, err := tableHeader(code, strs)
			if err != nil {
				return fmt.Errorf("line %d: %v", n, err)
			}
			q := qualify(name)
			if array {
				if tables[q] {
					return fmt.Errorf("line %d: [[%s]] redefines table [%s]", n, name, name)
				}
				arrays[q]++
				aot, aotN = name, arrays[q]
				cur = fmt.Sprintf("%s#%d", q, arrays[q])
				continue
			}
			if tables[q] || arrays[q] > 0 {
				return fmt.Errorf("line %d: duplicate table [%s]", n, name)
			}
			tables[q] = true
			cur = q
			continue
		}
		key, _, ok := strings.Cut(code, "=")
		if !ok {
			return fmt.Errorf("line %d: expected a key/value pair or a table header, got %q", n, code)
		}
		// Spaces are not part of a bare or dotted key (`a . b` is `a.b`);
		// a quoted key is a placeholder here, and no placeholder contains a
		// space, so the strip cannot reach inside one.
		key = restoreStrings(strings.ReplaceAll(strings.TrimSpace(key), " ", ""), strs)
		if key == "" {
			return fmt.Errorf("line %d: empty key", n)
		}
		full := cur + "\x00" + key
		if keys[full] {
			return fmt.Errorf("line %d: duplicate key %q", n, key)
		}
		keys[full] = true
	}
	switch {
	case st.inML:
		return fmt.Errorf("unterminated multi-line string (%s)", st.delim)
	case st.depth > 0:
		return fmt.Errorf("unclosed %d bracket(s) at end of file", st.depth)
	}
	return nil
}

// tableHeader splits a `[name]` or `[[name]]` header line. code carries
// placeholders where quoted key segments were, so the brackets it is checked
// for are real brackets and never characters inside a quoted name; strs puts
// those segments back, keeping `[a."x"]` and `[a."y"]` distinct names.
func tableHeader(code string, strs []string) (name string, array bool, err error) {
	body, ok := strings.CutPrefix(code, "[[")
	if ok {
		array = true
		body, ok = strings.CutSuffix(body, "]]")
	} else {
		body = strings.TrimPrefix(code, "[")
		body, ok = strings.CutSuffix(body, "]")
	}
	if !ok {
		return "", array, fmt.Errorf("malformed table header %q", restoreStrings(code, strs))
	}
	name = strings.ReplaceAll(strings.TrimSpace(body), " ", "")
	if name == "" || strings.ContainsAny(name, "[]") {
		return "", array, fmt.Errorf("malformed table header %q", restoreStrings(code, strs))
	}
	return restoreStrings(name, strs), array, nil
}

// strPlaceholder brackets the index of a string scan consumed, standing in
// for it in the structural code. \x01 cannot appear in a TOML file's text
// the parser cares about, so the token can never collide with real content.
const strPlaceholder = "\x01"

// restoreStrings puts the consumed strings back into structural code, so a
// name or a message reads the way the file does.
func restoreStrings(code string, strs []string) string {
	if !strings.Contains(code, strPlaceholder) {
		return code
	}
	for i, s := range strs {
		code = strings.ReplaceAll(code, strPlaceholder+strconv.Itoa(i)+strPlaceholder, s)
	}
	return code
}

// tomlScan carries the lexer state that spans lines: an open multi-line
// string, and the bracket depth of a multi-line array.
type tomlScan struct {
	inML  bool
	delim string
	depth int
}

// scan consumes one line and returns its structural code — comments removed
// and every string replaced by a placeholder — plus the strings it replaced,
// in order. The caller can then read headers and keys without mistaking a
// `[`, a `#` or an `=` inside a string for structure, and can still put a
// quoted key back where it belongs. Dropping the strings outright, which an
// earlier cut did, made `[a."b.c"]` and `"my key" = 1` — both legal TOML —
// look malformed.
func (s *tomlScan) scan(line string) (string, []string, error) {
	var b strings.Builder
	var strs []string
	placeholder := func(text string) {
		b.WriteString(strPlaceholder + strconv.Itoa(len(strs)) + strPlaceholder)
		strs = append(strs, text)
	}
	for i := 0; i < len(line); {
		if s.inML {
			j := strings.Index(line[i:], s.delim)
			if j < 0 {
				return b.String(), strs, nil
			}
			i += j + len(s.delim)
			s.inML = false
			s.delim = ""
			continue
		}
		switch c := line[i]; {
		case c == '#':
			return b.String(), strs, nil
		case c == '"' || c == '\'':
			d := string(c)
			if strings.HasPrefix(line[i:], strings.Repeat(d, 3)) {
				s.inML = true
				s.delim = strings.Repeat(d, 3)
				// A multi-line string is only ever a value, never a key, so
				// its text does not have to survive — only its position.
				placeholder(s.delim)
				i += 3
				continue
			}
			end, err := closeQuote(line, i)
			if err != nil {
				return "", nil, err
			}
			placeholder(line[i:end])
			i = end
		case c == '[' || c == '{':
			s.depth++
			b.WriteByte(c)
			i++
		case c == ']' || c == '}':
			s.depth--
			if s.depth < 0 {
				return "", nil, fmt.Errorf("unbalanced %q", string(c))
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), strs, nil
}

// closeQuote returns the index just past the single-line string that opens
// at line[start].
func closeQuote(line string, start int) (int, error) {
	q := line[start]
	for i := start + 1; i < len(line); i++ {
		if q == '"' && line[i] == '\\' {
			i++
			continue
		}
		if line[i] == q {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("unterminated string %q", line[start:])
}
