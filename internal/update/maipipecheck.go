package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
	dir, err := os.MkdirTemp("", "spine-maipipe")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	candidate := filepath.Join(dir, MaipipeFile)
	if err := os.WriteFile(candidate, []byte(content), 0o644); err != nil {
		return err
	}
	out, err := exec.Command(bin, "validate", candidate).CombinedOutput()
	if err == nil {
		return nil
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
	)
	for i, raw := range splitLines(content) {
		n := i + 1
		// A line that continues a multi-line string or array carries no
		// header or key of its own.
		continued := st.inML || st.depth > 0
		code, err := st.scan(raw)
		if err != nil {
			return fmt.Errorf("line %d: %v", n, err)
		}
		code = strings.TrimSpace(code)
		if code == "" || continued {
			continue
		}
		if strings.HasPrefix(code, "[") {
			name, array, err := tableHeader(code)
			if err != nil {
				return fmt.Errorf("line %d: %v", n, err)
			}
			if array {
				if tables[name] {
					return fmt.Errorf("line %d: [[%s]] redefines table [%s]", n, name, name)
				}
				arrays[name]++
				cur = fmt.Sprintf("%s#%d", name, arrays[name])
				continue
			}
			if tables[name] || arrays[name] > 0 {
				return fmt.Errorf("line %d: duplicate table [%s]", n, name)
			}
			tables[name] = true
			cur = name
			continue
		}
		key, _, ok := strings.Cut(code, "=")
		if !ok {
			return fmt.Errorf("line %d: expected a key/value pair or a table header, got %q", n, code)
		}
		key = strings.ReplaceAll(strings.TrimSpace(key), " ", "")
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

// tableHeader splits a `[name]` or `[[name]]` header line.
func tableHeader(code string) (name string, array bool, err error) {
	body, ok := strings.CutPrefix(code, "[[")
	if ok {
		array = true
		body, ok = strings.CutSuffix(body, "]]")
	} else {
		body = strings.TrimPrefix(code, "[")
		body, ok = strings.CutSuffix(body, "]")
	}
	if !ok {
		return "", array, fmt.Errorf("malformed table header %q", code)
	}
	name = strings.ReplaceAll(strings.TrimSpace(body), " ", "")
	if name == "" || strings.ContainsAny(name, "[]") {
		return "", array, fmt.Errorf("malformed table header %q", code)
	}
	return name, array, nil
}

// tomlScan carries the lexer state that spans lines: an open multi-line
// string, and the bracket depth of a multi-line array.
type tomlScan struct {
	inML  bool
	delim string
	depth int
}

// scan consumes one line and returns its structural code — string contents
// and comments removed — so the caller can read headers and keys without
// mistaking a `[` or a `#` inside a string for either.
func (s *tomlScan) scan(line string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(line); {
		if s.inML {
			j := strings.Index(line[i:], s.delim)
			if j < 0 {
				return b.String(), nil
			}
			i += j + len(s.delim)
			s.inML = false
			s.delim = ""
			continue
		}
		switch c := line[i]; {
		case c == '#':
			return b.String(), nil
		case c == '"' || c == '\'':
			d := string(c)
			if strings.HasPrefix(line[i:], strings.Repeat(d, 3)) {
				s.inML = true
				s.delim = strings.Repeat(d, 3)
				i += 3
				continue
			}
			end, err := closeQuote(line, i)
			if err != nil {
				return "", err
			}
			i = end
		case c == '[' || c == '{':
			s.depth++
			b.WriteByte(c)
			i++
		case c == ']' || c == '}':
			s.depth--
			if s.depth < 0 {
				return "", fmt.Errorf("unbalanced %q", string(c))
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), nil
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
