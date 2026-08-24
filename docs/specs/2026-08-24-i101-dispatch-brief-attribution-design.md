# I101 dispatch-brief attribution

Status: approved (grilled 2026-08-24)
Date: 2026-08-24
Tickets: I101
Decision record: ADR 0020; respects ADRs 0001, 0002; refines I090's C1 ruling

## Problem Statement

`spine audit routing` recognizes claude-team Bash spawns (I090) but attributes
one only when a ticket id appears in the spawn command or the prompt command
that follows it. The owner's real dispatch flow has neither: the lead writes a
**dispatch brief** to a file and starts the worker with the brief delivered by
reference — `herdr agent prompt lhc-implementer "$(cat
$WS/dispatch-task-2-implementer.md)"`.

Verified live at `25eb985` against the 2026-08-20 local-harness lead transcript:
27 of 27 spawns are recognized, none is attributable, every one lands in the
unmatched list, and the footer points at this ticket. The nine tickets that
effort built — I079–I087 — go unjudged, which is the routing contract's only
enforcement being silent in the owner's default working mode.

Two things make the naive fix wrong. Reading the brief from disk recovers **0
of 27**: `.superpowers/` is gitignored, so no brief was committed, and the
`spine-wt-local-harness` worktree was removed when the effort shipped. And the
audit cannot reach that transcript directory by default at all — the effort ran
in a worktree, and `DefaultTranscriptsDir` maps only the exact repo path.

## Solution

A dispatch brief is evidence **as the lead's transcript recorded it being
written**. While scanning a session the reader builds a table of heredoc writes
(normalized target path → body). When a spawn or its paired prompt command
references a path, attribution resolves from that table: the brief's first line
supplies the ticket token, its whole body may satisfy D28 repo qualification,
and the report discloses the brief path the attribution came from. The audit
never opens the referenced file and never invokes a shell.

Transcript discovery widens to the union of `git worktree list` and a
slug-prefix scan of `~/.claude/projects`, so an effort built in a worktree —
including one since removed — is scanned without a hand-passed `--transcripts`.

Measured on the same corpus, path-keyed recovery from recorded heredocs
attributes 25 of the 27 spawns. The remaining two have no prompt command at all
and stay honestly unmatched.

## User Stories

1. As a repo owner who dispatched a ticket through a claude-team lead with the
   brief delivered by `$(cat …)`, I want that spawn attributed to the ticket,
   so that the model it ran on is judged against the ticket's tier instead of
   listed as unmatched.
2. As a repo owner, I want attribution to come from what my lead's transcript
   recorded, so that a verdict does not change or vanish because a worktree was
   removed or a gitignored file was rewritten.
3. As a repo owner, I want a brief that names other tickets "for context" to
   attribute only the ticket its first line names, so that work nobody
   dispatched is never certified as routed.
4. As a repo owner, I want a spawn whose brief cannot be resolved to stay in
   today's unmatched listing, so that under-attribution stays visible and is
   never replaced by a guess.
5. As a repo owner reading a verdict, I want the brief path the attribution
   came from disclosed on the line, so that a wrong attribution is findable
   rather than invisible.
6. As a repo owner whose effort ran in a worktree, I want `spine audit routing`
   to find that lead's transcripts by default, so that the fix is reachable
   without knowing the `~/.claude/projects` slug convention.
7. As a repo owner, I want the scanned transcript directories named in the
   report's warnings, so that the audit's scope is stated rather than guessed.
8. As a developer, I want the audit to remain read-only over transcripts with
   no filesystem-reaching behavior, so that no containment, symlink or
   time-of-check rules are needed to trust it.

## Implementation Decisions

**D29 — brief table.** While scanning one session, record every heredoc write
in a Bash `tool_use` command as (target path → body). Target is the argument of
`cat >` / `cat >>`; `>>` appends to an existing entry. The table is
**per-session**: a brief written in one orchestrator session never attributes a
spawn in another.

**D30 — path normalization.** A referenced path is resolved textually in three
steps: expand `$NAME` / `${NAME}` from `NAME=value` assignments recorded earlier
in the same session (last write before this point wins); make it absolute
against the session cwd if relative; `path.Clean`. Table keys are normalized the
same way. No shell is ever invoked. A path that still contains an unexpanded
variable does not resolve.

**D31 — reference forms.** A spawn or prompt command references a brief via
`$(cat <path>)`, `--brief <path>`, or a bare argument ending in `.md`.
Recognition stays in command position under the existing `commandSegments` /
`segmentFields` discipline.

**D32 — temporal rule.** A spawn resolves a path to the most recent recorded
write of that path **at or before** the spawn's own position in the session. A
later rewrite of the same path never attributes an earlier spawn.

**D33 — first line attributes, body qualifies.** The brief's first line is the
ticket-attribution text (the I042 first-line rule). The whole body may satisfy
D28 repo qualification. This mirrors the asymmetry `matches()` already applies:
`firstLine(d.prompt)` for the token, full text for `repoQualifies`.

**D34 — precedence.** A spawn that already names a ticket in its own command
keeps that attribution (I090's rule, unchanged). Otherwise, if the paired prompt
command resolves a brief, the brief supersedes the prompt command's raw text as
the spawn's attribution source. An unresolvable reference falls back to today's
prompt-text behavior.

**D35 — disclosure.** A verdict whose evidence came from a resolved brief names
the brief path on the line, in the style of the codex reader's `source:` notes.

**D36 — transcript discovery.** `DefaultTranscriptsDir` grows a sibling that
returns the union of: the repo's own slug directory; slug directories for every
path in `git worktree list`; and slug directories matching `<repo-slug>-*`. A
`git` that fails or a repo that is not a git checkout degrades to the slug scan.
The union is safe because D28 already gates every record it sweeps in: a sibling
repo caught by the prefix fails repo qualification and its dispatches land
unmatched. Scanned directories are named in `rep.Warnings`.

**D37 — footer.** The unmatched footer's `see I101` pointer is removed; the note
keeps its count and reason.

## Testing Decisions

- Tests drive the audit package's public `Run` and the CLI seam against real
  temp repos and hand-built JSONL fixtures. No source-text assertions.
- A team fixture mirroring the real shape: a lead session whose Bash command
  writes `dispatch-task-1-implementer.md` by heredoc — first line naming the
  ticket and the repo path, body naming three other tickets — then starts a
  worker and prompts it with `$(cat $WS/…)`. Asserts exactly one ticket
  attributed, the other three untouched, and the brief path disclosed.
- Positive control for D32: the same path rewritten later in the session with a
  different ticket; the earlier spawn keeps the earlier body.
- Positive control for D33: a brief whose first line names the ticket but not
  the repo, and whose body names the repo — attributed.
- Degradation cases: unresolvable variable, no recorded write, first line with
  no ticket, `--brief` pointing outside any recorded write. Each stays unmatched
  with today's listing.
- **Negative control** (mirroring I090's `recognizeTeamSpawns`): a package var
  gating brief resolution, flipped off in one test to prove the team fixture
  falls back to unattributed. The fix must be load-bearing.
- D36 gets its own test over a temp `HOME` with slug directories for a repo, a
  live worktree and a removed worktree, plus a decoy sibling repo whose records
  must land unmatched via D28.
- Verification evidence: re-run `spine audit routing` against the live
  2026-08-20 corpus and record the attributed count in the ticket.

## Out of Scope

- Reading a brief from disk, in any form. Rejected on evidence (ADR 0020), not
  deferred.
- Briefs delivered by a `Write` tool call or authored in an earlier session.
- I102's unification of team-spawn pairing across flavors; this work leaves the
  claude/codex pairing difference exactly as it stands.
- Re-litigating the resulting I079–I087 verdicts. Attribution is the deliverable;
  an unwanted verdict is answered with an `ESCALATION` record or `tier: n/a`.

## Further Notes

I090's C1 ruling stands for command text: a heredoc body that nothing references
is still not attribution text. This work narrows "stripped" for a *referenced*
brief only, and narrows it to one line.
