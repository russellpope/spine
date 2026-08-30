# Pre-dispatch model validation (I051)

## Product requirement document

**Status:** approved for implementation

**Ticket:** I051, fail-closed pre-dispatch model validation

**Decision:** adopt Design C from
`.superpowers/sdd/I051-worker3-grill.md`. Spine will provide an atomic,
validated resolver. The controlled team skills will capture its output in a
guarded assignment immediately before each launch and will pass that exact
value to the launcher.

## Goal

Stop a controlled team launch before it consumes tokens when its model ID is
retired, ambiguous, unsafe, unknown, or different from the active route for
the requested `(flavor, tier)` key.

The result is model-ID validation, not full route-tuple validation. I075 owns
dispatch effort enforcement. Alternate model launches are outside I051.

## Authority and completion boundary

I051 crosses two repositories, but neither repository may write the other as
part of an implicitly broadened task.

- This Spine checkout owns the embedded policy, strict validation logic, CLI,
  audit matcher, and Spine documentation.
- `/Users/ldh/Projects/github.com/deepthought` owns the three team-skill source
  files and their shell regression tests.
- The installed `codex-team`, `claude-team`, and `handoff-to-codex` skills are
  symlinks into that deepthought checkout. A deepthought source edit is a live
  install change.
- Spine implementation, review, verification, shipment, and installation must
  finish before any deepthought source edit.
- Deepthought changes require a separate, explicit authorization. The Spine
  phase does not grant it.
- I051 remains open after the Spine phase. It may close only after both the
  Spine acceptance set and the separately authorized deepthought integration
  acceptance set pass.

This PRD authorizes no production code change by itself. It defines the work
that later implementation dispatches must perform.

## Current failure mode

`internal/model.Resolve` intentionally preserves compatibility. It resolves
embedded defaults, then a repository mirror, and returns historical shipped
rows as `Inherited`. That is correct for update and audit, but unsafe as a
launch decision. A mirror containing a retired ID can still resolve to that
retired ID.

Audit also accepts more evidence than a launch gate should. A
`resolvedTier` recognizes the active ID, current aliases, and selected
historical IDs. This keeps old transcripts readable. It must not permit a new
launch on an alias or historical value.

The controlled skills currently perform a one-time capability probe and then
resolve at launch sites. Some launch recipes put `spine model` inside command
substitution in the outer launcher. A failed command substitution does not
guarantee that the outer command stops. Other recipes resolve a value earlier
without proving that the checked value reaches the launcher.

## Atomic command contract

The new grammar is:

```text
spine model [--dir D] validate [--expect MODEL_ID] <flavor> <tier>
```

This is a two-layer flags-first grammar, matching the binding I119 contract.
`--dir` is an outer `model` flag and must precede the `validate` positional.
`--expect` belongs only to the nested validate parser and must follow
`validate` but precede the `<flavor>` and `<tier>` positionals. These forms are
valid:

```text
spine model validate codex primary
spine model --dir D validate codex primary
spine model --dir D validate --expect MODEL_ID codex primary
```

Passing the outer `--dir` flag to the nested validate parser is a usage
failure. It must not be accepted as a compatibility exception. An `--expect`
before `validate` or after the flavor/tier positionals is also a usage failure.
The existing resolver grammar remains byte-compatible:

```text
spine model [--dir D] [--alternate] [--effort|--json] <flavor> <tier>
```

The new verb does not accept `--alternate`, `--effort`, `--json`, `--force`,
or a bypass flag. An explicitly empty `--expect` is bad usage.

Without `--expect`, the command reads the repository policy once, resolves
the requested cell from that snapshot, validates its active ID, and prints
the validated ID. With `--expect`, it performs the same single-snapshot work,
then also classifies and compares the candidate. It prints the active ID only
when the candidate is byte-equal to it.

Success output is exactly the validated ID followed by one newline. Stderr is
empty. Every failure writes no stdout.

This command is atomic across policy read, resolution, validation, and output
inside one Spine process. It is not a transaction with the later vendor
process.

## Resolution and validation order

Validation follows this order against one in-memory snapshot:

1. Load and validate the embedded model table and `modelValidation` policy.
2. If `WORKFLOW.md` is absent, use the embedded current route.
3. If `WORKFLOW.md` is present, require it to be readable. Parse its template
   generation and the requested mirror row strictly.
4. Reject a supported-generation ambiguity or malformed requested row as a
   configuration error. Do not fall back to the embedded value.
5. Select the dotted `<flavor>.<tier>` row when present. For Claude only, use
   the legacy bare `<tier>` row when no dotted row exists.
6. Classify the selected ID. The embedded current ID is active. An ID in the
   requested cell's shipped history is retired. Any other ID is a deliberate
   repository override.
7. Apply the positive ID syntax and deny policy to the active ID. A safe
   deliberate override remains valid.
8. If `--expect` is absent, return that validated active ID.
9. If `--expect` is present, apply candidate syntax and deny checks, then use
   the candidate classification precedence below.

The existing `Resolve`, `HistoricalIDs`, `MirrorRows`, update provenance,
alias behavior, and old CLI modes remain unchanged.

## Strict repository mirror policy

The validation reader is intentionally stricter than `RoutingKeys` and
`readOverride`. It applies only to launch validation.

- An empty `repoDir`, a nonexistent directory, or an absent `WORKFLOW.md`
  means embedded defaults.
- Any error other than file absence while reading a present `WORKFLOW.md`
  refuses as configuration input.
- A missing `model_routing` block or missing requested row means the embedded
  current route.
- A missing `template_version` is accepted for legacy repositories. A present
  value must be one decimal integer no greater than the binary's compiled
  generation. Empty, duplicate, malformed, or newer values refuse.
- More than one `model_routing:` block refuses. The validator must not guess
  which block controls a launch.
- The exact dotted requested key may occur at most once. The selected legacy
  bare Claude key may occur at most once. One dotted row plus one bare row is
  legal and the dotted row wins.
- An exact-key line with no colon, an empty value, multiple model IDs,
  repeated effort separators, or malformed alternate syntax refuses. A
  malformed unrelated row does not block another requested key.
- The strict value parser applies the existing `CommentIndex` rule before it
  parses model, effort, and alternate fields. A `#` inside an ID remains data;
  the safe-ID grammar later rejects it for launch.
- The selected row uses the existing model/effort/alternate grammar only when
  that grammar is unambiguous. Existing effort-vocabulary errors still
  refuse. Validation does not compare the later launch effort.
- A selected row whose ID equals the embedded current ID is active even when
  its effort or alternate makes existing provenance report `Override`.
- A selected row whose ID equals any historical ID for the requested cell is
  `retired-model`, regardless of an edited effort or alternate.
- Any other selected ID is a deliberate override. It passes only when it
  satisfies the safe-ID and deny rules.
- A safe custom candidate passed only through `--expect` is not an override.
  The repository must declare it in the selected mirror row before it can be
  active.

The strict reader must read `WORKFLOW.md` once. Candidate classification for
other active tiers uses the same parsed snapshot, not a second file read.
Only unambiguous other-tier rows enter the `route-mismatch` set. A malformed
unrelated row remains non-blocking and contributes no candidate; a value that
depends on that row therefore falls through to another applicable refusal
reason, normally `unmapped-dispatch`.

## Active-ID-only launch matching

Launch matching uses byte-for-byte equality with the active ID for the
requested key.

It performs no whitespace trimming, case folding, substring matching, alias
expansion, family inference, or normalization. Current IDs shared by multiple
tiers are legal because validation includes the requested key.

Aliases and historical IDs remain audit-only evidence. A launch never passes
because its candidate is in `Entry.Aliases` or `HistoricalIDs`.

For an `--expect` candidate that is not byte-equal to the active ID, apply
these reasons in order:

1. `invalid-model-id` when it fails the positive syntax.
2. `forbidden-model` when an exact token or named pattern denies it.
3. `route-mismatch` when it is an active ID for another tier of the same
   flavor in the same snapshot.
4. `retired-model` when it is historical evidence for the flavor and is not
   currently active for any tier.
5. `unmapped-dispatch` for every other safe, non-forbidden candidate.

This ordering keeps a currently shared ID active even if the same string also
appears in old history. A shorthand alias such as `opus` reports the deny
rule rather than being softened into audit compatibility.

## Embedded safe-ID and deny policy

`models/defaults.json` gains this closed top-level object beside the table:

```json
"modelValidation": {
  "idPattern": "^[A-Za-z0-9][A-Za-z0-9._/:+-]{0,127}$",
  "forbiddenTokens": [
    "auto", "default", "latest",
    "fable", "opus", "sonnet", "haiku",
    "sol", "terra", "luna",
    "kimi-k3", "deepseek-v4-pro", "glm-5.2",
    "qwen3.8", "qwen"
  ],
  "forbiddenPatterns": [
    {"name": "generic-selector", "re": "(?i)^(auto|default|latest)$"},
    {"name": "bare-family", "re": "(?i)^(fable|opus|sonnet|haiku|sol|terra|luna|qwen)$"},
    {"name": "vendor-auto", "re": "(?i)(^|[-._/:])auto($|[-._/:])"}
  ]
}
```

The object accepts exactly `idPattern`, `forbiddenTokens`, and
`forbiddenPatterns`. Each pattern object accepts exactly `name` and `re`.
Unknown members are invalid embedded policy.

The ASCII allowlist runs first. It admits 1 through 128 bytes and rejects
whitespace, control bytes, quotes, dollar signs, backticks, semicolons,
parentheses, backslashes, and shell operators. Launchers must still pass the
result as a quoted argument.

`forbiddenTokens` uses byte-exact equality. Named regular expressions use
Go's RE2 engine against the original ID. Their anchoring or token boundaries
must be explicit in the expression. This allows the exact lowercase alias
`deepseek-v4-pro` to be forbidden while the current ID `DeepSeek-V4-Pro`
remains valid. `automatic-model` does not match the `vendor-auto` boundary
rule.

Embedded table validation rejects:

- a missing or malformed policy member;
- an invalid regular expression or empty/duplicate pattern name;
- an empty or exact-duplicate forbidden token;
- a current shipped ID that fails `idPattern` or matches any deny rule;
- a historical ID that fails `idPattern`;
- any shorthand alias that differs from its cell's full current ID and is
  absent from `forbiddenTokens`;
- any unknown field in the policy or pattern schema.

These are binary build invariants and follow the existing `validateTable`
panic convention. Runtime repository values produce typed refusals or
configuration errors instead.

## No bypass

I051 adds no `--force`, environment switch, allow file, ledger exception, or
fallback behavior.

`ESCALATION` and `FALLBACK` records authorize a tier choice. They do not make
an ambiguous or unsafe model ID deterministic. A dispatcher selecting another
tier validates that tier's active route and writes the existing routing
record. A dispatcher selecting a safe custom model first declares it as a
repository override, then validates it.

Validation failure stops the launch. The controlled skill must not retry with
plain `spine model`, omit the model flag, or let the vendor choose a default.

## Exit codes, reasons, and diagnostics

| Exit | Meaning | Output contract |
| --- | --- | --- |
| 0 | Active route validated, and `--expect` matched when supplied | Exact ID plus `\n` on stdout; empty stderr |
| 1 | Well-formed validation request refused by launch policy | Empty stdout; one escaped diagnostic line on stderr |
| 2 | Bad invocation, unknown flavor/tier, unreadable or malformed repository input, or unsupported generation | Empty stdout; diagnostic and usage when applicable |

An invalid embedded schema is a binary invariant failure in
`mustLoadDefaults`, not a runtime exit-2 case. Its tests must fail the build
before a binary can ship.

Exit-1 reason tokens are stable:

- `forbidden-model`
- `invalid-model-id`
- `retired-model`
- `route-mismatch`
- `unmapped-dispatch`

Every diagnostic begins `model validate: <reason>:` for exit 1 or
`model validate:` for exit 2. It names the `<flavor>.<tier>` key. Deny
failures name either `token:<value>` or the configured pattern name.
Untrusted values use Go `%q`, so newline and control bytes cannot forge log
lines. No refusal prints an offending value to stdout.

Examples:

```text
model validate: forbidden-model: claude.routine candidate "opus" (rule: token:opus)
model validate: retired-model: claude.routine resolves historical id "claude-sonnet-5"; refresh WORKFLOW.md with 'spine update --dir "/repo" --write'
model validate: route-mismatch: codex.primary expects "gpt-5.6-sol", got active codex.routine id "gpt-5.6-terra"
model validate: unmapped-dispatch: codex.primary does not map candidate "bespoke-safe"
model validate: invalid-model-id: codex.primary resolves "bad;id" (allowed: ASCII letters, digits, '.', '_', '/', ':', '+', '-')
```

The refresh hint prints the caller's quoted repository directory, not a fixed
path.

## Audit invariant

Audit verdict names and severity do not change in I051. Validation and audit
share one exact active-ID predicate from `internal/model`; audit then layers
aliases and history on top for evidence compatibility.

| Launch classification | Audit relationship |
| --- | --- |
| validated | The exact active ID must be in audit's candidate set for the same flavor and unchanged repository policy |
| route-mismatch | The candidate is another active tier and audit may judge its tier relationship |
| unmapped-dispatch | No active route for that flavor maps the candidate in the snapshot |
| forbidden-model or retired-model | Launch-only refusal; audit may still recognize retained evidence |

The binding invariant is state-scoped: an ID that passes validation cannot
later receive `unmapped-dispatch` for the same flavor when audit reads the
same repository policy. The invariant does not promise a particular audit
tier verdict and does not span a policy change.

Validation must not call audit. Audit must not copy a third active-ID matcher.
`resolvedTier.matches` retains its alias and history loops, but delegates its
active leg to `internal/model`.

## Controlled spawn-site dataflow

The separately authorized deepthought phase changes exactly these eight
controlled launch sites:

| Site | Source section | Captured variable | Required launcher use |
| --- | --- | --- | --- |
| 1 | `skills/codex-team/SKILL.md`, cmux worker workspace creation | one `WORKER_N_MODEL` per worker | each `codex -m` uses only its corresponding variable |
| 2 | `skills/codex-team/SKILL.md`, herdr Master lead start | `LEAD_MODEL` | `herdr agent start ... -m "$LEAD_MODEL"` |
| 3 | `skills/codex-team/SKILL.md`, herdr Lead worker start | `WORKER_MODEL` | `herdr agent start ... -m "$WORKER_MODEL"` |
| 4 | `skills/claude-team/SKILL.md`, cmux role launch | `ROLE_MODEL` | `claude ... --model "$ROLE_MODEL"` inside the cmux command |
| 5 | `skills/claude-team/SKILL.md`, herdr Master lead start | `LEAD_MODEL` | `herdr agent start ... --model "$LEAD_MODEL"` |
| 6 | `skills/claude-team/SKILL.md`, herdr Lead role start | `ROLE_MODEL` | `herdr agent start ... --model "$ROLE_MODEL"` |
| 7 | `skills/handoff-to-codex/SKILL.md`, cmux lead spawn | `LEAD_MODEL` | `cmux new-workspace ... codex -m "$LEAD_MODEL"` |
| 8 | `skills/handoff-to-codex/SKILL.md`, herdr lead spawn | `LEAD_MODEL` | `herdr agent start ... -m "$LEAD_MODEL"` |

Every site uses a separate guarded assignment immediately before launch:

```sh
if ! WORKER_MODEL=$(spine model --dir "$SDD_CWD" validate codex "$TIER"); then
  echo "codex-team: model preflight refused codex.$TIER; no worker spawned" >&2
  return 1
fi
test -n "$WORKER_MODEL" || return 1
herdr agent start "$SLOT" --kind codex --pane "$PANE" -- -m "$WORKER_MODEL"
```

The exact control keyword may be `exit 1` when the recipe runs in a script and
`return 1` when it runs in a sourced/function context. Each skill must state
the correct context. It may not rely on `${VAR:?}`, a prior capability probe,
or command substitution nested inside `cmux`, `herdr`, `claude`, or `codex`.

Claude launch sites continue to resolve effort through the existing
`spine model --effort` path in a separately guarded assignment. The model
variable comes only from `spine model validate`. The effort variable is not
proof that I051 validated effort.

The three shared preflights also probe the new verb and retain the install or
rebuild hint. They do not replace per-spawn validation.

Source tests must establish local flow at every named site. The presence of
the text `spine model validate` elsewhere in a skill is insufficient. Tests
must fail when one site removes `validate`, removes the guard, launches a
literal or different variable, or lets a failed validator reach a launcher.

## Plain mode and uncontrolled dispatches

`claude-team` currently falls back to plain subagent-driven development. That
path dispatches through an upstream Agent-tool interface and provides no
executable place to pass a validated model ID. I051 changes the `plain`
branch to refuse with this meaning:

```text
claude-team: plain mode cannot prove pre-dispatch model validation; run under cmux or herdr. No worker spawned.
```

The implementation may adjust punctuation to match the skill, but it must
retain `plain mode`, `cannot prove pre-dispatch model validation`, and
`no worker spawned` for regression tests.

`codex-team` and `handoff-to-codex` already refuse plain mode and keep doing
so. Arbitrary direct `spawn_agent` or Agent-tool calls outside these three
skills remain outside Spine's executable control. AGENTS prose cannot turn
the CLI into a global launch interceptor. I051 makes no such claim.

## TOCTOU and threat boundary

The guarded assignment closes the avoidable races:

- resolution and validation use one repository snapshot;
- the launcher receives the exact captured value;
- a failing substitution stops before the outer launcher;
- the safe-ID grammar prevents the captured ID from carrying shell or log
  control syntax, and the launcher still quotes it.

The repository may change after validation and before launch or audit. The
captured value remains syntactically safe, but a later audit may read a new
mapping. I051 adds no receipt, policy digest, file lock, or file-descriptor
handoff to the vendor process. A process with local write access can also
replace a skill or launcher after validation. The local user and checkout are
trusted for this ticket.

Effort is a second mutable read at Claude sites until I075 binds it. I051 does
not claim that the model and effort form one validated snapshot.

## Binary-first rollout and cross-repository sequence

The order is load-bearing:

1. Implement and test the Spine policy, strict validator, CLI, audit matcher,
   and documentation in `/Users/ldh/Projects/github.com/spine`.
2. Run a fresh requirements-attack spec review and independent verification
   against this PRD.
3. Ship the exact verified Spine SHA. Install it with `make install`, then
   copy `~/bin/spine` to `~/.local/bin/spine`.
4. Prove both installed paths recognize `model validate` and that a valid
   route succeeds.
5. Stop. Obtain separate authorization for the deepthought landing.
6. In `/Users/ldh/Projects/github.com/deepthought`, add failing site-scoped
   tests before editing skills. Change the three skill sources and the two
   shell tests only after the red controls are recorded.
7. Verify all eight launch sites, the plain-mode refusal, old-binary refusal,
   and no hardcoded launch IDs. Commit and land the deepthought change under
   that repository's authority.
8. Run a final cross-repository spec review and verification using the landed
   Spine and deepthought SHAs.
9. Only then close I051 in Spine, naming both repositories' implementation
   commits and the review evidence. Re-run Spine's final exact-SHA gate and
   refresh both installed binaries if the closure commit changes build
   provenance.

Reversing steps 3 and 6 would change live symlinked skills to call a verb the
installed binary does not know. That rollout is forbidden.

## Documentation and migration

The Spine phase updates `README.md`, `CONTEXT.md`, and `CHANGELOG.md`. It does
not edit `templates/current/AGENTS.md.tmpl`,
`templates/current/WORKFLOW.md.tmpl`, or bump `templates/VERSION`.

`models/defaults.json` receives embedded policy only. No repository
`WORKFLOW.md` rewrite follows from that object. Existing `spine model` calls,
JSON, effort, alternate output, aliases, history, and mirror rendering stay
compatible.

Historical mirror rows now block controlled new launches until the owner runs
`spine update --dir <repo> --write`. Old transcripts remain auditable.
Legacy bare Claude rows remain readable: a safe custom bare override passes,
and a historical bare value refuses.

## Exclusions

- Compare, normalize, or transport dispatch effort. I075 owns it.
- Validate or launch `Entry.Alternate` values.
- Delete aliases or historical IDs from audit evidence.
- Change audit verdict names, severity, escalation, or fallback semantics.
- Add a global interceptor for direct Agent-tool or `spawn_agent` calls.
- Add a validation receipt, policy digest, lock, or local-writer defense.
- Change workflow templates or their generation.
- Patch an installed plugin cache or copied skill outside deepthought source.
- Edit deepthought without separate authorization.

## Requirements attack and resolutions

| Attack | Resolution |
| --- | --- |
| `Resolve` says an inherited historical row is valid, so rejecting it contradicts the resolver. | `Resolve` remains the compatibility authority for update and audit. Launch validation adds a stricter history classification and does not change `Resolve`. |
| A safe custom ID would be unknown to the embedded table and therefore always fail. | A safe custom ID passes only when the repository's selected mirror row declares it as the active override. `--expect` alone cannot create an override. |
| Alias/history compatibility could make an old or shorthand ID launchable. | Launch comparison uses only the active ID. Audit adds aliases and history after the shared exact active-ID leg. |
| A denylist can never enumerate shell injection strings. | The positive ASCII grammar rejects control and shell syntax first. The deny policy handles syntactically safe but semantically ambiguous selectors. |
| Case-insensitive deny matching would reject the valid current `DeepSeek-V4-Pro`. | Exact tokens stay byte-sensitive. Only named patterns opt into `(?i)`, and table validation proves no current shipped ID matches them. |
| `auto` substring matching would reject legitimate names such as `automatic-model`. | `vendor-auto` uses explicit separators or string boundaries. A negative control pins `automatic-model` as legal when it is the active override. |
| An ESCALATION line could be treated as a model bypass. | Records choose or explain tiers. The selected tier still validates and there is no model-ID bypass path. |
| A capability probe at skill startup already proves Spine can resolve models. | A startup probe neither binds the later value nor sees every per-tier repository state. Every launch site captures and uses its own validated result. |
| `$(spine model validate ...)` inside `herdr agent start` looks atomic. | Shell command-substitution failure does not reliably stop the outer command. A separate guarded assignment is mandatory. |
| Validating model and effort in I051 would close the whole route. | Effort has separate grammar, normalization, and escalation decisions in I075. I051 preserves resolver vocabulary errors but makes no dispatch-effort claim. |
| Alternate is another model ID and should be accepted. | Team skills do not launch alternates, and audit does not map them as active routes. Alternate validation is excluded. |
| Plain SDD is still a spawn path, so keeping fallback would satisfy convenience. | The upstream Agent-tool path has no executable model handoff. `claude-team` plain mode refuses rather than making a false every-spawn claim. |
| Validation can race a mirror edit before audit. | The guarantee is explicitly limited to unchanged policy. Receipts and digests are future work. |
| Spine can ship the CLI and silently edit deepthought because the skills are symlinked. | The symlink makes the authority and rollout risk stronger, not weaker. Deepthought requires a separate authorization after binary installation. |
| Closing I051 after Spine ships would treat the consumer integration as optional. | The ticket remains open until the deepthought tests and eight controlled sites pass and a final cross-repository review verifies both SHAs. |
| The original I051 draft assigned `--dir` to the nested parser, which conflicts with I119's flags-first command contract. | `--dir` belongs to the outer `model` parser and precedes the `validate` positional. Only `--expect` belongs to the nested parser. Supplying an outer flag to the nested parser remains a usage failure. |

## Acceptance criteria

1. `spine model [--dir D] validate [--expect MODEL_ID] <flavor> <tier>`
   obeys flags-first parsing at both parser layers, leaves the existing model
   command unchanged, and prints exactly one validated ID only on exit 0.
   `spine model --dir D validate ...` succeeds for valid input;
   supplying outer `--dir` to the nested parser, an outer `--expect`, and a
   nested flag after flavor/tier are usage failures.
2. Validation reads one repository snapshot. Missing `WORKFLOW.md` selects
   embedded defaults; unreadable present input, duplicate relevant blocks or
   keys, malformed selected rows, malformed template versions, and newer
   generations exit 2 with no stdout.
3. Embedded current rows and exact current mirror rows pass. A safe deliberate
   mirror override passes. A historical requested-cell row refuses with
   `retired-model`, including legacy bare Claude history and history with an
   edited effort.
4. `--expect` uses byte equality with the active requested ID. Current aliases
   refuse, historical IDs refuse, another active tier reports
   `route-mismatch`, and an otherwise safe unknown reports
   `unmapped-dispatch`.
5. `models/defaults.json` carries the exact closed `modelValidation` schema.
   Build-time tests reject bad regexes, duplicate tokens or names, unsafe
   current/history IDs, deny overlap with a current ID, unknown schema fields,
   and any shorthand alias omitted from the exact token inventory.
6. Empty IDs, overlength IDs, whitespace, control bytes, quotes, backticks,
   dollar signs, `$()`, semicolons, backslashes, and shell operators refuse as
   `invalid-model-id`. Stderr is one escaped line and stdout is empty.
7. Exact forbidden tokens and all named pattern classes refuse as
   `forbidden-model` and name the rule. `automatic-model` is not rejected by
   an `auto` substring check when declared as the active override.
8. No CLI flag, environment variable, allow file, ESCALATION record, or
   fallback path bypasses validation. Failure never retries plain resolution
   or omits the launcher's model argument.
9. Existing resolver errors, including Pi effort-vocabulary errors, remain
   blocking. I051 adds no effort equality, normalization, launch transport,
   or alternate validation claim, and old `--effort`, `--alternate`, and
   `--json` outputs remain unchanged.
10. Audit uses the shared exact active-ID matcher, keeps its public verdicts,
    aliases, history, shared-ID rules, escalation, and fallback behavior, and
    proves that a validated default or safe override cannot become
    `unmapped-dispatch` for the same flavor under unchanged policy.
11. A failed validator, including one that prints an ID before exit 1, causes
    zero calls to fake cmux, herdr, Claude, or Codex launchers.
12. Each of the eight named deepthought sites has a guarded assignment local
    to that launch and uses only the captured variable as its model argument.
    Mutation tests fail when one site removes `validate`, removes its guard,
    or launches a literal or different variable.
13. The three skill preflights probe `model validate` and retain a clear
    binary install/rebuild refusal. A pre-I051 fake binary cannot reach a
    launcher.
14. `claude-team` refuses plain mode with the required message. `codex-team`
    and `handoff-to-codex` retain plain refusal. No artifact claims control of
    arbitrary direct Agent-tool or `spawn_agent` calls.
15. Spine documentation states the model-ID-only, alternate, effort, audit,
    TOCTOU, trusted-local-writer, and cross-repository boundaries. Workflow
    templates and generation remain unchanged.
16. The finished Spine diff receives a fresh primary-tier requirements-attack
    spec review and independent verification before shipment. Focused tests,
    full uncached tests, race tests for model/audit, vet, build, CLI fixtures,
    `make verify`, doctor, and routing/stage audits pass with recorded output.
17. The exact verified Spine binary is installed at both `~/bin/spine` and
    `~/.local/bin/spine`, and both paths pass a valid `model validate` smoke
    before deepthought changes begin.
18. Deepthought changes begin only after separate authorization. Its relevant
    shell tests pass, its known research stray and unrelated work remain
    unstaged, and the landing commit contains only the authorized skill/test
    paths.
19. A final fresh primary-tier review and verifier inspect both repository
    diffs and rerun the cross-repository fake-launch and installed-binary
    acceptance cases.
20. I051 closes only after criteria 1 through 19 pass. Its resolution records
    the Spine implementation SHA, deepthought integration SHA, final review,
    verification, rollout order, and stated boundaries.
