# Local-harness conventions — plan

**Date:** 2026-08-18 · **Effort:** local-harness-conventions · **Tickets:** I079–I087
**Spec (binding authority):** `docs/specs/2026-08-18-local-harness-conventions-design.md`
**ADRs:** 0015, 0016 (respect 0001, 0002, 0004, 0007, 0011, 0014) · **Glossary:** CONTEXT.md

This plan is the SDD wrapper over the ticket set. Each task IS one ticket
in `docs/issues/`; the ticket file is the requirements text and its
acceptance criteria are the task's exit test. Tasks are ordered so every
task's `blocked-by` is complete before it runs (serial execution). Where a
task's text and the spec disagree, the spec wins.

## Global Constraints

- Go stdlib only for spine's own dependencies (ADR 0001); a new module
  dependency needs an owner-ratified ruling recorded in the ledger and the
  ticket.
- Test at the CLI-dispatch seam against tempdir fixture repos (prior art
  `cmd/spine/main_test.go`, `internal/update` gen tests + testdata);
  package-level tests only for canonical-form byte determinism.
- Every gate check class ships a positive-control pair (known-good passes,
  seeded violation fails). Every advisory/doctor rule ships a negative
  control (canonical input → no finding).
- Results-contract JSON: `maipipe_results: 0`, `status`, `summary`,
  `findings[]` with `severity`, `message`, `file`, `line`, `code =
  "go@1/<check>"`; written only when `MAIPIPE_RESULTS` is set.
- Exit codes for `spine gate`: 0 pass, 1 findings, 2 misconfiguration.
- Facts region and cursor obey sole-writer + canonical-form; no hand-edited
  cursor blocks; use `spine cursor tick` only for stage completion.
- Machine-owned regions follow ADR 0002 (inherited refresh, override
  preserved, unrecognized reported); template generation bump is a single
  integer (ADR 0004) and happens once, in Task 6.
- Use CONTEXT.md vocabulary verbatim in code comments, docs, and CLI help:
  harness, alternate, checkpoint, model region, facts region, working
  home, reload preamble, gate pack, check class, positive control.
- Commit explicit paths only; never `git add -A`. Each task = one or more
  commits referencing its ticket id in the subject.
- Do not resolve tickets (`status: fixed`) inside a task — the controller
  does that at task completion after review, appending a `## Resolution`
  section.

## Task 1: I079 — pi harness rows + alternate cell in the model table

Read `docs/issues/I079-pi-harness-rows-and-alternate-cell.md` — it is your
requirements. Spec section: Implementation Decisions → Model table.
Deliver: `spine model pi <tier> [--alternate] [--json]`; pi cells with
explicit efforts (primary xhigh, routine medium, mechanical low, fallback
xhigh), `alternate: {id: qwen3.8-27b-q8_0, effort: xhigh}` on every cell;
pi vocabulary `low|medium|xhigh` (`high` rejected with a message); mirror
rows render/parse a trailing `alt: <id> @ <effort>` under inherited/override
rules; JSON key remains under the legacy `flavors` map. Acceptance criteria
in the ticket.

## Task 2: I082 — spine gate go skeleton + results-contract emitter + tskip, binary-hygiene

Read `docs/issues/I082-gate-command-skeleton-results-contract-tskip-binary-hygiene.md`.
Spec section: Gate pack. ADR 0015. Deliver the tracer bullet: `spine gate go
<check> [--dir D]`, exit-code contract, results-contract emitter, human
table fallback, pack version constant `go@1` in one place, check classes
`tskip` (allowlist via env) and `binary-hygiene`, each with a
positive-control fixture pair.

## Task 3: I080 — Checkpoint document + spine checkpoint new|latest|list

Read `docs/issues/I080-checkpoint-document-and-commands.md`. Spec section:
Checkpoint. New sibling package (handoff/cursor pattern). Deliver `spine
checkpoint new --from <narrative.md> --touched <csv> --gate
<pass|fail|none> --effort <level> [--slug s] [--facts-only]`, `latest`,
`list`; working home `.superpowers/sdd/checkpoints/NNN-<slug>.md`;
frontmatter (`ordinal`, `created`, `effort`, `narrative`); model region
(`## Task`, `## Conclusions`, `## Next moves`, strict) and canonical facts
region (`touched`, `gate`, `sha`, `effort_recommended`, `written`);
embedded reload preamble stating the trust split and the facts-only case;
`latest` byte-stable.

## Task 4: I083 — Gate classes: gitignore-control, fixture-manifest, test-enum-vs-spec

Read `docs/issues/I083-gate-config-driven-classes.md`. Blocked by Task 2.
Config inputs arrive as environment variables (names chosen here become
the `gate_pack_config` → env mapping Task 6 renders; document them in the
report). Two-arm gitignore control; manifest existence only; enum-vs-spec
extras on both sides. Positive-control pairs; missing config → exit 2.

## Task 5: I084 — Gate classes: deferred-cleanup-errcheck, dead-code-callgraph, n-plus-one

Read `docs/issues/I084-gate-analysis-classes.md`. Blocked by Task 2.
Dependency ruling required before adding `golang.org/x/tools` (ADR 0001):
prefer stdlib `go/ast`+`go/types`+`go/build`; if x/tools is judged
necessary, stop and report NEEDS_CONTEXT with the reason — the controller
rules and ledgers it. Positive-control pairs.

## Task 6: I085 — Template gen 11: gate_pack keys, maipipe.toml region (gate-go), docs/remediation scaffold, doctor region integrity

Read `docs/issues/I085-template-gen-11-gate-pack-keys-maipipe-region-remediation-dir.md`.
Blocked by Task 2 (and uses the env names from Task 4/5 reports for
`gate_pack_config`). ADR 0002, 0004, 0016. Deliver WORKFLOW.md keys
`gate_pack`, `gate_pack_disabled`, `gate_pack_config` (preserved choices);
`spine update` renders the `# spine:begin gate-pack go@1` … `# spine:end`
region into `maipipe.toml` (create-with-region-only when absent; user lanes
untouched when present) containing `[pipelines.gate-go]` (profile full, one
stage per enabled class, `run = "spine gate go <check>"`, env from config);
`mutation-go` NOT rendered here; gen 10→11 with migration test; scaffold
`docs/remediation/README.md`; doctor region-integrity finding; repos
without `gate_pack` write no `maipipe.toml`.

## Task 7: I081 — Handoff embeds newest checkpoint; doctor advisory for checkpoints

Read `docs/issues/I081-handoff-embeds-checkpoint-and-doctor-advisory.md`.
Blocked by Task 3. `handoff new` embeds newest checkpoint after the cursor
block (facts verbatim; model region under "Prior narrative (model-authored,
not evidence)"); doctor advisory: malformed/non-canonical facts region,
ordinal gaps, unignored `.superpowers/`. Negative controls.

## Task 8: I086 — Mutation battery Go port: spine gate go mutate + mutation-go pipeline

Read `docs/issues/I086-mutation-battery-go-port-and-mutation-go-pipeline.md`.
Blocked by Task 6. Port the `/model-eval` skill's Python runner behaviour
(source: `docs/mutation-battery-checklist.md`; runner lives at
`~/.claude/skills/model-eval/` — read, port, don't reinvent). Killed/
survived rows via results contract (`code = go@1/mutate`), unmutated
negative control mandatory, exit 1 only on control failure; `spine update`
renders `[pipelines.mutation-go]` (profile audit) into the region.

## Task 9: I087 — Remediation templates (hitlist, round record) + round-budget audit advisory

Read `docs/issues/I087-remediation-templates-and-round-budget-advisory.md`.
Blocked by Task 6. Embedded `hitlist.tmpl.md` and `remediation-round.tmpl.md`
per the spec's Remediation decisions (no fix text in the hitlist); records at
`docs/remediation/<effort>/round-N.md`; `spine audit stages` advisory rule
for round-4+ without `extension-ratified-by` (negative controls; exit code
unaffected); reconcile `docs/remediation/README.md`. A `spine remediation
new` verb is out of scope unless instantiation is impossible without one —
record the decision either way.

## Stage mapping

- **implement:** Tasks 1–9 serial via claude-team (herdr), routine-tier
  implementers from ticket prose; reviewer floor per ticket annotations
  (Tasks 3 and 6 review at primary).
- **functional-test:** CLI-seam tests per task; full `make test` before
  each task's review.
- **review:** independent blind final whole-branch review against the spec
  on the primary tier.
- **verify:** fresh acceptance verification: `make test`, `spine doctor`,
  `spine audit stages`, `spine audit routing`, diff integrity.
- **ship/docs/handoff:** resolve I079–I087 with `## Resolution` sections,
  commit explicit paths, `spine handoff new`, re-run audits.
