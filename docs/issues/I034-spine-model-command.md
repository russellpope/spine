---
id: I034
title: spine model command
severity: med
status: fixed
affects: [cmd, models]
blocked-by: [I033]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Design D12. A read-only `spine model <flavor> <tier>` subcommand: a thin printer over the resolver, matching the convention the routing audit already states — the boundary is the pure function, the CLI prints it.

Default output is the bare model id and nothing else, so a shell consumer can interpolate it into a spawn command with no parsing dependency. `--effort` prints the resolved effort instead; `--json` prints the whole entry. Flavor is a required positional argument and is never inferred from context — an inferred flavor is the same class of invisible resolution the estate is chasing elsewhere.

Emitting ready-made vendor CLI spawn arguments is explicitly out of scope: it would teach spine the flag syntax of every other tool.

## Acceptance criteria

- [ ] `spine model claude primary` prints exactly the id, one line, no decoration
- [ ] `--effort` prints the resolved effort; `--json` prints id and effort together
- [ ] Run inside a repo with an override, the command returns the override; run outside any repo, it returns embedded defaults
- [ ] Missing or unknown flavor/tier exits non-zero with a usage-grade message
- [ ] Output is stable enough to consume from shell without quoting surprises
- [ ] `go test ./...` green
