---
id: "0020"
title: "Dispatch-brief attribution reads the lead's transcript, never the file on disk"
status: Accepted
date: 2026-08-24
---

# 0020: Dispatch-brief attribution reads the lead's transcript, never the file on disk

## Context

I090 taught `spine audit routing` to recognize claude-team Bash spawns, but a
spawn is only *attributed* when a ticket id appears in the spawn command or in
the prompt command that follows it. The owner's real dispatch flow has neither:
the lead writes a **dispatch brief** to a file and starts the worker with the
brief delivered by reference — `herdr agent prompt lhc-implementer "$(cat
$WS/dispatch-task-2-implementer.md)"`. Measured live on the 2026-08-20
local-harness lead transcript, 27 of 27 spawns are recognized and none is
attributable, so the nine tickets that effort built (I079–I087) go unjudged.

I101 proposed resolving the referenced path against the session cwd and reading
the ticket id out of the file. Probing that against the corpus the ticket cites
recovers **0 of 27**: `.superpowers/` is gitignored, so no brief was ever
committed, and the `spine-wt-local-harness` worktree was removed when the effort
shipped. Attribution that depends on an uncommitted file in a deleted directory
decays to nothing on exactly the efforts it exists to judge.

The ticket ids are not lost, though. Every one of those briefs was written by a
heredoc — `cat > $WS/dispatch-task-2-implementer.md <<'EOF' … EOF` — inside a
Bash command in the lead's own transcript, which is durable. They are lost only
because I090's final review (C1) made the dispatch record carry the
*heredoc-stripped* command: a brief body names the ticket under work and
routinely several more "for context" (the I082 brief's body names I083, I084,
I085 and I086), so attributing on raw command text certified work nobody
dispatched as routed. C1 was right about the body and wrong only in throwing
away the first line with it.

Two further facts shaped the decision. Path-keyed recovery from recorded
heredocs scores **25 of 27**, where a cruder "first heredoc in the command"
rule scores 24 and disagrees with the path-keyed answer on four spawns —
disagreement being the misattribution this audit must never make. And the 27
spawns fail D28 repo qualification on their own text; only the brief body
carries `primary repo: /Users/ldh/Projects/github.com/spine`.

## Decision

A dispatch brief is evidence **as the lead's transcript recorded it being
written**. `spine audit routing` builds a per-session table of heredoc writes
(target path → body) while scanning a transcript, and when a spawn or its paired
prompt command references a path, resolves attribution from that table. It never
opens the referenced file, and never invokes a shell; paths are resolved
textually — variables expanded from `NAME=value` assignments recorded earlier in
the same session, made absolute against the session cwd, then cleaned.

Within a resolved brief, the split I090 C1 established is preserved by narrowing
rather than by stripping:

- the **first line** is the ticket-attribution text, matching the first-line
  rule ratified at I042 review for exactly this context-sentence bleed;
- the **whole body** may satisfy D28 repo qualification, mirroring the
  asymmetry `matches()` already applies to `d.prompt`.

A spawn resolves to the most recent recorded write of that path at or before its
own position in the session. Anything unresolvable — no recorded write, a path
that will not normalize, a first line naming no ticket — stays unattributed and
keeps today's visible unmatched listing. A spawn attributed this way discloses
the brief path the attribution came from.

Reading the file from disk is rejected, not deferred.

## Consequences

- The audit stays strictly read-only over transcripts. It gains no
  filesystem-reaching behavior, so no containment, symlink, size-cap or
  file-changed-since-dispatch rules are needed, and there is no window in which
  a brief edited after dispatch rewrites history.
- Attribution is reproducible: the same transcripts yield the same verdicts
  forever, independent of which worktrees still exist or what a gitignored
  directory currently holds.
- Efforts whose leads delivered briefs some other way — a `Write` tool call, a
  brief authored in an earlier session, a file that predates the transcript —
  are not recovered. They degrade to the honest unmatched listing rather than
  being guessed at. If such a corpus appears with evidence, it argues for a new
  source, not for reopening disk reads.
- I079–I087 become judged, and several reviewer spawns ran `claude-fable-5 @
  high` against `tier: routine` tickets, so quiet `unattributed-transcript`
  lines become loud `escalated-no-reason` ones. That is the fix working; the
  remedy for a wrong verdict is an `ESCALATION` record or `tier: n/a`, never a
  role-based carve-out from evidence.
- I090's C1 ruling stands for command text. This ADR narrows what "stripped"
  means for a *referenced* brief only; a heredoc body that nothing references
  is still not attribution text.
