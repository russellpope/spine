# Issue / Bug Ledger — convention

Each issue is a markdown file in this directory (copy `_template.md`). Frontmatter fields:

- `id` — stable, unique (e.g. I001)
- `title` — short title
- `severity` — low | med | high | critical
- `status` — open | in-progress | fixed | wontfix
- `affects` — issue ids whose fix this one changes/overlaps (e.g. [I009])
- `blocked-by` — issue ids that must be resolved first (e.g. [I003])

## Rationalize pass (before remediation)

When the ledger has many items (e.g. dozens of deck slides or audit findings), build the
dependency graph from `affects:` / `blocked-by:` so overlapping or conflicting fixes are
**batched and ordered** — fixed together, never revisited separately. Output an ordered
remediation plan that respects every `blocked-by` and groups items sharing `affects` links.
