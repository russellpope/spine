package gate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// This file is the gate pack's type-checking loader, shared by the check
// classes that cannot be decided syntactically (deferred-cleanup-errcheck
// needs the callee's signature; dead-code-callgraph needs callee identity).
//
// It is stdlib only, per ADR 0001: `go list` enumerates the module's
// packages and the export data of their dependencies, go/parser parses, and
// go/types type-checks each package against an export-data importer
// (go/importer with the "gc" compiler and a lookup over the files `go list
// -export` produced). No golang.org/x/tools dependency is taken.
//
// A target repo that does not compile is misconfiguration, not a finding: a
// gate cannot judge code the compiler has not agreed to.

// unit is one type-checked compilation unit — a package, or a package's
// test variant. A package with internal test files yields one unit holding
// both (that is how the compiler sees them); an external test package
// (foo_test) yields its own unit.
type unit struct {
	isMain  bool
	isXTest bool // an external test package (foo_test), not importable code
	files   []*ast.File
	info    *types.Info
}

// loaded is the whole module: every unit, sharing one FileSet so positions
// are comparable across packages.
type loaded struct {
	dir   string
	fset  *token.FileSet
	units []*unit
}

// goListPackage is the subset of `go list -json` output the loader reads.
type goListPackage struct {
	Dir          string
	ImportPath   string
	Name         string
	GoFiles      []string
	TestGoFiles  []string
	XTestGoFiles []string
	Export       string
	Error        *struct{ Err string }
}

// loadModule enumerates, parses and type-checks the Go module rooted at
// dir. Every returned error is misconfiguration: no packages, a `go list`
// failure, a parse error or a type error in the target repo.
func loadModule(dir string) (*loaded, error) {
	pkgs, err := goList(dir, "-json", "./...")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("--dir %s: no Go packages found (is there a go.mod?)", dir)
	}
	exports, err := exportData(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %q", path)
		}
		return os.Open(file)
	})
	out := &loaded{dir: dir, fset: fset}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ImportPath < pkgs[j].ImportPath })
	for _, p := range pkgs {
		if p.Error != nil {
			return nil, fmt.Errorf("--dir %s: package %s: %s", dir, p.ImportPath, p.Error.Err)
		}
		// One unit for the package plus its internal test files, one more
		// for its external test package when it has one.
		u, err := typeCheck(fset, imp, p, append(append([]string{}, p.GoFiles...), p.TestGoFiles...), p.ImportPath, false)
		if err != nil {
			return nil, err
		}
		if u != nil {
			out.units = append(out.units, u)
		}
		x, err := typeCheck(fset, imp, p, p.XTestGoFiles, p.ImportPath+"_test", true)
		if err != nil {
			return nil, err
		}
		if x != nil {
			out.units = append(out.units, x)
		}
	}
	return out, nil
}

// typeCheck parses and type-checks one set of files of one package.
func typeCheck(fset *token.FileSet, imp types.Importer, p goListPackage, names []string, importPath string, isXTest bool) (*unit, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var files []*ast.File
	for _, name := range names {
		path := filepath.Join(p.Dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		files = append(files, f)
	}
	info := &types.Info{
		Uses: map[*ast.Ident]types.Object{},
		Defs: map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: imp, Error: func(error) {}}
	pkg, err := conf.Check(importPath, fset, files, info)
	if err != nil {
		return nil, fmt.Errorf("type-checking %s: %w", importPath, err)
	}
	return &unit{
		isMain:  pkg.Name() == "main",
		isXTest: isXTest,
		files:   files,
		info:    info,
	}, nil
}

// exportData maps import path -> export data file for every dependency of
// the module's packages and their tests, which is what the "gc" importer
// reads. `go list -export` compiles what it must, so a target repo that
// does not build fails here — as misconfiguration, by the caller.
func exportData(dir string) (map[string]string, error) {
	pkgs, err := goList(dir, "-deps", "-export", "-test", "-json", "./...")
	if err != nil {
		return nil, err
	}
	exports := map[string]string{}
	for _, p := range pkgs {
		// `go list -test` also lists the synthesized test binaries and the
		// test-augmented variants ("foo [foo.test]"); no unit imports those.
		if p.Export == "" || strings.Contains(p.ImportPath, " ") {
			continue
		}
		if _, seen := exports[p.ImportPath]; !seen {
			exports[p.ImportPath] = p.Export
		}
	}
	return exports, nil
}

// goList runs `go list` in dir and decodes its JSON object stream.
func goList(dir string, args ...string) ([]goListPackage, error) {
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("--dir %s: go list %s: %s", dir, strings.Join(args, " "), msg)
	}
	var pkgs []goListPackage
	dec := json.NewDecoder(&stdout)
	for {
		var p goListPackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("--dir %s: decoding go list output: %w", dir, err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// rel reports a position's file relative to the module root, in the slash
// form every finding's file field uses.
func (l *loaded) rel(pos token.Pos) (string, int) {
	p := l.fset.Position(pos)
	return relSlash(l.dir, p.Filename), p.Line
}
