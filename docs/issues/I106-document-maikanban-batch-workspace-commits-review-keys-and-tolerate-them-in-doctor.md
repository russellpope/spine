---
id: I106
title: "Document the `batch:` / `workspace:` / `commits:` / `review:` ticket keys and tolerate them in `spine doctor`"
severity: low
status: fixed
batch: 2026-08-27-dhyg#2
commits: [f3827dc, 0aff8d4]
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

## Resolution

Fixed 2026-08-27 (batch 2026-08-27-dhyg#2, commits f3827dc + 0aff8d4, spec
docs/specs/2026-08-27-doctor-hygiene-batch-design.md). The four keys are
documented in the ledger convention (template + `spine init` scaffold) with
writer and lifecycle; the change was purely additive, so per the I065
gen-bump rule no superseded entries were owed. `spine doctor` gained its
first per-ticket checks (D13, all severity warn, exit semantics unchanged):
nonexistent `workspace:` path (any status), `workspace:` present on a closed
ticket, and malformed `batch:` on open/in-progress tickets only — with
negative controls proving each scoping. "Warn (not block)" was adjudicated
at the grill as "severity warn, not error": doctor warns still set exit 1,
and the convention prose states exactly that (corrected in fix round 1).
`commits:`/`review:` values carry no checks, and no `spine batch` helper was
built, per item 3.
