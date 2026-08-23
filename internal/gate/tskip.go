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
			recv, ok := skipReceiver(sel.X)
			if !ok {
				return true
			}
			line := fset.Position(sel.Sel.Pos()).Line
			if allow[rel] || allow[rel+":"+strconv.Itoa(line)] {
				return true
			}
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("skipped test: %s.%s call in a _test.go file", recv, sel.Sel.Name),
				File:     rel,
				Line:     line,
				Code:     cfg.Code("tskip"),
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

// tbNames are the receiver identifiers a testing.TB value conventionally
// carries. Restricting the ident case to these keeps the class from firing
// on an unrelated method that happens to be named Skip — an operator's only
// remedy for that would be allowlisting a line that is not a skip.
var tbNames = map[string]bool{"t": true, "b": true, "tb": true}

// skipReceiver reports whether expr is a testing.TB receiver for a Skip
// call, and its printed form for the finding message. Two shapes match: a
// conventionally named identifier (t, b, tb) and a T() accessor call, which
// is how suite-style tests reach the TB (s.T().Skip()).
func skipReceiver(expr ast.Expr) (string, bool) {
	switch x := expr.(type) {
	case *ast.Ident:
		if tbNames[x.Name] {
			return x.Name, true
		}
	case *ast.CallExpr:
		if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "T" {
			if recv, ok := sel.X.(*ast.Ident); ok {
				return recv.Name + ".T()", true
			}
			return "T()", true
		}
	}
	return "", false
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

// skipDir names directories the syntactic check classes do not descend
// into: the toolchain's own ignore set — testdata and any name starting
// with "_" or "." — plus vendor and node_modules. Go repos routinely keep
// template or deliberately broken sources under testdata, and `go build
// ./...` never sees them; a gate that parsed them would exit 2 on a tree
// the compiler is perfectly happy with.
func skipDir(name string) bool {
	return name == "testdata" || skipDirExceptTestdata(name)
}

// skipDirExceptTestdata is skipDir minus the testdata rule, for the one
// walker that must descend into testdata: gitignore-control's entry-point
// arm, where an ignored testdata entry point is hidden just the same.
func skipDirExceptTestdata(name string) bool {
	if name == "vendor" || name == "node_modules" {
		return true
	}
	return len(name) > 1 && (name[0] == '_' || name[0] == '.')
}

// underTestdata reports whether rel (a slash-separated path) has a testdata
// element, the paths the toolchain excludes from a build.
func underTestdata(rel string) bool {
	for _, elem := range strings.Split(rel, "/") {
		if elem == "testdata" {
			return true
		}
	}
	return false
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
