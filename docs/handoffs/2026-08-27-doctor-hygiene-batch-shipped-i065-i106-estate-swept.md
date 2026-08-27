---
title: "doctor hygiene batch shipped — I065+I106, estate swept"
created: 2026-08-27
handoff_ordinal: 25
---

# Handoff — doctor hygiene batch shipped — I065+I106, estate swept (2026-08-27)

## Context

**The doctor hygiene batch (I065 + I106) is shipped, pushed, and deployed
estate-wide.** Executed as batch `2026-08-27-dhyg` via the claude-team
transport on cmux (workspace:6 `spine: sdd-workers`) — spine's first live
`--batch` run and the first live use of the `batch:`/`workspace:`/`commits:`
keys the batch itself documents. Full SDD gating: blind task reviews with a
requirements-attack step, one fix round on I106, final whole-branch review on
fable-5 @ high ("Ready to merge: Yes", 0 Critical/Important).

Grill rulings that shaped it (all in the spec): I065 before I106; doctor
keeps `warn ⇒ exit 1` (I106's "warn not block" = severity warn, not
exit-neutral); full-history verbatim backfill (found a third retired line —
the `review-tier` bullet — neither ticket nor spec named); estate sweep as
this effort's deploy checklist; spine-only build.

What shipped: `supersededLines` backfill + binding gen-bump rule
(`internal/update/update.go`), three historical fixtures + negative control;
four ticket keys documented in the ledger convention (template + scaffold);
doctor D13 — first per-ticket checks (dangling `workspace:` any status,
`workspace:` on a closed ticket, malformed `batch:` on live tickets only),
ten tests incl. four negative controls. Follow-ups filed: **I114**
(cursor tickets grammar can't express a two-ticket batch) and **I115**
(D13/frontmatter hardening bundle from final-review triage).

## State (verify before relying)

- **main = `a1fc359`, pushed, in sync** (8 commits this session:
  `c3dd640` pre-work, `6f306a2`/`d389f17` I065, `f3827dc`/`0aff8d4`/`f06ce19`
  I106, `a1fc359` follow-through, plus this handoff's commit — which is one
  more lane run owed and run). maipipe full passed at `a1fc359`.
- `go test ./...` exit 0 (18 pkgs); gofmt/vet clean. Worktree
  `~/worktrees/spine-2026-08-27-dhyg` and branch `doctor-hygiene-batch`
  removed after ff-merge.
- `~/bin/spine` rebuilt at `a1fc359` (`make install`, sha256 prefix
  `cba771edf9f1d9b3`).
- **`spine update --dir .` exit 0 — I065's target achieved.** The D4 warn on
  `docs/issues/README.md` is gone.
- **`spine doctor` still exits 1, for a NEW, honest reason:** the D9 warn
  "tickets `I065,I106` does not resolve" — the grammar gap this batch
  surfaced, accepted at the grill, cured by I114. It persists until I114
  lands or the next effort's cursor replaces the tickets value. The adr
  README note is info (exit-neutral). This is NOT the old drift red.
- **Estate sweep (deploy checklist, per-repo `pre/-write/post/doctor` exits;
  full logs were in /tmp/sweep-*.out):**
  - Cured 1→0/0: ccq, home-lab-admin, jarvis, notetui, observability_notes,
    pure-automation, deepthought, hbmview (8/11). Each has uncommitted
    machine refreshes (WORKFLOW/CLAUDE/AGENTS/issues-README) — owner commits.
  - Residual skips, all verified GENUINE local edits (the guard working, not
    missed stock): praxis issues-README (ADR-0108 precedence note),
    moo-clone issues-README (id-band convention section), ultima WORKFLOW
    (dated model-pin comment). Owner reconciles only if wanted.
  - Pre-existing per-repo warns unrelated to I065: D11 `.superpowers` not
    gitignored (ultima, praxis, hbmview), praxis D9 issues-unticked,
    moo-clone D9 malformed old cursor block.
- Ledger: I065, I106 **fixed** (with `batch:`/`commits:`, workspace cleared —
  live D13 subjects passing); I114, I115 **open**.
- The SDD workspace (ledger, briefs, reports) was deleted with the worktree
  per SDD; rulings are recorded in the spec, tickets, and the session's
  final report.

## Next steps

1. **Owner: commit the 8 estate repos' machine refreshes** (each is a clean
   `spine update -write` result; `git diff` per repo before committing).
2. **I114** (comma-list tickets grammar) — small, precisely scoped, and it
   retires the one remaining doctor warn on spine. Natural next pick.
3. I115 (D13 hardening) — bundled, ~20 lines, non-blocking.
4. Owner call: reconcile praxis/moo-clone/ultima local edits or leave them
   (they are deliberate; the ADR-0108 note in praxis even says it expects to
   survive regeneration — it does, by blocking it, which may not be what its
   author intended).
5. openweights (I111, I112) remains parked.

## Gotchas

### Learned this session

- **A reworded line that never shipped in a generation is NOT a supersession
  case.** I106's fix round reworded a line its own branch introduced; `spine
  update -write` correctly skipped the render as locally edited. The cure is
  converging the render by hand to the template (identical outcome to the
  refresh) — NOT `--force` (repo-wide, would regenerate hand-authored files)
  and NOT a supersededLines entry (nothing shipped ever emitted the line).
- **`grep -c`/filters eat exit codes even inside `bash -c`** — the controller
  hit `go test ./... | grep -v ok; echo $?` reporting grep's status. Run the
  command unpiped into a file, echo `$?` immediately, inspect the file after.
- **cmux workspace reuse over recreation:** `spine: sdd-workers`
  (workspace:6) is the spine group's anchor — closing it would dissolve the
  group. Reused with explicit `cd` in every dispatch despite a stale cwd.
- The estate sweep's residual skips looked like backfill misses; inspecting
  the actual skipped lines proved all three are deliberate owner content.
  Always read the lines before concluding either way.

### Standing (unchanged, see prior handoff for detail)

- Read exit codes unpiped; heredocs unreliable (write scripts to scratchpad);
  macOS bash 3.2; fish quotes globs; `spine` wants flags before positionals;
  never tick `handoff`; `spine cursor start` needs `--force` mid-flight;
  audit routing judges against `tier` with ESCALATION grammar; maipipe stop
  hook wants `full --wait` on HEAD moves; stage explicit paths only
  (`docs/research/2026-08-26-fusion-harness-borrow-hitlist.md` is untracked
  and not ours); owner ban on claude-sonnet-5 (substitute opus-5 @ low —
  honored via `spine model` this session).

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
