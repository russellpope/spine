# Deployment on Kubernetes, not as a Kubernetes operator

- **Status:** Accepted
- **Date:** 2026-05-17
- **Deciders:** Russell Pope

**Technical Story:** Architecture-of-record §2 (Non-goals), §3 D2, §5 (High-Level Architecture), §23 (Phase 8 mentions the Helm chart).

## Context and Problem Statement

Praxis is built in Go and manages infrastructure that overlaps in spirit with what Kubernetes operators manage. The question: do we implement Praxis as a Kubernetes operator (CRDs + controllers in a target cluster), or as a standalone platform that happens to run on Kubernetes as its preferred deployment target?

## Decision Drivers

- We need our own API, our own auth model, our own audit pipeline, and finer-grained RBAC (per-field, per-operation) than the K8s API affords.
- We need our own datastores — Postgres for state, NATS for events, Temporal for workflows. K8s etcd is unsuitable as primary store for our expected scale.
- Kubernetes is a fine *execution substrate* — pods, services, secrets, ingress — but a constraining *API model*.
- Operators expect a single API surface that covers VMs, networks, storage, and third-party tools — not a sprawling CRD-per-kind defined inside the target cluster.
- We want the option to deploy outside Kubernetes (bare metal, Nomad, docker-compose for dev) without architectural surgery.

## Considered Options

- (A) Kubernetes operator — CRDs for VirtualMachine, NetworkPolicy, etc.; reconcilers running inside the cluster.
- (B) Standalone platform deployed on Kubernetes — own API server, own datastores, K8s only runs the deployments.
- (C) Hybrid — K8s operator for K8s-native concepts; separate API for non-K8s concepts.

## Decision Outcome

**Chosen option:** (B). Praxis runs as a Helm-deployed set of services on a K8s cluster (or anywhere else — Kubernetes is not a hard requirement; it is the v1 recommended deployment target). It exposes a REST API at its own host/port, owns its own Postgres + NATS + Temporal, and treats Kubernetes as an execution substrate rather than an API to subscribe to.

### Consequences

- **Good:** API model unconstrained by K8s API machinery (no per-CRD limits, no version-conversion plumbing, no etcd-derived scaling limits).
- **Good:** Operators interact via `praxis` CLI, not `kubectl`. No conflation of platform identity.
- **Good:** We retain the option to deploy outside Kubernetes — docker-compose for dev, bare metal for embedded scenarios.
- **Bad:** We rebuild some patterns Kubernetes ships for free — watch, informers, leader election. Accepted: the patterns have well-known shapes; reimplementing inside our model is a finite cost.
- **Bad:** Helm chart maintenance and Kubernetes upgrade testing become our responsibility from Phase 8 onward.

### Confirmation

- The platform must remain deployable in a docker-compose-only environment (validated in `deploy/compose/` as it matures).
- No production code imports `k8s.io/api` or `k8s.io/client-go` as a runtime dependency. Lint rule deferred; principle enforced by review. *(Revised — DC-M02: `internal/pluginhost/kubernetes.go` + `select.go` import k8s.io/api + client-go as direct runtime deps since the ADR-0066 Kubernetes Jobs plugin host. That usage is execution substrate — launching plugin Pods/Jobs when the operator selects `PRAXIS_PLUGIN_HOST=kubernetes` — not the operator/CRD/reconciler pattern this ADR rejects, so the decision's spirit holds: Praxis still deploys as a standalone platform, owns its own resource model, and runs fine in docker-compose. The criterion is narrowed to: no production code may use client-go to implement Praxis's own control loop or persist Praxis state in the kube API; execution-substrate use behind the pluggable host selector is permitted.)*
- Phase 8 produces and tests the Helm chart.

## Pros and Cons of the Options

### (A) Kubernetes operator

- **Good:** Free watch / informer / leader-election plumbing.
- **Good:** `kubectl` becomes a usable client.
- **Bad:** Stuck in K8s API model — no per-field RBAC, no per-operation RBAC of our shape.
- **Bad:** Per-tenant clusters become the only real isolation boundary.
- **Bad:** Audit pipeline is K8s-API-server-shaped, not what we need.

### (B) Standalone (chosen)

- Covered above.

### (C) Hybrid

- **Good:** Best-of-both in theory.
- **Bad:** Two API surfaces, two auth models, two RBAC models. Operators have to learn both. Conceptual drift inevitable.

## More Information

- Architecture doc §3 D2, §5, §23 Phase 8.
- Related ADRs: 0007 (own resource model), 0009 (Postgres state), 0014 (mTLS in-cluster).
