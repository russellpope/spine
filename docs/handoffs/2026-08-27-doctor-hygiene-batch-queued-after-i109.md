---
title: "doctor hygiene batch queued after I109"
created: 2026-08-27
handoff_ordinal: 24
---

# Handoff — doctor hygiene batch queued after I109 (2026-08-27)

## Context

**I109 is shipped, pushed, and closed** — see
`docs/handoffs/2026-08-26-i109-cursor-fence-anchoring-shipped.md` for its full
record. Cursor delimiters are now fences, recognized only as a whole line at
column 0, so a document may quote `<!-- spine:cursor -->` in prose and show a
complete block by indenting it. The standing "never write the literal marker"
rule is retired.

**The doctor hygiene batch (I065 + I106) is the chosen next work and has NOT
been started.** No effort cursor exists for it; the live cursor still belongs to
`i109-cursor-marker-anchoring` at `handoff[<]`, which is terminal and correct.

Reconnaissance was done before stopping, and it found three things that change
how the batch should be scoped. They are the reason this handoff exists.

### 1. I065's ticket text does not match spine's actual condition

The ticket says the unrecognized "local edit" is the **`tier`** bullet, from the
2026-08-10 estate sweep. On spine today it is the **`status`** bullet.
`docs/issues/README.md:8` reads:

    - `status` — open | in-progress | fixed | wontfix

while `templates/current/issues-README.md:8` reads:

    - `status` — open | in-progress | fixed | wontfix | superseded. `superseded` is

So spine's README is a generation behind on a **different line** than the ticket
describes, and it is missing `superseded` from the documented status vocabulary
entirely. Do not take the ticket's example as the live case — reproduce with
`spine update --dir .` first. (Spine was hand-reconciled once on 2026-08-09,
commit `0249f87`; it has drifted again since, which is itself evidence that
one-time reconciles do not hold.)

### 2. I106 edits the very file I065 is about

I106 adds four keys (`batch:`, `workspace:`, `commits:`, `review:`) to the
ledger convention — that is `docs/issues/README.md` plus `spine init`'s
scaffold. That is the same file whose stock-text drift is I065.

Sequencing is therefore a real design decision, not an ordering detail. Landing
I106 first creates new stock text that every estate repo must then migrate to,
re-arming the exact trap I065 exists to disarm. Landing I065 first means
building the known-stock machinery, then immediately adding a new generation of
stock text through it — which is arguably the correct proof that the machinery
works. **Grill this before building.**

### 3. I106's "warn, not block" still makes doctor exit 1

`cmd/spine/main.go:527` — `if f.Severity == "warn" || f.Severity == "error"` —
so **any warn drives exit 1**. I106 asks doctor to *warn* on a `workspace:` path
that does not exist and on a malformed `batch:`. That will keep doctor exiting 1
in ordinary use, which is precisely the condition I065 is meant to retire.

The two tickets pull against each other. Options, none yet chosen: make the new
I106 findings `info`; introduce a third severity that does not drive the exit
code; or accept that doctor's exit code means "look at me" rather than
"something is broken" and stop treating it as a gate. **This is an owner call.**

Note also that only the D4 **warn** on `docs/issues/README.md` drives the exit.
The second note, `docs/adr/README.md`, is severity **info** ("hand-authored file
preserved") and is harmless — fixing I065 should take doctor to exit 0 with that
info note still present. Verify that assumption rather than trusting it.

## State (verify before relying)

- **spine `main` = `5846c27`, pushed, in sync with origin.** Four commits this
  session: `8391b72` (I109 implementation), `fbd6ebd` (docs), `6efc443`
  (handoff), `5846c27` (handoff state correction).
- **maipipe full #39 passed @`5846c27`.** Nothing owed to the stop hook.
- `gofmt -l .` empty; `go vet ./...` exit 0; `go test ./...` exit 0 (18
  packages); `spine audit stages` **exit 0**; `spine audit routing` exit 0.
- `spine doctor` **exit 1** on two D4 notes — `docs/issues/README.md` (warn,
  drives the exit) and `docs/adr/README.md` (info). **This is I065 itself**, so
  for this batch it is the target, not background noise. It has been the
  standing "do not fix as a side quest" item in every prior handoff; that
  instruction no longer applies.
- `spine update --dir .` **exit 1**, skipping `docs/issues/README.md`. This is
  the cheapest live reproduction of I065.
- Ledger: **90 fixed / 21 open / 2 wontfix** (113 total). Frontier of 19;
  only `I073` and `I077` are blocked, both on `I072`.
- `~/bin/spine` built at `8391b72` and SHA-256-matched to a fresh `go build`.
  Nothing has changed code since.
- Cursor: `i109-cursor-marker-anchoring`, all stages `[x]` through `docs`,
  `handoff[<]`, `derivation: clean`.

## Next steps

1. **Grill the batch before building** — the gate is mandatory and there are
   three real questions: the I065/I106 sequencing above, the warn-versus-exit-1
   contradiction, and whether I065's scope is spine-only or the estate sweep the
   ticket describes (it names 11 repos; that is a much larger job than "fix the
   red locally").
2. `spine cursor start --force --effort <name> --tickets I065,I106 --prd <path>`
   **before** any `spine handoff new`.
3. Then `/grill-with-docs` → `/to-spec`. Both are `disable-model-invocation`;
   the model cannot invoke them, the owner must type them.
4. Dispatch a cold reviewer and a fresh-context verifier at the end. On I109 the
   cold gates found three real defects the build missed, including a regression
   the build introduced.

**Highest-severity open item, and it is parked:** `I111` (`high`) is the only
non-`med`/`low` ticket on the frontier. It owns the hazard carried from I047 —
the two D28 predicates at `internal/audit/audit.go:401` and `:453` gate on
**flavor**, not transcript source, with no test covering it. It only bites once
open-weights records are tagged, which is the parked programme. Do not start
openweights work (I111, I112) unless the owner reopens it.

Other frontier candidates: **I113** (the I109 follow-on — trailing whitespace
after the closing fence escapes the `NonCanonical` guard, pre-existing);
**I032** (mechanical, precisely specified, verified NOT shipped); **I072** (host
config schema, the only frontier ticket that unblocks others).

## Gotchas

### Learned this session

- **A testdata tree containing a `.superpowers/` path needs `git add -f`.** The
  repo gitignore swallows it, so the test passes locally off the on-disk file and
  fails on a fresh clone. The tell is the staged-file count against the fixture
  you copied from. Prove it with `git archive HEAD | tar -x` into a temp dir and
  run the suite there.
- **`spine init` needs an explicit `--profile library-cli`** in a bare directory,
  and **`spine cursor tick` wants `--dir` BEFORE the positional stage.** Same
  shape as `spine model`. Getting it wrong produced clean-looking, meaningless
  functional output.
- **`spine audit routing` judges a dispatch against `tier`, not `review-tier`.**
  Correctly dispatching review at primary for a `tier: routine` ticket reads as
  silent escalation until an `ESCALATION` line exists. Advisory, not blocking —
  which is how it quietly erodes.
- **A handoff cannot describe its own commit.** The I109 handoff's State block
  was accurate when written and false minutes later (it claimed unpushed, lane
  not run). Correct it in a follow-up commit rather than leaving the next session
  instructions to redo finished work.
- **`grep -c ""` counts lines**; `grep -c <pattern>` counts matches. Do not
  confuse them when sizing a file.

### Standing

- **Read exit codes unpiped.** fish reports the *last* pipeline command's status.
  Use `bash -c '...; echo $?'`. This still bit inside a `bash -c` block reading
  `$?` after a pipe.
- **Heredocs are unreliable here** — a `python3 - <<PY` block failed with
  `(eval):27: unmatched "`. Write the script to the scratchpad and run the file.
  Project rule says use Write/Edit for file content anyway, never shell heredocs.
- **macOS bash is 3.2** — no `declare -A`, no `${var,,}`. Use `python3`.
- **fish: quote globs** (`"--include=*.go"`) or use `bash -c`.
- **Never tick the `handoff` stage.** Terminal; recover with
  `spine cursor here handoff`.
- **`spine cursor start` refuses mid-flight** — `--force` to supersede, and run
  it before `spine handoff new`.
- **The `implement` tick needs a ledger line starting with the ticket id AND
  containing `done`/`complete` as a whole word.**
- **`.superpowers/sdd/progress.md` is gitignored** — evidence must land where a
  reviewer sees it.
- **A prescribed negative control is a hypothesis, not a fact.** Run both arms.
- **A ledger's `status:` field is not evidence** — check the artifact.
- **Stage explicit paths only.** `docs/research/2026-08-26-fusion-harness-borrow-
  hitlist.md` is untracked and **not ours**.
- **The maipipe stop hook wants `maipipe run full --wait` whenever HEAD moves**,
  docs-only included. Batch commits so one run covers them.
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @ low`.
- **Retired:** the "never write the literal cursor marker" rule. Write about the
  cursor freely; indent complete examples.

<!-- spine:cursor -->
effort: i109-cursor-marker-anchoring
prd: docs/specs/2026-08-26-cursor-marker-anchoring-design.md
tickets: I109
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
