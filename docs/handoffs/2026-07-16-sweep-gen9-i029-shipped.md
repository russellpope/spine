# Handoff Reference — gen-9 fleet sweep + I029 shipped (2026-07-16)

## Stage cursor

<!-- spine:cursor -->
effort: gen9-sweep-i029-i030
prd: docs/specs/2026-07-15-stage-cursor-controls-design.md
tickets: I029-I030
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[<]
<!-- /spine:cursor -->

(Filename deliberately sorts after `2026-07-16-stage-cursor-polish-shipped.md` — same-day handoffs tiebreak by
filename DESC; see [I031](../issues/I031-same-day-handoff-filename-tiebreak.md), filed this session.)

## State

- **SHIPPED:** main FF-merged 015d132..b836bc9 (4 branch commits: I030/I031 filings dfedba1, I029 fix 26de369,
  spine self-update 40e4753, I032 filing b836bc9) + ticket-close commit 3a898d5 + this handoff. Binary
  reinstalled from main (`spine version` → 9, now with I029's named-ids detail).
- **Fleet is 17/17 at gen 9** (I030): 14 clean sweep commits, objectstudio (244059a) + maipipe (1364d2b)
  hand-folded with local edits preserved verbatim, spine on the effort branch. Full per-repo table in the
  archived effort ledger. Every repo's delta = WORKFLOW.md stamp + tickets-grammar line + doctor-advises
  sentence, plus template-owned CLAUDE/AGENTS `spine:begin v9` marker bumps.
- **Pushes:** owner approved the full backlog this session — spine, deepthought, objectstudio, praxis,
  home-lab-admin pushed after this handoff landed (see PICKUP/final state). The OTHER 11 swept repos each
  carry 1 unpushed gen-9 commit (hbmview's is on `feat/header-redesign`) — those pushes were NOT in the
  approved list and remain owner-gated.
- **I029 behavior now live:** `ticked-missing` details name the missing ids (first 5 + "+N more"); an
  all-missing set from a resolvable `tickets:` value appends `— tickets: "<value>" resolved but every id is
  missing; check it for a typo`.

## Why (key decisions + rationale)

- **Grill/PRD rode the parent effort** (I024-I027 precedent): I029 was filed by the parent's final review, the
  sweep was flagged as follow-up at its ship, and the owner approved all four session items 2026-07-16 via an
  explicit scope question (sweep, I029, I031 filing, push backlog).
- **Final-review Minors accepted as-is, deferred to I032** (parent P2-P6 precedent): (M1) the typo hint also
  fires on the implement row where the issues row can prove `tickets:` correct — text-only, partly
  ticket-inherited (RA1); (M2) truncation test couples fixture size to the cap constant.
- **Sweep per-task review folded into the final whole-branch review** (I023 precedent) — mechanical regen,
  evidence = dry-run consistency + doctor + post-commit status checks, then 17/17 surface + 5 deep spot-checks
  by the primary reviewer.

## Review trail

Per-task review I029 (sonnet, routine): Approved, 0 findings; reviewer independently re-proved RED on the base
commit via temp worktree and eyeballed all three detail wordings. Final whole-branch review (fable, primary;
first dispatch died on API 529 pre-output, clean relaunch): READY TO MERGE: YES — 0 Critical, 0 Important,
2 Minor (→ I032). Requirements-attack RA1-RA6 all surfaced with resolutions (notably RA5: I031's candidate 1
as worded would weaken the I014 newest-doc invariant — constrain it or prefer candidate 2 at assignment).
Acceptance sim A-G live on the branch binary including the cap boundary (5 named, no "+N more"; 6 → 5 +
"+1 more") and the gen-9 bare-id grammar form. Fresh-clone full suite + vet + gofmt clean (standing step — no
gitignore-swallowed fixtures this batch). Verify: routing audit session-scoped exit 0 (all escalations
reasoned, records at dispatch time this build); audit stages blocked only on the expected I025 stale-effort
finding, cleared by this document.

## Open questions & risks

- **I031** (open, low): same-day handoff filename tiebreak — carry RA5's constraint into effort assignment.
- **I032** (open, low, mechanical): scope the typo hint to the issues row; decouple truncation test from cap.
- 11 swept repos with unpushed gen-9 commits (owner-gated; list = sweep table minus the 5 pushed repos, spine
  included in pushed).

## Gotchas & hard-won lessons

- `spine audit routing` against a multi-session project transcript dir drowns in other sessions' dispatches
  (`<synthetic>` models, prior-effort tickets) — copy THIS session's jsonl to a scratch dir and pass that
  (parent batch did the same; now twice, consider a --session flag ticket).
- Ticket presence in stage derivation requires `id:` frontmatter — empty fixture files don't count (bit FT1;
  documented issueIDs contract).
- A ticket id merely *named in a reviewer dispatch prompt* (I031 here) gets correlated by the routing audit and
  needs its own ESCALATION record — filed-only tickets included.
- Same-day second handoff: name it to sort after the existing one (this doc: `sweep-...` > `stage-...`).
- First final-review dispatch died on API 529 pre-output; read-only reviewer, so clean relaunch with the same
  prompt was safe — verify tree state before retrying any WRITING agent (praxis 2026-07-10 lesson).
