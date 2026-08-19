package gate

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"
)

// checkDeadCode reports functions and methods — exported and unexported —
// that no root can reach, across every package of the module under --dir.
//
// Roots (the documented rule):
//
//   - every `main` function of a `package main` package;
//   - every `init` function;
//   - every Test*/Benchmark*/Example*/Fuzz* function in a _test.go file, so
//     code reached only from tests is live;
//   - every reference made at package level (a var initializer naming a
//     function, a table of handlers), since that reference is not inside
//     any function body;
//   - every exported function and method of an importable package — a
//     non-main, non-test package whose import path has no `internal`
//     element. A library's exported API is its contract; a gate cannot see
//     the callers and must not call the contract dead. A package under
//     `internal/` has no such contract: no other module can import it, so
//     its exported declarations are candidates like any other.
//
// The call graph is built from types, not from names: every reference to a
// function object inside a body is an edge, whether it is a direct call, a
// method value, or a function passed as an argument. A call through an
// interface marks every concrete method with that name reachable — a
// deliberate over-approximation, because a false positive here accuses
// working code of being dead.
//
// Interface satisfaction is the same over-approximation, one step wider: a
// method is live if its name is a method of any interface declared in the
// module or in a package the module imports (and of the universe's error).
// Most methods reached through an interface are never named by the module
// that declares them — a String called only by fmt, a ServeHTTP called only
// by net/http, a Len/Less/Swap called only by sort — and the module's own
// source shows nothing but the concrete type. The residual limitation: a
// method reached only through an interface from a package the module does
// not directly import is still reportable.
//
// Only declarations in non-test files are candidates: an unused test helper
// is the test author's business.
func checkDeadCode(dir string, cfg Config) ([]Finding, error) {
	mod, err := loadModule(dir)
	if err != nil {
		return nil, err
	}
	g := newCallGraph(mod)
	reachable := g.reach()
	var findings []Finding
	for _, key := range g.candidates() {
		if reachable[key] {
			continue
		}
		d := g.decls[key]
		findings = append(findings, Finding{
			Severity: SeverityError,
			Message:  fmt.Sprintf("unreachable function: %s is not reachable from any main, init, test root or exported library API", d.display),
			File:     d.file,
			Line:     d.line,
			Code:     Code("dead-code-callgraph"),
		})
	}
	return findings, nil
}

// funcDecl is one declared function or method: where it is and how a
// finding names it.
type funcDecl struct {
	display string
	file    string
	line    int
	inTest  bool // declared in a _test.go file
}

// callGraph is the module's function reference graph, keyed by a stable
// identity string rather than by *types.Func pointer: the same function
// reached from a package's own source and from another package's export
// data is two objects but one function.
type callGraph struct {
	decls map[string]funcDecl
	edges map[string]map[string]bool
	roots map[string]bool
	// methodsByName lets an interface call reach every concrete method of
	// that name, the over-approximation the class prefers to a false
	// positive.
	methodsByName map[string][]string
}

func newCallGraph(mod *loaded) *callGraph {
	g := &callGraph{
		decls:         map[string]funcDecl{},
		edges:         map[string]map[string]bool{},
		roots:         map[string]bool{},
		methodsByName: map[string][]string{},
	}
	for _, u := range mod.units {
		for _, file := range u.files {
			path, _ := mod.rel(file.Pos())
			inTest := strings.HasSuffix(path, "_test.go")
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok {
					// A package-level declaration's references have no
					// enclosing function; treat them as roots.
					for _, key := range g.referenced(u, decl) {
						g.roots[key] = true
					}
					continue
				}
				fn, ok := u.info.Defs[fd.Name].(*types.Func)
				if !ok {
					continue
				}
				key := funcKey(fn)
				_, line := mod.rel(fd.Pos())
				g.decls[key] = funcDecl{display: display(fn), file: path, line: line, inTest: inTest}
				if recvName(fn) != "" {
					g.methodsByName[fn.Name()] = append(g.methodsByName[fn.Name()], key)
				}
				if isRoot(fn, fd, u, inTest) {
					g.roots[key] = true
				}
				if g.edges[key] == nil {
					g.edges[key] = map[string]bool{}
				}
				for _, callee := range g.referenced(u, fd) {
					g.edges[key][callee] = true
				}
			}
		}
	}
	for name := range interfaceMethodNames(mod) {
		g.roots[interfaceMethodKey(name)] = true
	}
	return g
}

// interfaceMethodNames is the name set that makes a concrete method live by
// interface satisfaction: every method of every interface declared in the
// module's own packages or in a package one of them imports, plus the
// universe's error. reach() turns each name into reachability for every
// concrete method that carries it.
func interfaceMethodNames(mod *loaded) map[string]bool {
	names := map[string]bool{}
	seen := map[*types.Package]bool{}
	collect := func(pkg *types.Package) {
		if pkg == nil || seen[pkg] {
			return
		}
		seen[pkg] = true
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			tn, ok := scope.Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			iface, ok := tn.Type().Underlying().(*types.Interface)
			if !ok {
				continue
			}
			for i := 0; i < iface.NumMethods(); i++ {
				names[iface.Method(i).Name()] = true
			}
		}
	}
	if iface, ok := errorType.Underlying().(*types.Interface); ok {
		for i := 0; i < iface.NumMethods(); i++ {
			names[iface.Method(i).Name()] = true
		}
	}
	for _, u := range mod.units {
		collect(u.pkg)
		for _, imported := range u.pkg.Imports() {
			collect(imported)
		}
	}
	return names
}

// referenced returns the identity of every function object referenced
// anywhere under n — calls, method values, function values. An interface
// method reference expands to every concrete method of that name.
func (g *callGraph) referenced(u *unit, n ast.Node) []string {
	var out []string
	ast.Inspect(n, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		fn, ok := u.info.Uses[ident].(*types.Func)
		if !ok {
			return true
		}
		if isInterfaceMethod(fn) {
			out = append(out, interfaceMethodKey(fn.Name()))
			return true
		}
		out = append(out, funcKey(fn))
		return true
	})
	return out
}

// interfaceMethodKey is the pseudo-identity an interface method reference
// records, so a concrete method declared after the reference was seen is
// still reached — reach() expands these keys once the whole module is in.
func interfaceMethodKey(name string) string { return "interface-method:" + name }

// reach returns the set of function identities reachable from the roots.
func (g *callGraph) reach() map[string]bool {
	seen := map[string]bool{}
	var queue []string
	push := func(key string) {
		if strings.HasPrefix(key, "interface-method:") {
			for _, impl := range g.methodsByName[strings.TrimPrefix(key, "interface-method:")] {
				if !seen[impl] {
					seen[impl] = true
					queue = append(queue, impl)
				}
			}
			return
		}
		if !seen[key] {
			seen[key] = true
			queue = append(queue, key)
		}
	}
	for key := range g.roots {
		push(key)
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for callee := range g.edges[key] {
			push(callee)
		}
	}
	return seen
}

// candidates returns the declarations a finding may name, sorted for
// determinism: everything declared outside a _test.go file.
func (g *callGraph) candidates() []string {
	var keys []string
	for key, d := range g.decls {
		if !d.inTest {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// isRoot applies the documented root rule to one declaration.
func isRoot(fn *types.Func, fd *ast.FuncDecl, u *unit, inTest bool) bool {
	if fn.Name() == "init" && fd.Recv == nil {
		return true
	}
	if u.isMain && fn.Name() == "main" && fd.Recv == nil {
		return true
	}
	if inTest && isTestRootName(fn.Name()) {
		return true
	}
	if !inTest && fn.Exported() && importablePkg(u) {
		return true
	}
	return false
}

// importablePkg reports whether another module could import u's package,
// which is what makes its exported API a contract the gate must treat as
// live. A main package is not importable, an external test package is not
// code, and a path with an `internal` element is importable only from
// inside this module — where the call graph can see every caller.
func importablePkg(u *unit) bool {
	if u.isMain || u.isXTest || u.pkg == nil {
		return false
	}
	for _, elem := range strings.Split(u.pkg.Path(), "/") {
		if elem == "internal" {
			return false
		}
	}
	return true
}

// isTestRootName reports the four function-name shapes `go test` calls.
func isTestRootName(name string) bool {
	for _, prefix := range []string{"Test", "Benchmark", "Example", "Fuzz"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// isInterfaceMethod reports whether fn is a method declared on an interface
// rather than on a concrete type.
func isInterfaceMethod(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	_, isIface := sig.Recv().Type().Underlying().(*types.Interface)
	return isIface
}

// funcKey is a function's stable identity across type-check units:
// import path, receiver base type (for methods), and name.
func funcKey(fn *types.Func) string {
	pkg := ""
	if fn.Pkg() != nil {
		pkg = fn.Pkg().Path()
	}
	return pkg + "." + recvName(fn) + "." + fn.Name()
}

// recvName is the receiver's base type name for a method, "" for a
// function.
func recvName(fn *types.Func) string {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return ""
	}
	t := sig.Recv().Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

// display is how a finding names a function: pkg.Func or (pkg.Type).Method.
func display(fn *types.Func) string {
	pkg := ""
	if fn.Pkg() != nil {
		pkg = fn.Pkg().Name()
	}
	if recv := recvName(fn); recv != "" {
		return fmt.Sprintf("(%s.%s).%s", pkg, recv, fn.Name())
	}
	return pkg + "." + fn.Name()
}
