# Architecture Decision Records — convention

One decision per file: `NNNN-short-slug.md` (numbering starts at 0001; `spine adr new` picks
the next number). Front-matter fields: `id`, `title`, `status`, `date`, optional `supersedes`.

Statuses: `Accepted` (default) or `Superseded by NNNN`.

ADRs are immutable once accepted. Reversing or amending a decision means a NEW ADR that
supersedes the old one (`spine adr new "..." --supersedes NNNN`) — the only permitted edit to
an existing ADR is the status flip that supersede performs. If resolving an issue changes the
architecture, record the change as an ADR and link it from the issue.
