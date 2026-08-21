package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// nPlusOneClientsKey is the gate_pack_config key holding the client call
// names; it reaches the stage as SPINE_GATE_N_PLUS_ONE_CLIENTS.
const nPlusOneClientsKey = "n_plus_one_clients"

// checkNPlusOne reports the call-in-loop pattern: a call to one of the
// configured client names (a method like db.Query or a package function
// like client.Fetch) lexically inside a for or range loop body, at any
// nesting depth. That is the shape of an N+1 — one round trip per
// iteration where one batched call would do.
//
// The class is syntactic on purpose: the name list is what makes a call a
// client call in a given repo, so no type information adds anything. The
// list is required configuration — with no names there is nothing to check,
// which is misconfiguration, not a clean pass.
//
// _test.go files are not read: a loop of client calls in a test is a test
// doing setup, not a production round-trip storm.
func checkNPlusOne(dir string, cfg Config) ([]Finding, error) {
	raw, _ := cfg.Get(nPlusOneClientsKey)
	names := map[string]bool{}
	for _, n := range splitList(raw) {
		names[n] = true
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s is unset or empty: list the client call names as comma-separated method or function names (e.g. Get,Query,Fetch)", EnvVar(nPlusOneClientsKey))
	}
	var findings []Finding
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel := relSlash(dir, path)
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return fmt.Errorf("parsing %s: %w", rel, perr)
		}
		for _, call := range callsInLoops(file, names) {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("call in loop: %s is a configured client call (%s) inside a loop body — one round trip per iteration", callName(call), EnvVar(nPlusOneClientsKey)),
				File:     rel,
				Line:     fset.Position(call.Pos()).Line,
				Code:     cfg.Code("n-plus-one"),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return findings, nil
}

// callsInLoops returns the calls to one of names that appear inside a for
// or range loop body, at any depth. Containment is lexical, so a call in a
// func literal declared inside a loop counts too: that literal runs per
// iteration.
func callsInLoops(file *ast.File, names map[string]bool) []*ast.CallExpr {
	type span struct{ from, to token.Pos }
	var loops []span
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ForStmt:
			loops = append(loops, span{node.Body.Pos(), node.Body.End()})
		case *ast.RangeStmt:
			loops = append(loops, span{node.Body.Pos(), node.Body.End()})
		}
		return true
	})
	var out []*ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !names[callName(call)] {
			return true
		}
		for _, l := range loops {
			if call.Pos() >= l.from && call.Pos() < l.to {
				out = append(out, call)
				break
			}
		}
		return true
	})
	return out
}

// callName is the name a call is matched against: the selected name for a
// method or qualified call (db.Query -> Query), the identifier for a plain
// call (fetch() -> fetch), and "" for anything else.
func callName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.Ident:
		return fun.Name
	}
	return ""
}
