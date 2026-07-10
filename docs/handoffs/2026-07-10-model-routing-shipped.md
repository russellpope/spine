# Handoff Reference — model-routing shipped + dogfooded (2026-07-10)

## Net state

Model-routing build COMPLETE and pushed. spine main @ 4b3ab04 (gen 6,
pushed), deepthought main @ 1377ee1 (gen 6, pushed), objectstudio @ dd44a02
(gen 6, local — hand-folded WORKFLOW.md, no remote push authorized).
I002–I005 fixed; I006 (fleet sweep + memory supersede) open, gated on one
real build exercising the machinery — praxis I001 is that build.

## Why (key decisions + rationale)

- Full decision record: docs/specs/2026-07-09-model-routing-design.md (D1–D10
  from the 2026-07-09 grill; amended 2026-07-10 by final review — verdict
  enum superset, fallback effort=high, notification channel pinned).
- Vocabulary: CONTEXT.md at spine root (execution modes, tiers, escalation /
  silent descent both directions, fallback routing, effort routing, routing
  audit, reviewer floor incl. inline n/a).
- ADR 0010: tiers-not-ids + contract lives in estate-owned surfaces.

## Review trail (what caught what — the pattern held, 6th consecutive)

- Task gates: 1 fix loop each. I002 direction-blind escalation excusal
  (false-negative through silent-descent); I003 two fleet-propagating
  contract wording gaps; I004 requirements-level contradictions in the
  controller's own supplement (review-tier n/a, effort omission — resolved
  per ratified D4/D7).
- Final review + acceptance sim (live binary, real-format transcripts):
  C1 vacuous-verify cross-artifact defect (/to-tickets local mode emitted
  tickets the audit can't see → exit-0 gate that judged nothing).
- Confirmation pass: caught NEW-1, a Critical the fix wave itself introduced
  (removed-key customized values silently destroyed on plain --write) —
  live-reproduced, guarded (7a75e13), re-verified at the write boundary.
  LESSON: confirmation passes re-run the original repro; keep that step.

## Dogfood evidence

- `spine audit routing` ran against THIS build's real transcripts: caught
  the controller's own unrecorded fix-wave descent (I002) and primary
  adjudication (I004); after writing 3 ESCALATION records → all
  escalated-with-reason, exit 0. Records live in spine
  .superpowers/sdd/progress.md (gitignored — full build ledger incl.
  deferred-minors list M1–M10, C3–C6).
- objectstudio audit smoke: C1b "nothing audited" warning fired correctly on
  its pre-convention unannotated ledger; default transcript-dir derivation
  worked.

## Open items & risks

- I006 (fleet sweep): gated on praxis I001 completing as the first real
  routed build. Sweep runbook notes: C3 objectstudio WORKFLOW.md carries
  deliberate local content (vfb gate) — doctor D4 warn is its standing
  state, never --force it; M9-class customized keys skip with named lines.
- Deferred minors (all recorded in spine .superpowers/sdd/progress.md):
  C4 transcript-slug derivation only flattens / and . (repos with other
  punctuation need --transcripts); C5 last-FALLBACK-reason-wins; C6
  lateral-tier detail wording; M1 dated model ids report unmapped (warn);
  M10 whole-file "auto" assertion fragile-but-loud.
- to-spec skill has the same installer-managed exposure to-tickets had
  (symlink into ~/.agents, mattpocock/skills lockfile) — vendor into
  deepthought before any estate modification touches it.
- Spec's "companion plan to be written by /to-tickets" header note is stale
  (tickets ARE the plan; no plan file was produced) — harmless, tidy at will.
- Memory supersede list for I006: ultracode/subagent-driven conflations;
  stale "~/.agents to-tickets clobber" warning (vendored 606871f).

## Gotchas & hard-won lessons (this session)

- Dispatch discipline is now CONTRACT: every subagent dispatch names an
  explicit model AND its description carries the ticket id token; reviewer
  dispatches above a ticket's tier need an ESCALATION record at dispatch
  time (post-hoc works but the audit flags it first).
- Ledger record grammar is exact: unspaced arrow, `reason:` required;
  malformed records excuse nothing (by design, documented in the template).
- spine update dry-run exits 1 on pending — never `&&` it ahead of the
  write step. Bash tool runs bash, login shell fish (quote globs; `==` in
  test brackets breaks).
- Push convention: spine + deepthought remotes are `origin`; push only on
  Russell's word (2026-07-10 authorization spent).
