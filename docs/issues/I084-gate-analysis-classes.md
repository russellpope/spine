---
id: I084
title: "Gate classes: deferred-cleanup-errcheck, dead-code-callgraph, n-plus-one"
severity: med
status: open
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
