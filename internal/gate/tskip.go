package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// tskipAllowKey is the gate_pack_config key holding the tskip allowlist; it
// reaches the stage as SPINE_GATE_TSKIP_ALLOW.
const tskipAllowKey = "tskip_allow"

// skipNames are the testing.TB skip methods. Any call to one of them from a
// _test.go file is a finding: the check class is zero tolerance by design,
// and an intentionally skipped test is expressed by allowlisting it, not by
// the skip call being invisible.
var skipNames = map[string]bool{"Skip": true, "Skipf": true, "SkipNow": true}

// checkTskip reports every t.Skip / t.Skipf / t.SkipNow (and the b.* forms)
// call in a _test.go file under dir. Entries in the SPINE_GATE_TSKIP_ALLOW
// allowlist are not findings; an unset allowlist means "no allowlist", not
// misconfiguration.
func checkTskip(dir string, cfg Config) ([]Finding, error) {
	raw, _ := cfg.Get(tskipAllowKey)
	allow, err := parseTskipAllow(raw)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	fset := token.NewFileSet()
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		rel := relSlash(dir, path)
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return fmt.Errorf("parsing %s: %w", rel, perr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !skipNames[sel.Sel.Name] {
				return true
			}
			recv, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			line := fset.Position(sel.Sel.Pos()).Line
			if allow[rel] || allow[rel+":"+strconv.Itoa(line)] {
				return true
			}
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("skipped test: %s.%s call in a _test.go file", recv.Name, sel.Sel.Name),
				File:     rel,
				Line:     line,
				Code:     Code("tskip"),
			})
			return true
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return findings, nil
}

// parseTskipAllow reads the allowlist format: a comma-separated list of
// entries, each either a repo-root-relative slash-separated file path
// (allowing every skip in that file) or path:line (allowing one call).
// Blank entries are ignored; a non-numeric line suffix is misconfiguration.
func parseTskipAllow(raw string) (map[string]bool, error) {
	allow := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if path, line, ok := strings.Cut(entry, ":"); ok {
			if _, err := strconv.Atoi(line); err != nil {
				return nil, fmt.Errorf("%s: entry %q is not path or path:line", EnvVar(tskipAllowKey), entry)
			}
			entry = path + ":" + line
		}
		allow[entry] = true
	}
	return allow, nil
}

// skipDir names directories no check class descends into.
func skipDir(name string) bool {
	return name == ".git" || name == "vendor" || name == "node_modules"
}

// relSlash reports path relative to dir in slash form, the form every
// finding's file field uses.
func relSlash(dir, path string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
