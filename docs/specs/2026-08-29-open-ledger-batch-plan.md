# Open-ledger batch implementation plan

> **For workers:** follow the live `open-ledger-batch` cursor and the detailed
> ticket PRDs. This plan coordinates work. It does not authorize an unresolved
> design.

**Goal:** Close the 17 in-scope ledger tickets through their individual gates,
without picking up I112 or conflating host, fleet, and owner decisions.

**Architecture:** Independent ticket streams run in severity order. I072 is
the only dependency root, and audit/model changes are serialized to keep one
reviewable truth for routing behavior.

**Spec:** [open-ledger batch design](2026-08-29-open-ledger-batch-design.md)

## Current checkpoint

- [x] I111 fixed and closed: `0723251`, `a7ee899`.
- [x] I032 fixed and closed: `2e75d5e`, `910e421`, `1d7786b`.
- [x] I102 fixed and closed: `35808b3`, `78ceeb1`.
- [x] I108 fixed and closed: `3eae6e8`, `72749d9`.
- [x] I050 design and plan committed: `5d2825e`.
- [x] I072 design and plan committed: `b963eb9`.
- [x] I105 research committed: `c06a896`; owner choice still pending.
- [ ] I050: implement the committed
  [design](2026-08-29-approved-untested-acceptance-design.md).
- [ ] I051: PRD/plan work is active but not committed at HEAD `5d2825e`; do
  not start code until its approved artifacts land.
- [ ] I072: implement the approved [design](2026-08-29-host-routing-config-design.md), then unblock I073 and I077.
- [ ] I073, I074, I075, I078, I066, I076, I077, I007, and I093: open.

## Work sequence

### 1. Keep the batch boundary intact

- [ ] Dispatch only the 17 cursor tickets. Leave I112 parked until its owner
  decides the OpenWeights axis.
- [ ] Record owner calls rather than inferring them: I075, I077, I093 items
  3 through 5, I105, and any cross-repo or fleet action.
- [ ] Stage explicit paths. Do not stage `.cache/`, the research stray, or
  concurrent work.

### 2. Run the independent ticket streams

- [ ] Complete I051 and I050 only after each committed ticket PRD is approved.
- [ ] Implement I072 from its committed feature plan. Do not add I074
  heterogeneous verdicts or I073 public renaming to its diff.
- [ ] After verified I072, implement I073 and I077 from their own PRDs.
- [ ] Serialize I111/I074/I078 audit work. Since I111 is closed, I074 and
  I078 must each rebase on the current audit result and receive a focused
  review before the next audit change begins.
- [ ] Check I007 and I075 for shared model/dispatch files before either starts.
- [ ] Keep I066 and I105 documentary. Their completion requires an accurate
  decision map or research recommendation, not product implementation.

### 3. Apply the per-ticket gate

- [ ] Grill before a new feature PRD. Link the PRD to its issue and this batch
  PRD without copying detailed design into this plan.
- [ ] For each code ticket, add a focused failing test, observe it fail,
  implement, run focused tests and `go test ./...`, then retain the evidence.
- [ ] For I066 and I105, validate sources and links, review scope and wording,
  and verify that unresolved owner calls stay unresolved.
- [ ] Run task review at the ticket's routed tier. A risk trigger or final
  whole-branch review uses primary tier.
- [ ] Fresh-context verification attacks the ticket PRD first, then checks the
  diff, test evidence, ticket closure, and routing records.

### 4. Integrate and ship

- [ ] Resolve every audit/model collision before starting a parallel change to
  the same files. Re-run the affected focused tests after each integration.
- [ ] Run the final spec review against the batch design and all detailed
  feature PRDs. Record any amendment before acceptance.
- [ ] Run `spine audit routing` with the batch transcripts, then run
  `maipipe run full --wait` at the final candidate SHA.
- [ ] Push only that tested SHA. Record it, the lane result, every ticket's
  closure SHA, and every remaining owner blocker in the final handoff.
- [ ] Deploy only after ship: `make install`, copy `~/bin/spine` to
  `~/.local/bin/spine`, and verify both binaries against the shipped SHA.
