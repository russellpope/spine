## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Repo invariants

- The git remote is `github`, NOT `origin` — push with `git push github main`.
- `gh` is not authenticated here; use plain git + the GitHub web UI.
- golangci-lint drifts between local and CI: CI is the gate of record — do NOT chase local-only lint failures.
- Gates before every commit: `make test` + golangci-lint for BOTH GOOS (linux+darwin; linux-only build tags exist).
- Ignore `.claude/scheduled_tasks.lock` if it appears in status.
