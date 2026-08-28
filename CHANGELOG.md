# Changelog

Behaviour changes visible to repos that consume `spine`. Format follows
[Keep a Changelog](https://keepachangelog.com/); entries reference the ticket
(`docs/issues/`) or ADR (`docs/adr/`) that carries the detail.

## Unreleased

### Added

- **A comma-list cursor `tickets:` form.** `tickets: I0NN,I0MM[,...]` resolves to
  exactly the listed ids, in input order — the form a non-adjacent ticket batch
  needs, which previously ran with its issues/implement evidence degraded to
  not-judged. Strict by design: internal whitespace, a malformed or empty
  element, or a duplicate makes the whole value unresolvable (no partial
  resolution), and the unresolvable-tickets note names the new form. The
  WORKFLOW template's grammar line is updated in the same change, with the
  outgoing line joining the updater's superseded set, so estate repos refresh
  cleanly on their next `spine update`. The template also gains the binding
  gen-bump authoring note: any content-changing template edit appends its
  predecessors' dropped lines to the superseded set in the same change. (I114)

### Fixed

- **Trailing whitespace after the closing cursor fence now reports
  non-canonical form.** The canonical-form guard compared bytes only up to the
  closing tag text, so a hand edit that padded the closing fence line went
  undetected while the same padding on the opening line was caught. The
  compared span now ends at the closing fence line's end. What `spine cursor`
  writes is unchanged. (I113)
- **D13 per-ticket checks hardened.** A non-absolute `workspace:` value now
  warns and is never stat'd (removing the false-warn dependence on the process
  CWD under `--dir`); one pair of surrounding quotes is stripped from ticket
  frontmatter values before validation, so YAML-quoted `batch:` ids no longer
  read as malformed; the parser's deliberate no-comment-stripping divergence is
  guarded by a comment; and a fence-less ticket's silence is pinned by a test.
  (I115)

- **An `openweights` model flavor.** `spine model openweights <primary|routine|mechanical|fallback>`
  resolves to `FW-Kimi-K3`, `DeepSeek-V4-Pro`, `FW-GLM-5.2` and `FW-Kimi-K3`, every
  tier at effort `high` — `routine` and `mechanical` included, which the global tier
  defaults would otherwise give `medium` and `low`. `fallback` deliberately shares
  `primary`'s model: the flavor exists to measure open-weights models, so a refusal
  re-run must not silently leave open weights. Repos override these rows in
  `WORKFLOW.md` like any other flavor's. Resolution for `claude`, `codex` and `pi` is
  unchanged. (I110)

### Changed

- **Every repo's `model_routing` block reflows on the next `spine update`.** The mirror
  pads its key column to the longest `flavor.tier` key, and `openweights.mechanical:`
  is longer than anything that came before, so all pre-existing rows gain padding.
  This is whitespace only — no id, effort or provenance changes, and no row is
  refreshed, overridden or reported — but it does mean one unavoidable diff in a file
  most repos keep checked in. (I110)
- **A panic inside a gate check is now a misconfiguration exit, not a crash.**
  `spine gate` recovers panics at two seams and returns them through `gate.Run`'s
  documented contract — message on stderr, exit 2, no results document — instead of
  letting the Go runtime kill the process. The exit code an operator sees was
  already 2, but only by accident: it was the runtime's status for an unrecovered
  panic, and the missing `$MAIPIPE_RESULTS` file was the same accident. Both are now
  real guarantees, for every check class rather than the two that prompted this.
  Two new message classes, deliberately distinguishable from the existing
  `--dir %s does not type-check` on their first line: an export-data mismatch names
  the verbatim panic text, the toolchain the binary was built with, the toolchain on
  PATH, and `make install`; anything else is an internal error carrying the panic
  value and stack verbatim, and never suggests a rebuild. This matters after a Go
  upgrade — `dead-code-callgraph` and `deferred-cleanup-errcheck` would fail a lane
  with a `gcimporter` stack trace on a commit that had nothing to do with it. A
  module that genuinely fails to type-check is unchanged. No toolchain version
  comparison is performed anywhere in the gate; detection keys on the importer
  actually failing. (I107, ADR 0021)
- **`spine doctor` exit code on unstamped fleet repos.** New check D12 warns when a
  repo carrying `docs/issues/` lacks a maikanban repository slug (or has a malformed
  one). Warnings affect the exit code, so `spine doctor` now exits non-zero on such
  repos until `spine init` stamps the slug. No stage pack or template runs
  `spine doctor`, so no CI lane goes red on its own. (I094)
- **`spine update` refuses a `maipipe.toml` that maipipe cannot load.** The
  hand-rolled spine-side TOML scanner is gone; `maipipe validate` is the sole
  grammar authority for the rendered gate-pack region. A file without a top-level
  `schema` key, or with a duplicate stage outside the managed region, is rejected
  before anything is written. (I104, ADR 0018)
- **With `gate_pack` set, `spine update` needs `maipipe` on PATH to touch
  `maipipe.toml`.** Without the binary the pre-flight cannot run, so the file is
  skipped with a named preflight skip and exit 0; other managed files still update.
  (I104, ADR 0018)
- **The pack pin now reaches every stage.** `spine update` renders
  `run = "spine gate go@1 <check>"` (mutation included), and `spine gate` with a
  versioned pack argument codes findings with that pin, refuses a check outside the
  pin's class list, and refuses a pin the binary does not ship — both exit 2, no
  findings document. Bare `spine gate go <check>` is unchanged. Every adopter's region
  is rewritten once; the plan now says "N stage(s) changed" for byte-only rewrites and
  names the `maipipe gate approve-definition` cost. Old bare run lines read as stale,
  not unrecognized; a foreign pin is unrecognized. (I103, ADR 0019)
- **Clearing `gate_pack` now has an uninstall path.** If any stage outside the
  managed region still composes `gate-go` or `mutation-go`, `spine update` refuses
  and names the pipeline and stage to fix. With no such consumers, the plan shows a
  marker-inclusive deletion of the region and `--write` removes it. `spine doctor`
  D10 warns on a stale region left behind and errors on damaged markers. (I097)
- **`spine audit routing` attributes team spawns whose brief was delivered by file
  reference.** A claude-team lead that starts a worker with `$(cat <brief>)`,
  `--brief <path>` or a bare `.md` argument previously left every spawn in the
  unmatched list. Attribution now resolves the referenced path against the heredoc
  writes recorded in the lead's own transcript — the brief's first line supplies the
  ticket, its body may satisfy repo qualification, and the verdict discloses the
  brief path it came from. The file on disk is never opened and no shell is ever
  invoked, so verdicts stay reproducible after a worktree is removed. Anything
  unresolvable stays unattributed, as before. (I101, ADR 0020)
- **Transcript discovery now covers worktrees.** `spine audit routing` scans the
  union of the repo's own slug directory, the slugs of every path in `git worktree
  list`, and slug directories matching `<repo-slug>-*`, so an effort built in a
  worktree — including one since removed — is found without `--transcripts`. Scanned
  directories are named in the report's warnings; `--transcripts` still overrides
  entirely. (I101)
