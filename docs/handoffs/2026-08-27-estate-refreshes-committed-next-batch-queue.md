---
title: "estate refreshes committed — next batch queue"
created: 2026-08-27
handoff_ordinal: 26
---

# Handoff — estate refreshes committed — next batch queue (2026-08-27)

## Context

Closes out the doctor-hygiene batch session. **Read the previous handoff
first** — `docs/handoffs/2026-08-27-doctor-hygiene-batch-shipped-i065-i106-estate-swept.md`
(ordinal 25) carries the full batch record: what shipped, review chain,
rulings, sweep results, gotchas. This handoff adds only what happened after
it: the estate refreshes are now **committed** in all 11 repos, and the next
batch is queued for the owner's grill.

## State (verify before relying)

- spine `main` pushed and in sync; this handoff's commit is the only one
  after `e793061`. `spine update --dir .` exit 0; `spine doctor` exit 1 on
  exactly one warn — the D9 tickets-grammar gap (I114's target), plus the
  exit-neutral adr info note.
- **Estate sweep commits (local only, NOT pushed — pushing is per-repo
  owner's call):** ccq `ee4efe1` (main), deepthought `eb99794` (main),
  hbmview `d160640` (**feat/header-redesign**), home-lab-admin `d48f433`
  (main), jarvis `3f8b0bb` (main), moo-clone `37e2c0b` (**m4b-war-screens**),
  notetui `ec43774` (main), observability_notes `77e7666` (main), praxis
  `05bbc31` (**authz-batch-2026-08**), pure-automation `27e4ca9` (main),
  ultima-dci-edition `82a6875` (**residual-wave-2026-08**). Four landed on
  feature branches (bolded) because that's where those repos sat — machine
  refreshes only, safe to merge or cherry-pick wherever the owner prefers.
  Staging was exact-file-list guarded; remaining dirt in those repos is
  pre-existing scratch, untouched.
- Deliberate local edits preserved (NOT drift): praxis issues-README
  (ADR-0108 note), moo-clone issues-README (id-band section), ultima
  WORKFLOW (model-pin comment).
- Ledger: I065/I106 fixed; **open frontier candidates: I114, I115, I113,
  I032, I072** (I073/I077 blocked on I072). openweights (I111, I112) parked.

## Next steps

1. **Owner review of the shipped batch:** the diff is `c3dd640..a1fc359` on
   main; handoff 25 + the spec's Further Notes carry the rulings. An
   independent pass is one command away: `/spec-review` against
   `docs/specs/2026-08-27-doctor-hygiene-batch-design.md`, or
   `/review-ticket I065` / `I106`.
2. **Next batch — owner's grill call.** The natural candidate is a
   cursor/doctor hygiene micro-batch: **I114** (comma-list tickets grammar —
   retires spine's last standing doctor warn; touches internal/cursor +
   internal/stages), **I115** (D13/frontmatter hardening, ~20 lines), and
   optionally **I113** (trailing whitespace after the closing cursor fence
   escapes NonCanonical — pre-existing I109 follow-on). All small, all
   adjacent machinery. Alternative: **I072** (host config schema) — the only
   frontier ticket that unblocks others.
3. Cursor: this effort is terminal at `handoff[<]`. The next effort starts
   with `spine cursor start --force --effort <name> --tickets <...>` BEFORE
   any `spine handoff new`. Note the tickets-grammar trap below.

## Gotchas

- **The `tickets:` grammar cannot express a non-contiguous multi-ticket
  batch** (I114). `I114,I115` will warn evidence-not-judged (accepted last
  time); `I114-I115` works ONLY for adjacent ids — `I113-I115` would be
  three tickets, which happens to be exactly the candidate batch, so a range
  may genuinely fit this time.
- Estate repos: four commits sit on feature branches (see State) — don't
  assume main when following up.
- Everything else: handoff 25's Gotchas section (learned + standing) still
  applies verbatim.

<!-- spine:cursor -->
effort: doctor-hygiene-batch
prd: docs/specs/2026-08-27-doctor-hygiene-batch-design.md
tickets: I065,I106
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[<]
<!-- /spine:cursor -->

## Checkpoint (newest): 003-dogfood-the-shipped-local-harness-conventions-on-spine-itsel.md

<!-- spine:checkpoint:facts -->
touched:
- internal/update/gatepack.go
- internal/gate/results.go
- internal/gate/mutate.go
- maipipe.toml
- WORKFLOW.md
gate: pass
sha: 265efc9ede4c229f135c38b558bfe722ec918427
effort_recommended: medium
written: 2026-08-19T16:31:36Z
<!-- /spine:checkpoint:facts -->

### Prior narrative (model-authored, not evidence)

## Task

Dogfood the shipped local-harness conventions on spine itself (deepthought handoff 2026-08-19 §1a–h) and close the cross-repo follow-through (§2).

## Conclusions

- go@1 pack is self-enabled on spine (I089); five classes + mutation-go pass under maipipe at the pinned commit.
- First live maipipe seam found four defects, all fixed: region TOML grammar + schema (I091); results line 0 / file "." / severity "warn" and battery env leak (I092).
- Bake-off positive control: hygiene classes catch committed binaries on 3/3 arms (docs/research/2026-08-19-…).
- Checkpoint round-trip, model alternate provenance, routing blind spot (I090) verified; minor follow-ups in I093.
- Cross-repo: maipipe I201 filed; deepthought spine PRD amended; /model-eval runs the binary.

## Next moves

- Owner: push spine (main ahead of origin, unpushed since 2132d89); close herdr team workspace; remove worktree spine-wt-local-harness.
- Owner call on I093 items 3–5 (unconfigured-class stages, --force scoping, D11 value tamper).
- Phase 1 continues: `/grill-with-docs` in maipipe with deepthought's maipipe execution-floor PRD.
