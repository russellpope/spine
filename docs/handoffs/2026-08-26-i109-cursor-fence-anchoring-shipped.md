---
title: "I109 cursor fence anchoring shipped"
created: 2026-08-26
handoff_ordinal: 23
---

# Handoff — I109 cursor fence anchoring shipped (2026-08-26)

## Context

**I109 shipped.** The cursor block was found by a bare substring scan, so its
delimiters matched anywhere — mid-sentence, inside backticks. A handoff that
quoted the opening delimiter in prose hijacked its own parse, and
`spine audit stages` went red against a byte-perfect snapshot. The workaround
was a standing rule in every handoff's gotchas: never write the literal marker.
That is not a rule a tool for documenting conventions can ask its authors to
keep.

Delimiters are now **fences** — recognized only as a whole line starting at
column 0. This sentence contains `<!-- spine:cursor -->` inline, and the
indented example below is a complete block; neither is a fence, and this
document parses clean. That is the fix, demonstrated by the document you are
reading rather than asserted:

    <!-- spine:cursor -->
    effort: <kebab-name>
    prd: docs/specs/<file>.md
    <!-- /spine:cursor -->

**The standing rule is retired.** Write about the cursor freely. To show a
*complete* block, indent it, as above.

## State (verify before relying)

- **spine `main` = `6efc443`, PUSHED and in sync with origin.** Three commits:
  `8391b72` implementation, `fbd6ebd` docs, `6efc443` this handoff. (This block
  was written before the push and corrected immediately after — a handoff cannot
  describe its own commit, so verify with `git status --short --branch`.)
- `gofmt -l .` empty, `go vet ./...` **exit 0**, `go test ./...` **exit 0**
  (18 packages). `spine audit routing` **exit 0**, I109 `escalated-with-reason`.
- **`maipipe run full #38 PASSED @`6efc443`.** One run covered all three commits,
  batched deliberately because the stop hook wants a run whenever HEAD moves.
- `~/bin/spine` rebuilt at `8391b72` and SHA-256-matched against a fresh
  `go build`. It carries the CRLF fix; the earlier install did not.
- `spine doctor` **exit 1** on the two long-standing D4 notes
  (`docs/issues/README.md` warn, `docs/adr/README.md` info). **Pre-existing,
  I065.** Not a side quest.
- Ledger: I109 `fixed`, **I113 filed** (open). Ticket count 90 fixed / 21 open /
  2 wontfix.
- Cursor: effort `i109-cursor-marker-anchoring`, all stages `[x]` through
  `docs`, handoff terminal.

### What the gates caught that the build did not

Three findings came from the cold gates, and all three were real:

1. **Reviewer:** `Save` refused more broadly than decision D8 authorized. Owner
   ratified widening D8; code stands.
2. **Verifier:** a **regression this change introduced** — the scanner trimmed
   only space and tab, so a CRLF document's fence line (`<tag>\r`) matched
   nothing and a valid block reported as *missing*. The substring scan it
   replaced handled CRLF. Fixed here with its own test and both control arms.
   The verifier recommended deferring it; that was declined, because a
   regression introduced by a diff does not belong in a follow-up ticket.
3. **Ship:** the new fixture's `.superpowers/sdd/progress.md` silently did not
   stage — `.gitignore:2` ignores `.superpowers/`. The test passed locally off
   the on-disk file and would have **failed on a fresh clone**. Caught by
   comparing the staged-file count against the fixture it was copied from.

Three owner-ratified spec amendments came out of review (story 9, D4, D8), each
recorded inline in the PRD with its reason. No code changed for any of them.

## Next steps

Nothing is left over from I109 — the lane passed and main is pushed. Pick from:

1. **I113** (`low`, `mechanical`) — the natural follow-on, found by I109's verify
   probe and confirmed **pre-existing** by reading the original source: trailing
   whitespace after the *closing* fence escapes the `NonCanonical` comparison,
   so one hand-edit shape passes the sole-writer guard. Opening-fence padding is
   caught; closing is not. `scanFences` already computes the boundary the fix
   needs.
2. **Doctor hygiene as one batch** — I065 (the permanent red; a gate nobody reads
   is not a gate) + I106 (tolerate `batch:`/`workspace:`/`commits:`/`review:`
   keys). Both small, same surface.
3. **I032** (`mechanical`) — two accepted Minors, verified NOT shipped:
   `stages.go:288` still passes `c.Tickets` where the fix says pass `""`, and
   `maxNamedMissingIDs` appears nowhere in the test file.
4. **I093 items 3–5 need an owner call**, not a build.
5. Larger forward work: the I066 map's open children — **I072** (host config
   schema; unblocks I073 + I077), I074, I075, I076, I078.

Frontier: I007 I032 I050 I051 I065 I066 I072 I074 I075 I076 I078 I093 I102 I105
I106 I108 I111 I112 **I113**. Blocked: I073, I077 (both on I072).

**Still parked:** the openweights programme. I111 and I112 park with it. Do not
start it unless the owner reopens it. I111 still owns the carried hazard —
I047's two D28 predicates (`internal/audit/audit.go:401` and `:453`) gate on
**flavor**, not transcript source, and no test covers it.

## Gotchas

### Learned this session

- **A testdata tree containing a `.superpowers/` path needs `git add -f`.** The
  repo-level gitignore swallows it; the older `clean` fixture is tracked only
  because it was force-added once, after which gitignore stops applying. The
  cheap tell is the staged-file count against the fixture you copied. Prove it
  with `git archive HEAD | tar -x` into a temp dir and run the suite there —
  that is what distinguishes "passes here" from "ships".
- **`spine init` needs an explicit `--profile library-cli`** in a bare
  directory, and **`spine cursor tick` wants `--dir` BEFORE the positional
  stage**. Same flags-before-positionals shape as `spine model`. Getting it
  wrong produced clean-looking, meaningless functional output.
- **`spine audit routing` judges a dispatch against `tier`, not
  `review-tier`.** A ticket annotated `tier: routine` / `review-tier: primary`
  reads as silent escalation when you correctly dispatch review at primary, until
  an `ESCALATION` line exists. Advisory, not blocking — which is how it quietly
  erodes.

### Standing

- **Read exit codes unpiped.** fish reports the *last* pipeline command's status.
  Use `bash -c '...; echo $?'`. This bit again inside a `bash -c` block that read
  `$?` after a pipe.
- **macOS bash is 3.2** — no `declare -A`, no `${var,,}`. Use `python3`.
- **fish: quote globs** (`"--include=*.go"`) or use `bash -c`.
- **Never tick the `handoff` stage.** Terminal; recover with
  `spine cursor here handoff`.
- **`spine cursor start` refuses mid-flight** — `--force` to supersede, and run
  it before `spine handoff new`.
- **The `implement` tick needs a ledger line starting with the ticket id AND
  containing `done`/`complete` as a whole word.** The error blames the tickets
  field instead of the wording.
- **`.superpowers/sdd/progress.md` is gitignored** — evidence must land where a
  reviewer sees it.
- **A prescribed negative control is a hypothesis, not a fact.** Run both arms.
- **A ledger's `status:` field is not evidence** — check the artifact.
- **Stage explicit paths only.** `docs/research/2026-08-26-fusion-harness-borrow-
  hitlist.md` is untracked and **not ours** — another session's.
- Owner ban: never route to `claude-sonnet-5`; substitute `claude-opus-5 @ low`.
- **Retired this session:** the "never write the literal cursor marker" rule.
  I109 fixed the parser instead.

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
