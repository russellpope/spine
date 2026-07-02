# spine — workflow runtime CLI (v1 design)

**Date:** 2026-07-01 · **Status:** approved-pending-plan · **Repo:** github.com/russellpope/spine

## Problem

The unified workflow's tooling is a 41-line `scaffold.sh` behind a skill invoked once, with templates
canonical in the deepthought repo. The 2026-07-01 diagnostic (deepthought `FINDINGS.md`) found:

- **No upgrade path.** `emit()` skips existing files, so template improvements strand every adopter.
  hbmview — the only scaffolded repo — is already one template generation stale, three days after shipping.
- **The fleet's highest-volume artifact is unserved.** ~135 ADRs exist across praxis/ultima/hbmview, every
  one under a hand-rolled convention; the scaffold creates `docs/adr/` empty.
- **Two competing homes for specs/plans.** Five repos write to `docs/superpowers/{specs,plans}` (skill
  defaults); the scaffold says `docs/specs/`.
- **Rules live in prompts, not tools.** Conventions get re-explained to the model every session — tokens
  spent remembering rules instead of doing work. A deterministic binary that agents call is cheaper,
  testable, and allowlistable with one rule.

## Decisions (settled during brainstorm)

1. **Spine absorbs workflow-init entirely.** Templates and scaffold logic move here; the deepthought
   skill becomes a thin shim that calls the `spine` binary. One source of truth.
2. **Go, stdlib only.** Flat 4-command surface doesn't exercise cobra; spine has no config by design
   (its config *is* the repo artifacts), so viper would create a second source of truth. Revisit cobra
   if v2 grows nested command trees. Zero third-party dependencies.
3. **v1 scope: `init`, `update`, `adr`, `doctor`.** `adopt`, `handoff`, `eval` are v2, designed after
   v1 feedback.
4. **Distribution: `make install` → `~/bin/spine`** (static binary, on PATH). No remote required during
   development; chezmoi may own the `~/bin` entry later under Plan B.
5. **Acceptance includes the live hbmview un-stranding** — not just fixtures.
6. **Fleet convention going forward: `docs/specs/` absorbs plans.** `<date>-<topic>-design.md` +
   `-plan.md` pairs, PRDs alongside (the convention hbmview invented). Templates and doctor steer
   superpowers skills there via CLAUDE.md instruction. Existing `docs/superpowers/` trees are never
   mass-moved.
7. **Update mechanism: ownership split + config-preserving regeneration** (Approach 1; see below).
   Managed-blocks-everywhere and 3-way template merge were considered and rejected as noise/overkill
   for five small, mostly machine-owned files.

## Non-goals (v1)

- No network access, ever. No telemetry. No config file.
- No `adopt` retrofit intelligence (v2) — `init` in a conventioned repo just skip-if-exists as today.
- No machine-level doctor checks (model-picker cache etc. stay in FINDINGS P2 tooling).
- No mass migration of existing `docs/superpowers/` artifacts.
- No file deletion by any command, ever.

## Architecture

```
spine/
  cmd/spine/main.go          # dispatch only (map[string]func + flag.NewFlagSet per command)
  internal/tmpl/             # go:embed of templates/ + {{KEY}} rendering + VERSION
  internal/scaffold/         # init
  internal/update/           # key extraction, regeneration, diff, atomic write
  internal/adr/              # new / list / supersede mutation
  internal/doctor/           # read-only checks, human + --json output
  templates/                 # moved from deepthought + new: adr-README.md, adr.tmpl.md, VERSION
  docs/{specs,adr,issues,handoffs}   # self-scaffolded on day one
  Makefile                   # build / test / install
  CLAUDE.md, WORKFLOW.md     # dogfood (spine init run on this repo)
```

- **Templates compile into the binary** (`go:embed`) — no path coupling to any checkout.
- **`templates/VERSION`**: single monotonic integer, compiled in. Stamped by init/update into
  `WORKFLOW.md` (`template_version: N`) and the CLAUDE.md marker block. Staleness = stamped < compiled.
  No template-history archive, with one exception: the 2026-06-28 generation ships embedded as **gen-0**
  for legacy claiming (the only unstamped generation in the wild).
- **File ownership model:**
  - *Machine-owned:* `WORKFLOW.md`, `docs/harness-interface.md`, `docs/issues/README.md`,
    `docs/issues/_template.md`, `docs/adr/README.md`. Update regenerates them wholesale from the
    current template, preserving extracted config keys.
  - *Mixed:* `CLAUDE.md` — a spine-managed block delimited by `<!-- spine:begin -->` /
    `<!-- spine:end -->` markers holding the workflow header; everything outside is user territory,
    never touched.
  - *User-owned (spine never modifies):* everything in `docs/specs/`, `docs/handoffs/`, issue entries,
    and ADR bodies (single exception: the supersede status flip, below).
- **Preserved config keys** (extracted from existing `WORKFLOW.md`):
  `profile`, `reviewers`, `functional_harness`, `model_routing.*`, `effort`, `model_default`,
  `security_routing`, `gates`, `stages`.
- **Choice-vs-default rule:** an extracted value equal to its own generation's default (the value the
  file's template would have rendered for that profile) is *not* a user choice — regeneration gives it
  the current template's default. Only values differing from their generation's default are reapplied.
  `profile` is always preserved. Rationale: hbmview's `model_default: claude-opus-4-8` is gen-0's
  default, not a decision; preserving it verbatim would defeat the un-stranding.

## Commands

### `spine init [--profile P] [--dir D] [--name N]`
Ports scaffold.sh behavior: profiles `go-service | py-tool | rust | library-cli | presentation | ui`,
auto-detected (go.mod → go-service or library-cli, pyproject/setup.py → py-tool, Cargo.toml → rust,
*.pptx/*.key → presentation, package.json + UI framework → ui), `--profile` overrides; per-file
skip-if-exists; creates `docs/{specs,adr,issues,handoffs}`; keeps emitting the issues README +
`_template.md` pair unchanged. New over scaffold.sh: stamps `template_version` and emits
`docs/adr/README.md` (numbering, `Accepted`/`Superseded` statuses, immutability + supersede rules —
lifted from ultima's convention). Exit 0 on success.

### `spine update [--dir D] [--write] [--force]`
Dry-run by default: per-file unified diffs to stdout; exit **1 if changes pending, 0 if current**
(scriptable by agents). `--write` applies atomically (temp file + rename), warning first if the target
file has uncommitted git changes.

- Machine-owned files: extract preserved keys → re-render current template → diff/write.
  Lines that are neither current-template nor gen-0 content nor recognized keys are printed as
  **unrecognized local edits; the file is skipped** unless `--force` (the diff shows exactly what force
  would drop — nothing is silently lost).
- `CLAUDE.md`: rewrite only the marker block. Legacy files (no markers): if content matches rendered
  gen-0 (modulo values), claim cleanly (replace wholesale, re-render, add markers); otherwise insert a
  marker block at top and preserve all existing content below it.
- `WORKFLOW.md` legacy (no `template_version`): treated as gen-0 claim — extract keys, regenerate.
- Update never downgrades: a stamped `template_version` greater than the compiled generation is a hard
  error (upgrade spine first) rather than being treated as current-gen. Non-integer stamp values are
  treated as current-gen, as before.

### `spine adr new "Title" [--supersedes NNNN]` / `spine adr list`
`new`: scan `docs/adr/` for next `NNNN`, slugify title, render `NNNN-slug.md` from embedded template
(front-matter: `status: Accepted`, `date`, optional `supersedes`), print path. `--supersedes` performs
the single permitted mutation of an existing ADR: flip its status line to `Superseded by NNNN`.
`list`: table of number / title / status. Exit 2 if `--supersedes` target missing.

### `spine doctor [--dir D] [--json]`
Read-only. Checks:

| ID | Check |
|----|-------|
| D1 | Required dirs/files present (`docs/{specs,adr,issues,handoffs}`, `WORKFLOW.md`, `CLAUDE.md`, `docs/harness-interface.md`) |
| D2 | `template_version` stamped < binary's compiled version → "run spine update" |
| D3 | CLAUDE.md marker-block integrity (present, balanced, parseable) |
| D4 | Unrecognized local edits in machine-owned files |
| D5 | New files accumulating in `docs/superpowers/{specs,plans}` → nudge toward `docs/specs/` |
| D6 | ADR numbering collisions / invalid status values. Pre-spine, hand-rolled ADRs with no front matter (e.g. hbmview's) are `info`, not `warn` — spine conventions apply to new ADRs, not retrofit existing ones. |

Human-readable by default; `--json` emits `{findings: [{id, severity, path, message}]}`.
Exit 0 clean or info-only / 1 warn-or-error findings / 2 execution error.

## Error handling

- No network. No deletion. Atomic writes only.
- Errors → stderr; output/diffs/JSON → stdout. Exit codes as specified per command (0 ok,
  1 findings-or-pending, 2 hard error) — uniform across commands.

## Testing

TDD (red → green per task). Table-driven unit tests per package. Golden-file integration tests run the
built binary in `t.TempDir()`:

1. Fresh `init` per profile → golden trees.
2. **hbmview-drift fixture** — copies of hbmview's actual current `WORKFLOW.md` / `CLAUDE.md` /
   `harness-interface.md` → `update` produces current-generation output with hbmview's keys preserved.
3. Legacy CLAUDE.md claim (gen-0 match and no-match paths).
4. Unrecognized-edits refusal + `--force` behavior.
5. `adr` numbering, supersede flip, collision handling; `doctor` findings + exit codes + `--json` shape.

`make test` gates everything.

## Acceptance (v1 done means)

1. `spine init` self-scaffolds this repo (dogfood, day one).
2. Spine's own design decisions are recorded via `spine adr new` (ADR 0001+: Go/stdlib, ownership
   model, specs-home convention, cobra-reconsidered-at-v2).
3. Live `spine update --write` on hbmview un-strands its stale files with config preserved —
   verified by clean `spine doctor` and human-reviewed git diff.
4. `make test` green; binary installed at `~/bin/spine`.

## Migration end-state (deepthought handover)

After acceptance: deepthought `skills/workflow-init/` thins to a shim (SKILL.md keeps the trigger
description; step 2 becomes `spine init --profile <p>`; `scaffold.sh` + `templates/` deleted from
deepthought). The `~/.claude/skills/workflow-init` symlink continues pointing at deepthought — the
skill remains the discovery surface, the binary is the implementation. All future spine artifacts
(specs, plans, ADRs, handoffs) live in this repo.

## v2 candidates (explicitly deferred)

`adopt` (detect-and-map retrofit for praxis/ultima conventions), `handoff list|latest`,
`eval` (docs/evals/ run-record convention, pairs with the planned /model-eval skill), new profiles
(swift, infra, knowledge), machine-level doctor checks, cobra migration if the command tree deepens.
