---
title: "ledger reconciliation and the herdr detection spike"
created: 2026-08-26
handoff_ordinal: 22
---

# Handoff — ledger reconciliation and the herdr detection spike (2026-08-26)

## Context

Session 2026-08-26. Two things landed, in two repos, and one programme thread
was parked by owner decision.

**The openweights programme is off the critical path.** The owner built
`/openweights-team` on the second laptop. It is **not pushed** — `git fetch` on
deepthought showed no incoming commits to `main` — so it exists on exactly one
machine. Stated intent: "we'll probably refine it here later." Do not start
openweights work unless the owner reopens it; **I111 and I112 are parked with
it**, since both exist only to serve it.

**The herdr detection spike passed** (deepthought, committed `836920d`). The
`/openweights-team` design carried an UNVERIFIED claim gating its herdr path,
with a stated fallback of "redesign the playbook, ship cmux alone". That
fallback is not needed.

**The spine ledger was reconciled** (committed `dfc573d`). It reported 32 open
tickets; 11 were not open. This mattered because every "what's next" read of
this repo — human or `/wayfinder` — trusts that list.

## State (verify before relying)

- **spine** `main` = **`dfc573d`**, docs-only. Verify push state — it was
  unpushed while this was written.
- **deepthought** `main` = **`836920d`**, **ahead 1, unpushed**. Pushing it is
  what turns a future blocked pull into an ordinary merge, since the second
  laptop may hold the same spec path.
- Ledger now: **89 fixed / 21 open / 2 wontfix**. Invariant verified — no
  `fixed` ticket blocked by an unclosed one, no dangling ids.
- `go test ./...` **exit 0** (18 packages). `spine audit stages` **exit 0**.
- `spine doctor` **exit 1**, on the same two D4 notes as always
  (`docs/issues/README.md` warn, `docs/adr/README.md` info). **Pre-existing —
  byte-identical to the baseline captured before any change today.** Filed as
  **I065**. Do not "fix" it as a side quest.
- Cursor: effort `i110-openweights-flavor`, all stages `[x]` through `docs`,
  handoff terminal, derivation clean. **Today's work was deliberately outside
  the cursor** — unticketed maintenance with no PRD, so no new effort was
  started. The next real feature batch should open its own effort.
- `~/bin/spine` is still built at the I110 commit; today changed no code.
- **`claude-auto` is not installed on this laptop** (searched PATH, shell
  configs, chezmoi). Second-laptop-only. No end-to-end openweights run is
  verifiable here.

### What closed, and what pointedly did not

Closed as `fixed` — each verified against the working tree, not assumed:
**I001** (epic; children I002–I006 fixed, `spine audit routing` live,
`WORKFLOW.md:76` mandates it at verify), **I040–I047**, **I049** (the codex
cluster). The tell was that their own acceptance ticket **I048** was already
`fixed` (2026-07-27, dated live-run evidence) while listing all of them in
`blocked-by` — an inconsistency that could not be true.

Closed as `wontfix`: **I064**, a duplicate of **I109**. The defect is real and
unfixed; I109 owns it. Cross-linked both ways.

Left open on purpose, having checked: **I032** (verified NOT shipped —
`stages.go:288` still passes `c.Tickets` where the fix says pass `""`, and
`maxNamedMissingIDs` appears nowhere in the test file), **I093** (its own note
records items 1–2 done; 3–5 need owner calls), **I065** (verified still live),
**I066** (a `wayfinder:map`; maps close only when children do, and I072–I078
are open).

**Carried hazard:** I047's Resolution records that its two D28 predicates
(`audit.go:401`, `:453`) are gated on **flavor**, not transcript source.
Closing I047 did not close that. **I111** owns it, no existing test covers it,
and it only bites once open-weights records are tagged — which is parked.

## Next steps

The owner's stated direction: **"a bunch of review and cleanup for the next
batch of features."** The frontier is now trustworthy enough to plan from.

1. **I109** (`med`) — the strongest cleanup candidate. A bare `strings.Index`
   scan in `parse()` (`internal/cursor/cursor.go:267`) matches the cursor
   marker anywhere, so quoting it in prose hijacks the block and turns
   `spine doctor` D9 and `spine audit stages` red against a pristine block. It
   has earned a standing "never write the literal marker" rule in every handoff
   — fixing it retires a recurring footgun. I064 carries the field
   reproduction.
2. **Doctor hygiene, as one batch** — **I065** (the permanent red; a gate
   nobody reads is not a gate) and **I106** (tolerate `batch:` / `workspace:` /
   `commits:` / `review:` ticket keys). Both small, both in the same surface.
3. **I032** (`mechanical`) — two accepted Minors, precisely specified, verified
   unshipped. Good filler.
4. **I093 items 3–5 need an owner call**, not a build — unconfigured
   gate classes rendering as exit-2 stages, `update --force` being
   all-or-nothing, D11 being shape- not value-evident.
5. Larger forward work, if the "next batch of features" means building: the
   I066 map's open children **I072** (host config schema; unblocks I073 and
   I077), **I074**, **I075**, **I076**, **I078**.

Frontier (open, every blocker closed): I007, I032, I050, I051, I065, I066,
I072, I074, I075, I076, I078, I093, I102, I105, I106, I108, I109, I111, I112.
Blocked: I073 and I077, both waiting on I072.

## Gotchas

### Learned this session

- **macOS bash is 3.2 — no `declare -A`, no `${var,,}`.** An associative-array
  script for the ledger invariant ran with an empty map and printed
  `none — consistent` plus a frontier listing `fixed` tickets. It looked like a
  clean pass. Anything non-trivial goes in `python3`; the working checker is
  worth rebuilding if needed (status/blocked-by parsed from frontmatter only).
- **A ledger's `status:` field is not evidence.** Ten tickets read `open` while
  their code was in the binary. Check the artifact — a named test file, the
  function, the CLI verb — before believing either direction.
- **herdr detects a `pane run` agent fine** (0.8.2, verified). Detection is a
  *screen-scrape manifest* (`live_prompt_box` / `live_blocked_form`), not
  spawn-time registration; `agent explain <pane-id>` names the matching rule.
  A pane-run agent starts **unnamed** — `herdr agent rename <pane-id> <name>`
  gives it the same session identity `agent start` confers.
- **A worker in an untrusted cwd starts `blocked`, not `idle`** — Claude Code's
  "do you trust this folder?" confirm. `agent prompt` refuses a blocked agent
  (`agent_blocked`) before sending input, so it stalls rather than misfires.
  Clear with `agent send-keys <target> enter`. This is what the allowlist seed
  exists for, and it bites hardest under an N>=2 worktree policy where every
  worker gets a fresh directory.
- **`herdr agent send-keys` takes a restricted key vocabulary** — `ctrl-c` is
  rejected `invalid_key`. Tear down via the pane or workspace.
- **herdr exit codes are honest** (1 on error, 0 on success) — an earlier
  reading of "exit 0 on failure" was the fish pipe-status trap below, not a
  herdr quirk.
- Use an isolated `herdr workspace create` for any live herdr experiment. The
  owner's main workspace runs a dozen real agents; never prompt one.

### Standing

- **Read exit codes unpiped.** fish reports the *last* pipeline command's
  status, so `cmd | tail; echo $status` hides failures. Use
  `bash -c '...; echo $?'`.
- **fish: quote globs** (`"--include=*.go"`) or use `bash -c`. An unquoted glob
  aborts the command with `no matches found`.
- **Never write the literal cursor marker in prose.** The parser finds the
  block by substring scan, so a quoted marker mid-sentence hijacks it. This is
  **I109** — until it is fixed, describe the marker, never type it.
- **Never tick the `handoff` stage.** It makes the handoff doc it describes a
  stale snapshot and blocks `spine audit stages`. Terminal state; recover with
  `spine cursor here handoff`.
- **`spine cursor start` refuses while an effort is mid-flight** — `--force` to
  supersede, and run it before `spine handoff new`.
- **`spine model` wants flags BEFORE positionals.** `spine model --effort
  openweights primary` works; trailing flags print usage and exit 2, which
  reads like a broken flavor.
- **The `implement` tick needs a ledger line starting with the ticket id AND
  containing `done`/`complete` as a whole word** (`implementDoneWordRe`,
  `internal/stages/stages.go`). The error blames the tickets field instead of
  the wording.
- **`.superpowers/sdd/progress.md` is gitignored** — evidence must land
  somewhere a reviewer sees before ticking.
- **The maipipe stop hook demands `maipipe run full --wait` whenever HEAD
  moves**, docs-only included. Batch commits so one lane run covers them. No
  daemon restart after `make install`.
- **A prescribed negative control is a hypothesis, not a fact.** Run both arms
  and record observed output. (Today's did discriminate: reverting I041 to
  `open` on a throwaway copy produced 4 violations and exit 1, including the
  exact `I048(fixed) blocked-by I041(open)` anomaly that existed before.)
- **Stage explicit paths only** — never `git add -A`. Two untracked files sit
  in these repos that are not ours: `docs/research/2026-08-26-fusion-harness-
  borrow-hitlist.md` (spine, another session) and
  `docs/research/2026-08-24-paperclip-steal-list.md` (deepthought).
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @ low`.

<!-- spine:cursor -->
effort: i110-openweights-flavor
prd: docs/specs/2026-08-25-openweights-flavor-spine-design.md
tickets: I110
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
