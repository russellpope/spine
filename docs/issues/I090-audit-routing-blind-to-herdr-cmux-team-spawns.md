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

`internal/audit/teamspawn.go` (new): recognizes `herdr agent start … --model
<id> [--effort <e>]` and a `cmux send` whose payload invokes `claude …
--model <id>` inside a Bash tool_use block's command text, and the follow-up
prompt commands (`herdr agent prompt`, a non-claude `cmux send`) a spawn
borrows ticket attribution from when its own command names no ticket.
Recognition is confined to command position and to text outside heredoc
bodies, so a query command, a non-claude send, and a spawn quoted in prose
are not dispatches. `parseLine` emits these as ordinary `dispatch` records
(plus the extracted `effort`, surfaced on unmatched dispatches) — repo
qualification, attribution and the model-vs-tier judgement are the existing
logic, unforked. The interim footer disclosure the ticket proposed was not
needed: the blind spot is closed rather than documented.

Fixtures: `testdata/team` (three team-built tickets — herdr spawn attributed
by the following prompt → match; herdr spawn naming its ticket, routine model
on a `tier: primary` ticket → silent-descent; cmux send → match) and
`testdata/teamnoise` (one lookalike command per negative bullet, all
no-transcript). Tests in `internal/audit/i090_test.go`.

## Evidence

- `go test ./internal/audit/ -run 'I090|Team' -v` → all pass, including
  `TestTeamSpawnsAreJudged`, `TestTeamSpawnLookalikesAreNotDispatches`,
  `TestParseTeamSpawn` (12 subtests).
- Load-bearing check `TestTeamSpawnRecognitionIsLoadBearing`: with
  `recognizeTeamSpawns = false`, all three team fixture tickets report
  `no-transcript` — the pre-fix behaviour.
- Negative controls, one mutation each, all caught:
  - relax `isTool(f, "herdr", "agent", "start")` to `…"agent"` → `I501
    verdict = escalated-no-reason …, want no-transcript`.
  - drop the `claude`-payload requirement in `cmuxClaudeSend` → `I502
    verdict = escalated-no-reason …, want no-transcript`.
  - skip `stripHeredocBodies` → `I503 verdict = escalated-no-reason …, want
    no-transcript`.
- `go vet ./...` clean; `make test` green across all 18 packages.
