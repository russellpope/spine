# Design — Codex as a first-class harness for the spine workflow

Date: 2026-07-10
Profile: library-cli (spine repo)
Status: draft (awaiting review)

## Problem

Claude Code is fully wired into the unified workflow: `spine init`/`update`
emit `CLAUDE.md` with a machine-owned `<!-- spine:begin vN -->` block, and
Claude reads it on every session. **Codex is blind to all of it.** Codex reads
`AGENTS.md` (global `~/.codex/AGENTS.md` plus per-repo `AGENTS.md`, merged), and:

- No spine repo emits an `AGENTS.md`, so Codex opens a spine repo with zero
  knowledge of spine, `WORKFLOW.md`, the mandatory gates, model routing, or the
  `docs/` conventions.
- The global `~/.codex/AGENTS.md` is a 3-line graphify stub with a real bug
  (`~/.Codex/` — wrong-cased path) and no workflow context.
- deepthought's project skills (`spec-review`, `to-tickets`, `model-eval`,
  `inline-rigor`, `aside`, `workflow-init`) are Claude Code project skills and
  are not invocable from Codex.

Goal: make Codex a first-class citizen of the spine workflow — the same way
Claude is — with the durable mechanism living in the spine tool so every repo
benefits, proven first on deepthought.

## Non-goals

- Not changing the workflow itself (gates, stages, model routing) — only making
  a second harness aware of it.
- Not migrating Claude off `CLAUDE.md`. Both files coexist; both carry the same
  machine-owned block.
- Not re-authoring the skills themselves — workstream C is packaging/registration
  only.

## Locked decisions

Settled with the owner on 2026-07-10:

1. **Source of truth:** spine generates *both* `CLAUDE.md` and `AGENTS.md`.
   `spine update` keeps them in sync across every repo. (Not a symlink, not a
   hand-maintained file.)
2. **Depth:** full — workflow context + subagent support + project skills
   invocable in Codex. (Subagent support turns out to be free; see B.)
3. **Scope:** deepthought is the proving ground; the mechanism is reusable
   everywhere via spine.
4. **Spec home:** this repo (spine), under spine's own gates.
5. **Generation:** bump template generation 6 → 7. Adding a machine-owned file
   is a generation event; stamping it keeps spine's generation history legible
   and lets `doctor`/`audit` reason about it.
6. **Content:** the Codex block is *Codex-tuned*, not a verbatim mirror of the
   CLAUDE.md block (see A.1).

## Environment findings (2026-07-10)

- Codex CLI 0.144.1, model `gpt-5.6-sol`, `~/.codex/config.toml`.
- `codex features list`: `multi_agent` is **`stable / true`** already — the
  superpowers `codex-tools.md` advice to set `[features] multi_agent = true` is
  a no-op here. `skill_mcp_dependency_install` is also `stable / true`.
  → Workstream B needs **no** feature-flag change; subagent-driven and
  parallel-agent skills already work in Codex.
- Codex loads skills through *marketplaces* (`codex plugin marketplace add`,
  `codex plugin add`). `~/.codex/config.toml` already registers local-source
  marketplaces (e.g. `openai-bundled`, `openai-primary-runtime`) alongside git
  ones — so a local marketplace pointing at deepthought's `skills/` is the
  natural registration path for workstream C.
- spine generation model: a single integer in `templates/VERSION` (currently
  `6`) selects `templates/current`; there is no per-N template directory. The
  `genXtoY_*.go` files are migration/unrecognized-line logic, not template
  dirs. `templates/current/WORKFLOW.md.tmpl` and `CLAUDE.md.tmpl` already render
  the stamp/marker from `{{VERSION}}`, so a "gen7 bump" = edit
  `templates/VERSION` → `7`, nothing else.

## Architecture

### A. spine emits `AGENTS.md` (durable core — this repo)

The existing `CLAUDE.md` machinery is the template. Four concrete changes:

**A.1 — New template `templates/current/AGENTS.md.tmpl`.**
Same `<!-- spine:begin v{{VERSION}} -->` / `<!-- spine:end -->` markers as
`CLAUDE.md.tmpl` so the marker-surgery in `update` works unchanged. Content is
Codex-tuned:
- States it is the Codex-facing workflow brief (read by Codex; the CLAUDE.md
  twin carries the same facts for Claude).
- Same workflow facts: unified workflow → `WORKFLOW.md`; `docs/specs`,
  `docs/adr`, `docs/issues`, `docs/handoffs`; mandatory gates (PRD up front,
  spec-review of the diff, verification before completion); model routing lives
  in `WORKFLOW.md` (`primary`/`routine`/`mechanical`/`fallback`, tiers not
  model ids).
- Drops Claude-only slash-command trigger syntax (`/grill-with-docs`,
  `/to-spec`, `/spec-review`, `/wayfinder`) as *literal invocations*; instead
  references the stages/gates by name. The initial block does **not** point at
  the skills — that pointer is deferred to workstream C, which is the one that
  registers them (see §C's added obligation to add it to
  `AGENTS.md.tmpl` once the marketplace registration lands). Rationale: an
  unresolvable `/command` in Codex is noise until C lands, and a pointer to
  skills Codex can't yet invoke would be equally misleading.
- Notes Codex specifics: subagent tools (`spawn_agent`/`wait_agent`/
  `close_agent`) are available (multi_agent on); worktree/branch detection per
  the superpowers `codex-tools.md` environment-detection guidance.

**A.2 — Register in the scaffold manifest.**
Add `{"AGENTS.md.tmpl", "AGENTS.md"}` to `scaffold.Files` (scaffold.go:23) so
`spine init` emits it for new repos. `ProfileOwns` returns `true` for all
profiles for this path (Codex-awareness is universal, including `knowledge`).

**A.3 — `planAgents` in update.go.**
A near-clone of `planClaude` (update.go:248): render the current block, then
- file missing → `Pending` + `Created` (this is how *existing* gen-6 repos pick
  up `AGENTS.md` on their next `spine update` — no migration code required);
- file present with markers → `replaceMarkerBlock` surgery, preserving
  hand-authored content outside the block;
- file present without markers → claim-on-top (`block + "\n" + old`).
Wire the call into `Run` (after `planClaude`, update.go:82-86).
`replaceMarkerBlock`/`markerBegin`/`markerEnd` are reused as-is.

**A.4 — Generation bump 6 → 7.**
Edit `templates/VERSION` → `7`. Nothing else: `WORKFLOW.md.tmpl`
(`template_version: {{VERSION}}`) and `CLAUDE.md.tmpl` (`v{{VERSION}}` marker)
already render from the embedded VERSION. No `supersededLines` entries: no
previously-emitted line changes (AGENTS.md is new; the CLAUDE.md/WORKFLOW.md
version stamps are regenerated keys, not local-edit candidates). Verify the
`n > tmpl.Version()` downgrade guard (update.go:193) still behaves: gen-6 repos
render as `current` and advance to 7; a hypothetical gen-8 repo still errors.

**A.5 — Tests.**
Mirror the CLAUDE.md coverage:
- scaffold: `AGENTS.md` created by `init`, skipped when present
  (`gen6_scaffold_test.go` analog — rename/extend for gen7).
- update: created when missing; marker block replaced when present;
  hand-authored content outside markers preserved; unbalanced markers →
  `Unrecognized`, not clobbered.
- version: `spine doctor` is extended to treat `AGENTS.md` as machine-owned on
  the same terms as `CLAUDE.md` — marker check (D3) and the non-misleading
  "--force cannot repair" preserve-hint (D4) both run over `AGENTS.md` too,
  with tests (final-review fix wave, `internal/doctor/doctor.go` +
  `doctor_test.go`). `internal/audit/testdata` fixtures carrying
  `template_version: 5`/`6` are deliberate historical fixtures representing
  older repos, not assertions about the compiled generation — they are
  correctly left unchanged, and `gen6_scaffold_test.go` stays green at v7
  without edits.

### B. Codex config hygiene (global, deepthought-side)

- Rewrite `~/.codex/AGENTS.md`: fix the `~/.Codex/` path bug; make it a lean
  *global* pointer (graphify + the fact that per-repo `AGENTS.md` carries the
  spine workflow). Global stays thin; repo-level `AGENTS.md` carries specifics.
- `multi_agent`: already on — verify with `codex features list`, no change.

### C. deepthought project skills invocable in Codex (needs a spike)

Register deepthought's `skills/` as a **local Codex marketplace** so `spec-review`,
`to-tickets`, `model-eval`, `inline-rigor`, `aside`, `workflow-init` are
invocable from Codex, not just Claude.

**Spike first (unknown):** the exact marketplace manifest format Codex expects
for a `source_type = "local"` marketplace that exposes a directory of skills.
The spike inspects an existing local marketplace snapshot (e.g.
`~/.codex/.tmp/bundled-marketplaces/openai-bundled`) to learn the manifest
schema, then produces a manifest over deepthought's `skills/`. Registration via
`codex plugin marketplace add` + `codex plugin add`. If Codex cannot consume a
project-local skills dir without publishing, fall back to documenting the skills
in the repo `AGENTS.md` and revisit. **C does not block A or B.**

**C's obligation on `AGENTS.md.tmpl`:** once the skills are registered,
workstream C must add the skills pointer to
`templates/current/AGENTS.md.tmpl` (the block A.1 ships without it — see A.1).
This is a content-bearing template change, not a mechanical one: it changes
what the marker block renders for every profile, which interacts with
`supersededLines` semantics (existing repos' prior rendered text becomes a
carry-forward/local-edit candidate on their next `spine update`) and may
warrant its own generation bump. Scope that interaction when C starts rather
than assuming a same-generation edit is safe.

## Sequencing

A → B → C. A is the durable core and ships independently. B is a ~10-minute
config edit. C starts with a feasibility spike and may land later.

## Data flow

```
spine update --write   (gen7 binary)
  ├─ CLAUDE.md   <!-- spine:begin v7 --> …  (Claude reads)
  ├─ AGENTS.md   <!-- spine:begin v7 --> …  (Codex reads)   ← new
  └─ WORKFLOW.md  template_version: 7
```
Codex session in deepthought → reads merged `~/.codex/AGENTS.md` (global) +
`deepthought/AGENTS.md` (workflow brief) → follows the same gates/routing Claude
does; project skills resolve once C registers them.

## Risks

- **Generation-bump blast radius.** Bumping VERSION rewrites the marker/stamp in
  *every* repo's next `spine update`. Mitigated: only the version stamp changes
  on existing files; content is otherwise identical. Covered by the update tests
  (A.5) and a dry-run (`spine update` without `--write`) on deepthought + spine
  itself before `--write`.
- **Two files drift.** CLAUDE.md and AGENTS.md carry the same facts in different
  words; a future workflow change must touch both templates. Mitigated: both
  render from `tmpl.Values`; a test asserts both blocks reference the same gate
  set / routing tiers.
- **C infeasible as designed.** Codex's local-skills packaging is unproven.
  Mitigated: spike-gated; documentation fallback; C is decoupled from A/B.

## Testing strategy

- Unit: scaffold + update + tmpl, per A.5.
- Integration: `spine init` in a temp dir → assert `AGENTS.md` present and
  marker-bounded; `spine update` on a fixture gen-6 repo → asserts `AGENTS.md`
  created and version advanced to 7.
- Manual verify (deepthought): `spine update` dry-run, then `--write`; open a
  Codex session and confirm it can state the gates and model tiers from
  `AGENTS.md`; confirm a registered skill (C) is invocable.
