---
id: I090
title: "spine audit routing is blind to herdr/cmux claude-team spawns (no-transcript for team-built tickets)"
severity: med
status: fixed
affects: [I002, I009, I078]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

`spine audit routing --dir ~/Projects/github.com/spine` (2026-08-19):

```
I079    routine     -    no-transcript    no dispatch or transcript evidence found
… (identical for I080–I087)
```

I079–I087 were built by a claude-team on herdr (lead fable-5 @ high, routine
implementers opus-5 @ low). The audit parses only Task/Agent tool-use blocks
in the lead's transcript; claude-team dispatches are Bash `herdr agent start
… --model <id>` (and the first `agent prompt` after), so a whole team-built
effort reports as unjudged. The routing contract's only enforcement is
therefore silent exactly in the mode (visible worker panes) the owner's
CLAUDE.md makes the default inside cmux/herdr.

## Fix

Treat `herdr agent start … --model <id>` / cmux equivalents inside Bash
tool-use blocks as dispatch records: extract model id (and `--effort` if
present), attribute to the ticket id named in the same command or the
following `agent prompt` text, then judge against the ticket's `tier` as
today. Positive control: a fixture transcript with one team spawn per
ticket; negative: a Bash block running `herdr` without `agent start` is not
a dispatch. Until then, document the blind spot in the audit's footer when
any Bash block mentions `herdr agent start`.

## Resolution (2026-08-20)

**What this delivers, precisely:** claude-team Bash spawns are recognized
and judged **when the spawn command or its following prompt command names
the ticket**. When the brief is delivered by file reference, the spawn is
surfaced as an unmatched dispatch with its model and effort, and counted in
a footer line — a stated gap instead of silence. Both halves of the ticket's
"fix" and "until then" clauses therefore shipped; the footer clause is not
moot.

`internal/audit/teamspawn.go` (new) recognizes `herdr agent start … --model
<id> [--effort <e>]` and a `cmux send --surface|--pane <id>` whose payload
invokes `claude … --model <id>`, plus the follow-up prompt commands
(`herdr agent prompt`, a non-claude `cmux send`) a spawn borrows ticket
attribution from when its own command names none. Recognition is confined to
command position, outside heredoc bodies, and outside trailing comments.
`parseLine` emits these as ordinary `dispatch` records; repo qualification,
attribution and the model-vs-tier judgement are the existing logic, unforked.
Effort is reported, never judged. `cmd/spine` prints `[model @ effort]` and
the footer count.

**Attribution reads only the heredoc-stripped command** (final-review C1).
A lead routinely writes the worker's brief and starts the worker in one Bash
call, and the brief names the ticket under work plus others "for context";
attributing on the raw text manufactured `match` verdicts for tickets nobody
dispatched. Fail-open was the one direction this audit must not fail in.

**Residual gaps, both filed:**

- **I101** — the owner's real dispatch flow inlines the brief
  (`herdr agent prompt lhc-implementer "$(cat $WS/dispatch-task-2-implementer.md)"`),
  so no ticket token appears in the command or the prompt. Against the real
  local-harness lead transcript, 27 of 27 spawns are recognized and none is
  attributable: I079–I087 remain `no-transcript`, now alongside a footer
  saying 27 spawns were seen. Closing that needs brief-file resolution.
- **I101 (related note)** — `DefaultTranscriptsDir` maps only the exact repo
  path, so a lead that ran in a worktree (`spine-wt-local-harness`) is not
  scanned by default discovery at all; the live check below needs an explicit
  `--transcripts`. Out of scope here, recorded for honesty.
- **I102** — `codex.go` carries a second recognizer for the same commands
  with different pairing semantics (accumulate-all vs first-prompt-only).
  Cross-referenced in both files; unification deferred.

`status: fixed` is kept on that narrowed reading: the blind spot the ticket
names — team spawns producing no dispatch record at all — is closed, and
what remains is a distinct, filed, and now-visible attribution gap.

## Evidence

- `go test ./internal/audit/ -run 'Team|I090|Attribute'` → all pass:
  `TestTeamSpawnsAreJudged`, `TestTeamSpawnPairsByWorkerHandle`,
  `TestTeamSpawnLookalikesAreNotDispatches`, `TestAttributeTeamPrompt`
  (7 rows), `TestParseTeamSpawn` (15 rows),
  `TestUnmatchedTeamSpawnCarriesEffort`.
- Load-bearing check `TestTeamSpawnRecognitionIsLoadBearing`: with
  `recognizeTeamSpawns = false` every team fixture ticket reports
  `no-transcript` — the pre-fix behaviour.
- One mutation per guard, every one caught:
  - raw command as the attribution text → `I504`/`I505` judged
    `escalated-no-reason`, want `no-transcript` (the C1 regression).
  - relax `isTool(…, "agent", "start")` → `I501` judged.
  - drop the claude-payload requirement → `I502` judged.
  - skip `stripHeredocBodies` → `I503` judged.
  - drop `--surface` from the handle flags → `I605` back to `no-transcript`
    and `cmux_send_addressing_a_surface` fails.
  - ignore the worker handle → `I601 actuals = "claude-fable-5", want
    "claude-sonnet-5"`; scan forward → `I604` silent-descent; drop
    first-prompt-only → `I603` judged; drop spawn-token-wins →
    `spawn 0 (impl-b) prompt = "now do I407", want ""`; drop the `#` guard →
    `spawn_parenthesized_inside_a_trailing_comment` fails.
- Live check against the real lead transcript (2026-08-20), the run that
  found C1:

  ```
  $ spine audit routing --dir ~/Projects/github.com/spine \
      --transcripts ~/.claude/projects/-Users-ldh-Projects-github-com-spine-wt-local-harness
  I079    routine     -    no-transcript    no dispatch or transcript evidence found
  … (identical for I080–I089)
  unmatched dispatches (no ticket id or not repo-qualified; not judged):
    herdr agent start lhc-implementer --kind claude --pane w19:p2 -- --permission-mode auto
      --model claude-opus-5 --effort low  [claude-opus-5 @ low]
    …
    note: 27 team spawn(s) recognised but unattributable (brief delivered via `$(cat file)`
    names no ticket in the command); see I101
  ```

  The manufactured `match` / `escalated-no-reason` rows on I079/I082–I086 are
  gone.
- `go vet ./...` clean; `make test` green across all 18 packages.

