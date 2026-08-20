---
id: I095
title: "spine audit routing cannot attribute a team spawn whose brief is delivered by file reference"
severity: med
status: open
affects: [I090]
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

I090 taught the audit to recognize claude-team Bash spawns, but attribution
still needs a ticket id in the spawn command or in the prompt command that
follows it. The owner's real dispatch flow has neither: the lead writes a
brief file, then starts the worker with the brief inlined by shell
substitution, e.g.

```
herdr agent prompt lhc-implementer "$(cat $WS/dispatch-task-2-implementer.md)"
```

The transcript records the command, not what `cat` expanded to, so the
ticket id lives only in a file on disk. Measured against the real lead
transcript for the local-harness effort (2026-08-20): **27 of 27** spawns
are recognized and none is attributable — every one lands in the audit's
unmatched list with its model and effort, and the footer says so.

That is a visible gap rather than a silent one (and strictly better than the
pre-I090 `no-transcript`), but it means the motivating tickets I079–I087
still go unjudged.

## Fix

Attribute from the brief file: when a spawn or prompt command references a
readable path (`$(cat <path>)`, `--brief <path>`, a bare `.md` argument),
resolve it against the session cwd and read the ticket id out of the file.
Shell variables the transcript does define (`WS=…` earlier in the same
command) can be expanded literally; anything unresolvable stays unattributed.

Constraints: the audit is read-only and must never execute transcript text —
path resolution only, no shell invocation. A file that no longer exists, or
resolves outside the audited repo, degrades to today's unmatched listing.
Never guess: a wrong attribution certifies unrouted work as compliant, which
is the failure mode I090's C1 review round exists to prevent.

## Related

- I090 — the recognizer this builds on.
- `DefaultTranscriptsDir` maps only the exact repo path, so a lead that ran
  in a worktree (`spine-wt-local-harness`) is not scanned by default
  discovery and needs an explicit `--transcripts`. Separate gap, same
  practical effect on team-built efforts; worth folding into this work.
