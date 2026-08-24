# Changelog

Behaviour changes visible to repos that consume `spine`. Format follows
[Keep a Changelog](https://keepachangelog.com/); entries reference the ticket
(`docs/issues/`) or ADR (`docs/adr/`) that carries the detail.

## Unreleased

### Changed

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
