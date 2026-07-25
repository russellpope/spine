# Flavor-aware model table (template gen 10) — design

Source: grilled 2026-07-24 (deepthought session, decisions Q1–Q13 + seam check). Build tickets: [I033](../issues/I033-model-table-resolver.md)–[I039](../issues/I039-fleet-sweep-gen10.md) — the ledger carries the plan; there is no separate plan document. Triage: ready-for-agent.

Supersedes the placement clause of [ADR 0010](../adr/0010-model-routing-tiers-not-ids-contract-lives-in-estate-owned-surfaces.md); its headline decision (artifacts name tiers, never ids) is preserved and strengthened. Interacts with [ADR 0002](../adr/0002-ownership-split-with-config-preserving-regeneration-and-choice-vs-default.md) (choice-vs-default).

## Problem Statement

The tier→id mapping is materialized into every repo's `WORKFLOW.md`, so a new model release is a fleet-wide file migration rather than a config change. Worse, it cannot actually be migrated: the choice-vs-default rule classifies any on-disk value differing from the current template default as a deliberate per-repo choice and carries it forward. This was verified empirically — a probe with a changed fallback value returned it as a choice. Changing a model id in the template therefore propagates **nothing**; each repo's stale value silently wins. The prior generations never exposed this because gen 5→6 changed only comments and structure, never a value.

Half the mapping is not in `WORKFLOW.md` at all. Codex model ids live in skill prose: the codex-team playbook pins a single worker model as a shell variable, and the handoff playbook pins the lead with a literal `-m` argument. Because that half is prose, nothing validates it, and it has already failed in the field — the pinned worker model flattens per-tier routing so every spawned worker runs the routine-tier model regardless of its ticket's declared tier, under-modelling primary-tier tickets. The owner caught it manually mid-run.

`spine audit routing` is blind to all of this. It re-parses `WORKFLOW.md` with a second, independent parser, and has no flavor dimension whatsoever, so codex-executed work cannot be routing-audited at all. It also has no `template_version` gate, so an older binary will read a newer file and emit confident verdicts from a misparse.

Finally, effort is not expressible where it is needed. Effort is defined as a function of tier alone, but the owner needs it per flavor — codex primary at xhigh while claude primary stays high — because the tier's default effort does not always produce the expected behavior. And the top-level `effort:` key encodes the whole tier-default table inside a trailing comment, where it can disagree with reality and nothing notices.

Underneath all four is one economic need the current design cannot serve: choosing, per project, which real model backs the primary tier — the most capable model for genuinely hard work, a cheaper capable one for projects that do not warrant it — without editing seventeen repos or touching a single plan or ticket.

## Solution

Move the tier→id mapping out of per-repo files and into spine, keyed by **flavor** (`claude`, `codex`) as well as tier, with effort as part of each entry.

Spine ships the defaults in a versioned, embedded data file that also records every default it has ever shipped. Each repo's `WORKFLOW.md` keeps a **mirror** of the resolved table, rendered and marked as spine-managed; editing a value in place turns that entry into a per-repo override. Because spine knows its own shipped-default history at runtime, it can distinguish an inherited old default from a deliberate override by direct lookup rather than by re-rendering a template — which is precisely what the choice-vs-default rule cannot do, and precisely why the current design cannot migrate a value. Inherited defaults are refreshed on update and itemized in the plan; overrides are preserved untouched.

A new read-only `spine model <flavor> <tier>` resolves an entry and prints it. The team skills stop hardcoding ids and call it per dispatch, with a spine-presence check joining the preflight they already run. `spine audit routing` consumes the same resolver instead of its own parser, gains flavor-scoped resolution with explicit aliases, and gains the missing version gate.

Template generation 10 carries the migration: the mirror gains its flavor axis in a syntax chosen so that an un-upgraded binary degrades to a loud warning rather than a silent misparse, and the top-level `effort:` key is retired into the table.

The Opus 5 refresh that prompted this work becomes a one-line data change in the defaults file — the first entry the new mechanism carries, not a special case.

## User Stories

1. As a repo owner, I want to set which real model backs the primary tier per project, so that a project that does not warrant the most capable model can run a cheaper one without touching any plan or ticket.
2. As a repo owner, I want that per-project choice to survive `spine update`, so that regenerating machine-owned files never silently reverts a deliberate decision.
3. As the fleet owner, I want a new model release to be one edit to a committed data file plus a rebuild, so that adopting a new model is not a seventeen-repo sweep.
4. As the fleet owner, I want repos still carrying a previous default to be refreshed automatically on update, so that the fleet converges on the current model without me visiting each repo.
5. As the fleet owner, I want each refreshed value itemized in the update plan, so that model-id changes are reviewable rather than lost among template prose churn.
6. As a repo owner who deliberately pinned an older model, I want that value preserved and reported as an override, so that automatic refresh cannot overwrite an intentional choice.
7. As a codex team lead, I want each worker spawned at its ticket's declared tier, so that a primary-tier ticket is never executed by the routine-tier model.
8. As a codex team lead, I want the lead's own model resolved from the table, so that the orchestration tier is not a literal pinned in prose that drifts from the routing contract.
9. As a claude team lead, I want my model resolved from the project's primary tier, so that the credit-sizing judgment currently written as prose guidance becomes a per-project setting.
10. As a skill maintainer, I want no model id hardcoded in any team skill, so that the routing-flattening bug cannot be reintroduced by an ordinary edit.
11. As a skill author, I want a missing spine binary to refuse early with an install hint, so that a dispatch never silently falls back to a stale or guessed model.
12. As the owner, I want codex primary to run at xhigh while claude primary runs at high, so that each flavor gets the effort that actually produces the expected behavior.
13. As a repo owner, I want an entry that omits effort to inherit its tier's default, so that every dispatch resolves to a determinate effort and nothing is left to an unread config file.
14. As a repo owner, I want to override effort on any single entry, so that a tier whose default effort underperforms can be corrected without changing the tier's meaning everywhere.
15. As the verify gate, I want routing resolution to be flavor-scoped, so that a model id is interpreted against the flavor that actually ran it rather than matched blindly across all of them.
16. As the verify gate, I want aliases declared explicitly in the table, so that tier resolution does not depend on substring matching that will collide as model names multiply.
17. As the verify gate, I want the audit to refuse a `WORKFLOW.md` stamped newer than the binary compiles, so that an un-upgraded binary cannot emit confident verdicts from a misparse.
18. As a fleet owner mid-sweep, I want an un-upgraded binary reading a new-format file to report every dispatch as unmapped with a warning, so that the failure is loud and obviously wrong rather than quietly plausible.
19. As a human reading `WORKFLOW.md`, I want the resolved table visible in the file, so that I can see what each tier means without running a command.
20. As a human reading `WORKFLOW.md`, I want spine-managed values marked as such, so that I can tell an inherited default from a deliberate override at a glance.
21. As a ticket author, I want tickets to keep naming only a tier, so that no artifact is ever coupled to a flavor or a model family.
22. As a dispatcher, I want to supply the flavor at dispatch time, so that the same ticket can be executed by either flavor without rewriting it.
23. As an agent resolving a model, I want a single resolver shared by the CLI and the audit, so that what gets dispatched and what gets verified cannot disagree.
24. As a shell consumer, I want the bare model id on stdout by default, so that interpolating it into a spawn command needs no parsing dependency.
25. As a shell consumer, I want effort and the full entry available behind flags, so that richer consumers are served without burdening the common case.
26. As a repo owner, I want resolution to honor the current repo's overrides, so that the same command run in two projects returns each project's own answer.
27. As a user outside any spine repo, I want resolution to fall back to embedded defaults, so that the command is still useful without a repo context.
28. As the estate's glossary owner, I want the claude-vs-codex axis called `flavor` everywhere, so that one concept does not accumulate a second name across repos.
29. As a future maintainer, I want the schema keyed by flavor generically rather than by two hardcoded names, so that a third flavor can be added without a schema change.
30. As a repo owner, I want the top-level `effort:` key retired into the table, so that no key can silently contradict the resolved effort.
31. As a ticket author, I want per-ticket `effort:` annotations to keep working unchanged, so that retiring the repo-level key does not disturb escalation records.
32. As a repo owner with a customized `effort:` value, I want the migration to carry it into per-entry overrides, so that retiring the key does not discard my setting.
33. As a future reader of the ADR trail, I want the placement change recorded as a refinement of the tiers-not-ids decision, so that it is not mistaken for a reversal.
34. As the owner of the deferred drift effort, I want the audit's verdict vocabulary left open for a drift verdict, so that this work does not foreclose distinguishing platform-caused descent from dispatcher-caused descent.

## Implementation Decisions

**D1 — Flavor is the axis name.** The claude-vs-codex distinction is `flavor`, the term already defined in deepthought's glossary for exactly this concept. It is promoted into spine's glossary. `vendor` is rejected as inaccurate (the vendors are the model providers, not the runtimes) and `harness` is rejected as already meaning the functional-test harness in this repo.

**D2 — Entry shape.** A table entry is keyed by `(flavor, tier)` and carries a model id plus an optional effort. Tiers remain the existing four; flavors are open-ended map keys, not an enum of two.

**D3 — Effort resolution.** An entry omitting effort inherits its tier's default. Resolution always yields a determinate effort; there is no "pass nothing and let the runtime decide" outcome, because that would make the effort half of the routing contract unverifiable against a config spine never read. The existing tier defaults become data in the defaults file rather than prose in a comment. `xhigh reserved for final verification and security-critical passes` degrades from a rule to advisory guidance.

**D4 — Two layers only.** Precedence is embedded defaults, then per-repo override. Defaults live in a versioned data file in this repo, embedded at build time alongside the existing template assets. **Format must be stdlib-parseable** — this repo carries zero dependencies per ADR 0001, so TOML is unavailable and the file is JSON. The TOML shown while grilling this design was illustrative of structure, not of format. A machine-local user config layer is explicitly rejected for now: the owner already rebuilds from source routinely, and an uncommitted layer would make fleet state unauditable from git. The precedence chain is designed so such a layer could be inserted later without disturbing either end.

**D5 — Shipped-default history.** The defaults file records not only the current default per `(flavor, tier)` but every default previously shipped. This history is what makes automatic refresh possible: it converts "is this value an inherited default or a deliberate override?" from a re-render-and-diff inference into a direct lookup.

**D6 — Refresh rule.** On update, an on-disk value matching any historical default for its entry is treated as inherited and refreshed to the current default. A value matching no known default is a deliberate override and is preserved untouched. Every refresh is itemized in the update plan, distinct from unrelated template churn. The irreducible ambiguity — a deliberate choice that happens to equal a prior default — is accepted, mitigated by the existing plan-then-`--write` flow that makes it visible before it lands.

**D7 — Model keys leave the choice-vs-default path.** The routing keys and the retired `effort` key must be removed from the generic choice-extraction machinery, which would otherwise classify them by the old re-render rule and fight the new resolver. This is the seam where the original trap would otherwise reappear.

**D8 — Mirror syntax: dotted flat keys.** The mirror renders one line per entry under the existing routing block, keyed `<flavor>.<tier>`, with effort appended to the value when set. This syntax was chosen for its failure mode, not its looks: an un-upgraded binary finds no recognized bare tier key, so the mapping comes back empty and the existing "no tier mapping found" warning fires. Nested flavor blocks were rejected because the current parser breaks only on a non-two-space line, so nested tier lines would parse as bare tiers with the flavor stripped and the last flavor silently winning.

```
model_routing:                      # spine-managed defaults; edit a value to override
  claude.primary:    claude-fable-5
  claude.routine:    claude-sonnet-5
  claude.mechanical: claude-haiku-4-5
  claude.fallback:   claude-opus-5
  codex.primary:     gpt-5.6-sol @ xhigh
  codex.routine:     gpt-5.6-terra
  codex.mechanical:  gpt-5.6-luna
  codex.fallback:    gpt-5.6-terra @ xhigh
```

**D9 — Value encoding.** An entry's value is the model id, optionally followed by ` @ <effort>`. Parsing splits on the separator before the existing comment-stripping applies. The separator was chosen to not collide with the comment character the current parsers already handle.

**D10 — Codex gets a uniform four-tier table.** Codex has a real fallback entry meaning "where codex-refused work re-runs, still on codex", keeping the table uniform across flavors with no special case for a flavor lacking a tier. Cross-flavor escalation — codex-refused work moving to claude or to the owner — is a separate routing concern and is not modelled as a table row. Note the motivating observation for this request (an apparent security-triggered downgrade) was diagnosed during grilling as unrelated mid-session drift; the row is justified on its own merits, since codex refusals are real, not by that observation.

**D11 — New resolver module.** A new module exposes a pure resolution function taking a repo directory, a flavor, and a tier, and returning the resolved entry plus its provenance: **default, inherited, or override** (three values — an earlier two-valued phrasing contradicted D5's whole purpose and was corrected 2026-07-24 during I033 review). A value is **inherited** when its id *and* effort together match the current default or any shipped historical pair; an entry diverging in either is an **override**, so a deliberate effort choice on a default id is never auto-refreshed away. The defaults file needs no schema-version stamp: it is embedded in the binary, so asset and reader always ship together and cannot skew. This is the single new seam. Repo context comes from the working directory as with other spine commands, overridable by flag; outside a spine repo, resolution returns embedded defaults.

**D12 — CLI is a thin printer.** `spine model <flavor> <tier>` prints the bare id and nothing else, so shell consumers need no parsing dependency. `--effort` prints the resolved effort; `--json` prints the whole entry. Flavor is a required argument and is never inferred from context — an inferred flavor is the same class of invisible resolution the estate is already chasing elsewhere. Emitting ready-made vendor CLI spawn arguments was rejected as coupling spine to other tools' flag syntax.

**D13 — Audit consumes the resolver.** The routing audit stops parsing `WORKFLOW.md` itself and calls the shared resolver, collapsing two independent parsers into one. Tier resolution becomes flavor-scoped, and alias matching moves from substring containment to explicit aliases declared per entry in the table.

**D14 — Audit gains a version gate.** The audit refuses a `WORKFLOW.md` whose stamped generation exceeds what the binary compiles, matching the gate the update path already has. Its absence is the reason an un-upgraded binary can currently emit confident verdicts from a misparse.

**D15 — Flavor of a dispatch.** Flavor is derived from the transcript source. While codex transcript parsing remains out of scope, this resolves to claude for every audited dispatch; the derivation point is named explicitly so the deferred codex-audit effort has a defined seam rather than a redesign. Where a model id is declared under more than one flavor, the transcript-derived flavor decides; the table should avoid such collisions.

**D16 — Template generation 10.** The migration renders the flavor-axis mirror, retires the top-level `effort:` key, and stamps generation 10. A repo carrying a customized `effort:` value has it migrated into per-entry effort overrides on that repo's entries rather than discarded. The retired key's prior emitted line joins the superseded-line set so it is recognized as machine-emitted rather than read as a local edit.

**D17 — Per-ticket effort annotations are untouched.** Retiring the repo-level `effort:` key does not affect per-ticket `effort:` frontmatter or the escalation-record grammar. The two are distinct and must not be conflated during migration.

**D18 — Skills call spine.** The team and handoff skills drop their hardcoded ids and resolve per dispatch through the CLI. A spine-presence check joins the existing shared preflight, refusing early with an install hint exactly as those skills already do for a missing codex binary or frontend. The claude-team lead-model paragraph — prose instructing the lead to size its own model to project difficulty and remaining credits — is replaced by resolution of the project's primary tier, since the per-repo override is that sizing decision made explicit.

**D19 — ADR trail.** A new ADR supersedes ADR 0010's placement clause only. Its headline decision stands: artifacts name tiers, never ids. The ADR must state that tickets remain flavor-neutral and that the dispatcher supplies flavor, so the change is not misread as a reversal of tier indirection.

## Testing Decisions

A good test here asserts external behavior at an existing package boundary: given a repo state on disk, what does the resolver return, what does the update plan contain, what verdicts does the audit produce. Tests must not reach into parsing internals or assert on intermediate structures — the parsers are being consolidated, and tests coupled to them would obstruct exactly the change being made.

**Resolver module.** Tested directly as a pure function: defaults with no repo, defaults with a repo carrying no override, override honored, effort inherited when omitted, effort overridden when present, unknown flavor and unknown tier rejected, historical-default recognition (a value matching a prior default reports as inherited; an unrelated value reports as override). Prior art: the existing key-extraction tests, which exercise a pure function over content with no filesystem or CLI involvement.

**Migration.** Tested through the update entry point using a captured real-repo fixture at generation 9, following the established generation-migration pattern exactly: an allowlist of sanctioned content-diff lines, asserting the diff from the fixture to current contains only those lines and nothing else. This is the pattern that has caught unintended template churn on every prior generation bump, and it is the test that will catch a migration silently rewriting an override. Additional cases: a repo with a customized `effort:` value migrates into per-entry overrides; a repo with a deliberate model override retains it; a repo carrying a prior default has it refreshed and itemized.

**Audit.** Tested through the audit entry point against scenario fixture directories, following the existing clean/degraded/mixed/vacuous convention. New scenarios: dotted-key mapping resolves correctly; an un-gated newer generation is refused; explicit aliases resolve where substring matching would previously have been relied on; a dispatch whose model maps to no entry reports unmapped rather than guessing. The existing silent-descent and escalation-record scenarios must continue to pass unchanged — this work must not alter what the audit blocks on.

**Template rendering.** Tested through the render entry point: the generation stamp is 10, the mirror renders every flavor and tier, and the retired key is absent. Prior art: the existing template tests asserting rendered content and version.

**Skills.** A grep-style shell regression test beside the existing preflight test asserts that no team skill contains a hardcoded model id outside a documented example block, and that the dispatch paths invoke the resolver. This is deliberately a weak test of behavior and a strong guard against one specific regression — a pinned literal creeping back into a worker-model variable, which has already happened once in the field. Prior art: the existing frontend-preflight shell test is the only precedent for testing skill behavior in that repo.

**Fleet verification.** After the sweep, a routing audit on a repo with real transcripts confirms resolution end to end against real-format data rather than synthetic fixtures.

## Out of Scope

- **Mid-session model drift.** A team lead observed running at the primary-tier model and later found on a mechanical-tier model, with no corresponding event in its own context, is a separate defect with a separate PRD. The leading hypothesis is platform-side quota degradation, unverified at time of writing. This spec deliberately does not model it: a table declares intent, and drift overrides intent at runtime. The audit's verdict vocabulary is left open so a drift verdict — distinct from dispatcher-caused silent descent, with a different remedy and different gate behavior — can be added without rework.
- **Codex transcript parsing.** Making codex-executed tickets actually auditable end to end requires parsing a transcript format of unknown shape. This spec completes the mapping side only, and names the flavor-derivation seam that work picks up.
- **Cross-flavor fallback routing.** Codex-refused work escalating out of codex entirely is a routing rule, not a table row, and is not designed here.
- **Local models as a third flavor.** Deferred at the owner's direction pending investigation of local infrastructure. The schema is keyed generically so adding one later is data, not a schema change.
- **A machine-local user config layer.** Rejected for now under D4; the precedence chain accommodates it later.
- **Changing what the audit blocks on.** Reasoned escalations stay advisory, silent descent stays blocking. This spec changes how models resolve, not what constitutes a violation.
- **Upstream plugin changes.** Per ADR 0010's still-standing placement principle, none of this contract is patched into the upstream plugin cache.

## Further Notes

The problem statement's central claim was verified empirically during grilling rather than inferred: a probe against the current template returned a changed fallback value as a deliberate choice, confirming that a template value change propagates nothing. Any implementation that leaves the routing keys inside the generic choice-extraction path will reproduce this exactly, which is why D7 is called out as its own decision rather than folded into the resolver work.

The mirror-syntax decision (D8) is the one place where a cosmetic-looking choice has a correctness consequence. Both candidate syntaxes require the same parser work; they differ only in how an un-upgraded binary fails. One degrades to a loud, obviously-broken warning; the other to confident wrong verdicts. Reviewers should treat any later proposal to "clean up" the dotted keys into nested blocks as a correctness regression unless the version gate is proven to cover every read path.

The sequencing matters for the sweep: skills gain a hard dependency on the resolver (D18) while repos gain the new mirror (D16). Repos not yet migrated still resolve correctly, since resolution falls back to embedded defaults when a repo carries no override — so skill updates and the fleet sweep do not have to land atomically.

The Opus 5 refresh that prompted this work is deliberately not given special treatment anywhere in this spec. It is one value in the defaults file, carried by the same mechanism as every future release. If implementing it requires a special case, the mechanism is wrong.
