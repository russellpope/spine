# Flavor-to-harness migration (I073)

## Product requirement document

**Status:** owner-ratified for implementation and the named migration sequence
once I072's independent verifier records PASS for its exact final SHA. No I073
implementation or fleet write may begin before that prerequisite passes.

**Ticket:** I073, flavor-to-harness rename migration.

**Decision:** rename the model table's first axis from `flavor` to `harness`.
The harness selects the execution route; model ID and effort remain the route
data. The values `claude`, `codex`, `pi`, and `openweights` do not change.

I073 is a terminology and interface migration, not a model-table redesign. It
does not decide I112's openweights tension. It preserves I072's host-routing
schema v1, host-blind mirrors and update behavior, `--alternate` rule, D16,
and preference-only audit mapping.

## Goals and boundaries

- Make `harness` the current public CLI, JSON, source, template, audit, and
  fleet term for the model table's first axis.
- Allocate generation 14, after current generation 13, and sweep the complete
  named primary estate without force writes.
- Keep old repositories, transcripts, and JSON consumers readable through a
  bounded compatibility period.
- Keep routing verdict semantics and all raw transcript interpretation intact.

I073 does not change host config, gateway execution, credentials, model-family
classification, I074 heterogeneous audit verdicts, I075 effort transport, I076
review-record grammar, or historical records. Historical ADRs, closed issues,
handoffs, and old specs stay as evidence. New docs can explain their old term.

## Compatibility contract

### CLI

Canonical forms become:

```text
spine model [--dir D] [--alternate] [--effort|--json] <harness> <tier>
spine model [--dir D] validate [--expect MODEL_ID] <harness> <tier>
```

The invocation is positional-compatible. Callers still pass the same first
value and tier, so no `--flavor` alias is needed or added. Usage, help, errors,
and active documentation say `harness`, including `unknown harness`.

### JSON, defaults, and source symbols

Generation 14 is the compatibility release. `spine model --json` adds the
canonical top-level `harness` and keeps top-level `flavor` with the same value
as a documented deprecated field. Existing fields retain their exact meanings:
`tier`, `id`, `effort`, `aliases`, `alternate`, `provenance`, `requested`,
`host`, and `pin`. Any JSON object that names the first axis follows the same
additive rule. Text and effort output remain unchanged apart from wording in
diagnostics.

`models/defaults.json` writes canonical `harnesses` and
`tierDefaultEffortByHarness`. During gen 14 the embedded defaults reader accepts
exactly one top-level `harnesses` or legacy `flavors` object and rejects
both/neither. This reader is a build/input compatibility affordance, not a
third runtime configuration source. Repository mirror keys stay unchanged,
because `claude.primary` names a value, not the word "flavor". Host config v1
already uses `harnesses` and is unchanged.

Rename all source terms: `Flavors`, `Entry.Flavor`, `LaunchRequest.Flavor`,
flavor-named resolver parameters/helpers, `tierDefaultEffortByFlavor`, and
audit token/derivation names become harness equivalents. They are in Go
`internal/` packages, so I073 has no supported external Go API to retain.

The old JSON field and legacy defaults reader remain through the gen-14 fleet
compatibility period. They can be removed only by a separately approved
generation-15-or-later change after the I073 fleet ledger proves all 20 named
primaries are at gen 14 and JSON consumers were checked. I073 must not remove
them early.

### Audit and transcripts

Audit continues to parse existing Claude and Codex transcript formats
unchanged. Raw `message.model`, aliases, history, verdict strings, actual-model
values, row order, exit behavior, D28 source qualification, Codex behavior,
and I111's model-derived selector with transcript-source tiebreaker remain
unchanged. Only the first-axis wording and identifiers become `harness`.

Source remains distinct from harness. I073 must not infer one from the other,
rewrite session files, call host-aware resolution while building audit maps, or
make a divergent I072 host pin launchable. `--alternate` remains host-blind.

## Generation-14 contract

Generation 14 is required because template-owned active wording changes.
`MirrorRows` becomes a harness-named renderer but emits identical dotted keys,
ordering, IDs, effort values, alternates, and padding. The current content-based
mirror tests remain the guard against treating column alignment as a model
change. Templates distinguish routing harness from `functional_harness`
(`cli`, `rest`, `framebuffer`, `none`), which stays unchanged.

Update must recognize all prior supported generations through 13, preserve
recognized overrides, flag unrecognized edits, and stay host-blind. No host
pin, host config, or local route may appear in generated `WORKFLOW.md`.

## Touchpoint inventory

| Area | Current touchpoints | I073 treatment |
| --- | --- | --- |
| Embedded table | `models/defaults.json`, `models/embed.go` | Write harness keys; retain the one-release legacy reader; preserve data. |
| Model resolver | `internal/model/model.go`, `model_test.go` | Rename public/internal symbols, diagnostics, strict resolution, history, effort, override, and mirror terminology. |
| I072 host path | `internal/hostconfig/*`, host resolution in `internal/model`, `internal/doctor/doctor.go` | Rename variables/messages only. Preserve schema-v1 `harnesses`, `Load`, D16, and exact host behavior. |
| CLI | `cmd/spine/main.go`, `main_test.go`, `strictargs_test.go`, `i072_host_test.go` | Canonical usage/errors and additive JSON field compatibility. |
| Audit | `internal/audit/audit.go`, `codex.go`, `teamspawn.go`, `resolve_test.go`, `i047_test.go`, `i111_test.go`, `i072_host_test.go` | Rename first-axis identifiers while preserving source, transcript parsing, D28, and verdict output. |
| Rendering | `templates/VERSION`, `templates/current/{WORKFLOW,AGENTS,CLAUDE}.md.tmpl`, `internal/tmpl/*`, `internal/scaffold/*` | Generation 14 and routing-harness wording; leave functional harness unchanged. |
| Update | `internal/update/{modelrouting,keys,update,diff}.go` plus generation fixtures/tests | Rename implementation terms and prove old generations migrate without changed route values. |
| Active docs | `README.md`, `CONTEXT.md`, `CHANGELOG.md`, root generated docs, I066-I073/I076/I102/I110-I112 where forward-facing | Update live instructions and I073 closure. Preserve history and I112's unresolved warning. |
| Fleet | primary generated `WORKFLOW.md`, `AGENTS.md`, `CLAUDE.md`, `docs/issues/{README.md,_template.md}` | Review-first gen-14 sweep, one primary at a time, no force. |

Tests with flavor-named identifiers, string expectations, JSON shapes, model
mirror keys, or template-line literals are touchpoints too. They change only
where they exercise current behavior; one legacy-reader and JSON-key test keeps
the compatibility window honest.

## Fleet scope and execution order

The fixed scope is the following 20 primary repositories, observed 2026-08-30
from their primary `.git` directory and `WORKFLOW.md`:

1. `spine`
2. `ai-virt-framebuffer`
3. `ai_infra_notes`
4. `ccq`
5. `deepthought`
6. `hbmview`
7. `home-lab-admin`
8. `jarvis`
9. `ladderbench`
10. `maikanban`
11. `maipipe`
12. `moo-clone`
13. `notetui`
14. `objectstudio`
15. `observability_notes`
16. `obsidian-ep-vault`
17. `pi-pack`
18. `praxis`
19. `pure-automation`
20. `ultima-dci-edition`

All `*-wt-*` worktrees are excluded. They share their primary history and
receive the migration by merge or rebase. The fleet ledger must list them as
exclusions, not silently count them as missing.

Proposed sequence:

1. Merge and verify the I073 primary implementation, then build the exact
   candidate binary without installing it over the owner binary.
2. Update and verify `spine` first.
3. Sweep clean primaries: `ai_infra_notes`, `home-lab-admin`, `jarvis`,
   `ladderbench`, `maikanban`, `moo-clone`, `notetui`, `pi-pack`.
4. Sweep active/dirty or branch-bearing primaries only after an owner identifies
   migration-only paths: `ai-virt-framebuffer`, `ccq`, `deepthought`, `hbmview`,
   `observability_notes`, `praxis`, `pure-automation`.
5. Sweep exceptions individually last: `maipipe`, `objectstudio`,
   `obsidian-ep-vault`, `ultima-dci-edition`. Their non-stock or stale state
   requires dry-run review and, if needed, owner-approved hand folding.

The owner may reorder to protect active work. Every deferred repository needs a
ledger reason, owner, and recovery order. `--force` is prohibited.

## Per-repository verification and rollback

For each primary, record pre-update HEAD, branch, generation, dirty paths,
dry-run report, reviewed diff, post-update commit, and results of:

1. `spine update --dir <repo>` dry run, reviewed before any write.
2. `spine update --dir <repo> --write`, staging only reviewed generated paths.
3. `spine doctor --dir <repo>`, with standing findings separated from new ones.
4. `spine model --dir <repo>` text, `--effort`, `--json`, and `validate` for
   each declared harness/tier. JSON must show equal `harness` and deprecated
   `flavor` during gen 14.
5. `spine audit routing --dir <repo>` when scoped transcript evidence exists.
   Record the scope and verdict; use existing transcripts, do not manufacture
   new ones.

Stop on the first unexpected dry-run, doctor, model, or audit result. Do not
continue the roster. Revert only the affected repository's migration commit,
rerun that verification with the prior binary, and fix a primary regression
before retrying. Never reset a repository, edit unrelated dirty work, remove a
host config to hide a failure, or use `--force`.

## Owner-ratified write boundary

The 2026-08-30 owner ruling ratifies I073 implementation and the named fleet
migration sequence once I072 has an independent verifier PASS for its exact
final SHA. That prerequisite is the remaining start gate; no further owner
round trip is required for the in-scope implementation or roster.

Use only an exact, independently verified I073 candidate binary for fleet
writes, follow the sequence and per-repository stop conditions above, and stop
if a repository no longer matches its recorded preflight state. This authority
does not permit force writes, unrelated repository edits, or early removal of
the generation-14 compatibility layer.

## Requirements attack

| Attack | Resolution |
| --- | --- |
| A blanket rename changes `functional_harness`. | Keep it unchanged and document the distinction. |
| JSON consumers break although CLI position works. | Add canonical `harness`, retain equal deprecated `flavor` in gen 14, and bind delayed removal. |
| New defaults keys make input fixtures unreadable. | Accept exactly one canonical/legacy key during gen 14; reject both/neither. |
| The rename changes I072 behavior. | Preserve schema-v1, `Load`, host-blind mirror/update, alternate, D16, and preference-only audit regressions. |
| Audit confuses source and harness. | Retain I111 derivation, source tiebreak, D28, and raw-token fixtures. |
| Template wording strands older repos. | Allocate generation 14 and test all prior supported generations. |
| Worktrees inflate or hide fleet scope. | Use the named 20 primaries and record all worktrees as exclusions. |
| Dirty exceptions require unsafe writes. | Review dry runs, hand-fold only by owner decision, never force. |
| I072 was merged but not independently verified. | I073 remains blocked until the verifier records PASS for the exact SHA. |

## Acceptance criteria

1. Active CLI help/errors, JSON canonical fields, source symbols, templates,
   docs, and audit wording call the first axis `harness`; only the declared
   gen-14 legacy JSON key and defaults reader retain `flavor`.
2. CLI positional behavior and all four current harness values are unchanged.
   JSON emits equal `harness` and `flavor` values without removing old fields.
3. Defaults write harness keys, read exactly one legacy/canonical key during
   the window, and preserve IDs, efforts, aliases, history, alternates, and
   mirror rows.
4. I072 regression coverage proves host schema-v1, `--alternate`, launch
   validation, D16, and preference-only audit behavior did not move.
5. Audit coverage proves old transcripts, I111 selection, source tiebreak,
   D28, verdict tokens, and raw model strings are stable.
6. Generation 14 is embedded; supported prior generations update to it without
   lost overrides or host-dependent generated content.
7. Fleet ledger contains all 20 primary results, explicit worktree exclusions,
   no force updates, and the prescribed verification evidence.
8. Compatibility removal requires a separate owner-approved gen-15-or-later
   effort after the fleet and JSON-consumer proof gate.
9. A fresh requirements-attack review and independent verification pass before
   I073 closes or the ratified fleet sequence starts.
