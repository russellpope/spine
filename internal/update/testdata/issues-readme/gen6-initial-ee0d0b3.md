# Issue / Bug Ledger — convention

Each issue is a markdown file in this directory (copy `_template.md`). Frontmatter fields:

- `id` — stable, unique (e.g. I001)
- `title` — short title
- `severity` — low | med | high | critical
- `status` — open | in-progress | fixed | wontfix
- `affects` — issue ids whose fix this one changes/overlaps (e.g. [I009])
- `blocked-by` — issue ids that must be resolved first (e.g. [I003])

Optional model-routing annotation fields (build tickets only; plain bug issues stay valid
without them — `spine audit routing` reports unannotated tickets, never judges them):

- `execution-mode` — inline | subagent-driven | ultracode; how the work runs
- `tier` — primary | routine | mechanical | fallback; the model tier the work is dispatched at
- `effort` — override of the tier's default effort; set only on deviation
- `risk-triggers` — zero or more of cross-task-integration, concurrency-subtle-state,
  security-surface, plan-flagged-ambiguity; any present forces primary-tier review
- `review-tier` — the tier review runs at; never below `tier`

See `WORKFLOW.md` `model_routing` for the tier→model mapping and the full dispatch contract.

## Rationalize pass (before remediation)

When the ledger has many items (e.g. dozens of deck slides or audit findings), build the
dependency graph from `affects:` / `blocked-by:` so overlapping or conflicting fixes are
**batched and ordered** — fixed together, never revisited separately. Output an ordered
remediation plan that respects every `blocked-by` and groups items sharing `affects` links.

## Wayfinding operations

`/wayfinder` and `/to-tickets` treat this ledger as the repo's issue tracker. Wayfinder
issues use the same files and ids, plus three optional frontmatter fields:

- `labels` — `[wayfinder:map]` on the map; `[wayfinder:research|prototype|grilling|task]` on tickets
- `parent` — a ticket's map id (e.g. [I020]); the map has none
- `assignee` — who has claimed the ticket; empty + `status: open` = unclaimed

Operations:

- **Map** — one issue labelled `wayfinder:map`; its body carries the map sections
  (Destination / Notes / Decisions so far / Not yet specified / Out of scope).
- **Tickets** — child issues (`parent:` = the map id); the body is the `## Question`.
- **Claim** — set `assignee:` and `status: in-progress` before any work.
- **Blocking** — `blocked-by:` frontmatter is the native blocking edge.
- **Frontier** — open, unclaimed tickets whose `blocked-by` ids are all `fixed`.
- **Resolve** — append the answer as `## Resolution`, set `status: fixed`, and index the
  gist in the map's Decisions so far. Out-of-scope tickets close as `wontfix` with a line
  in the map's Out of scope section.
