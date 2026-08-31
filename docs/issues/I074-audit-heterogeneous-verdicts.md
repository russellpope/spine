---
id: I074
title: Audit routing verdicts for heterogeneous dispatches
severity: med
status: fixed
commits: [5836ef6, 86d4c55, be2e3e9, 9914e8e, b665095, 430e54b, 8e482a5]
affects: [audit]
blocked-by: [I070]
labels: [wayfinder:task]
parent: I066
assignee:
---

## Question

Extend `spine audit routing` for declare-then-confirm
([I069](I069-attribution-declare-then-confirm.md)): the dispatch declares
(harness, model, effort); the audit confirms against the transcript. What is
the verdict vocabulary — confirmed; declared-vs-observed mismatch (the new
silent-descent analogue for proxied pools); unconfirmable (transcript carries
no usable id for a known combo — distinct from silent-descent and from
unmapped-dispatch)? Does effort get confirmed too, or declared-only? Verdicts
must respect host-pinned equivalences ([I068](I068-host-scoped-availability-and-tier-pins.md)):
a routine ticket running the host's pinned gpt-5.6-sol is conformant, not an
escalation. Blocked on [I070](I070-proxied-model-ids-in-claude-transcripts.md)
for what observed ids exist to confirm against.

## Resolution

Fixed 2026-08-30 at final I074 product SHA `8e482a5`. Heterogeneous judgment
consumes only complete I075 declarations, validated I072 host-local observed
IDs, the final pinned route, and a completely correlated linked worker event.
Exact different-route or unauthorized-effort proof blocks; missing, unmapped,
unavailable, unreachable, or effort-unobserved evidence remains explicit and
nonblocking `unconfirmable`. Same-identity legacy evidence is not double judged,
while independent legacy silent descent still blocks. No current transcript
format proves observed effort, so production retains `observed-effort=-` and
adds no extractor. A fresh routine review and a different independent routine
verifier passed the exhaustive matrix, hostile host/identity/output probes,
full/race/vet/build, Windows compile, and template-migration gates. I073 may now
perform the public naming migration. The batch-final exact-SHA maipipe lane
remains the ship gate.
