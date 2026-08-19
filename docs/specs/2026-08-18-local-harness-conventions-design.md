# Local-harness conventions — checkpoint doc, gate packs, remediation, pi harness

Status: ready-for-agent
Date: 2026-08-18 (grilled same day from the deepthought PRD
`deepthought/docs/specs/2026-08-17-spine-local-harness-conventions-design.md`;
decisions owner-ratified in-session)
Glossary: CONTEXT.md — "harness", "alternate" (Model routing); "Checkpoint";
"Gate pack" (all added this grill)
ADRs: 0015 (checks in the binary, per-pack versioning, maipipe pipelines),
0016 (spine-managed region in `maipipe.toml`); respects 0002, 0004, 0007,
0011, 0014
Related work: deepthought map I023 (source decisions I026, I028, I030,
I033); spine map I066 (I067 harness split ratified — this spec adds the
`pi` harness; I072 host-scoped reachability is deliberately **not** pulled
forward); maipipe execution-floor PRD (consumer of the gate pack and the
checkpoint/hitlist briefs); Pi extension pack PRD (harness half of the
checkpoint dual-writer contract)

## Problem Statement

Weak local models running long dev tasks lose work at two seams spine
owns. First, context: one auto-compaction cost 5h04m (20.1% of a run), and
the one thing measured lost across a context clear was *forward intent* — a
deliverable that existed only as a plan in the model's head. There is no
document a session can distil itself into before a reload, no writer
discipline for it, and no cache-stable reload prompt. Second, enforcement
content: maipipe can run a verify gate, but there is no versioned,
attributable battery of deterministic checks for it to run, so the corpus's
recurring defect classes (t.Skip'd tests, ignored deferred-cleanup errors,
dead code, enum/spec drift, committed binaries, hidden entry points, missing
fixture manifests) ship again and again. Downstream of both, remediation
rounds have no shared shape — hitlists were free prose, do-not-regress was
oral tradition, budgets were unenforced — and the model table cannot express
"critique with a different model/effort than the author", which the corpus
shows is structurally necessary (zero self-flagged shortcuts, ever; the
full-tier bake-off's fabricated environment-facts Critical happened on the
primary driver).

## Solution

Four spine-owned conventions, each a small CLI surface plus templates, no
enforcement in spine itself (maipipe executes; the Pi extension triggers):

1. **Checkpoint document** — a new doc kind with a model region and a
   harness-written facts region, an uncommitted ordinal-numbered working
   home, a spine-shipped reload preamble, and `spine checkpoint new|latest|
   list` as sole writer of the facts region.
2. **Gate packs** — deterministic checks as `spine gate <pack> <check>`
   subcommands, versioned per pack (`go@1`), delivered as two maipipe
   pipelines inside a spine-managed region of `maipipe.toml`, opt-out and
   config via preserved WORKFLOW.md keys, positive controls in spine's own
   tests. The mutation battery is ported to Go under the pack.
3. **Remediation conventions** — a hitlist template (findings without
   fixes) and a round-record convention under `docs/remediation/`, budget
   derived by counting, dose escalation mechanically decidable from
   per-finding status.
4. **`pi` harness rows** in the model table (data-only) with the new
   optional `alternate` cell.

## User Stories

Checkpoint

1. As a Pi-extension author, I want `spine checkpoint new` to accept the
   model's narrative file plus machine facts and write a canonical
   checkpoint, so that the extension stays thin and never hand-formats
   spine documents.
2. As a reloaded session, I want `spine checkpoint latest` to print the
   reload preamble followed by the newest checkpoint byte-stably, so that
   the prefix is cacheable and I resume with my forward intent intact.
3. As a reloaded session, I want the preamble to tell me explicitly that
   the model region is my own prior claims and the facts region is
   evidence, so that I do not treat a stale narrative as verified truth.
4. As an operator, I want the model region validated for its three
   sections (task, conclusions-with-why, next moves) and the write refused
   when one is missing, so that a hollow checkpoint is never mistaken for a
   good one.
5. As a Pi-extension author, I want a facts-only fallback flagged
   `narrative: missing` after a failed retry, so that reload still has the
   machine facts and the gap is visible.
6. As an operator, I want checkpoints to accumulate as `NNN-<slug>.md` in an
   uncommitted working home, so that forensics and telemetry can see the
   sequence without polluting history.
7. As a session author, I want `spine handoff new` to auto-embed the newest
   checkpoint (facts verbatim, narrative under a "prior narrative — not
   evidence" heading), so that forward intent survives the session end too.
8. As an operator, I want `spine doctor` to advise on malformed or
   non-canonical facts regions and ordinal gaps, so that hand edits surface
   without blocking mid-effort.
9. As an operator, I want the checkpoint's git sha, timestamp and ordinal
   computed by spine and files-touched supplied by the extension, so that
   each fact comes from the party that actually knows it.
10. As a telemetry consumer, I want the facts region to carry gate status
    and recommended per-leg effort, so that checkpoint fires join the fact
    db with meaning.

Gate pack

11. As a maipipe pipeline author, I want each check class to be one stage
    invoking `spine gate go <check>`, so that a finding is attributable to
    `go@1/<check>` and I can drop a class by name.
12. As a repo owner, I want `spine update` to render the `gate-go` and
    `mutation-go` pipelines into a `# spine:begin gate-pack …` region of
    `maipipe.toml`, so that pack refreshes are one command and my own lanes
    are untouched.
13. As a repo owner, I want to compose the pack into my lanes with
    `pipeline = "gate-go"` (full) and `pipeline = "mutation-go"` (audit), so
    that enforcement vs advisory placement is my choice.
14. As a repo owner, I want `gate_pack`, `gate_pack_disabled` and
    `gate_pack_config` in WORKFLOW.md to survive `spine update` as choices,
    so that opting out of a class or pointing a check at my spec file is
    durable.
15. As a repo owner, I want `spine update` to create a `maipipe.toml`
    containing only the region when none exists, so that adoption is
    additive.
16. As a maipipe stage, I want every check to emit the results contract
    (`maipipe_results: 0`, status, findings with `code = go@1/<check>`,
    file/line) to `MAIPIPE_RESULTS` and a human summary otherwise, so that
    findings flow into evidence and hitlists unchanged.
17. As a spine maintainer, I want every check class to ship a positive
    control pair (known-good passes, seeded violation fails) in spine's
    test suite, so that a check that cannot fail cannot ship.
18. As a repo owner, I want `spine doctor` to check region integrity only
    (markers, canonical content), so that per-repo doctor cost is one file.
19. As a Go repo owner, I want the eight go@1 classes — `tskip`,
    `deferred-cleanup-errcheck`, `dead-code-callgraph`, `test-enum-vs-spec`,
    `n-plus-one`, `binary-hygiene`, `gitignore-control`, `fixture-manifest`
    — so that the corpus's measured defect classes are caught mechanically.
20. As a repo owner, I want `gitignore-control` to check two arms — every
    declared build output is ignored at the path actually used, and no
    entry-point source is ignored — so that the hidden-entry-point bug
    (three occurrences) cannot recur.
21. As a repo owner, I want `binary-hygiene` to detect committed binaries by
    content, so that a 27 MB executable never lands in history again.
22. As a remediation author, I want `spine gate go mutate` (Go port of the
    battery) to emit killed/survived rows through the results contract with
    an unmutated negative control, so that do-not-regress blocks are
    generated from machine rows.
23. As a repo owner, I want to pin `gate_pack: go@1` and move to `go@2`
    deliberately, so that a pack release never silently changes my gate.
24. As a repo owner, I want per-check inputs (spec path for
    `test-enum-vs-spec`, manifest path for `fixture-manifest`, binary list
    for `gitignore-control`) declared once in `gate_pack_config` and passed
    to stages as env, so that configuration lives with the other spine
    choices.

Remediation

25. As a remediation author, I want a hitlist template (file:line, finding,
    why it matters, do-not-regress block) with no fix text, so that the
    default dose is findings-without-fixes.
26. As a remediation author, I want round records at
    `docs/remediation/<effort>/round-N.md` with dose, hitlist ref, maipipe
    run id, verdict, and a per-finding status table keyed by the results
    contract `code`, so that "escalate only after a round fails on the same
    finding" is a lookup, not a judgment.
27. As an operator, I want the 3-round budget derived by counting records
    and a 4th round to require `extension-ratified-by:` in frontmatter, so
    that overrun is visible and owner-attributed.
28. As an operator, I want `spine audit stages` to advise (not block) on
    a round beyond budget without ratification, so that the convention has
    teeth without new blocking gates before evidence.
29. As a remediation author, I want the rescore-as-fresh-submission rule to
    be a cross-reference to the eval seam (ADR 0007), so that there is one
    definition.
30. As a repo owner, I want `spine init`/`update` to scaffold
    `docs/remediation/` (with a README stating the convention), so that
    round records have a home from day one.

Model table

31. As a dispatcher, I want `spine model pi <tier>` to resolve to
    qwen3.8-27b-q8_0 with an explicit per-cell effort, so that the pi
    harness participates in routing today.
32. As an evaluator stage, I want `spine model pi <tier> --alternate` to
    return the owner-tuned `(id, effort)` alternate, so that the critic
    differs from the author by data, not dispatch-time heuristics.
33. As an owner, I want `alternate` to accept the same model at a different
    effort, so that "opus @ low over a smaller model"-style tuning is
    expressible.
34. As a repo owner, I want pi rows in the WORKFLOW.md mirror
    (`pi.primary: … @ …`) with the same inherited/override semantics, so
    that per-repo remap works without I072.
35. As an owner on a differently-equipped host, I want the spec to leave
    reachability to I072, so that host-scoped pins land once, in one place.

## Implementation Decisions

Checkpoint

- New sibling package following the handoff/cursor pattern (own package,
  embedded template, `main.go` subcommand, doctor finding); no generic
  document-family abstraction is introduced.
- Working home: `.superpowers/sdd/checkpoints/NNN-<slug>.md`, uncommitted
  by convention (spine does not manage `.gitignore`; doctor advises if
  `.superpowers/` is not ignored). Ordinals reserved with the same
  exclusive-marker technique as handoffs.
- Document shape: frontmatter (`ordinal`, `created`, `effort`, `narrative:
  present|missing`), then two marked regions: `<!-- spine:checkpoint:model
  -->` (headed sections exactly `## Task`, `## Conclusions`, `## Next
  moves`) and `<!-- spine:checkpoint:facts -->` (canonical block: `touched`
  list, `gate: pass|fail|none`, `sha`, `effort_recommended`, `written`).
  The facts block obeys the cursor's sole-writer and canonical-form rules;
  a byte-drifted block is a doctor advisory (not an audit block, in v1).
- CLI: `spine checkpoint new --from <narrative.md> --touched <csv>
  --gate <pass|fail|none> --effort <level> [--slug s] [--facts-only]`;
  `spine checkpoint latest` prints preamble + doc (exit 1 when none);
  `spine checkpoint list`. Strict validation: missing/empty section →
  refuse (exit 2, names the section); `--facts-only` writes with
  `narrative: missing`. sha/timestamp/ordinal computed by spine; touched
  list is caller-supplied.
- Reload preamble is an embedded template (`checkpoint-preamble.md`),
  byte-stable across a template generation, stating the model/facts trust
  split and the facts-only case ("reconstruct intent from facts").
- `handoff new` embeds the newest checkpoint when the working home is
  non-empty, after the cursor block: facts verbatim, model region under a
  fixed "Prior narrative (model-authored, not evidence)" heading.
- Doctor: new advisory finding covering malformed facts region, non-canonical
  facts region, ordinal gaps, unignored `.superpowers/`.

Gate pack (per ADR 0015/0016)

- `spine gate <pack> <check> [--dir D]`; pack `go`, version `1`. Exit 0 =
  pass, 1 = findings, 2 = misconfiguration. When `MAIPIPE_RESULTS` is set the
  check writes the results-contract JSON there (`maipipe_results: 0`,
  `status`, `summary`, `findings[]` with `severity`, `message`, `file`,
  `line`, `code = "go@1/<check>"`); otherwise a human table on stdout.
- Check definitions (v1): `tskip` — any `t.Skip*` call in `_test.go` is a
  finding (zero tolerance; allowlist via config); `deferred-cleanup-errcheck`
  — deferred calls whose error return is discarded on cleanup-class
  functions (Close/Remove/Flush/etc.); `dead-code-callgraph` — exported
  and unexported functions unreachable from any main/test root via
  `go/packages`; `test-enum-vs-spec` — enum/const set in code vs values
  enumerated in the configured spec file; `n-plus-one` — call-in-loop
  pattern against configured client method names; `binary-hygiene` —
  tracked files that are executables/archives by content, and stray second
  module trees; `gitignore-control` — arm 1: each configured build output
  path is ignored at that path; arm 2: no `package main` file is ignored;
  `fixture-manifest` — configured manifest path exists and is non-empty
  (content judgment is the evaluator's, never here); `mutate` — Go port of
  the behavioural mutation battery, killed/survived rows as findings, an
  unmutated negative control mandatory, exit 1 only on control failure
  (advisory lane).
- Delivery region in `maipipe.toml`: `# spine:begin gate-pack go@1` …
  `# spine:end`, containing `[pipelines.gate-go]` (profile full; one stage
  per enabled class, `run = "spine gate go <check>"`, `env` from
  `gate_pack_config`) and `[pipelines.mutation-go]` (profile audit, one
  stage). Regenerated by `spine update` under ADR 0002 rules; created with
  only the region when the file is absent.
- WORKFLOW.md keys (preserved choices): `gate_pack: go@1`,
  `gate_pack_disabled: [..]`, `gate_pack_config: {test_enum_spec, fixture_manifest,
  build_outputs, n_plus_one_clients, tskip_allow}`. Rendering omits disabled
  classes. Template generation bumps to 11 for the new keys and
  `docs/remediation/`.
- Doctor: region markers present and content canonical for the pinned pack
  version; nothing else per repo.
- Positive controls: fixture repos per check under spine's testdata, each
  with a passing and a seeded-violation variant, run at the CLI seam.
- Supersedes ADR 0013 items 2 and 4; the `/model-eval` skill's Python
  runner remains until the port ticket lands, then calls the binary.

Remediation

- Templates embedded: `hitlist.tmpl.md` (header: effort, round, dose,
  source run id; per finding: `code`, file:line, finding, why-it-matters,
  do-not-regress block listing killed mutation rows by `code`; no fix
  text), `remediation-round.tmpl.md` (frontmatter `round`, `dose:
  findings-only|prescriptive|raw-review`, `hitlist`, `run_id`, `verdict`,
  optional `extension-ratified-by`; body: per-finding table `code | status
  open|fixed|regressed | note`), and `docs/remediation/README.md`
  (convention text: 3-round budget, dose escalation rule, rescore-as-fresh
  cross-ref to the eval seam).
- `spine init`/`update` scaffold `docs/remediation/README.md`; round records
  and hitlists are hand-authored from templates (no renderer in spine —
  rendering from run facts is maipipe's evidence renderer's job).
- `spine audit stages` gains one advisory rule: for the cursor's effort,
  a `round-4+` record without `extension-ratified-by` is reported, never
  blocking.
- Stable finding ids come from the results contract `code`; the round
  record keys on it, which is what makes "same finding failed last round →
  prescriptive dose" mechanical.

Model table

- `models/defaults.json` gains a `pi` harness (JSON key remains under the
  legacy `flavors` map until I073's rename; the resolver's flavor→harness
  rename is I073's, not this spec's) with explicit per-cell efforts:
  primary qwen3.8-27b-q8_0 @ xhigh, routine @ medium, mechanical @ low,
  fallback @ xhigh; every cell carries `alternate: {id: qwen3.8-27b-q8_0,
  effort: xhigh}`. Model ids are the served identifier verbatim (must match
  the `id` under Pi's `lmstudio` provider); aliases `qwen3.8`, `qwen`.
- Cell schema gains optional `alternate: {id, effort}`; `history` entries
  may carry an alternate too so inherited-vs-override detection covers it.
- pi effort vocabulary is `low | medium | xhigh` (no `high`); resolution
  for pi rejects `high` with a clear error rather than mapping it. Effort
  translation to the model's reasoning aliases is the harness's job (Pi
  pack / dispatch), not spine's.
- CLI: `spine model pi <tier> [--alternate] [--effort|--json]`; `--json`
  includes `alternate` when present. `MirrorRows()` renders pi rows into
  WORKFLOW.md with alternate as a trailing `alt: id @ effort` on the same
  line, parsed back with the same inherited/override rules.
- No reachability/doctor check for pi models (I072).

## Testing Decisions

- A good test drives the public CLI against a tempdir fixture repo and
  asserts observable outcomes — files written, bytes of canonical blocks,
  results-contract JSON, exit codes, stdout — never internal call shapes.
- Seam: the CLI dispatch (prior art `cmd/spine/main_test.go`) plus fixture
  repos regenerated by `spine update` (prior art `internal/update` gen
  tests and testdata). Package-level tests only for canonical-form byte
  determinism of the facts region (prior art `internal/cursor`).
- Checkpoint: new/latest/list round-trip; strict section validation
  (refuse + names section); facts-only path sets `narrative: missing`;
  latest output is byte-identical across two invocations (cache premise);
  handoff embeds newest checkpoint; doctor advisory fires on a mutated
  facts block (negative control: canonical block → no finding).
- Gate pack: every check has a passing fixture and a seeded-violation
  fixture (positive control pair) at the CLI seam, asserting exit code and
  the `code` on each finding; results-contract JSON validated against the
  documented required keys; `spine update` renders the region into an
  absent file, an existing file with user lanes (lanes untouched), and
  respects `gate_pack_disabled`; doctor detects a broken marker.
- Remediation: scaffold creates `docs/remediation/README.md`; audit stages
  advisory on round-4 without ratification (negative control: with
  ratification → clean).
- Model: `spine model pi routine` / `--alternate` / `--json`; `high` for pi
  errors; mirror rows round-trip inherited vs override for a cell with an
  alternate; existing claude/codex tests unchanged.
- Fleet negative control: `spine update` on a repo without `gate_pack`
  writes no `maipipe.toml`.

## Out of Scope

- Triggering checkpoints, computing files-touched, or reloading (Pi
  extension pack); executing gates, relaying claims, rendering evidence,
  running the evaluator (maipipe); session observation (watchdog).
- Any blocking audit rule for checkpoints or remediation rounds (advisory
  first; blocking waits on the joint A/B).
- The flavor→harness rename (I073), host-scoped reachability and pins
  (I072), heterogeneous audit verdicts (I074).
- A hitlist/round renderer from run facts (maipipe evidence renderer's
  territory).
- Non-Go packs; a maipipe include mechanism; spine managing `.gitignore`.
- The measurement itself: the checkpoint A/B pre-registration is joint
  with the Pi pack and lives in deepthought `docs/research/`; the
  remediation-dose A/B likewise. This spec ships the instruments those
  A/Bs need.

## Further Notes

- Cross-product assumptions on maipipe (region tolerated in `maipipe.toml`,
  `spine` on the daemon PATH, `code` field usage, pipeline names,
  fixture-manifest split) are to be filed as a maipipe ticket; the
  deepthought PRD gets a dated amendments section (R4 "local flavor" →
  `pi` harness; scripts → binary; snippet → region).
- The full-tier bake-off's fabricated environment-facts Critical is the
  concrete adversary behind the model/facts split and the strict preamble
  wording; the typed-claims gap for environment claims is maipipe's to
  grill.
- Suggested build order inside this spec: model-table rows (smallest,
  unblocks maipipe evaluator) → checkpoint package (unblocks Pi pack R2) →
  gate pack static classes + region + WORKFLOW keys (unblocks maipipe R2)
  → mutation port → remediation templates + audit advisory (Phase 3 per
  the estate build order).
