package hermetic

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/russellpope/spine/internal/hostconfig"
)

func TestMain(m *testing.M) { os.Exit(Main(m)) }

// The host routing file is the read that made the suite depend on the
// developer's machine: under Isolate its default path is inside the fresh
// home, and restore hands back the previous resolution.
func TestIsolateMovesTheHostRoutingFile(t *testing.T) {
	before, err := hostconfig.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	root, restore, err := Isolate()
	if err != nil {
		t.Fatal(err)
	}
	got, err := hostconfig.DefaultPath()
	if err != nil {
		restore()
		t.Fatal(err)
	}
	cache := os.Getenv("GOCACHE")
	restore()
	if !within(root, got) || got == before {
		t.Errorf("DefaultPath() = %q, want a path under %q (was %q)", got, root, before)
	}
	if cache == "" || within(root, cache) {
		t.Errorf("GOCACHE = %q, want the developer's cache outside %q", cache, root)
	}
	if after, _ := hostconfig.DefaultPath(); after != before {
		t.Errorf("after restore DefaultPath() = %q, want %q", after, before)
	}
	if _, err := os.Stat(root); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("isolated home %q survives restore: %v", root, err)
	}
}

// TestEveryTestPackageIsHermetic fails when a package's tests can run
// against the real user config dir: every directory in the module holding
// _test.go files must have a TestMain that calls hermetic.Main.
func TestEveryTestPackageIsHermetic(t *testing.T) {
	root := moduleRoot(t)
	self, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	seen := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
			return filepath.SkipDir
		}
		if path != root {
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
		}
		tests, err := filepath.Glob(filepath.Join(path, "*_test.go"))
		if err != nil || len(tests) == 0 {
			return err
		}
		seen++
		ok, err := callsMain(tests, path == self)
		if err != nil {
			return err
		}
		if !ok {
			rel, _ := filepath.Rel(root, path)
			missing = append(missing, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen < 2 {
		t.Fatalf("found %d test packages under %s; the walk is not seeing the tree", seen, root)
	}
	sort.Strings(missing)
	for _, dir := range missing {
		t.Errorf("%s: tests can read the real user config dir; add func TestMain(m *testing.M) { os.Exit(hermetic.Main(m)) }", dir)
	}
}

// callsMain reports whether a TestMain in files calls hermetic.Main (or
// Main, inside this package).
func callsMain(files []string, self bool) (bool, error) {
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
		if err != nil {
			return false, err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "TestMain" || fn.Body == nil {
				continue
			}
			found := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return !found
				}
				switch fun := call.Fun.(type) {
				case *ast.SelectorExpr:
					if x, ok := fun.X.(*ast.Ident); ok && x.Name == "hermetic" && fun.Sel.Name == "Main" {
						found = true
					}
				case *ast.Ident:
					if self && fun.Name == "Main" {
						found = true
					}
				}
				return !found
			})
			if found {
				return true, nil
			}
		}
	}
	return false, nil
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test directory")
		}
		dir = parent
	}
}
