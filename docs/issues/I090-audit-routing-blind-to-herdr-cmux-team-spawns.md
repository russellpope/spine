---
id: I090
title: "spine audit routing is blind to herdr/cmux claude-team spawns (no-transcript for team-built tickets)"
severity: med
status: open
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
