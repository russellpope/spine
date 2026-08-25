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
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
)

// This file is the gate pack's type-checking loader, shared by the check
// classes that cannot be decided syntactically (deferred-cleanup-errcheck
// needs the callee's signature; dead-code-callgraph needs callee identity).
//
// It is stdlib only, per ADR 0001: one `go list` invocation enumerates the
// module's packages and the export data of every dependency, go/parser
// parses, and go/types type-checks each package against an export-data
// importer (go/importer with the "gc" compiler and a lookup over the files
// `go list -export` produced). No golang.org/x/tools dependency is taken.
//
// A target repo the loader cannot judge is misconfiguration, not a finding,
// in two distinguishable shapes: the module cannot be loaded at all (a
// broken go.mod, no `go` on PATH, no packages), or it loads but does not
// type-check. A gate cannot judge code the compiler has not agreed to.

// unit is one type-checked compilation unit — a package, or a package's
// test variant. A package with internal test files yields one unit holding
// both (that is how the compiler sees them); an external test package
// (foo_test) yields its own unit.
type unit struct {
	isMain  bool
	isXTest bool // an external test package (foo_test), not importable code
	pkg     *types.Package
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

var newLoaderImporter = importer.ForCompiler

// goListPackage is the subset of `go list -json` output the loader reads.
type goListPackage struct {
	Dir          string
	ImportPath   string
	Name         string
	GoFiles      []string
	CgoFiles     []string
	TestGoFiles  []string
	XTestGoFiles []string
	Export       string
	Module       *struct{ Main bool }
	Error        *goListError
	DepsErrors   []*goListError
}

// goListError is `go list -e`'s per-package error. A package that does not
// compile carries one in Error; every package that imports it carries the
// same text in DepsErrors (and no export data).
type goListError struct{ Err string }

// firstLine strips the `# importpath` header `go build` prefixes to a
// compile failure and returns the first diagnostic line, so the refusal
// reads `internal/inv/inv.go:3:29: undefined: x` and not a transcript.
func (e *goListError) firstLine() string {
	for _, line := range strings.Split(e.Err, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "# ") {
			return line
		}
	}
	return strings.TrimSpace(e.Err)
}

// isMainModule reports whether p is a package of the module under --dir and
// not one of the artefacts `go list -test` synthesizes: the test binary
// ("foo.test") and the test-augmented variants ("foo [foo.test]"), whose
// files the plain package listing already covers.
func (p goListPackage) isMainModule() bool {
	return p.Module != nil && p.Module.Main &&
		!strings.Contains(p.ImportPath, " ") && !strings.HasSuffix(p.ImportPath, ".test")
}

// loadModule enumerates, parses and type-checks the Go module rooted at
// dir. One `go list` call serves both purposes — the module's own packages
// and every dependency's export data — because each check class is its own
// pipeline stage, so every extra subprocess is paid again per enabled class.
func loadModule(dir string) (out *loaded, err error) {
	packageUnderCheck := dir
	defer func() {
		if recovered := recover(); recovered != nil {
			value := fmt.Sprint(recovered)
			if strings.Contains(value, "export data version") {
				out = nil
				err = fmt.Errorf("spine could not decode export data: %s; binary toolchain: %s; rebuild spine with make install", value, runtime.Version())
				return
			}
			out = nil
			err = fmt.Errorf("spine internal error while loading package %s: panic: %s\n%s", packageUnderCheck, value, debug.Stack())
		}
	}()
	pkgs, err := goList(dir)
	if err != nil {
		return nil, err
	}
	exports := map[string]string{}
	var own []goListPackage
	for _, p := range pkgs {
		if p.Export != "" && !strings.Contains(p.ImportPath, " ") {
			if _, seen := exports[p.ImportPath]; !seen {
				exports[p.ImportPath] = p.Export
			}
		}
		if p.isMainModule() {
			own = append(own, p)
		}
	}
	if len(own) == 0 {
		return nil, fmt.Errorf("cannot load the module under --dir %s: no Go packages found (is there a go.mod?)", dir)
	}
	// Refuse on the first compile error before type-checking anything:
	// the packages are checked in import-path order, and an importer that
	// sorts before its broken dependency would otherwise fail first with
	// the downstream symptom ("could not import … (no export data)")
	// instead of the cause (I093.2). A package's own Error outranks the
	// DepsErrors it inherits, so the named error is the one to fix.
	sort.Slice(own, func(i, j int) bool { return own[i].ImportPath < own[j].ImportPath })
	for _, p := range own {
		if p.Error != nil {
			return nil, fmt.Errorf("--dir %s does not type-check: %s: %s", dir, p.ImportPath, p.Error.firstLine())
		}
	}
	for _, p := range own {
		if len(p.DepsErrors) > 0 {
			return nil, fmt.Errorf("--dir %s does not type-check: dependency of %s: %s", dir, p.ImportPath, p.DepsErrors[0].firstLine())
		}
	}
	fset := token.NewFileSet()
	imp := newLoaderImporter(fset, "gc", func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %q", path)
		}
		return os.Open(file)
	})
	out = &loaded{dir: dir, fset: fset}
	for _, p := range own {
		packageUnderCheck = p.ImportPath
		// One unit for the package plus its internal test files, one more
		// for its external test package when it has one. CgoFiles belong to
		// the first: a cgo package whose C-backed declarations are missing
		// from the file set does not type-check.
		files := append(append(append([]string{}, p.GoFiles...), p.CgoFiles...), p.TestGoFiles...)
		u, err := typeCheck(fset, imp, p, files, p.ImportPath, false)
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
// FakeImportC is set so a cgo package's `import "C"` resolves: the loader
// reads the .go sources as written, not the cgo-generated ones.
func typeCheck(fset *token.FileSet, imp types.Importer, p goListPackage, names []string, importPath string, isXTest bool) (*unit, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var files []*ast.File
	for _, name := range names {
		path := filepath.Join(p.Dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("--dir %s does not type-check: parsing %s: %w", p.Dir, name, err)
		}
		files = append(files, f)
	}
	info := &types.Info{
		Uses: map[*ast.Ident]types.Object{},
		Defs: map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: imp, FakeImportC: true, Error: func(error) {}}
	pkg, err := conf.Check(importPath, fset, files, info)
	if err != nil {
		return nil, fmt.Errorf("--dir %s does not type-check: %s: %w", p.Dir, importPath, err)
	}
	return &unit{
		isMain:  pkg.Name() == "main",
		isXTest: isXTest,
		pkg:     pkg,
		files:   files,
		info:    info,
	}, nil
}

// goList runs the loader's one `go list` invocation in dir and decodes its
// JSON object stream. -e keeps a package that does not compile from failing
// the whole command, so a type error arrives as that package's Error field
// and only a module that cannot be loaded at all fails here.
func goList(dir string) ([]goListPackage, error) {
	args := []string{"list", "-deps", "-export", "-test", "-e", "-json", "./..."}
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("cannot load the module under --dir %s: go %s: %s", dir, strings.Join(args, " "), msg)
	}
	var pkgs []goListPackage
	dec := json.NewDecoder(&stdout)
	for {
		var p goListPackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("cannot load the module under --dir %s: decoding go list output: %w", dir, err)
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
