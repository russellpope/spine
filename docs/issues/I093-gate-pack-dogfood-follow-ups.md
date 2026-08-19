---
id: I093
title: "Gate pack dogfood follow-ups: sortFindings ordering untested, type-check error names downstream import, unconfigured classes render as exit-2 stages, update --force is all-or-nothing"
severity: low
status: open
affects: [I082, I084, I085]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Minor findings from the 2026-08-19 dogfood (I088–I092), none blocking:

1. **`sortFindings` file-order has no test.** spine's own mutation spec
   probe `gate-sort-findings-reversed` (reverse the file comparator in
   `internal/gate/gate.go`) SURVIVED: `TestGateResultsDeterministic`
   asserts byte-equality across runs, not the documented order. Add a
   two-file fixture asserting `file asc, line asc, message asc`.
2. **Type-check refusal message names the downstream symptom.** On the
   bake-off `claude` arm (does not compile), `deferred-cleanup-errcheck` and
   `dead-code-callgraph` say `could not import vsphere-inventory/cmd (no
   export data …)` instead of the first package compile error
   (`internal/inventory/inventory.go:215: ext.CanonicalName undefined`).
   `go list -e -json` carries per-package `Error`/`DepsErrors` — surface the
   first one.
3. **Unconfigured config-driven classes render as stages that exit 2.**
   A fresh `gate_pack: go@1` with empty `gate_pack_config` renders
   `fixture-manifest`, `gitignore-control`, `n-plus-one`,
   `test-enum-vs-spec` stages that fail the lane as misconfiguration until
   the owner sets config or lists them in `gate_pack_disabled`. Spec/ADR
   0015 say "the rendered region omits disabled classes" and nothing about
   unconfigured ones; spine's own first render needed three disables plus
   one config (I089). Options: `spine update` advisory line naming the
   classes that need config; or omit-with-comment; or doctor info. Owner
   decides — filed so the trap is on record.
4. **`spine update --force` is all-or-nothing.** Reverting a tampered
   region (D10 round-trip) also dropped the pre-existing local edit in
   `docs/issues/README.md` (D4, I065); had to restore by hand. A per-file
   `--force <path>` or "force only files named" would make the D10 remedy
   safe in a repo that carries other local edits.

5. **D11 is shape-evident, not value-evident.** Editing `gate: pass` →
   `gate: fail` in a checkpoint's facts region re-renders canonically and
   doctor stays silent; only malformed/non-canonical edits fire (spec
   scope). If the facts region should be tamper-evident, a content hash in
   the frontmatter is the smallest change — owner call.

## Fix

Five small tickets' worth; batch when convenient. (1) and (2) are code +
test; (3) and (4) need an owner call first.
