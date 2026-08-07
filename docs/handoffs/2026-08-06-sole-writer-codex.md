# Handoff — cursor-writes build (codex team), 2026-08-06

You are the codex team lead for `/Users/ldh/Projects/github.com/spine`. This
doc is authoritative. Grill → PRD → tickets are done; your team executes
implement → functional-test → review → verify → ship for tickets I057–I061.

## Context to read first (in order)

1. `AGENTS.md` — repo conventions (workflow profile `library-cli`, gates).
2. `docs/specs/2026-08-06-cursor-writes-design.md` — the PRD. All decisions
   are owner-ratified; do not relitigate them. The Testing Decisions and
   Further Notes sections are binding (zero new seams; sequencing).
3. `docs/issues/I057-*.md` … `I061-*.md` — the five tickets, frontmatter
   carries execution-mode / tier / risk-triggers / review-tier. Blocking
   edges: I057 and I058 are the frontier (independent, parallelizable);
   I059 needs I057; I060 needs I057+I058; I061 needs I060.
4. `CONTEXT.md` "Stage cursor" section — the glossary for every term the PRD
   uses (working home, committed snapshot, sole-writer rule, canonical form,
   write-time tripwire).
5. `WORKFLOW.md` — `model_routing` block: resolve worker models per ticket
   `tier:` via `spine model codex <tier>` (never hardcode ids).

## Current state (verified 2026-08-06)

- Branch `main` at `460c570`; origin is 3 commits behind (owner pushes, not
  you).
- **Uncommitted effort docs on main** (deliberate — commit them as your first
  act, one docs commit, explicit paths only): `CONTEXT.md` (modified),
  `docs/specs/2026-08-06-cursor-writes-design.md`, `docs/issues/I057-…I061-*.md`,
  and this handoff file. Precedent: `86cf362`. Do NOT stage `PICKUP.md`,
  `.DS_Store`, or `docs/research/2026-08-05-routing-yield-feasibility.md`
  (unrelated, stays untracked).
- Code state: `internal/cursor/cursor.go` already carries parse + canonical
  serialization (`StagesLine`); `internal/stages` has the derivation engine
  and audit; template generation is 9 (`internal/update` testdata
  `spine-gen9`). Nothing of this effort's code exists yet.
- `spine audit stages` currently blocks ONLY on the newest-handoff-stale
  check against the pre-existing `2026-08-06-mutation-battery-shipped.md`;
  this very file resolves that once committed. Per-stage derivation is clean
  (prd match 1/1, issues match 5/5).
- `spine doctor` has one pre-existing D4 warn (`docs/issues/README.md` local
  edit) — not yours, leave it.

## What's next (the build)

Work the frontier: I057 ∥ I058, then I059, I060, I061. Each ticket's
acceptance criteria are the contract; `go test ./...` green is table stakes
everywhere. Verify stage: `spine audit stages` and `spine audit routing`
exit 0, plus each ticket's live-verify evidence.

## Task breakdown hints

- I057 (verbs) is the big one — four verbs sharing parse→mutate→re-serialize→
  splice on the working home (`.superpowers/sdd/progress.md` head). The
  tripwire must emit the audit's OWN finding text: reuse the derivation
  engine, never a copy of its strings.
- I058 (embed) is read-side only; it must not depend on I057's writer.
- I060: the gen 9→10 migration must REPLACE the "MUST embed the verbatim
  output of `spine cursor`" WORKFLOW.md line via the supersededLines path —
  replaced, never duplicated. Prior art: gen4to5 test, hbmview fixture.
- I061 sweep: estate repos = every repo whose WORKFLOW.md carries
  `template_version`. Dry-run diff → `--write` → commit, per repo. Skills to
  rewrite: `/Users/ldh/Projects/github.com/deepthought/skills/handoff/SKILL.md`
  and `…/skills/handoff-to-codex/SKILL.md` (live via `~/.claude/skills`
  symlinks — edits are live immediately; deepthought has its own uncommitted
  backlog, commit only your two skill files, explicit paths).
- Dogfood rule: the moment I057 merges, run `make install` (refreshes
  `~/.local/bin/spine`) and mutate THIS effort's cursor only via the new
  verbs (`spine cursor tick implement`, etc.). The seeded block in
  `.superpowers/sdd/progress.md` was the last legal hand edit.

## Gotchas

- Reviewer floor: I057/I059/I060 carry `review-tier: primary` — resolve via
  `spine model codex primary`. Never review below a ticket's tier; record
  any dispatch-time deviation as an ESCALATION line in the ledger
  (`.superpowers/sdd/progress.md`, grammar in WORKFLOW.md).
- Live verification convention: never claim done from source edits — run the
  reinstalled binary (`make install` first) against a real repo and paste
  command + output as evidence in the ledger.
- Commit hygiene: explicit paths always, never `git add -A`/`.`;
  scratch/handoff-in-progress files stay out; `PICKUP.md` stays untracked.
- Cursor grammar findings BLOCK `audit stages` (2026-07-16 amendment) — a
  half-written writer that emits a malformed block will wedge the audit;
  keep writes atomic (serialize fully, then splice).
- `.superpowers/sdd/` is gitignored — ledger and dispatch files never land
  in commits; they are the local audit trail.
- Same-date handoffs tie-break lexicographically on filename (this file was
  renamed from `…-cursor-writes-codex.md` to out-sort
  `…-mutation-battery-shipped.md`). If you ship today, name the shipped
  handoff to sort after `sole-writer-codex` (e.g.
  `2026-08-06-sole-writer-shipped.md`) so `audit stages` sees it as newest.
  Day-granular dates making same-day successor efforts fight over sort order
  is a real quirk — worth an observation line in your shipped handoff's open
  items, owner decides if it becomes a ticket.
- Owner is not watching live; completion/blockage signal per your codex-team
  playbook (report in pane, trigger-flash the surface).

## Stage cursor (verbatim `spine cursor` output at handoff time)

<!-- spine:cursor -->
effort: cursor-writes
prd: docs/specs/2026-08-06-cursor-writes-design.md
tickets: I057-I061
stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] deploy[ ] docs[ ] handoff[ ]
<!-- /spine:cursor -->
