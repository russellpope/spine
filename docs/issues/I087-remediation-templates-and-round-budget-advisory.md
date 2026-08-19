---
id: I087
title: "Remediation templates (hitlist, round record) + round-budget audit advisory"
severity: med
status: fixed
affects: [templates, stages, audit]
blocked-by: [I085]
execution-mode: subagent-driven
tier: routine
effort:
risk-triggers: []
review-tier: routine
labels: [ready-for-agent]
parent: local-harness-conventions
---

## Parent

Spec: `docs/specs/2026-08-18-local-harness-conventions-design.md`
(Remediation). ADR 0007 (rescore rule lives in the eval seam). Estate build
order places this in Phase 3; it is unblocked as soon as I085 ships.

## What to build

A remediation author has spine-shipped templates and one advisory audit
rule. Embedded templates: `hitlist.tmpl.md` (header: effort, round, dose,
source run id; per finding: `code`, file:line, finding, why-it-matters,
do-not-regress block listing killed mutation rows by `code`; **no fix
text** — the default dose is findings-without-fixes) and
`remediation-round.tmpl.md` (frontmatter `round`, `dose: findings-only|
prescriptive|raw-review`, `hitlist`, `run_id`, `verdict`, optional
`extension-ratified-by`; body: per-finding table `code | status open|fixed|
regressed | note`). Records live at `docs/remediation/<effort>/round-N.md`
(dir scaffolded by I085). `spine audit stages` gains one advisory rule: for
the cursor's effort, a `round-4+` record without `extension-ratified-by` is
reported — never blocking. Budget is derived by counting records; no cursor
grammar change. `docs/remediation/README.md` text (from I085) is reconciled
with the templates.

## Acceptance criteria

- [ ] Both templates embedded and reachable (documented how they are
      instantiated — a `spine remediation new` verb is NOT in scope unless
      the implementer finds instantiation impossible without one; record the
      decision).
- [ ] `audit stages` reports round-4 without ratification; silent with
      `extension-ratified-by` set and for rounds 1–3 (negative controls);
      exit code unaffected (advisory).
- [ ] Round record table keys on results-contract `code`; example in the
      template uses `go@1/<check>` ids.

## Blocked by

- I085 (docs/remediation scaffold, gen 11). Do-not-regress content assumes
  I086's killed rows but the templates do not depend on it landing first.

## Resolution

Fixed 2026-08-19 on branch `local-harness-conventions` (commit 55cba1c).
Embedded `templates/current/hitlist.tmpl.md` (header effort/round/dose/source
run id; per finding `code`, file:line, finding, why-it-matters, do-not-regress
block of killed `go@1/mutate` rows by probe id; no fix text — dose-scoped to
the `findings-only` default) and `remediation-round.tmpl.md` (frontmatter
round/dose/hitlist/run_id/verdict/optional extension-ratified-by; per-finding
table `code | status open|fixed|regressed | note` keyed on results-contract
codes). Decision: NO `spine remediation new` verb — both templates are
scaffolded as machine-owned `docs/remediation/_hitlist.template.md` and
`_round.template.md` (init + update; knowledge profile excluded) and
instantiated by copying to `docs/remediation/<effort>/`; README names them.
`spine audit stages` gains one advisory: `docs/remediation/<effort>/
round-N.md` with N ≥ 4 lacking a non-empty `extension-ratified-by:` is printed
as `warning:` from a dedicated `Report.RoundBudget` (not `Notes`, which doctor
would surface as D9 and turn into a gate); exit code proven unchanged for
blocking and non-blocking cursors; rounds 1–3 and ratified rounds silent.
Ruling: the rule keys on the record's filename ordinal (round-N), which is
the count under the sequential-numbering convention. Review clean, no fix
round.
