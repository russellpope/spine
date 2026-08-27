---
id: I106
title: "Document the `batch:` / `workspace:` / `commits:` / `review:` ticket keys and tolerate them in `spine doctor`"
severity: low
status: in-progress
batch: 2026-08-27-dhyg#2
workspace: /Users/ldh/worktrees/spine-2026-08-27-dhyg
affects: [I094]
blocked-by: [I065]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## Problem

Filed 2026-08-21 from maikanban's I035/I034 design (maikanban ADR 0010,
`docs/specs/2026-08-21-i034-i035-review-and-batch-design.md`). maikanban and the claude-team
skill now write four frontmatter keys the ledger convention does not mention:

| key | writer | shape |
|---|---|---|
| `batch` | board, at batch claim | `<YYYY-MM-DD>-<4 alnum>#<n>` — never cleared |
| `workspace` | team lead, while a worktree exists | absolute path — cleared at close |
| `commits` | team lead, in the close commit | list of SHAs |
| `review` | board, human verdict only | `pending` \| `approved` \| `changes-requested`; absent ≡ pending |

The schema belongs to spine; maikanban's ADR 0002/0006 "invents no keys" rule was narrowed to
"no keys without a spine-side contract", which is this ticket.

## What to do

1. Add the four keys to the ledger convention (`docs/issues/README.md` template and `spine
   init`'s scaffold), with writer and lifecycle as above.
2. `spine doctor`: accept the keys silently; warn (not block) on a `workspace:` path that does
   not exist and on a malformed `batch:` value.
3. Nothing else — no `spine batch` helper until `grep -l 'batch: <id>'` proves fragile.
