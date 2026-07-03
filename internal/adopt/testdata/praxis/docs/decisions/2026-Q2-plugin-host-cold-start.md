# Plugin host cold-start operator survey — Q2 2026

> Companion data for [ADR-0071](../adr/0071-long-lived-per-plugin-deployments.md).
> The survey ran from `2026-04-15` through `2026-05-15` against
> operators self-identifying as running `PRAXIS_PLUGIN_HOST=kubernetes`
> in production. The numbers below are the anonymized aggregate; the
> verbatim per-operator responses live in the operator-feedback
> Slack channel referenced by the ADR-0071 Open Questions block.

## Respondent profile

- **N = 11** operators identified as running the Kubernetes plugin
  host (ADR-0066) in production at the time of the survey.
- **9 of 11** ran at least one plugin with an Apply-heavy workload
  (>1 Apply/min sustained over 24 hours during the survey window).
- **2 of 11** ran very-high-frequency Apply workloads (>3 Apply/sec
  sustained) — both were NetBox-style ingest plugins.

## Latency profile

End-to-end Apply latency (Job submitted → Pod Ready → Apply RPC
completes → Job cleanup completes):

| Percentile | Latency |
|---|---|
| median | 1.8s |
| p95 | 4.2s |
| p99 | 7.1s |

Image-pull on cold node caches dominated the long tail (>5s).
Operators on warm node caches (image pre-pulled by a node-local
sidecar) saw a median Apply latency closer to **0.6s**.

## Apply rate per plugin per deployment

| Percentile | Apply/min |
|---|---|
| median | 6 |
| p95 | 14 |
| p99 | 41 |

The p99 rate (41 Apply/min, or roughly one every 1.5 seconds) is
the threshold where per-invocation Job overhead becomes visible to
the operator's downstream consumer. Two operators reported sustained
rates above this threshold.

## Operator-rated impact of cold-start

Question: "On a scale of 0 (blocking — you cannot use the Kubernetes
host because of cold-start) to 10 (invisible — you do not notice
cold-start), how does cold-start affect your deployment?"

| Percentile | Rating |
|---|---|
| median | 8 |
| p95 | 7 |
| p99 | 6 |
| min | 3 |
| max | 10 |

The two operators rating ≤4 (cold-start as a concrete blocker) were
both the very-high-frequency Apply respondents.

## What the two blocked operators did

- **Operator A** (NetBox ingest, ~200 Apply/min sustained): switched
  to `PRAXIS_PLUGIN_HOST=subprocess` on a dedicated worker Pod (one
  Pod per operator-managed worker, many plugin subprocesses inside).
  Reported the latency profile they wanted (~50ms p95 per Apply).
- **Operator B** (derivation hook calling an internal CIDR
  allocator): the workload is in scope for the Phase-14 WASM host
  with the new capability policy (ADR-0069) — one allow-listed
  network endpoint, pure-compute everything else. Operator B's
  follow-up reported the WASM host meets their latency requirements
  (~5ms p95 per Apply, no Pod scheduling overhead).

## The remaining 9 operators

Rated cold-start between 6 and 10 on the invisibility scale. The
median 1.8s Apply latency is "fine" for their workloads (typical
Apply rate: 6/min). Across this group:

- **0 operators** named cold-start as a top-3 platform pain point in
  the Phase-14 free-text question.
- **3 operators** named the lack of per-plugin RBAC as their top-3
  pain point (resolved by ADR-0068 / Stage 15.1 in Phase 14).
- **2 operators** named the lack of WASM snapshot streaming as their
  top-3 pain point (resolved by ADR-0070 / Stage 15.3).

## Decision

ADR-0071 closes the long-lived per-plugin Deployments deferral as a
**non-goal** based on this data. The two affected operators have
working alternatives (subprocess host on dedicated worker; WASM
host with the Phase-14 capability + snapshot features). The
remaining 9 operators do not name cold-start as a pain point worth
the operational complexity of a new host kind.

The decision is reversible — a future Phase-N ADR can re-open it
when operator-survey data shows ≥25% of Kubernetes-host operators
rating cold-start as a concrete blocker (≤4 on the invisibility
scale).

## Methodology notes

- The survey was distributed via the operator-feedback Slack channel
  + the GitHub Discussions thread tagged `operator-survey`.
- Response rate: 11 of 17 known Kubernetes-host operators (65%).
- Self-identification was used for the production-deployment
  qualifier; no telemetry was collected from operator clusters.
- Latency percentiles were operator-supplied (operators reported
  their dashboard values); the platform team did not aggregate raw
  metric data.
- The 41 Apply/min p99 is a single-operator outlier (Operator A's
  NetBox ingest); the next-highest reported rate was 22 Apply/min.
