---
title: "estate maintenance pickup"
created: 2026-08-09
---

# Handoff — estate maintenance pickup (2026-08-09)

## Context

The cursor-writes effort (sole-writer rule, PRD
`docs/specs/2026-08-06-cursor-writes-design.md`, tickets I057–I061) is fully
shipped: verbs + tripwire, handoff auto-embed, canonical-form gate, gen 10
template, 17-repo fleet sweep, deepthought skills. See
`2026-08-06-sole-writer-shipped.md` for the build record. On 2026-08-09 the
estate was pushed, D4 reconciled (doctor fully clean), and I062 filed; this
handoff packages the remaining loose ends for the next session.

## State (verify before relying)

- spine main pushed through the ledger-dedupe/handoff commit after `b0c3451`
  (this file's commit); `spine doctor` exits 0, `spine audit stages` and
  `audit routing` exit 0. Template generation 10; live binary
  `/Users/ldh/bin/spine` embeds `482bc31`.
- Estate: 13 repos pushed to origin/main with their gen-10 migration commits
  (obsidian-ep-vault via a clean merge over 13 remote vault-backup commits).
- NOT pushed: **maipipe** (pre-push verify gate: HEAD `155e68b` unverified,
  gate demands `maipipe run full --wait`; 175-commit backlog behind it),
  **hbmview** (migration `c7b87c5` only on WIP branch `feat/header-redesign`,
  no upstream; main's 14 unpushed commits lack it), **moo-clone** (migration
  `6c41621` on `m4b-war-screens`; repo has no remote), **praxis** (migration
  `c9020be` already on the pushed `authz-followups-2026-08` branch; main's 5
  unpushed commits are unrelated).
- Ledger: I062 is the canonical same-date-tiebreak ticket (I031 superseded →
  wontfix). Statuses observed stale: I053–I056 still `open` though
  mutation-battery shipped 2026-08-06; three old tickets use nonstandard
  `status: closed`.

## Next steps

1. **I062 build** — frontier ticket, unblocked: same-date newest-handoff
   tiebreak (routine tier, primary review; candidates incl. I031's
   effort-matched preference are in the ticket).
2. **Ratify or loosen the complete-snapshot gate** — the final-review
   correction made `audit stages` compare the full snapshot, so any mid-effort
   cursor write blocks the audit until a fresh `spine handoff new`. Flagged to
   the owner 2026-08-07, not yet explicitly ratified; if the rhythm chafes
   mid-effort, it is a one-ticket loosening.
3. **maipipe push** — run `maipipe run full --wait` (owner judges pipeline
   cost), then `git push origin main`; largest unpublished backlog in the
   estate (175).
4. **Ledger hygiene sweep** — flip I053–I056 to `fixed` with Resolution
   blocks (evidence: mutation-battery shipped handoff), normalize the three
   `status: closed` tickets; one mechanical commit.
5. **claude.routine remap decision** — WORKFLOW.md still maps
   `claude.routine: claude-sonnet-5`; owner banned sonnet-5 (substitute
   claude-opus-5 @ low). Inherited default, refreshed by every sweep — if the
   ban is permanent it needs an owner override edit in the mirror (and
   ideally the estate default changed in spine).
6. Small: gitignore `PICKUP.md`; hbmview/moo-clone migrations land whenever
   those WIP branches merge (or cherry-pick if urgent).

## Gotchas

- Sole-writer rule is live estate-wide: never hand-edit a cursor block —
  `spine cursor start/tick/here/set` only; `spine handoff new` embeds the
  snapshot automatically. A hand edit fails `audit stages` (canonical-form
  gate).
- After any cursor write, `audit stages` blocks until a fresh handoff
  snapshot exists (see next-step 2) — expected, not a regression.
- The auto-mode permission classifier currently denies
  `cmux close-workspace` and compound rebase+push chains — state the exact
  action and let the owner run it or allowlist it (see project memory).
- maipipe's pre-push hook emits a JSON verdict (`stale_run`) — it is the
  repo's own verify gate, not a git failure; do not bypass.
- Estate survey scripts live in the session scratchpad pattern
  (bash, not fish — fish chokes on POSIX `for/do`; macOS bash 3.2 lacks
  `declare -A`).

<!-- spine:cursor -->
effort: cursor-writes
prd: docs/specs/2026-08-06-cursor-writes-design.md
tickets: I057-I061
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
