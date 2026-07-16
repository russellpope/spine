# Stage-cursor controls (template gen 8) — plan

Design: [2026-07-15-stage-cursor-controls-design.md](2026-07-15-stage-cursor-controls-design.md). Tickets: I018–I023 in `docs/issues/`. Execution: subagent-driven, overnight (map I010, decision I016). Every dispatch carries an explicit model and the ticket id in its description; records per WORKFLOW.md grammar in `.superpowers/sdd/progress.md`.

## Task 1 — Cursor format + parser + `spine cursor` (I018)

- Define the cursor grammar (single canonical form, documented once, reused verbatim in the gen 8 template section):

  ```
  <!-- spine:cursor -->
  effort: <kebab-name>
  prd: docs/specs/<file>.md
  tickets: I0NN-I0MM | prefix I0
  stages: grill[x] prd[x] issues[x] implement[<] functional-test[ ] review[ ] verify[ ] ship[ ] ...
  <!-- /spine:cursor -->
  ```

  `[x]` done, `[<]` = YOU ARE HERE (exactly one among non-done), `[ ]` pending. Stage names must match the repo's WORKFLOW.md `stages:` list.
- New `internal/cursor` package: parse from `.superpowers/sdd/progress.md` head; strict grammar, parse errors are findings not panics.
- New `spine cursor` subcommand: prints parsed cursor + advisory derivation verdict (verdict wiring lands in Task 2; until then prints cursor + "derivation: n/a"). Exit 0 always. `--quiet` prints nothing and exits 0 when no spine repo / no cursor (hook-friendly).
- Fixture tests in the doctor/audit testdata style: valid cursor, malformed block, missing file, two-HERE-markers, unknown stage name.

## Task 2 — Derivation engine + `spine audit stages` + doctor check (I019)

- `internal/stages` (or extend `internal/audit`): derive per-stage evidence for the anchored effort — `prd`: the cursor's PRD path exists; `issues`: ledger tickets in the cursor's id set exist; `implement`: commits/branch state touching those tickets (heuristic documented in code, conservative — absence of evidence never blocks, only presence-contradiction does).
- Bidirectional comparison: ticked-but-missing blocks; present-but-unticked blocks; no `progress.md` ⇒ warn, exit 0.
- Newest-handoff check: newest `docs/handoffs/*` must contain a `spine:cursor` block when a cursor exists — blocking in `audit stages`, advisory in doctor.
- `spine audit stages`: table output like `audit routing`, non-zero exit on any blocking finding. Wire `spine cursor` to print the real verdict.
- Doctor: new advisory D-check (next free D-code) reusing the engine; severity `warn`, never `error`.
- Fixture tests per the design's Testing Decisions list.

## Task 3 — Template gen 8 + ultima reconciliation (I020)

- Bump `tmpl.Version()` to 8; add "Stage cursor (consistency rule)" section to `WORKFLOW.md.tmpl` — adapted from ultima's hand-written section, embedding the Task 1 grammar and the handoff rule (verbatim `spine cursor` output in every handoff/resume prompt).
- Capture ultima's current WORKFLOW.md section lines **verbatim** (repo at `/Users/ldh/Projects/github.com/ultima-dci-edition`, lines ~20–32) into `supersededLines` in `internal/update`.
- Gen7→8 migration tests in the per-gen seam; ultima fixture test (hbmview_test.go style) proving plain update yields zero unrecognized lines and the gen 8 section supersedes the hand-written one.
- Update `spine doctor`/`update` gen-mismatch messages if they hardcode 7.

## Task 4 — SessionStart hook (I021)

- Add to `~/.claude/settings.json` SessionStart hooks: a command that, when cwd (or `$CLAUDE_PROJECT_DIR`) is a spine repo (WORKFLOW.md with `template_version`), runs `spine cursor --quiet`; output (if any) lands as session context. Keep it a one-liner shell wrapper; no new script file unless quoting demands one.
- Document in deepthought (the estate repo) alongside the other hook notes.
- No CI test (owner-approved); verified live in morning verify.

## Task 5 — /handoff skill hardening (I022)

- Edit `~/.claude/skills/handoff/SKILL.md`: for spine repos, the handoff MUST include the verbatim output of `spine cursor` in a dedicated section; prose paraphrases of stage state are defined incomplete; resume prompts inherit the same rule.
- Mirror the rule's one-line summary in the gen 8 template section (done in Task 3; this task only touches the skill).

## Task 6 — Fleet sweep to gen 8 (I023)

- For each of the 17 spine repos (list from the 2026-07-15 audit; enumerate live via `grep -l template_version */WORKFLOW.md`): `spine update` (dry-run) → review diff/unrecognized report → `spine update --write` → `spine doctor` → commit with a uniform message. Order: dormant gen-5 notes repos first, active repos next, **ultima-dci-edition last** via the supersededLines path (`--force` + reviewed diff only as fallback, flagged to owner if used).
- Skip nothing; a repo that fails to update cleanly is reported, not forced silently.
- Output: per-repo table (gen before/after, state) appended to the build ledger for morning review.

## Verify stage (morning, with owner)

- `go test ./...` green in spine; `spine audit routing` on this build's transcripts; `spine audit stages` self-check on spine's own ledger; live hook demo (open a session in a swept repo, cursor appears); /spec-review of the whole diff against the design doc.
