---
title: "i062-tiebreak-codex"
created: 2026-08-09
---

# Handoff — i062-tiebreak-codex (2026-08-09)

## Context

You are the codex team lead for the **I062 build** in
`/Users/ldh/Projects/github.com/spine`. Read first, in order:

1. `docs/issues/I062-same-date-handoff-tiebreak.md` — the ticket (routine tier,
   `review-tier: primary`, risk-trigger cross-task-integration). It carries the
   full problem statement, three candidate mechanisms, and acceptance criteria.
2. `docs/specs/2026-08-06-cursor-writes-design.md` — the PRD behind the
   sole-writer/handoff machinery I062 extends (complete-snapshot gate reads
   `handoff.Latest`).
3. `AGENTS.md` + `WORKFLOW.md` — dispatch contract, escalation record grammar,
   stage list. Model routing: tiers only, resolve ids via `spine model`.

## State (verify before relying)

- main at `07f9c5d`, 4 commits ahead of origin/main (push is the owner's call;
  do not push). Working tree clean except untracked `.DS_Store` and
  `docs/research/2026-08-05-routing-yield-feasibility.md` — leave both alone.
- `spine doctor`, `spine audit stages`, `spine audit routing` all exit 0 at
  `07f9c5d`. `go test ./...` green.
- The cursor block below still shows the shipped cursor-writes effort — that is
  correct at handoff time. Starting the I062 effort is YOUR first move (see
  next steps); the block in this file is spine-owned, never edit it.
- Relevant code: `internal/handoff/` (`Latest`, date parse + lexicographic
  tiebreak), consumers in `internal/stages/` (complete-snapshot gate) and
  `internal/doctor/`. `docs/issues/I031-same-day-handoff-filename-tiebreak.md`
  is the superseded predecessor (wontfix, pointer to I062) — its candidate
  discussion is background, I062 is authoritative.

## Next steps

1. `spine cursor start` for the i062 effort (sole-writer rule: only
   `spine cursor start/tick/here/set` may touch cursor state). Expect
   `spine audit stages` to block right after any cursor write until a fresh
   `spine handoff new` snapshot exists — expected behavior, not a bug; it
   resolves when you cut your shipped handoff at the end.
2. Build I062 per the ticket: pick ONE candidate mechanism (frontmatter ordinal
   written by `spine handoff new`, mtime-with-degradation, or
   live-effort-match preference), record the choice + rationale in the ticket's
   Resolution when done.
3. Acceptance: the ticket's four criteria verbatim, including the fresh-clone
   determinism requirement (git does not preserve mtimes — an mtime tiebreak
   must degrade to current lexicographic order, never nondeterminism) and
   different-date ordering untouched.
4. Review at primary tier (ticket `review-tier: primary`); routing audit reads
   dispatch records — every dispatch names an explicit model and carries the
   ticket-id token `I062`.
5. Ship: commit with explicit paths only (never `git add -A`), run
   `spine doctor` + both audits + `go test ./...`, `spine handoff new` for the
   shipped handoff, leave the push to the owner.

## Task breakdown hints

- This is plausibly a one-implementer + one-reviewer team: single ticket,
  small blast radius, but the risk-trigger forces primary review.
- Worker tiers: implementer at codex routine (`spine model codex routine` →
  currently gpt-5.6-terra), reviewer at codex primary (`spine model codex
  primary` → gpt-5.6-sol). Any deviation needs an ESCALATION/FALLBACK line in
  `.superpowers/sdd/progress.md` — exact grammar in WORKFLOW.md (unspaced
  `->`, `reason:` required).
- If you choose the frontmatter-ordinal mechanism, check what `spine handoff
  new` already writes (`internal/handoff`, `templates/current/handoff.tmpl.md`)
  and whether existing handoffs without the field need a documented fallback —
  backward compatibility with the 30+ existing handoff files is part of
  acceptance criterion 2.

## Gotchas

- NEVER hand-edit any spine cursor comment block, in any file — the canonical
  form gate fails the audit on hand edits. (Not even to quote one: the literal
  opening marker anywhere in a handoff body confuses the block parser.)
- Default shell on this machine is fish, but your panes run what you spawn;
  if you write multi-repo scripts, use bash files (macOS bash 3.2: no
  `declare -A`) — POSIX `for/do` typed into fish breaks.
- The auto-mode permission classifier denies `cmux close-workspace` and
  compound rebase+push chains — surface the exact command to the owner
  instead of retrying.
- Do not touch `PICKUP.md` (gitignored scratch) or the untracked research doc.

## Owner-facing note

Spine main is 4 ahead of origin (ledger hygiene, PICKUP.md gitignore,
claude.routine remap + I063). Pushing spine, and ratifying I063 (estate
default claude.routine remap), remain owner calls.

<!-- spine:cursor -->
effort: cursor-writes
prd: docs/specs/2026-08-06-cursor-writes-design.md
tickets: I057-I061
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
