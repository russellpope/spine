---
id: I093
title: "Gate pack dogfood follow-ups: sortFindings ordering untested, type-check error names downstream import, unconfigured classes render as exit-2 stages, update --force is all-or-nothing"
severity: low
status: fixed
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

### Done 2026-08-19 — items 1 and 2

1. `internal/gate/gate_test.go` `TestSortFindingsOrder`: four findings tying
   at each level, asserts `file asc, line asc, message asc`. Negative
   control = the mutation probe itself: `MAIPIPE_RESULTS=… spine gate go
   mutate --dir .` → `gate-sort-findings-reversed SURVIVED` with the test
   untracked (the battery copies tracked files only), `KILLED` once staged.
   Remaining survivor is the report-only `mutate-workcopy-leak-on-copy-error`.
2. `internal/gate/load.go`: `loadModule` now pre-passes every main-module
   package for `go list -e`'s `Error`, then `DepsErrors`, before any
   type-check, and reports the first diagnostic line (the `# importpath`
   header stripped). New refusal case in
   `TestGateTypeCheckedClassesRejectNonCompilingRepo` ("importer sorts
   first": `cmd` imports broken `internal/inv`) — red before the fix with
   the exact downstream text (`could not import … (no export data …)`),
   green after; every case now also asserts `no export data` is absent.
   `make test` 18/18 ok, `go vet` clean.

Open: 3 (unconfigured-class advisory), 4 (`--force` scoping), 5 (D11 value
hash) — owner call.

## Owner ruling 2026-08-30

- Item 3 is filed as I123 with the pre-write update-advisory design.
- Item 4 is filed as I124 with additive repeatable `--force-file` authority;
  global `--force` remains compatible.
- Item 5 remains shape-only. No I125 is filed.

## Resolution

Fixed 2026-08-30. Items 1 and 2 landed in
`ab204e5` (`fix(gate): I093.1/.2 — pin sortFindings order (kills mutation
probe); type-check refusal names the first compile error, not the importer
symptom`), with the completed-test and diagnostic evidence recorded above.
The owner accepted the remaining disposition: I123 records item 3, I124
records item 4, and item 5 remains shape-only with no I125. I123 and I124
remain open implementation tickets; this resolution does not claim their work
has shipped.
