# Architecture Decision Records — convention

One decision per file: `NNNN-short-slug.md` (numbering starts at 0001; `spine adr new` picks
the next number). Front-matter fields: `id`, `title`, `status`, `date`, optional `supersedes`.

Statuses: `Accepted` (default) or `Superseded by NNNN`.

The *decision* in an accepted ADR is immutable. Reversing or amending what was decided means a
NEW ADR that supersedes the old one (`spine adr new --supersedes NNNN "..."`); the superseded
ADR's only decision-level edit is the status flip that supersede performs. Edits that do not
change the decision — corrected citations and cross-references, typo fixes, a dated note
pointing at the issue or ADR that refined a consequence — are permitted in place (practised by
ADR 0013 and ADR 0016's I091 note). When in doubt whether an edit changes the decision, write a
new ADR. If resolving an issue changes the architecture, record the change as an ADR and link it
from the issue.
