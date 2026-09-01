---
id: I073
title: Flavor → harness rename migration
severity: med
status: fixed
commits: [09f4d2c, 966bb89, b604674, 1467637, 32b5437, 54c33ea, 703d220, 46b2324]
affects: [model, cli, workflow, fleet]
blocked-by: [I072]
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

[I067](I067-harness-vs-flavor-axis.md) ratified the rename: flavor becomes
harness. Inventory every touchpoint — `models/defaults.json` keys,
`spine model <flavor> <tier>`, `MirrorRows()` → WORKFLOW.md mirror rows,
CONTEXT.md "artifacts never name a flavor", WORKFLOW.md §Model routing
language, audit output, estate repos carrying mirror rows — and execute the
migration in an order that keeps `spine doctor`/`audit` green throughout,
including the fleet sweep. Sequenced after
[I072](I072-host-config-schema-and-precedence.md) so the rename lands the new
schema's names once, not twice.

## Resolution

Fixed 2026-08-31 at exact product SHA `46b2324`. Generation 14 makes
`harness` canonical across defaults, model resolution, CLI, audit internals,
templates, update, and active documentation while retaining equal deprecated
JSON `flavor` output and the legacy defaults reader for the bounded
compatibility window. Positional CLI behavior, the four harness values,
model/effort data, I072 host precedence, host-blind mirrors/update,
`--alternate`, D16, transcript parsing, I111 attribution, D28, and routing
verdict bytes remain stable. Compatibility removal still requires a separate
owner-approved generation-15-or-later effort.

Fresh primary requirements reviews and independent verification passed the
product at `46b2324`, including focused/full/race/vet/build checks, generation
10–13 migrations, hostile defaults compatibility, compiled CLI matrices, and
the exact trimpath candidate SHA-256
`2ae78cf2732464d9d662147cb68f403d1600b981c1fec2b34266e13046402eb4`.

The owner-ratified, no-force fleet sweep completed all 20 named primaries.
Spine was already converged and required no fleet commit; the other 19 landed
as follows:

| Primary | Migration commit |
| --- | --- |
| ai-virt-framebuffer | `f91603923da01e2817fd21eb68e1c041bbae1b04` |
| ai_infra_notes | `ed41bfd7d518c3095dc4831340dc8b31042af380` |
| ccq | `515fb354b3653dd77a98560232e27ec83c3454be` |
| deepthought | `c08fb46c0916eeb61c2e74c73d2a9e3b0bcfbf83` |
| hbmview | `a1890621c30d1a5cd67382cb1137ca51bebfa0e4` |
| home-lab-admin | `ea4056cde3dcbf97b51612bffd6647230d9f1613` |
| jarvis | `3b0a1f84905cb6a90f1fff9a5d78aebe771c5eb9` |
| ladderbench | `f14cf63e4495bee457855c84cee7d6ff88f95399` |
| maikanban | `e717c499698b709d182e20949581db9bdac7a5fb` |
| maipipe | `9eee2e1329d88f63d10040f08d9088fd416578c7` |
| moo-clone | `bc4135753e4a2a8bcd0e751301e0cfe14a3173bb` |
| notetui | `881da24c274bb2d9c61ed2cf25a6685368963355` |
| objectstudio | `67bf3080ac8dc4a9926f14f587e269528867fce6` |
| observability_notes | `e5225ba77693d47b0f8c04bcb4de5cebf257cc5e` |
| obsidian-ep-vault | `e8045c11ae05f3e4ec44d0f6ea8f0a2d20c5e42e` |
| pi-pack | `db2ad7ce4048a765423605b6ddbc8be944363573` |
| praxis | `75ddbb84738c790ebfcb876de1f24b1e59938a84` |
| pure-automation | `22a31a6a87cbd034765820db2f0886b00e090602` |
| ultima-dci-edition | `f2a0f22c5156d77d745977b4c859fe01946436e7` |

Repository-specific owner rulings preserved maipipe's merge-floor policy,
moo-clone's ID-allocation policy, and objectstudio's vfb gate in hand-authored
files with byte-identical hash proofs; retained praxis's accepted ADR-0108;
and preserved ultima's exact `claude-opus-5 @ high` override. Every primary
finished at generation 14 with a converged dry update, 64/64 declared
text/effort/JSON/validate checks, equal `harness == flavor`, standing doctor
findings separated from regressions, and all linked `*-wt-*` worktrees
excluded. No fleet command used `--force`.

The first independent post-fleet review found one evidence-only error: Spine's
doctor row recorded exit 0 despite D9/D11 warnings. The ledger was corrected
to exit 1 and a fresh primary re-review passed, authorizing acceptance criteria
7 and 8 on the fleet/compatibility axis. Per the owner-directed batch order,
the single exact-SHA `maipipe run full --wait` remains the batch-final ship
gate after I076 rather than a separate I073 lane.
