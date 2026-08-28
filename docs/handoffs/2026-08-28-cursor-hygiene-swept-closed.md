---
title: "cursor-hygiene-swept-closed"
created: 2026-08-28
handoff_ordinal: 30
---

# Handoff — cursor-hygiene-swept-closed (2026-08-28)

## Context

Session 2026-08-28, end of the **cursor-hygiene batch (I113+I114+I115)**. The
full arc: resumed the paused grill from PICKUP, settled rounds 1–2, wrote the
PRD pair (`docs/specs/2026-08-28-cursor-hygiene-batch-design.md` + `-plan.md`),
filed I116/I117 from the to-tickets sweep, dispatched a codex team
(sol lead, terra/luna workers) via herdr, the team shipped all three tickets,
and this session verified independently, ran the eight-repo estate sweep, and
closed deploy/docs.

The team's full evidence trail is `.superpowers/sdd/team-report.md` (per-task
red/green + observed-red negative controls, blind reviews, fable-5 @ high
final review with requirements-attack, fresh primary verification,
`spine audit routing` exit 0 with all rows `match`). Ticket bodies carry
`## Resolution` sections; the lead's own handoff is
`docs/handoffs/2026-08-28-cursor-hygiene-batch-shipped.md`.

Mid-session incident worth knowing: a macOS configd/opendirectoryd outage
broke DNS + user lookup for every pre-outage process tree (this session's,
herdr's, the first codex lead's). Daemons were bounced (`sudo killall
opendirectoryd configd; sudo killall -HUP mDNSResponder` — `launchctl
kickstart` is SIP-blocked), then herdr's terminal daemon and this Claude
session had to be restarted too because stale process trees never recover.
Symptom signature for next time: curl exits 000 in microseconds, `id -un`
prints `501`, ssh says "No user exists for uid 501".

## State (verify before relying)

- `main` = **`754da7c`** before this session's docs commit (changelog + this
  handoff land on top; check `git log --oneline -3`). **Ahead of origin by
  14+ commits — NOT pushed.** ssh-agent lost its identities in the daemon
  bounce (`ssh-add -l` → none); load keys (`ssh-add --apple-load-keychain`)
  before pushing.
- Tickets **I113, I114, I115 fixed** with `batch: 2026-08-28-chyg#{1,2,3}`,
  `commits:` written, workspaces cleared — the I106 key convention's second
  live exercise, lead as sole writer throughout.
- `~/bin/spine` rebuilt at ship (sha256 prefix `61b7fa7ba63f`). Comma-list
  grammar live: doctor's old D9 on the dhyg ledger value is gone.
- `spine doctor` on spine: **exit 0** + the standing D4 adr info note.
- Lane: runs #48 (`a3f371b`) and #49 (`754da7c`) passed; this session's
  docs commit needs its own `maipipe run full --wait` (done if the commit
  below says so; verify with the maipipe daemon otherwise).
- **Estate sweep done (8/8 cured, pre=1 → write=0 → post=0):** ccq,
  home-lab-admin, jarvis, notetui, observability_notes, pure-automation,
  deepthought, hbmview. Each diff is the grammar line + the 5-line gen-bump
  authoring note in WORKFLOW.md, **uncommitted — owner reviews and commits
  per repo** (logs: `/tmp/sweep-chyg-<repo>.out`). Doctor exit 0 everywhere
  except hbmview's pre-existing D11 (`.superpowers` not gitignored) — not a
  regression. praxis/moo-clone/ultima untouched by design; ultima's WORKFLOW
  is now one grammar-line stale on top of its model-pin edit.
- Open tickets: I116 (spine-model flag order), I117 (implement-tick
  misdirected message), I102, I105, I108, I109 (ticket open; its code
  shipped), I111, I112 (owner decision), I072. Next free id: **I118**.
- herdr `spine: codex team` workspace (w1A) still open — owner reviews the
  lead pane, then closes by hand.

## Next steps

1. **Owner: commit the 8 estate WORKFLOW refreshes** (`git diff` per repo
   first), push spine and estate repos as wanted (ssh keys first).
2. **Owner: close the herdr team workspace** after reviewing the report.
3. **I112** remains the standing owner decision (openweights axis); nothing
   blocks on it. I111 is the behavioural half, still unstarted, carries the
   D28 hazard (audit predicate must test transcript source, not flavor).
4. Next batch candidates: I116+I117 (both small, message/ergonomics), or
   I72/I102/I105 routing work, or the openweights programme via deepthought
   (`/openweights-team` still unbuilt — the real critical path there).
5. Lead's deferred ruling to fold into the next template-content change:
   align `prefix I0` (canonical example) vs `prefix <str>` (diagnostic
   metasyntax) — recorded in the team report.

## Gotchas

- **The comma-list form is strict**: no spaces, no duplicates, no partial
  resolution — a spaced hand edit fails the whole value, by design.
- **Gen-bump rule is now binding and written in the WORKFLOW template's
  authoring notes**: any content-changing template edit appends its
  predecessors' dropped lines to `supersededLines` in the same change.
- The maipipe stop hook demands `maipipe run full --wait` on every HEAD
  move, docs-only included — batch commits.
- **Stage explicit paths only**; never tick the handoff stage; `spine
  cursor` is the sole cursor writer; never write the literal cursor marker
  text in prose.
- `spine cursor start` refuses mid-effort — `--force` to supersede, and run
  it BEFORE `spine handoff new`.
- Read exit codes unpiped under fish. Flags before positionals on spine
  subcommands (I116). Ledger implement evidence needs a done/complete
  whole word (I117).
- herdr codex dispatch: first `agent prompt` after `agent start` stalls
  roughly half the time; a repeated stall can mean the text was TYPED but
  not submitted — read the pane before re-prompting; one
  `herdr agent send-keys <name> enter` submits it (memory:
  herdr-claude-team-transport-quirks, updated 2026-08-28).
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @
  low`.

<!-- spine:cursor -->
effort: cursor-hygiene-batch
prd: docs/specs/2026-08-28-cursor-hygiene-batch-design.md
tickets: I113-I115
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
