# Handoff Reference — derivation polish batch I024-I027 shipped (2026-07-16)

## Stage cursor

<!-- spine:cursor -->
effort: derivation-polish-i024-i027
prd: docs/specs/2026-07-15-stage-cursor-controls-design.md
tickets: I024-I027
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[<]
<!-- /spine:cursor -->

(Verbatim `spine cursor` at write time reported `derivation: blocking` on exactly one finding — the previous effort's handoff was newest, a stale-effort block, which is I025's own new rule working as designed. THIS document carries the effort-matched block, clearing it; re-run `spine audit stages` to confirm exit 0.)

## State

- **SHIPPED:** main FF-merged 8e50ec2..fdad11c (9 branch commits) + ticket-close commit 38aae16 + this handoff. NOT pushed — main is ahead of origin by 12; push is the owner's call.
- **Template generation is now 9** (I026 grammar line + I027 doctor-advises clause). Gen-9 binary installed at ~/bin/spine (`spine version` → 9). The SessionStart cursor hook now runs the gen-9 binary everywhere.
- **Fleet is entirely gen 8** (17/17, zero residue as of this morning's I028 closure) and now one generation behind the binary. Deliberate: no sweep ticket existed in this batch. Sweep to gen 9 is a queued owner decision; each repo's update is a two-line diff (grammar line + handoff-rule clause) + stamp, and the hand-folded repos (objectstudio, maipipe) will again skip with unrecognized-edits listings (verified non-destructive by the gen8to9 recognition tests + FT6 dry-run on deepthought).
- The spine repo's own WORKFLOW.md is also still gen 8 — it rides the same sweep decision.

## Why (key decisions + rationale)

- **Gen-9 bump ruling:** I026's template change had an owner-flagged choice (bump vs ride-next-gen); ADR 0004 (single integer generation compiled into the binary) makes "edit template text without bumping" incoherent, and I027's Fix explicitly pairs its template text with I026's bump. One bump, both changes.
- **I025's fresh-effort consequence accepted:** a new effort now blocks on `audit stages` until its own handoff exists (stale-effort block = same finding as absent block). This matches the design-mandated handoff-absent exception and the parent effort's observed mid-build state; the design doc's I014 bullet carries a dated amendment recording it.
- **Spec-review gate adjudication:** this batch's PRD is its four tickets; the fable whole-branch review (requirements-attack over tickets + design doc, live acceptance simulation, two rounds) served as the verify-stage spec-review — a separate dispatch would have re-run the identical comparison.

## Review trail

Per-task reviews (sonnet, review-tier routine): I024 Approved (1 Minor), I025 Approved (2 Minors), I026 Needs-fixes → doc-comment fix b2c0277 → Approved, I027 Approved (1 Minor). Final whole-branch review (fable, primary, per-WORKFLOW rule): READY WITH FIXES → fix wave 1441a78+4a09016 (cursor prints Report.Notes + handoff detail under malformed header; untracked the convention-violating I024 report; /handoff skill + design-doc syncs; filed I029) → re-review found 1 Critical (F1 fixture ledgers gitignore-swallowed, fresh clone red — proven by clone-and-test) → controller fixed inline fdad11c (git add -f, bookkeeping-only) and re-proved via fresh clone (suite exit 0) → verdict READY TO MERGE: YES. Minors P2-P6 accepted as-is with reasoning in the effort ledger.

## Open questions & risks

- **Fleet sweep to gen 9** — owner call (see State). Low risk, mechanical, but 16 repos × 1 commit of unpushed noise.
- **I029** (open, low): resolvable-but-wrong `tickets:` ranges (`I01-I04` typo) block correctly but don't name the missing ids or hint the typo; grammar discoverability exists only for the unresolvable class.
- Push backlog: spine ahead 12 after this session; deepthought ahead 3; objectstudio ahead 70; praxis ahead 21; home-lab-admin ahead 2. All owner-gated.

## Gotchas & hard-won lessons

- **The `.superpowers/` gitignore gotcha bit its THIRD build** — and in the worst way: the fix wave that *removed* a wrongly-tracked `.superpowers` file simultaneously *forgot to force-add* two rightly-tracked fixture ledgers. Naming the gotcha in the dispatch prompt is not sufficient; the final re-review's fresh-clone test is what caught it. Recommend: make "fresh-clone `go test ./...`" a standing verify-stage step for spine (it is what proved both the bug and the fix).
- Routing audit + fable final review: the "final review always runs primary" WORKFLOW rule still needs per-ticket `ESCALATION <id> mechanical->primary` records when the review dispatch description contains ticket ids — recorded at verify this time (second effort running this lesson; consider a template/audit affordance).
- fish pipeline exits: `$status` after `cmd | head` is head's status — capture exits without pipes.
- `spine cursor` now prints `warning:` lines (Report.Notes) and `n/a (cursor malformed)` — session-start hook output shape changed slightly with gen 9.
- Same-day handoffs tiebreak by filename DESC (handoff.List, documented): this doc was first written as `...-derivation-polish-shipped.md` and lost "newest" to the morning's `...-stage-cursor-controls-built.md` — renamed to sort after it. When writing a second handoff on one day, check the filename sorts after the existing one (or `spine audit stages` keeps blaming the older doc).
