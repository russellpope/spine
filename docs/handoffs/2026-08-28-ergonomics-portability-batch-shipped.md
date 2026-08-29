---
title: "ergonomics-portability batch shipped"
created: 2026-08-28
handoff_ordinal: 32
---

# Handoff — ergonomics-portability batch shipped (2026-08-28)

## Context

Session 2026-08-28 (same day as the cursor-hygiene close). Opened by clearing
the chyg housekeeping, then ran the **ergonomics + portability batch
(I116+I117+I118)** solo inline, full stage gates, TDD per ticket.

Housekeeping first: the 8 estate WORKFLOW.md refreshes from the chyg sweep
were reviewed (all identical: grammar line + gen-bump note), committed
per-repo with explicit paths, and **pushed** — along with spine itself and the
prior estate backlog. ssh was dead (`ssh-add --apple-load-keychain` → "No
identity found in the keychain", no keys in `~/.ssh`, both agent sockets
unusable); pushes went over https via `gh auth git-credential` without
touching any remote config (memory: estate-push-gh-https-workaround).
hbmview's sweep commits sit on its `feat/header-redesign` branch, NOT pushed —
owner decides that branch's fate.

The batch, from the grill (Q1–Q9 all ratified): I116 option B — `spine model`
keeps strict stdlib ordering but a flag-like token among the positionals now
errors `flags must precede positionals (saw … after …)` + usage, exit 2,
helper wired into cmdModel only. I117 — the zero-evidence implement miss
names the done/complete/completed whole-word requirement when ledger lines
anchor the ids; the typo hint survives only the no-line-at-all case; mixed
case: wording wins, missing list intact. I118 (filed at the grill, owner's
portability ask) — README `go install …/cmd/spine@latest` one-liner and
`spine version` gains a `build:` provenance line via debug.ReadBuildInfo.

Spec-review (fresh sub-agent, requirements-attack first): 0 missing/partial;
three spec-text defects (C1 example named the wrong positional, C2 fallback
scope too narrow, C3 first-position exemption uncodified) amended inline in
the PRD, no code changed. This effort's cursor `tickets: I116,I117,I118` was
the comma-list grammar's second live exercise; the I117 evidence gate fired
for real when this batch's own implement tick was refused until the ledger
carried done-words.

## State (verify before relying)

- main = `b81292d` + this docs/handoff commit on top; pushed through
  `b81292d`'s predecessor set earlier — push the final docs commit (gh-https
  workaround if ssh is still keyless).
- Tickets I116/I117/I118 **fixed**, `batch: 2026-08-28-ergo#{1,2,3}`,
  commits + Resolutions written. Next free id: **I119**.
- `~/bin/spine` rebuilt at `b81292d` (sha256 prefix `b6150787d94b`);
  `spine version` now prints `build: … b81292de0263 …` — the drift baseline;
  future handoffs can drop sha256 prefixes.
- Lanes: ergo-batch #1 @6d2a03c, main #51 @bd67922, main #52 @b81292d all
  passed; the docs commit below needs its own `maipipe run full --wait`.
- `spine doctor`: expect exit 0 + standing D4 adr info once this handoff
  lands (stale-snapshot warn resolves with it).
- herdr `spine: codex team` workspace may still be open from the chyg batch —
  owner closes by hand.

## Next steps

1. Owner: decide hbmview's `feat/header-redesign` (push / merge / leave).
2. **I112 decision** (openweights axis) still standing; I111 carries the D28
   hazard. Nothing blocks on either.
3. Next candidates: I072/I102/I105 routing work, or deepthought
   `/openweights-team` (the critical path), or a small ticket generalizing
   the I116 helper (see gotcha below) — not yet filed.
4. Push cadence: this session pushed everything reviewable; keep it.

## Gotchas

- The I116 fix covers `spine model` ONLY. The same trap lives elsewhere:
  observed live this session, `spine cursor show --dir X` silently IGNORES
  the trailing `--dir` (cursor's flagset stops at `show`) and reads the CWD
  repo — worse than an error. Flags before subcommand: `spine cursor --dir X
  show`. Candidate I119.
- The retired handoff gotchas are now code behavior: a wording miss on
  implement evidence names the done-word rule at the point of failure
  (I117), and a trailing flag on `spine model` names the ordering rule
  (I116). Stop re-teaching them in handoffs.
- ssh keychain is empty and `~/.ssh` holds no keys — either restore keys or
  keep using the gh-https push workaround (exact commands in the memory
  file).
- Batch commits so one `maipipe run full --wait` covers each HEAD move; read
  exit codes unpiped under fish; stage explicit paths; never tick the
  handoff stage; `spine cursor` is the sole cursor writer.

<!-- spine:cursor -->
effort: ergonomics-portability-batch
prd: docs/specs/2026-08-28-ergonomics-portability-batch-design.md
tickets: I116,I117,I118
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
