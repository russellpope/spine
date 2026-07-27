---
id: "0012"
title: "FALLBACK records excuse the fallback reading of shared-id tokens"
status: Accepted
date: 2026-07-26
---

# 0012: FALLBACK records excuse the fallback reading of shared-id tokens

The codex table deliberately declares the same id on `codex.routine` and
`codex.fallback` (ADR 0011 / D10: the codex fallback is "re-run on codex"), so
an observed terra token on an above-routine ticket is ambiguous: ordered-tier
reading (silent descent, blocking) or fallback reading (lateral, excusable).
The original resolution rule always preferred the ordered reading — which
turns a properly recorded refusal-rerun into a standing false blocker, with no
in-band remediation because records cannot excuse descent by design.

Decided 2026-07-26 (codex-audit grill): **when the audited ticket carries a
FALLBACK record, an ambiguous token whose candidates include fallback resolves
as fallback** and judges escalated-with-reason (advisory). Record consultation
therefore moves *before* tier resolution — a ledger record participates in
deciding what a token means, not just how the verdict reads. Without a record,
the ordered reading stands and real descent still judges silent-descent.

This trusts operator-authored records, consistent with the existing posture
(an ESCALATION record excuses exactly its to-tier). Rejected alternatives:
always-ordered (recurring false blockers on every legitimate refusal-rerun,
the failure class I008 showed is the tool's worst); a new ambiguous-fallback
verdict (more information, but every downstream gate must learn a value whose
meaning is "the audit declined to decide").
