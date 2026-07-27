---
id: I043
title: Codex repo scoping — cwd inside repo or commit known to repo
severity: high
status: open
affects: [audit, I008, I009]
blocked-by: [I041]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
---

## What to build

Design D22. A codex session belongs to the audited repo iff its recorded cwd
resolves inside the repo, OR its session_meta git commit hash is a commit
known to the audited repo (one git object-existence probe per distinct
hash). This keeps worktree-cwd teams (/private/tmp team dirs) visible and
makes cross-repo ticket-token collision impossible unless repos share
history — ticket ids restart at I001 per repo across the estate, and
praxis/maipipe each carry their own I024.

A failing or absent git probe degrades to cwd-only plus a report warning,
never an error. Sessions outside scope are invisible to attribution — they
are not "unattributed", they simply do not exist for this audit.

## Acceptance criteria

- [ ] Session with cwd inside the repo is in scope; sibling repo's session with the same ticket token is not
- [ ] Worktree fixture (cwd outside repo, commit hash present in repo history) is in scope — tiny real git repos in test temp dirs
- [ ] Commit hash unknown to the repo + cwd outside → out of scope
- [ ] git probe failure degrades to cwd-only with a warning naming the degradation
- [ ] Cross-repo collision fixture: two repos, same ticket id, each audit sees only its own evidence
- [ ] `go test ./...` green

## Blocked by

- I041
