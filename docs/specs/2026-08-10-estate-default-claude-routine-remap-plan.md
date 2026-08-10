# Estate default Claude routine remap — plan

**Date:** 2026-08-10 · **Effort:** i063-estate-remap · **Ticket:** I063

## Build sequence

1. Change the embedded Claude routine entry to Opus 5 at low effort, move the
   shipped Sonnet 5 pair to history, and replace current aliases.
2. Update model resolver and MirrorRows expectations, including pair-aware
   historical classification and the explicit low-effort suffix.
3. Add current-generation update tests for inherited refresh/itemization,
   override preservation, and already-current idempotence. Adjust only the
   centralized sanctioned migration diff allowances if strict fixture locks
   require it; do not rewrite historical fixtures. Correct the legacy
   top-level effort migration so a customized value replaces the new default
   suffix instead of being skipped.
4. Run focused and full tests, then obtain an independent blind review of the
   finished diff against the paired design document. Correct and re-review any
   findings.
5. Run fresh verification and workflow gates, resolve I063, commit explicit
   paths, create the shipped handoff, and re-run the final audits.

## Stage mapping

- **implement:** one coherent routine-tier worker owns the table and tests.
- **functional-test:** exercise the real CLI update path against disposable
  current-generation repositories for stale inherited, deliberate override,
  and already-current cases.
- **review:** independent primary-tier final whole-branch blind review against
  the PRD; no implementer report is supplied.
- **verify:** fresh primary whole-branch acceptance verification, full Go
  suite, doctor, stage/routing audits, and diff integrity checks.
- **ship/docs/handoff:** record the no-generation-bump decision and evidence in
  I063, commit only explicit paths, and create a fresh cursor-bearing handoff.

## Risks and controls

- The effort change makes history pair-sensitive: tests must prove the old
  medium pair is inherited while an old id at low effort is not.
- Routine and fallback intentionally share Opus 5 after this change. Routing
  remains tier-based; aliases stay scoped to resolved entries and historical
  ids remain exact-only.
- Strict migration locks may expose the new table-driven diff. Centralize any
  sanctioned allowance and leave captured historical fixture content intact.
- Cursor writes intentionally block `audit stages` until the shipped handoff
  captures the final snapshot (ADR 0014).
