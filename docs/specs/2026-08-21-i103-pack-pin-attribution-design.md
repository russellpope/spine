# I103 pack pin attribution

Status: approved (grilled 2026-08-21)
Date: 2026-08-21
Tickets: I103
Decision record: ADR 0019; respects ADRs 0015, 0017, 0018

## Problem Statement

A repo owner pins `gate_pack: go@1` so that a pack release never silently
changes their gate. Since I098 the pin freezes the check-class list, and the
owner can read exactly which stages their region runs. But the code on every
finding those stages emit still comes from whatever pack version the spine
binary on the machine happens to be. Today that is invisible, because one pack
exists. The day go@2 ships, a go@1 repo's remediation records, do-not-regress
blocks and telemetry start filling with `go@2/<check>` strings produced by go@1
stages, and the history the owner keys on is mislabelled.

## Solution

The **pack pin** freezes both halves of what `<pack>@<v>` names. The rendered
region's stages carry the pin on their run line (`spine gate go@1 tskip`), and
`spine gate` honours a versioned pack argument: findings are coded with the
pin, a check outside the pin's class list is refused, and a pack the binary
does not ship is refused rather than approximated. A hand run with a bare
pack name still attributes as the binary's own pack. The one-time region
rewrite every adopter takes is named in the update plan as changed stages, so
the re-approval cost is visible even though no stage is added or removed.

## User Stories

1. As a repo owner pinned at `go@1`, I want findings from my region's stages
   coded `go@1/<check>` whatever spine binary runs them, so that my
   remediation history and do-not-regress blocks stay keyed to the pack I
   chose.
2. As a repo owner, I want the pin to reach each stage as part of the stage's
   own definition, so that the plan diff alone tells me what changed when I
   move the pin.
3. As a repo owner reading `spine update`'s plan, I want to be told "N
   stage(s) changed: …" when the region's bytes change without a stage being
   added or removed, so that I know I must re-run `maipipe gate
   approve-definition` before the plan is applied.
4. As a repo owner moving from `go@1` to `go@2`, I want the plan to show both
   the class-list delta and the run-line change, so that the move is a
   deliberate, reviewable act.
5. As a repo owner whose region still carries the pre-pin run lines, I want
   `spine update` and `spine doctor` to treat my region as stale (D2), not as
   hand-edited (D4), so that migrating never requires `--force`.
6. As a repo owner, I want a region line naming a pin other than my
   `WORKFLOW.md`'s to be flagged as unrecognized region content, so that a
   drifted or hand-patched region cannot silently run a different pack.
7. As a developer running `spine gate go tskip` by hand, I want the finding
   coded with the binary's own pack, so that the command stays stateless and
   never depends on the cwd's `WORKFLOW.md`.
8. As a developer running `spine gate go@1 n-plus-one` where `n-plus-one` is
   not a go@1 class, I want a refusal (exit 2) naming the pin and the check,
   so that a mis-rendered or hand-edited stage fails loudly.
9. As a maipipe operator whose old spine binary does not ship the pack a
   region names, I want the stage to fail as misconfiguration (exit 2) with
   no findings, so that a wrongly-attributed finding never enters the
   results stream.
10. As a maipipe operator, I want the `mutation-go` stage pinned exactly like
    the check-class stages, so that mutation findings and check findings
    share one attribution rule.
11. As a fleet maintainer, I want `<pack>@<v>` to be the single form of the
    pin everywhere it appears (WORKFLOW.md, run line, finding code), so that
    nothing has to translate between forms.
12. As a reader of `CONTEXT.md`, I want "pack pin" defined, so that "the pin"
    in tickets and specs is unambiguous.
13. As a reader of spec story 23 and I098's Resolution, I want them to say
    the pin covers both the class list and the attribution string, so that
    the record matches the code.
14. As a future spine maintainer, I want the run-line-not-env decision
    recorded with its trade-off, so that nobody re-litigates the carrier
    without knowing the second rewrite it costs (ADR 0019).

## Implementation Decisions

- **Carrier.** The pin rides the run line: every stage in the managed region,
  including the mutation stage, renders `spine gate <pack>@<v> <check>` where
  `<pack>@<v>` is the repo's `gate_pack` value verbatim. No env var carries
  the pin; the `SPINE_GATE_*` namespace remains per-check configuration.
  (ADR 0019)
- **Gate command contract.** `spine gate <pack>[@<v>] <check>`. With a
  versioned argument the pin is authoritative for that run: the finding code
  is `<pin>/<check>`; a check not in the pin's frozen class list is refused;
  a pin the binary does not ship is refused. Both refusals are
  misconfiguration: exit 2, a stderr message naming the pin and the check,
  no findings document. With a bare pack name the run attributes as the
  binary's own pack, exactly as today. `spine gate` never reads
  `WORKFLOW.md`.
- **Pack resolution lives in the gate package.** The gate package exposes a
  way to resolve a pin to the attribution id and frozen class list, and the
  code helper takes the resolved pin rather than reading the binary's
  constant. The existing per-version class table remains the single source
  for what the binary ships.
- **Region reader.** Recognises both run-line forms as spine's own content:
  the bare form (any binary-shipped class) and the pinned form only when the
  pin equals the repo's current `gate_pack`. A pinned line naming any other
  pin is unrecognized region content (D4). A region in the bare form is
  therefore stale (D2), never unrecognized.
- **Changed-stages notice.** The plan's existing added/removed stage delta
  gains a third list: stages present in both the existing and rendered
  region whose lines differ in any byte. Plan output prints "maipipe.toml:
  this render changes N stage(s) not added or removed: …" alongside the
  existing notice, naming the `maipipe gate approve-definition` cost. Plan
  output only; it composes with the all-or-nothing write refusal. Computed
  only where a region already exists.
- **Doctor.** No new check. D2 reports the un-migrated region as stale via
  the update dry-run, and D10 is unchanged.
- **Records.** Spec story 23 and I098's Resolution gain a dated note that the
  pin now covers the attribution string; I103 is closed with its Resolution.
  `CONTEXT.md` gains **pack pin** (done during the grill).

## Testing Decisions

A good test drives a public entry point with real inputs and asserts on
observable output: the rendered bytes, the plan report, the finding code, the
exit code. No test asserts on how the pin is threaded internally.

- `internal/update`, through `update.Run` on real temporary repos, using the
  existing `packClassesFor` seam to stub a later pack (prior art:
  `gatepin_test.go`): every stage including mutate renders the pinned run
  line; a bare-form region is stale not unrecognized; a foreign-pin line is
  unrecognized; the changed-stages notice fires when only bytes differ and
  stays quiet when stages are added/removed only.
- `internal/gate`, through the package's public resolve/code surface, adding
  a stub later version to the existing per-version class table: a `go@1` pin
  on a "version 2" binary codes `go@1/<check>`; a class outside the pin is
  refused; an unshipped pin is refused; no pin attributes as the binary's
  own pack.
- `cmd/spine`, through the existing run-with-args helpers (prior art:
  `main_test.go` gate tests), real pack only: `spine gate go@1 tskip` → code
  `go@1/tskip`; `spine gate go@9 tskip` → exit 2 and no findings document;
  `spine gate go tskip` unchanged.
- Negative controls, one per behaviour: removing the pinned-form
  recognition must make the bare→pinned migration unrecognized; removing the
  pin-equality check must let a `go@2` line pass in a `go@1` repo; removing
  the changed-stages delta must silence the notice; removing the pin's
  class-list check must let an out-of-pin class run.
- Verification: gofmt, `go vet`, `SPINE_REQUIRE_MAIPIPE=1 make test`, and
  `maipipe run full --wait` at the final commit — spine's own region is an
  adopter and takes the rewrite, so the lane proves the pinned run line
  works under real maipipe.

## Out of Scope

- Shipping go@2 or any second pack; the per-version table gains a stub only
  in tests.
- Any env-var carrier for the pin, or reading `WORKFLOW.md` from `spine gate`.
- Changing how the pin freezes the class list (I098) or the opt-out path
  (I097).
- Migrating other adopters' regions; each takes the rewrite through its own
  `spine update --write` and re-approval.
- A new doctor check.

## Further Notes

- Exit-code vocabulary is the existing one: 1 = findings, 2 =
  misconfiguration.
- spine's own `maipipe.toml` changes in this work (it is pinned at `go@1`);
  commit it with the code so the maipipe lane runs against the new region.
- Next free ticket id at the time of writing: I107.
