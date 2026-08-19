---
id: I084
title: "Gate classes: deferred-cleanup-errcheck, dead-code-callgraph, n-plus-one"
severity: med
status: fixed
affects: [gate]
blocked-by: [I082]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md` (Gate
pack — check definitions). ADR 0015. ADR 0001 (stdlib only) applies to
spine's own deps: prefer `go/ast`, `go/types`, and `golang.org/x/tools/go/
packages` only if the owner ratifies the dependency in the ticket record;
otherwise stdlib `go/build`+`go/types` loading.

## What to build

Three analysis check classes on the I082 skeleton: `deferred-cleanup-
errcheck` — deferred calls on cleanup-class functions (Close/Remove/Flush/
Sync and a configurable list) whose error return is discarded;
`dead-code-callgraph` — functions unreachable from any `main` or `_test`
root across the module; `n-plus-one` — call-in-loop patterns against a
configured list of client method names. Each with a positive-control
fixture pair, findings with file:line and `code = go@1/<check>`.

## Acceptance criteria

- [ ] All three pass on good fixtures and fail on seeded ones at the CLI
      seam.
- [ ] `dead-code-callgraph` treats test roots as live and does not flag
      exported API of a library module (documented rule + fixture).
- [ ] Dependency decision recorded in the ticket resolution (stdlib vs
      x/tools), consistent with ADR 0001.

## Blocked by

- I082 (gate skeleton + emitter).

## Resolution

Fixed 2026-08-18 on branch `local-harness-conventions` (commits 5747276,
f741c98). **Dependency decision (ADR 0001): stdlib only — no
`golang.org/x/tools`.** Loading is one `go list -deps -export -test -e -json
./...` (`internal/gate/load.go`) + `go/parser` + `go/types` with
`importer.ForCompiler(fset, "gc", …)` over the export files; `CgoFiles`
included with `FakeImportC`. Loader errors are split into "cannot load the
module under --dir" vs "--dir does not type-check" (both exit 2, one CLI-seam
fixture each; the cgo path itself is untested — needs a C toolchain).
`deferred-cleanup-errcheck`: `defer` directly calling Close/Remove/RemoveAll/
Flush/Sync (or names in env-only `SPINE_GATE_CLEANUP_FUNCS` — not a
`gate_pack_config` key, the spec's key set is closed) whose error result is
discarded. `dead-code-callgraph`: roots = main, init, Test*/Benchmark*/
Example*/Fuzz*, exported identifiers of non-main packages in a library module,
plus every method whose name is in the method set of any interface visible in
the module or its direct imports (universe `error` included) — "when in doubt
reachable"; residual limitation (interfaces from packages the module does not
directly import) documented in usage. `n-plus-one`: syntactic call-in-loop
against required `SPINE_GATE_N_PLUS_ONE_CLIENTS`. Positive-control pairs and
the AC2 root-rule fixture at the CLI seam. Observation: the class finds two
real deferred-cleanup violations in spine itself (internal/audit/audit.go,
internal/gate/binaryhygiene.go) — left for the repo's own gate enablement.
Review + scoped re-review clean.
