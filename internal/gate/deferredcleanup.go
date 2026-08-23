package gate

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

// CleanupFuncsVar is the environment variable that extends the cleanup-class
// name set. It is deliberately NOT a gate_pack_config key — the spec's key
// set is closed, so `spine update` never renders it — and is documented in
// the gate usage as an env-only tuning knob.
const CleanupFuncsVar = "SPINE_GATE_CLEANUP_FUNCS"

// DefaultCleanupFuncs is the cleanup-class name set: a deferred call to a
// function or method by one of these names, whose signature returns an
// error, is a call whose failure the program has decided not to hear about.
var DefaultCleanupFuncs = []string{"Close", "Remove", "RemoveAll", "Flush", "Sync"}

// checkDeferredCleanup reports deferred calls to cleanup-class functions
// whose error return is discarded — `defer f.Close()` — as opposed to the
// two shapes that do handle it: a deferred func literal that inspects the
// error (`defer func() { if err := f.Close(); err != nil { … } }()`) and a
// deferred call to a wrapper whose own signature returns nothing.
//
// The name alone is not enough: go/types confirms the callee's signature
// really has an error result, so a Close() that returns nothing is not a
// finding. That is why this class type-checks the module, and why a target
// repo that does not compile is misconfiguration.
//
// The default name set is Close, Remove, RemoveAll, Flush, Sync. Extra
// names come from SPINE_GATE_CLEANUP_FUNCS, comma-separated; unset means
// the defaults alone.
func checkDeferredCleanup(dir string, cfg Config) ([]Finding, error) {
	names := map[string]bool{}
	for _, n := range DefaultCleanupFuncs {
		names[n] = true
	}
	for _, n := range splitList(cfg.env(CleanupFuncsVar)) {
		names[n] = true
	}
	mod, err := loadModule(dir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, u := range mod.units {
		for _, file := range u.files {
			ast.Inspect(file, func(n ast.Node) bool {
				def, ok := n.(*ast.DeferStmt)
				if !ok {
					return true
				}
				ident, printed := deferredCallee(def.Call)
				if ident == nil || !names[ident.Name] {
					return true
				}
				fn, ok := u.info.Uses[ident].(*types.Func)
				if !ok || !returnsError(fn) {
					return true
				}
				file, line := mod.rel(def.Pos())
				findings = append(findings, Finding{
					Severity: SeverityError,
					Message:  fmt.Sprintf("deferred cleanup call discards its error: defer %s() returns an error no one reads; defer a func literal that inspects it instead", printed),
					File:     file,
					Line:     line,
					Code:     cfg.Code("deferred-cleanup-errcheck"),
				})
				return true
			})
		}
	}
	return findings, nil
}

// deferredCallee returns the identifier naming the deferred call's callee
// and the call's printed form for the message. A deferred func literal has
// no callee identifier and so is never a finding — it is one of the two
// shapes that handle the error.
func deferredCallee(call *ast.CallExpr) (*ast.Ident, string) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun, fun.Name
	case *ast.SelectorExpr:
		return fun.Sel, exprString(fun)
	}
	return nil, ""
}

// exprString prints a selector chain (f.Close, s.file.Close) for a message,
// falling back to the selected name when the receiver is not a plain chain.
func exprString(sel *ast.SelectorExpr) string {
	var parts []string
	for {
		parts = append([]string{sel.Sel.Name}, parts...)
		switch x := sel.X.(type) {
		case *ast.Ident:
			return strings.Join(append([]string{x.Name}, parts...), ".")
		case *ast.SelectorExpr:
			sel = x
		default:
			return strings.Join(parts, ".")
		}
	}
}

// returnsError reports whether fn's signature has an error result.
func returnsError(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		if types.Identical(results.At(i).Type(), errorType) {
			return true
		}
	}
	return false
}

// errorType is the universe's error interface, the type a cleanup-class
// signature has to return for its discarded result to matter.
var errorType = types.Universe.Lookup("error").Type()
