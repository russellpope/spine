# Fable 5.1 remap rollout — plan

**Date:** 2026-09-02 · **Effort:** i128-remap-rollout · **Ticket:** I128

## Build sequence

1. Docs first: the retroactive design/plan pair for 68aa28f; a dated note on
   ADR 0011 pointing at I128; the `retired override` glossary term in
   CONTEXT.md.
2. Model package: a successor lookup for a historical id (same tier first,
   then tier order), test-first.
3. Update: the retired-override migration in the model-routing pass, the
   `ModelRefresh` kind marker, the CLI plan line, and the
   `modelDefaultDivergence` fix. Tests: stuck override at xhigh, retired id
   with edited alternate, current-id override preserved (negative control),
   inherited refresh unchanged, gen-9 bare row with customized effort.
4. Doctor: D18 retired-mirror check; D16 unreachable message with the
   host-file remedy and retired hint; fixtures moved to the current id.
   Tests: stale mirror, refreshed mirror (silent), current-id override
   (silent), host file listing only the retired id, host file listing no
   historical id (no hint).
5. Test precision: `sanctionedRefreshLine` helper; the ten skip blocks
   switched to it; static allowlists deleted; exact-row locks; gen-13 to 14
   converse assertion; negative control.
6. Deepthought: both preflight blocks capture the exit code and relay a
   refusal; guard script gains the refusal arm; run it.
7. Full Go suite, vet, doctor, stage and routing audits; blind spec-review
   of the finished diff against this design and of 68aa28f against the
   retroactive design; fresh-context verifier at primary; fix and re-review.
8. Ship: commit explicit paths, run the maipipe lane once at the final SHA,
   push, rebuild both binaries.
9. Deploy: sweep the 14 stale primary checkouts with the installed binary,
   validate each, record the list.
10. Docs and handoff: resolve I128 with evidence, handoff with cursor.

## Stage mapping

- **implement:** steps 1 to 6, inline under test-driven development (the
  chain is tightly coupled: update, doctor, and locks all key off the same
  successor predicate). Justification recorded in I128.
- **functional-test:** compiled-CLI runs against the jarvis stale mirror and
  the scratch stuck-override copy, plus a disposable current-generation repo
  per case.
- **review:** primary blind review of the whole diff against this design;
  the retroactive spec-review of 68aa28f.
- **verify:** fresh primary verification, full suite, doctor, audits, live
  binary evidence.
- **ship/deploy/docs/handoff:** as above.

## Risks and controls

- The retired-override rule changes update's disposition of a value the
  I063 tests may lock as preserved. Read those locks before editing; a lock
  on a historical-id override at a foreign effort is the case this plan
  changes deliberately and is rewritten with the reason.
- Consolidating ten locks can widen or narrow what they admit. The negative
  control proves narrowing; running every gen lock proves nothing widened.
- The deepthought guard script's old-binary arm greps for specific strings;
  keep them.
- The sweep writes into other repos' working trees. Dry-run each first and
  refuse any checkout whose WORKFLOW.md is already modified.
