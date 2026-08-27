---
title: "Doctor hygiene batch: known-stock backfill + batch-key convention"
tickets: I065, I106
created: 2026-08-27
status: draft
---

# Doctor hygiene batch (I065 + I106) — design

## Problem Statement

`spine doctor` and `spine update` exit 1 on repos that are actually healthy.

The updater compares a machine-owned file against only two renders: gen0 and
current. Any line an *intermediate* generation emitted — retired stock text —
reads as an owner customization, so the file is skipped every sweep, exits 1,
and stays a generation behind indefinitely. This is live on spine today
(`docs/issues/README.md` still carries the pre-`superseded` status bullet) and
across the estate (11 repos stranded on the pre-I046 tier bullet since the
2026-08-10 sweep). Spine was hand-reconciled once on 2026-08-09 and has
drifted again — one-time reconciles do not hold. A permanently red doctor
teaches operators to ignore it, which destroys its value as a signal.

Separately, maikanban and the claude-team skill now write four ticket
frontmatter keys (`batch:`, `workspace:`, `commits:`, `review:`) that the
ledger convention does not mention. The schema belongs to spine (maikanban ADR
0002/0006: no keys without a spine-side contract), so the convention is
currently violated by its own tooling, and nothing watches the keys' invariants
(a `workspace:` path that should have been cleared, a malformed `batch:` id).

The two tickets collide: I106 changes the very file whose drift is I065, and
I106's requested "warn (not block)" findings would keep doctor exiting 1 — the
exact condition I065 retires.

## Solution

Land I065 first: teach the updater every retired line the issues-README has
ever emitted, sourced verbatim from template history, so `spine update`
refreshes the file cleanly everywhere and doctor's D4 warn disappears. Then
land I106 through the fixed machinery — documenting the four keys is a new
template generation, and its clean propagation is the live proof that the
known-stock cure works. Doctor gains its first per-ticket checks, warning only
on genuine anomalies, so post-batch a healthy repo is exit 0 and a red doctor
always means "act".

## User Stories

1. As the estate operator, I want `spine update` to refresh a README whose only
   divergence is retired stock text, so that a sweep exits 0 without hand
   reconciliation.
2. As the estate operator, I want `spine doctor` to exit 0 on a healthy repo,
   so that a non-zero exit is always worth stopping for.
3. As the estate operator, I want the D4 warn on spine's own issues-README
   gone, so that the standing "known red, ignore it" caveat leaves handoffs.
4. As a session resuming work, I want doctor's exit code to be trustworthy, so
   that I never have to carry a "doctor is red but that's expected" exception.
5. As the template maintainer, I want a stated rule that every
   content-changing generation appends its predecessors' dropped lines to the
   known-stock set, so that this drift class cannot recur on any managed file.
6. As the estate operator, I want the known-stock lines captured verbatim from
   template git history rather than from ticket prose, so that the fix matches
   what repos actually contain (the I065 ticket itself mis-named the drifted
   line).
7. As an owner with genuine local edits in a machine-owned file, I want those
   edits still reported as unrecognized and the file still skipped, so that
   the backfill never widens into silently dropping my customizations.
8. As a maikanban board, I want the `batch:` key documented in the ledger
   convention with its writer and lifecycle, so that writing it no longer
   violates the no-undocumented-keys rule.
9. As a claude-team lead, I want `workspace:` and `commits:` documented the
   same way, so that the close protocol has a spine-side contract to cite.
10. As a human reviewer, I want the `review:` key documented (absent ≡
    pending, human verdict only), so that a board can render review state
    without inventing semantics.
11. As a repo adopting the workflow, I want `spine init`'s scaffold to carry
    the four keys' documentation, so that new repos start on the current
    convention.
12. As the estate operator, I want doctor to warn on a `workspace:` path that
    does not exist, so that a crashed or abandoned worktree close surfaces
    instead of rotting.
13. As the estate operator, I want doctor to warn when a closed ticket still
    carries `workspace:`, so that an incomplete close protocol is visible.
14. As the estate operator, I want doctor to warn on a malformed `batch:` id
    on a live ticket, so that a writer violating the contract is caught near
    the write.
15. As the owner of long-closed tickets, I want `batch:` shape checked only on
    open/in-progress tickets, so that a historical malformation cannot warn
    eternally on tickets nobody will reopen.
16. As a script consuming doctor, I want `warn ⇒ exit 1` semantics unchanged,
    so that existing gates keep working and info-level notes stay exit-neutral.
17. As the estate operator, I want the 11-repo sweep recorded as a deploy-stage
    checklist with per-repo exit codes, so that the rollout is verified rather
    than assumed.
18. As a future effort lead, I want a batch glossary entry in the domain
    model, so that "batch" means one thing across maikanban, claude-team, and
    spine.

## Implementation Decisions

- **Sequencing is I065 then I106**, as two landings inside one effort. I106's
  template bump propagating cleanly through the backfilled updater is the
  batch's end-to-end proof; a merged landing would destroy that evidence.
- **I065 is a known-stock backfill only.** The retired issues-README lines —
  every line any prior generation emitted that the current one does not,
  identified by auditing the template's full history (six commits) and copied
  verbatim from the historical renders — join the updater's superseded-lines
  set. No change to unrecognized-edit detection semantics, no new grammar.
  The two known members are the pre-`superseded` status bullet and the
  pre-I046 tier bullet; history is authoritative over both ticket text and
  this spec.
- **The generation-bump rule becomes explicit**: any bump that changes emitted
  content appends the dropped lines in the same change. I106's bump complies
  in this batch (if purely additive, the appended set is empty and the test
  proves it).
- **I106 adds the four keys to the ledger convention** (template and scaffold)
  as a writer/lifecycle table: `batch` — board, at claim, never cleared;
  `workspace` — team lead, while a worktree exists, absolute path, cleared at
  close; `commits` — team lead, in the close commit, list of SHAs; `review` —
  board, human verdict only, `pending | approved | changes-requested`, absent
  ≡ pending.
- **"Accept the keys silently" is already true** — no component validates
  unknown ticket frontmatter keys (verified against doctor, stage derivation,
  and the routing audit) — so I106's tolerance work is purely the new checks.
- **Doctor gains its first per-ticket checks**, under a new check id in the
  D-series, all severity `warn`: (a) `workspace:` path does not exist — all
  tickets; (b) `workspace:` present on a `fixed`/`wontfix`/`superseded`
  ticket — all tickets, the presence itself is the finding; (c) `batch:`
  value not matching `<YYYY-MM-DD>-<4 alnum>#<n>` — open/in-progress tickets
  only.
- **Doctor exit semantics are unchanged**: warn and error drive exit 1, info
  is exit-neutral. Decided at the grill: the new warns fire only on genuine
  anomalies, so the healthy-repo-exits-0 goal is met by fixing the findings,
  not by weakening the signal. No third severity, no ADR needed.
- **The hand-authored ADR-README info note remains** and remains exit-neutral;
  post-I065 spine's doctor is exit 0 with that note still printed.
- **Deploy stage includes the estate sweep**: rebuild the installed binary,
  then run `spine update` per estate repo named in I065, recording each exit
  code. The build itself is spine-only.
- The **batch** glossary entry is already captured in the domain model
  (CONTEXT.md, "Ticket batch", 2026-08-27).

## Testing Decisions

External behavior only: assert on update reports, doctor findings, and CLI
exit codes — never on the contents of internal tables.

- **Updater seam (existing)**: the generation-migration test pattern
  (gen5→6/gen9→10 style). New tests: a README frozen at each retired
  generation refreshes cleanly with zero unrecognized lines — these positive
  fixtures are what prove the backfill load-bearing (they fail if it is
  removed); the negative control — a genuine local edit in the same file
  still reads as unrecognized and skips the file — proves it is not
  over-broad. The two test groups together carry the claim; neither alone
  does. (Wording corrected at Task 1 review, which caught the original
  sentence attributing both halves to the negative control.)
- **Doctor seam (existing)**: the doctor package's findings-table tests. New
  tests per check: each warn condition fires with the expected severity and
  message; a healthy fixture yields no per-ticket findings; a malformed
  `batch:` on a fixed ticket does NOT fire (negative control for the
  status scoping); unknown keys alone produce no finding.
- **Dogfood seam (existing)**: spine's own repo — after I065, `spine update
  --dir .` and `spine doctor` both exit 0 (the latter with the info note
  still present). This is the live reproduction converted into a regression
  guard.
- **I106 propagation proof**: after the bump, the migration test for the new
  generation passes through the same updater seam — the machinery proof the
  sequencing decision exists to produce.

## Out of Scope

- Comma-list cursor tickets grammar — filed as I114, separate effort.
- A `spine batch` helper (I106 item 3: not until `grep -l` proves fragile).
- Any doctor severity/exit-code rework or third severity level.
- Fixing estate repos' *other* machine-owned files; this batch's sweep only
  runs the standard update per repo.
- The openweights programme (I111, I112) — parked.

## Further Notes

- This effort's cursor carries `tickets: I065,I106`, which the current grammar
  cannot resolve — the issues/implement evidence rules run not-judged with a
  visible note. Accepted at the grill; I114 is the fix.
- The I065 ticket names the tier bullet as the drifted line; on spine the
  drifted line is the status bullet. Both are real in different repos; the
  full-history backfill covers both without adjudicating the ticket text.
