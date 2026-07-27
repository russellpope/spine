---
id: I048
title: Codex audit — live acceptance against the estate, close I008/I009
severity: high
status: closed
affects: [audit, I008, I009]
blocked-by: [I041, I042, I043, I044, I045, I046, I047, I049]
execution-mode: inline
tier: primary
effort: xhigh
risk-triggers: []
review-tier: n/a
---

## What to build

Design "Live acceptance" + calibration notes. Run the finished audit against
the real session store and the real estate — this is the step that validates
synthetic fixtures against the undocumented format. Inline justification:
requires the live ~/.codex session store and operator-machine git state; no
per-task review cycle, verify-stage gates apply.

Expected honest outcomes (from the 2026-07-25 ground-truth investigation):

- moo-clone I024 → match (lead sol excluded, workers terra attributed)
- moo-clone M4a I008–I015 → unmapped-dispatch (gpt-5.5 was never a declared
  id; history reported honestly, non-blocking)
- moo-clone I021/I022 → match (sol on primary tickets)
- guardian threads contribute nothing anywhere
- praxis and maipipe audits unaffected by moo-clone's tokens and vice versa
- `tier: n/a` applied to moo-clone's pre-convention tickets kills the
  unannotated noise; empty-tier tickets stay loud

Close I008 and I009 with evidence (audit output quoted in the tickets),
update the issue ledger, and record any format surprise the live run
exposes as a new dated fact in I009 before closing.

## Acceptance criteria

- [x] Live run on moo-clone matches every expected outcome above, or each deviation is explained and ratified
- [x] Live run on praxis and maipipe shows no moo-clone contamination
- [x] A deliberately mis-scoped run (praxis tokens vs moo-clone) produces no false blocking verdict
- [x] I008 and I009 closed with quoted evidence; ledger updated
- [x] Fleet handoff notes the new flags and verdicts for operators (team report + handoff notes)

## Blocked by

- I041, I042, I043, I044, I045, I046, I047

## Closed 2026-07-27 — live acceptance results and ratified deviations

Expected outcomes, all hit after four live-driven wire-shape fix rounds
(each preceded by a dated I009 fact, each red/green-tested, rounds re-reviewed):
I024 match (lead sol excluded, terra workers + dispatch records attributed);
M4a I008-I015 unmapped-dispatch on gpt-5.5 (honest history, non-blocking);
I021/I022 match (sol); guardian threads contribute nothing anywhere (zero
codex-auto-review strings in any output); praxis exit 0 with its own I024
unannotated and uncontaminated despite the shared store (the mis-scope
proof); tier: n/a applied to moo-clone I001-I007 -> 7 exempt rows, post-
convention I100-I106 stay loud (edits left uncommitted in moo-clone for its
operator to ratify).

Format surprises found live, recorded dated in I009, and fixed on branch:
(1) exec_command arguments key `cmd`; (2) injected user-message preambles
(AGENTS.md / <recommended_plugins> / <hook_prompt>) ahead of the real brief;
(3) polymorphic session_meta.source (string on top-level sessions — this one
had hidden the ENTIRE store from the parser); (4) cmux leads dispatching via
custom_tool_call script text (orchestrator latch extended, evidence never
extracted from script blobs). Plus a ratified D21 second narrowing:
multi-token opening lines are ambiguous and attribute to none (single-line
briefs made "first line" the whole brief; a real I043 routine worker had
been blocking maipipe's primary I044).

RATIFIED DEVIATIONS (rows outside the expected table, all verified genuine
single-token worker sessions with correct cwd and no dispatch records):
moo-clone I023 + I035 and maipipe I060 judge silent-descent (blocking)
because their sessions' per-turn models include a lower-tier id (terra or
luna) alongside the declared-tier model — the mid-session model-switch/drift
phenomenon the design explicitly scopes OUT of interpretation (per-turn
evidence reports what ran). These are true findings, not attribution errors:
the audit now has the teeth it was built for. Remediation belongs to the
repo operators (FALLBACK/ESCALATION records where reruns were legitimate)
or to the deferred model-drift effort. Exit 1 on those repos is correct.

Near-miss visibility carry-forward (I044 review Minor 3): moot in practice —
M4a rows attribute via the worker scan (unmapped-dispatch), better than
unattributed; no live case surfaced where model-less spawn text was the only
signal.
