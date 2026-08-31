# Changelog

Behaviour changes visible to repos that consume `spine`. Format follows
[Keep a Changelog](https://keepachangelog.com/); entries reference the ticket
(`docs/issues/`) or ADR (`docs/adr/`) that carries the detail.

## Unreleased

### Fixed

- **Routing audit now attributes every member of a ticket range without accepting malformed hyphen chains.** Shared bounded range matching covers Claude, Codex, workflow, and discovery paths, including interior-only Codex evidence and huge ranges without materializing them. Chained, partial, or surrounding-hyphen forms attribute no endpoint; the exact `dispatch-task-I###.md` carrier and existing lowercase/later-message behavior remain compatible. (I121)
- **`spine doctor` now advises on meaningful Go toolchain skew.** D14 compares
  the binary's Go release with `go env GOVERSION` on PATH at major/minor
  precision, names both values and `make install` when they differ, and leaves
  patch-only skew quiet. It is a warning-only proxy for the importer condition,
  not a gate preflight or a claim that the binary cannot run. (I108, ADR 0021)
- **`spine audit routing` recognizes openweights models in Claude-layout transcripts.**
  The audit now derives each evidence token's flavor from its observed model id,
  so mixed Claude and openweights sessions resolve against the correct model
  tables. Transcript source still breaks ambiguous-id ties and preserves unknown-id
  behavior. D28 repo qualification remains attached to the Claude transcript
  layout, including openweights records. (I111, ADR 0022)
- **`adr new --supersedes` no longer parses zero-padded ids as octal.** The
  flag went through `flag.Int` (base-0), so the conventional zero-padded ids
  misparsed — `--supersedes 0011` silently flipped ADR **0009** — and
  `0x11`-style values parsed as hex. The id is now parsed base-10 with a
  digits-only check; non-digit, explicitly empty, zero, and out-of-range
  values error naming the rule, and a successful supersede prints
  `superseded: NNNN` so a wrong target is visible immediately. The ADR-convention README's example is corrected to the
  flags-first form in the same change. (I120)

### Changed

- **No subcommand silently discards input.** The I116 ordering guard is
  generalized to every parsing subcommand via a shared strict-parse helper:
  a flag after a positional errors `flags must precede positionals` naming
  the token, a stray positional errors `unexpected argument`, and an
  unknown `spine cursor` sub-subcommand errors naming the real verbs
  (previously `spine cursor show --dir X` silently answered for the CWD
  repo with exit 0). Flag-only `spine cursor` invocations keep the exit-0
  hook contract. **Breaking:** `spine gate` now takes flags first —
  `gate [--dir D] <pack>[@<v>] <check>`; the old trailing-`--dir` form
  errors naming the rule (bare maipipe run lines are unaffected).
  Trailing `--force` on `cursor start`/`tick` and `version`/`help`
  leniency are unchanged. (I119)

### Added

- **Routing audit can judge complete heterogeneous dispatch declarations.** Exact final host routes and byte-exact host-local observed IDs distinguish nonblocking `unconfirmable` evidence from blocking declared-effort or declared-observed mismatches, while preserving independent legacy silent descent. Complete event identity and a linked worker are mandatory; current transports still expose no observed effort, so production reports `-` rather than inventing confirmation. (I074)
- **Dispatch effort is now an explicit raw declaration.** Final target resolution can apply a byte-exact `--dispatch-effort` in JSON mode, and routing audit retains `(harness, model, effort)` per dispatch/retry with exact ticket-local effort authorization records. Output remains declared-only—observed effort is `-`, existing model verdicts and blocking behavior are unchanged, and no cross-family effort ordering is inferred. (I075)
- **Pinned routes can cite exact repo-local eval evidence, with warning-only doctor advice.** `evidence_refs` may point to the pinned model's `docs/evals/.../runs/...` record; D17 warns on missing, malformed, stale, mismatched, missing-battery, or failing evidence. Reads are bounded and physically contained under the repository, including symlink/atomic-replacement defenses. No advisory de-ratifies, blocks, or gates an owner pin or changes model, validation, or routing audit behavior. (I077)
- **`spine update --force-file PATH` scopes overwrite authority to exact managed files.** The repeatable flag accepts canonical paths in the current plan, preserves unselected local edits, and keeps standalone `--force` global. Malformed, duplicate, unknown, unmanaged, or mixed requests fail deterministically before writes; selected marker damage still requires manual repair, and candidate-preflight refusal remains whole-plan no-write. (I124)
- **Host-scoped routing capability and equivalence pins.** An owner-local,
  closed JSON file can constrain repository preferences to available harnesses
  or select an exact model-effort pin without leaking host state into
  templates, mirrors, or updates. `spine model` exposes the requested/final
  trail, controlled validation remains fail-closed for divergent pins until
  heterogeneous audit support lands, doctor D16 diagnoses bad or unreachable
  routes, and routing audit performs structural preflight only. (I072)

- **`spine update` now advises before rendering enabled gate checks with
  missing required configuration.** The deterministic stdout lines name the
  class, missing `gate_pack_config` key, and both remedies without implicitly
  disabling a stage or changing update exits, diffs, or configured writes.
  Optional `tskip_allow` and config-free checks remain quiet. (I123)

- **Identity-scoped discarded dispatch records.** `spine audit routing` now
  recognizes `DISCARDED` ledger records for one exact Claude or direct Codex
  dispatch event and reports the event as advisory `discarded-with-reason`.
  Malformed, duplicate, zero-match, and multi-match records warn without
  excusing work; identity-less Codex worker evidence remains fail-closed, and
  a separate lower-tier event still blocks as `silent-descent`. Template
  generation 13 publishes the workflow grammar. (I078)

- **Fail-closed active model validation.** `spine model [--dir D] validate
  [--expect MODEL_ID] <flavor> <tier>` reads one repository snapshot and emits
  only the exact active ID. It refuses forbidden, unsafe, retired,
  wrong-tier, and unmapped IDs with stable exit-1 reasons; malformed
  invocation or repository policy exits 2; failures write no stdout. I119's
  flags-first contract remains binding: outer `--dir` precedes `validate`, so
  `spine model validate --dir ...` is a usage error. There is no bypass, and
  existing plain, JSON, effort, alternate, audit-alias, and audit-history
  behavior is unchanged. The controlled codex-team, claude-team, and handoff
  launch sites now validate locally before every spawn, pass only the captured
  model (and separately resolved effort where supported), and refuse plain
  modes that cannot prove the handoff. (I051)

- **Ticket-local `APPROVED-UNTESTED` acceptance records.** An applicable
  criterion may stay unchecked while recording a dated approver token,
  repository-local Markdown reference, and reason. Doctor D15 warns on
  malformed records or invalid provenance, while `spine audit stages` scans
  only cursor-resolved tickets and prints a nonblocking valid/invalid summary.
  Spine checks the referenced file but does not authenticate the approver or
  resolve the fragment. Template generation 12 publishes the grammar and
  migrates pristine generation-11 repositories without rewriting tickets.
  (I050)

- **`spine version` prints build provenance.** A second `build:` line carries
  the module version, 12-char vcs revision, vcs time, and a dirty flag from
  the binary's embedded build info (`build: (no build info)` when absent), so
  two devices compare installs with one command instead of hashing binaries.
  The first line is byte-identical to before. The README documents the
  portable install path: `go install github.com/russellpope/spine/cmd/spine@latest`.
  (I118)

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

- **`spine model` names the flag-ordering rule at the point of failure.** A
  flag placed after the positionals (or standing in for one) previously
  printed bare usage or an unknown-tier error, reading as a broken flavor;
  it now errors `flags must precede positionals (saw … after …)` naming the
  offending token. The leading-flag form is unchanged. (I116)
- **The implement-tick zero-evidence message no longer misdirects to a
  tickets typo.** When ledger lines for the anchored ids exist but none
  carries done/complete/completed as a whole word, the derivation detail now
  names that requirement; the typo hint remains for ids with no ledger line
  at all. (I117)
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
