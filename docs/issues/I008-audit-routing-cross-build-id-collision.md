---
id: I008
title: audit routing — cross-build ticket-id collision in shared controller project dirs
severity: med
status: open
affects: []
blocked-by: []
execution-mode:
tier:
effort:
risk-triggers: []
review-tier:
---

## Problem

Found live during the praxis I001 build (2026-07-10) — the gen-6 acceptance
exercise. `spine audit routing` correlates dispatches to tickets by ticket-id
token across EVERY session JSONL in the transcript dir. Controller sessions
for different builds often share one project dir (both the spine
model-routing build and the praxis serialization-retry build ran from
deepthought), and ticket ids restart at I001 per repo — so yesterday's spine
"I003: gen-6 dispatch contract templates" dispatch (sonnet, correct FOR THAT
BUILD) matched praxis's unrelated I003 (tier: primary) and produced a FALSE
BLOCKING silent-descent verdict (exit 1) plus two false escalated-no-reason
advisories. False silent-descent is the worst failure class for this tool:
records cannot excuse descent by design, so there is no in-band remediation —
the operator must hand-scope transcripts (the workaround used: copy the
build's session files to a scratch dir and pass --transcripts).

## Fix

Give the audit build scoping. Candidate mechanisms (pick at design time):

- `--session <id>` / `--since <time>` filters on the transcript set — minimal,
  operator-driven, matches the existing --transcripts escape-hatch philosophy.
- Repo-qualified correlation: only count a dispatch when its description's
  ticket id exists in the audited repo's ledger AND the session touched that
  repo (e.g. cwd/tool-path evidence in the same transcript) — automatic but
  heuristic.
- Build-ledger anchoring: read the audited repo's .superpowers/sdd/progress.md
  `Started:` date and scope transcripts to sessions at/after it — zero new
  flags, uses data the contract already mandates.

Whichever lands, a silent-descent verdict should name the source session file
in its detail line so a false positive is diagnosable in one glance rather
than via manual transcript grepping.

Related deferred minors from the gen-6 build: C4 (transcript-slug
derivation), C5 (last-FALLBACK-reason-wins) — same audit surface, batch if
convenient.
