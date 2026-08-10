---
title: "i063 estate remap shipped"
created: 2026-08-10
handoff_ordinal: 3
---

# Handoff — i063 estate remap shipped (2026-08-10)

## Context

I063 is shipped locally. The estate Claude routine default is now
`claude-opus-5 @ low`; `claude-sonnet-5` remains only as the exact historical
medium-effort pair. Current aliases are Opus-only. Existing inherited Sonnet
rows refresh through the normal itemized model-table sweep, while unrelated
ids and mismatched effort pairs remain overrides.

The accepted design and plan are
`docs/specs/2026-08-10-estate-default-claude-routine-remap-design.md` and its
paired plan. No template-format mechanism changed: `templates/VERSION` stays
10, no ADR was added, and the generation decision is recorded in the fixed
I063 Resolution.

## State (verify before relying)

- Main contains the up-front specs (`01a85bb`, `ce86a11`), implementation
  (`b71a20b`), primary-review correction (`c84c5bd`), and fixed ticket
  Resolution (`ff0c0f2`). This handoff is the only subsequent shipped artifact.
- The initial primary blind review failed only AC5: spine's own correct value
  still had override-era spacing and normalized on the first update. The
  correction applied the generated alignment and added a root-WORKFLOW
  two-write byte-stability regression. Primary re-review passed with no
  findings.
- Fresh primary verification passed with no findings: uncached `go test ./...`,
  focused race tests, `go vet ./...`, a clean offline build, real compiled-CLI
  acceptance probes, generation-boundary checks, formatting/diff integrity,
  and routing audit all passed.
- Real CLI probes covered new init, inherited refresh/itemization, unrelated
  override preservation, legacy customized `xhigh` and `medium`, exact-only
  historical recognition, first/second update stability, and refusal to
  downgrade a generation-11 file.
- The pre-handoff doctor/stage result contained only ADR 0014's expected stale
  predecessor snapshot. With the generated complete snapshot below, the
  current-tree CLI reports doctor healthy and both stage/routing audits exit
  zero; I063 is `escalated-with-reason`, with no silent descent.
- The installed pre-I063 binary was deliberately not replaced and reports its
  generic D2 behind-version warning against this new table state. Final gates
  above used the current-tree CLI; `make install` is the owner's deployment
  action.
- The pre-existing untracked `.DS_Store` and
  `docs/research/2026-08-05-routing-yield-feasibility.md` were left untouched.

## Next steps

1. Owner: review and push the local commits when ready.
2. Owner: install the updated binary (`make install`) and run the ordinary
   estate `spine update` sweep when ready; neither action was performed here.
3. No additional code work is open for I063 unless the post-install estate
   sweep exposes a repository-specific override decision.

## Gotchas

- Do not sweep-edit historical tickets, old handoffs, or generation migration
  captures that mention Sonnet 5; those are records, not live defaults.
- A bare historical `claude-sonnet-5` means the shipped medium-effort pair and
  refreshes. `claude-sonnet-5 @ low` and shorthand `sonnet` are overrides, not
  inherited history.
- Routine and fallback now share the Opus 5 id/alias. Routing audit resolves
  ambiguity toward the ticket's declared tier; fallback/descent fixtures use
  exact historical ids to retain their original guard semantics.
- Do not hand-edit the embedded cursor block. Use the `spine cursor` verbs and
  regenerate a handoff if stage state ever changes.

<!-- spine:cursor -->
effort: i063-estate-remap
prd: docs/specs/2026-08-10-estate-default-claude-routine-remap-design.md
tickets: I063
stages: grill[x] prd[x] issues[x] implement[x] functional-test[x] review[x] verify[x] ship[x] deploy[x] docs[x] handoff[x]
<!-- /spine:cursor -->
