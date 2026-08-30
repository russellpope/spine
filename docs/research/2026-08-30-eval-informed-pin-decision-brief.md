# Eval evidence for host equivalence pins: decision brief

I077 asks how evaluation evidence should accompany owner-ratified host pins
without turning evidence into an automatic gate. This brief presents three
choices and does not change runtime behavior.

## Options

### A. Record an eval reference at ratification

Each pin stores one or more `eval:` references. Spine validates syntax and
preserves them in the resolution trail, but does not warn when evidence later
goes missing or stale. This gives durable provenance with little runtime
policy, but it can age silently.

### B. Add a doctor advisory

Pins remain valid with or without stored references. Doctor inspects eligible
evidence in an explicitly allowed read scope and warns on missing, stale,
malformed, or failing evidence. This finds drift but records weaker provenance.

### C. Store references and advise on their health

Ratification stores references and doctor checks only those references. This
combines provenance and drift detection without searching unrelated repos or
choosing evidence for the owner. It requires explicit grammar and freshness.

## Four decisions to bind

| Axis | Choices |
| --- | --- |
| Eligible evidence | Completed `docs/evals/` records, local-model mutation-battery results, and later I076 per-model yield evidence. Decide whether one passing source is enough or every reference must be healthy. |
| Missing or stale evidence | Silent provenance; warn on missing; or warn on missing, stale, and failing. Define freshness by a date/age rule, never file mtime. |
| Allowed read scope | Recommended default: repo-relative `docs/evals/` plus explicitly named local roots. Never crawl the fleet or follow arbitrary URLs during doctor. |
| Pin authority | Findings are advisory only. They never remove, replace, de-ratify, or block an owner pin. Only malformed host config remains an error. |

## Decision prompt

Choose A, B, or C, then bind eligible evidence, freshness/missing behavior,
and allowed roots. Option C gives the clearest provenance and drift signal,
but only with the advisory-only rule above.
