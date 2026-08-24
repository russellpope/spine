---
id: I101
title: "spine audit routing cannot attribute a team spawn whose brief is delivered by file reference"
severity: med
status: fixed
affects: [I090, I047]
blocked-by: []
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: [cross-task-integration]
review-tier: primary
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

**Amended 2026-08-24 (grill; ADR 0020).** This section originally said to
resolve the referenced path and read the ticket id *out of the file on disk*.
Measured against the corpus this ticket cites, that recovers **0 of 27**:
`.superpowers/` is gitignored so no brief was ever committed, and the
`spine-wt-local-harness` worktree was removed when the effort shipped. The
ticket named a fix that cannot satisfy its own acceptance evidence. Disk
reading is rejected, not deferred.

The ticket ids survive in the lead's transcript: every brief was written by a
heredoc (`cat > $WS/dispatch-task-2-implementer.md <<'EOF' … EOF`) inside a
Bash command, and I090's C1 round strips exactly those bodies.

Attribute from the brief **as the transcript recorded it being written**. While
scanning a session, build a table of heredoc writes (normalized target path →
body). When a spawn or prompt command references a path (`$(cat <path>)`,
`--brief <path>`, a bare `.md` argument), resolve it against that table:
variables expanded literally from `NAME=value` assignments recorded earlier in
the same session, made absolute against the session cwd, cleaned. The brief's
**first line** supplies the ticket token (the I042 first-line rule — the I082
brief's body names four other tickets "for context"); the **whole body** may
satisfy D28 repo qualification, without which none of the 27 qualifies. A spawn
resolves to the most recent write at or before its own position. Path-keyed
recovery scores **25 of 27**; the two misses have no prompt command at all.

Fold in the discovery gap below: union `git worktree list` with a
`<repo-slug>-*` scan of `~/.claude/projects`. `git worktree list` alone does not
find it — the worktree is gone — and the prefix scan is safe because D28 gates
what it sweeps in.

Constraints: the audit is read-only and must never execute transcript text —
textual path resolution only, no shell invocation, and no opening of the
referenced path. Anything unresolvable degrades to today's unmatched listing.
Never guess: a wrong attribution certifies unrouted work as compliant, which
is the failure mode I090's C1 review round exists to prevent.

Spec: `docs/specs/2026-08-24-i101-dispatch-brief-attribution-design.md` (+ plan).
Decision: ADR 0020.

## Verification

Verified 2026-08-24 on the integrated branch at `a716e0a`:

- `gofmt -l .` and `go vet ./...` were clean; `SPINE_REQUIRE_MAIPIPE=1 make test` passed.
- After `make install`, ran from the primary repo root with default transcript
  discovery and no `--transcripts` override:

  ```text
  spine audit routing --session a39329db-3a0d-4dd3-bc4d-6217f0c3509b
  ```

  The worktree-union scan included the removed local-harness slug directory.
  The I079–I087 rows disclose **25 unique** `dispatch-*.md` brief sources.
  The raw row list has 27 source occurrences because
  `dispatch-final-fixer.md` and `dispatch-final-reviewer.md` each appear for
  both I079 and I087; deduplicating those two repeated paths gives 25
  attributed starts.
- The scoped unmatched footer reports two remaining team spawns, both lacking
  a ticket-bearing paired prompt:

  ```text
  herdr agent start spine-wt-local-harness-implementer --kind claude --pane w19:p2 -- --permission-mode auto --model claude-opus-5 --effort low
  WS=.superpowers/sdd/2026-08-18-local-harness-conventions-plan; SK=/Users/ldh/.claude/plugins/cache/claude-plugins-official/superpowers/6.3.0/skills/subagent-driven-development; git log --oneline edc4588..HEAD; H=$(git rev-parse --short HEAD); git status --short | head -3; $SK/scripts/review-package docs/specs/2026-08-18-local-harness-conventions-plan.md edc4588 HEAD …
  ```

  Thus the cited corpus measures **25 attributed + 2 promptless unmatched =
  27 starts**.
- I079–I087 changed from unjudged to five `match` rows (I081–I084, I086) and
  four `escalated-no-reason` rows (I079, I080, I085, I087). The latter are the
  expected `claude-fable-5 @ high` / primary-model evidence against routine
  tickets with no `ESCALATION` record; the routing records were left intact.

## Related

- I090 — the recognizer this builds on.
- `DefaultTranscriptsDir` maps only the exact repo path, so a lead that ran
  in a worktree (`spine-wt-local-harness`) is not scanned by default
  discovery and needs an explicit `--transcripts`. Separate gap, same
  practical effect on team-built efforts; worth folding into this work.
