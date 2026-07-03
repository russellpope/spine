# Declarative-primary paradigm with imperative operations as first-class

- **Status:** Accepted
- **Date:** 2026-05-17
- **Deciders:** Russell Pope

**Technical Story:** Architecture-of-record §1 (Mission), §3 D1, §4 (Principles 1, 10), §6 (Operation, Composite), §7 (Lifecycle), §23 (Phased plan).

## Context and Problem Statement

Praxis manages heterogeneous infrastructure — VMs, networks, storage, third-party tools. Two operational paradigms dominate this space: declarative (Kubernetes, Terraform, Crossplane — `spec → reconciler → status`) and imperative (Ansible, AWX, runbook engines — invoke a procedure, observe result). Real infrastructure work mixes both: a VM's *configuration* (CPU, memory, disk) is naturally declarative; `reboot` is not — you do not declare "I want the machine rebooted" as state. The model must admit both without forcing one to masquerade as the other.

## Decision Drivers

- Reality of operator workflows — most teams already think declaratively for config and imperatively for actions.
- Auditability and convergence — drift detection requires a desired-state contract.
- Agent operability (see ADR-0021) — agents need to plan over both state and actions.
- Avoid the "everything-is-a-CRD" awkwardness where reboots become a sub-resource you `create` to trigger a side effect.
- GitOps fit — declarative resources are what git tracks.

## Considered Options

- (A) Pure declarative (K8s/Crossplane-style) — every action expressed as a desired-state change; controllers reconcile.
- (B) Pure imperative (Ansible/AWX-style) — every action a named procedure; no first-class desired state.
- (C) Declarative-primary with imperative operations as first-class on resource types.
- (D) Two parallel APIs — one for state, one for jobs (Terraform Cloud + Run-style).

## Decision Outcome

**Chosen option:** (C). Resources have `spec` (declarative) and `status`; TypeDefinitions also declare `operations` (imperative, named, permission-scoped, Temporal-backed). The API serves both. Reconcilers drive state convergence; operations are explicit user/agent invocations whose progress is surfaced into `status.operations[]`.

### Consequences

- **Good:** Operators express config naturally (declarative) and actions naturally (imperative) without contortions.
- **Good:** Operations are typed, permissioned, audited, and durable (Temporal workflows) instead of fire-and-forget shell.
- **Good:** Agents can plan over both surfaces — read state via spec/status, take action via operations.
- **Bad:** Two execution paths inside the platform (reconciler + workflow engine) — more moving parts than a single paradigm.
- **Bad:** API surface is larger (resources + operations) than a pure CRD or pure RPC model.
- **Neutral:** Composites can declare composite-level operations (e.g., `WebStack.rollingReboot`) that orchestrate child operations.

### Confirmation

- Every TypeDefinition that ships with a long-running or side-effectful action MUST express it as an operation, not as a mutation to `spec` that exists only to trigger a side effect.
- Code review checklist item: "If this would mutate spec just to trigger a side effect, is it actually an operation?"
- ADR-0020 (testability) requires both reconciler paths and operation paths to be exercised by the test harness.

## Pros and Cons of the Options

### (A) Pure declarative

- **Good:** Simpler mental model; one execution path.
- **Bad:** Forces ergonomic compromise for inherently imperative actions; "reboot" becoming a state field reads as a workaround.
- **Bad:** Audit becomes "what changed" rather than "what was done."

### (B) Pure imperative

- **Good:** Matches Ansible mental model that many operators already hold.
- **Bad:** No drift detection, no convergence, no GitOps fit, no idempotent re-apply.

### (C) Hybrid (chosen)

- Combines the strengths of both at the cost of one extra subsystem.

### (D) Two parallel APIs

- **Good:** Clean conceptual separation.
- **Bad:** Doubles authentication, RBAC, audit, observability surfaces; operators have to learn two clients.

## More Information

- Architecture doc §1, §3 D1, §6 (Operation), §9 (Workflow Engine), §23 (Phase 4 brings operations live).
- Related ADRs: 0007 (resource model), 0004 (Temporal), 0021 (autonomous operation).
