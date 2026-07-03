# Architecture Decision Records

This directory holds the **Architecture Decision Records (ADRs)** for Praxis. ADRs capture the reasoning behind decisions that shape the system, so future contributors understand not only *what* was decided but *why* — including what was considered and rejected.

## Format

We use the [MADR](https://adr.github.io/madr/) template in two variants:

- **Long-form** for architecturally load-bearing decisions where future contributors need full context: drivers, considered options, consequences, and how compliance will be confirmed. Template at [`0000-template.md`](0000-template.md).
- **Short-form** for routine choices where the decision is clear and the alternatives are obvious. Same header and outcome sections; pros/cons-per-option section omitted.

## Practice

- One ADR per substantive decision. New decisions get a new ADR; revisions are *new* ADRs that supersede prior ones (mark the prior ADR's status as `superseded by [ADR-NNNN](...)`).
- ADRs are immutable once accepted, except for status updates. Wrong decision? Write a new ADR that supersedes the old one.
- File names: `NNNN-kebab-case-title.md`, sequential, never reused, never reordered.
- Commit messages: `docs(adr): Dn <short description>` (where Dn references the architecture doc Decisions Log row, when applicable) or `docs(adr): <slug>` for new decisions outside the original log.
- Update the index in this file when adding a new ADR.

## Status legend

- **Proposed:** Under discussion; not yet in force.
- **Accepted:** In force. Code, tests, and documentation should respect it.
- **Rejected:** Considered and turned down. Useful for "we already discussed that."
- **Deprecated:** No longer relevant; not yet replaced.
- **Superseded:** Replaced by a newer ADR; the status line links to the successor.

## Index

This index mirrors the Decisions Log in [`../design/praxis-architecture.md`](../design/praxis-architecture.md) §3.

| # | Decision | Form | Status |
|---|---|---|---|
| [0001](0001-paradigm-declarative-with-imperative-operations.md) | Declarative-primary paradigm with imperative operations as first-class | long | Accepted |
| [0002](0002-deployment-on-kubernetes-not-as-operator.md) | Deployment on Kubernetes, not as a Kubernetes operator | long | Accepted |
| [0003](0003-event-bus-nats-jetstream.md) | Event bus: NATS JetStream | short | Accepted |
| [0004](0004-workflow-engine-temporal.md) | Workflow engine: Temporal | short | Accepted |
| [0005](0005-provider-plugins-via-hashicorp-go-plugin.md) | Provider plugins via HashiCorp go-plugin | long | Accepted |
| [0006](0006-wasm-wazero-for-v2-extensions.md) | WASM (Wazero) reserved for v2 in-process extensions | short | Accepted |
| [0007](0007-resource-model-k8s-style-typedefinition-and-controllers.md) | Resource model: K8s-style TypeDefinition + instance + controller | long | Accepted |
| [0008](0008-composition-as-first-class-resource.md) | Composition as a first-class resource | long | Accepted |
| [0009](0009-single-postgres-for-state-audit-and-temporal.md) | Single Postgres for state, audit, and Temporal | long | Accepted |
| [0010](0010-api-protocols-rest-openapi-primary.md) | API protocols: REST + OpenAPI primary; Connect-RPC deferred | short | Accepted |
| [0011](0011-cli-go-cobra-viper.md) | CLI built on Go + Cobra + Viper | short | Accepted |
| [0012](0012-config-formats-json-yaml-toml.md) | Config formats: JSON, YAML, TOML all accepted | short | Accepted |
| [0013](0013-user-auth-oidc.md) | User authentication via OIDC | short | Accepted |
| [0014](0014-service-to-service-auth-mtls.md) | Service-to-service authentication via mTLS | short | Accepted |
| [0015](0015-rbac-with-opa-for-advanced-policy.md) | Built-in RBAC with OPA for advanced policy | short | Accepted |
| [0016](0016-hierarchy-tenant-project-environment-resource.md) | Hierarchy: Tenant → Project → Environment → Resource | short | Accepted |
| [0017](0017-source-of-truth-layered-ownership-model.md) | Source-of-truth: layered field-ownership model | long | Accepted |
| [0018](0018-field-derivation-first-class.md) | Field derivation is a first-class TypeDefinition feature | short | Accepted |
| [0019](0019-audit-hash-chained.md) | Audit: every mutation, hash-chained, append-only | short | Accepted |
| [0020](0020-testability-as-load-bearing-architectural-concern.md) | Testability as a load-bearing architectural concern | long | Accepted |
| [0021](0021-architected-for-autonomous-agent-operation.md) | Architected for autonomous agent operation | long | Accepted |
| [0022](0022-patch-content-type-negotiation.md) | PATCH via Content-Type negotiation (RFC 6902 + RFC 7396); defer SMP and SSA | long | Accepted |
| [0023](0023-resource-version-opaque-base64-token.md) | ResourceVersion is an opaque base64url token | short | Accepted |
| [0024](0024-audit-chain-genesis-install-fingerprint.md) | Audit chain genesis via install fingerprint; Phase 8 adds signed genesis | long | Accepted |
| [0025](0025-typedefinition-names-plural-singular-shortnames.md) | TypeDefinition Names: plural, singular, shortNames | short | Accepted |
| [0026](0026-anonymous-fallback-until-rbac.md) | Anonymous-fallback authentication until Phase 6 RBAC | long | Superseded by [0044](0044-authorization-audit-and-retire-anonymous-fallback.md) |
| [0027](0027-transactional-store-api-for-atomic-mutation-audit-event.md) | Transactional store API for atomic mutation + audit + event | long | Accepted |
| [0028](0028-serviceaccount-builtin-typedefinition-token-endpoint.md) | ServiceAccount as a built-in TypeDefinition; tokens minted via dedicated endpoint | short | Accepted |
| [0029](0029-watch-via-websocket-with-opaque-resume-tokens.md) | Watch via WebSocket with opaque resume tokens | long | Accepted |
| [0030](0030-reconciler-framework-single-loop-k8s-queue-deferred.md) | Reconciler framework: single-loop now, K8s-style queue deferred | short | Accepted |
| [0031](0031-plugin-protocol-grpc-via-go-plugin.md) | Plugin protocol: gRPC via HashiCorp go-plugin | long | Accepted |
| [0032](0032-plugin-host-one-process-per-plugin.md) | Plugin host: one process per plugin; restart-on-crash; no in-process plugins in v1 | short | Accepted |
| [0033](0033-fakes-first-every-plugin-ships-a-fake.md) | Fakes-first: every plugin ships a fake in the same module | long | Accepted |
| [0034](0034-sourceconfig-as-first-class-resource.md) | `SourceConfig` as a first-class resource; sources own ingestion lifecycle | short | Accepted |
| [0035](0035-workflow-engine-temporal-embedded-and-external.md) | Workflow engine: Temporal embedded for dev, external for prod | long | Accepted |
| [0036](0036-operations-first-class-on-typedefinition.md) | Operations are first-class on TypeDefinition | short | Accepted |
| [0037](0037-operations-api-surface-post-operations.md) | Operations API surface: `POST /operations/{op}` returning an operation ID | short | Accepted |
| [0038](0038-workflow-status-mirroring-via-store-withtx.md) | Workflow → status mirroring via the same transactional contract | long | Accepted |
| [0039](0039-composite-reconciler-extends-controller-framework.md) | Composite reconciler extends the Phase 2 controller framework | short | Accepted |
| [0040](0040-operationdefinition-dispatch-plugin-or-workflow.md) | OperationDefinition.Dispatch: plugin \| workflow | short | Accepted |
| [0041](0041-composite-update-semantics-diff-and-converge.md) | Composite update semantics: diff and converge | long | Accepted |
| [0042](0042-oidc-integration-coreos-go-oidc-and-pkce-cli-flow.md) | OIDC integration: `coreos/go-oidc` + authorization-code-with-PKCE CLI flow | long | Accepted |
| [0043](0043-role-and-rolebinding-admission-and-most-specific-wins-evaluation.md) | `Role` / `RoleBinding` admission and most-specific-wins evaluation | long | Accepted |
| [0044](0044-authorization-audit-and-retire-anonymous-fallback.md) | Authorization audit events; retire anonymous fallback | short | Accepted |
| [0045](0045-fieldownership-typedefinition-admission-and-write-path-enforcement.md) | `fieldOwnership[]` on TypeDefinition: admission rules and write-path enforcement | long | Accepted |
| [0046](0046-derivation-pipeline-at-admission-inside-store-withtx.md) | Derivation pipeline runs after schema validation and inside `store.WithTx` | long | Accepted |
| [0047](0047-drift-events-cause-sourcesync-on-the-existing-bus-and-audit-chain.md) | Drift events ride the existing bus + audit chain via `Cause.SourceSync` | short | Accepted |
| [0048](0048-outbox-pattern-for-post-commit-bus-publish.md) | Outbox pattern for post-commit bus publish | long | Accepted |
| [0049](0049-mtls-service-to-service-spiffe-shaped-identity.md) | mTLS service-to-service: SPIFFE-shaped identity, file-mounted trust bundle, Helm-rotated certs | long | Accepted |
| [0050](0050-observability-conventions-otel-and-prometheus.md) | Observability conventions: OTel spans + Prometheus metrics + correlated request ID | long | Accepted |
| [0051](0051-provider-apply-rpc-for-writethrough-fan-out.md) | Provider `Apply` RPC for live writeThrough fan-out | short | Accepted |
| [0052](0052-multi-pod-outbox-ha-via-skip-locked.md) | Multi-pod outbox HA via `SELECT … FOR UPDATE SKIP LOCKED` | long | Accepted |
| [0053](0053-plugin-host-mtls-per-plugin-spiffe-identity.md) | Plugin host mTLS with per-plugin SPIFFE identity | long | Accepted |
| [0054](0054-snapshot-rpc-on-provider-interface.md) | Snapshot RPC on the Provider interface | long | Accepted |
| [0055](0055-watch-endpoint-rbac-scoping-and-clusterrole-admission.md) | Watch endpoint RBAC scoping + `ClusterRole` admission | short | Accepted |
| [0056](0056-spire-workload-api-for-short-lived-cert-rotation.md) | SPIRE Workload API for short-lived cert rotation | long | Accepted |
| [0057](0057-parameter-graph-parallel-derivation.md) | Parameter-graph parallel derivation | long | Accepted |
| [0058](0058-continuous-wal-archive-for-audit-chain.md) | Continuous WAL archive for audit chain backup | long | Accepted |
| [0059](0059-real-plugin-apply-shape-for-netbox-vcenter.md) | Real-plugin Apply shape for NetBox + vCenter | long | Accepted |
| [0060](0060-per-synckind-cursor-sub-resource.md) | Per-syncKind cursor sub-resource | long | Accepted |
| [0061](0061-vcenter-http-client-govmomi-and-rest.md) | vCenter HTTP client: govmomi vs REST adapter | long | Accepted |
| [0062](0062-spire-attested-bootstrap-token.md) | SPIRE-attested bootstrap token | long | Accepted |
| [0063](0063-multi-region-active-active-coordination.md) | Multi-region active-active coordination | long | Accepted |
| [0064](0064-audit-chain-formal-verification.md) | Audit chain formal verification | long | Accepted |
| [0065](0065-full-text-audit-search.md) | Full-text audit search index | long | Accepted |
| [0066](0066-plugin-host-on-kubernetes-jobs.md) | Plugin host on Kubernetes Jobs | long | Accepted |
| [0067](0067-wasm-in-process-plugin-path.md) | WASM in-process plugin path | long | Accepted |
| [0068](0068-per-plugin-rbac-inside-kubernetes-jobs.md) | Per-plugin RBAC inside Kubernetes Jobs | long | Accepted |
| [0069](0069-wasm-capability-policy.md) | WASM capability policy | long | Accepted |
| [0070](0070-wasm-snapshot-streaming-abi.md) | WASM snapshot streaming ABI | long | Accepted |
| [0071](0071-long-lived-per-plugin-deployments.md) | Long-lived per-plugin Deployments — non-goal | long | Accepted |
| [0072](0072-cross-tenant-authorization-boundary.md) | Cross-tenant authorization boundary | long | Accepted |
| [0073](0073-tenant-scoped-admin-surface.md) | Tenant-scoped admin surface | long | Accepted |
| [0074](0074-per-tenant-resource-quotas.md) | Per-tenant resource quotas | long | Accepted |
| [0075](0075-audit-search-http-surface.md) | Audit search HTTP surface | long | Accepted |
| [0076](0076-plugin-marketplace-metadata-surface.md) | Plugin marketplace metadata surface | long | Accepted |
| [0077](0077-tenant-first-class-admission.md) | Tenant first-class admission | long | Accepted |
| [0078](0078-billing-metering-surface.md) | Billing / metering surface | long | Accepted |
| [0079](0079-tenant-portal-architecture.md) | Tenant-portal architecture | long | Accepted |
| [0080](0080-federated-plugin-registry.md) | Federated plugin registry | long | Accepted |
| [0081](0081-sigstore-pluginpackage-verifier.md) | Sigstore PluginPackage verifier | long | Accepted |
| [0082](0082-cross-region-tenant-set-and-quota-aggregation.md) | Cross-region tenant-set + quota aggregation | long | Accepted |
| [0083](0083-multi-cloud-control-plane.md) | Multi-cloud control plane | long | Accepted |
| [0084](0084-tenant-identity-broker.md) | Tenant identity broker | long | Accepted |
| [0085](0085-paid-plugin-marketplace.md) | Paid-plugin marketplace | long | Accepted |
| [0086](0086-per-tenant-pluginregistry-curation.md) | Per-tenant PluginRegistry curation | long | Accepted |
| [0087](0087-cross-region-aggregation-expansion.md) | Cross-region aggregation expansion | long | Accepted |
| [0088](0088-multi-cloud-region-failover-orchestration.md) | Multi-cloud region failover orchestration | long | Accepted |
| [0089](0089-per-tenant-cross-cloud-replication.md) | Per-tenant cross-cloud replication | long | Accepted |
| [0090](0090-managed-tenant-data-plane.md) | Managed tenant data plane | long | Accepted |
| [0091](0091-kms-backed-identity-broker-keystore.md) | KMS-backed IdentityBrokerKeyStore | long | Accepted |
| [0092](0092-spire-federation-peering.md) | SPIRE Federation peering for the identity broker | long | Accepted |
| [0093](0093-external-audit-search-tier.md) | External audit-search tier | short | Accepted |
| [0094](0094-wasm-https-host-import.md) | WASM HTTPS host import | short | Accepted |
| [0095](0095-strict-hsm-identity-broker.md) | Strict-HSM identity broker (Signer abstraction) | short | Accepted |
| [0096](0096-strict-rpo-failover-gate-and-conflict-policy.md) | Strict-RPO failover gate + TenantReplication conflict policy | short | Accepted |
| [0097](0097-fully-managed-runtime.md) | Fully-managed runtime (containerd-backed scheduler) | short | Accepted |
| [0098](0098-billing-event-fraud-detection.md) | BillingEvent fraud / abuse detection | short | Accepted |
| [0099](0099-self-service-hosted-operator-onboarding.md) | Self-service hosted-operator onboarding | short | Accepted |
| [0100](0100-retire-getprivatekey-from-identity-broker.md) | Retire `IdentityBrokerKeyStore.GetPrivateKey` (Signer-only broker) | short | Accepted |
| [0101](0101-multi-region-tenant-workload-placement.md) | Multi-region TenantWorkload placement | short | Accepted |
| [0102](0102-persistent-abuse-detector-storage.md) | Persistent abuse-detector storage | short | Accepted |
| [0104](0104-onboarding-device-code-and-observability-followups.md) | Onboarding device-code + observability follow-ups | short | Accepted |
| [0105](0105-container-image-packaging-and-mtls-opt-in-default.md) | Container image packaging, version alignment, and mTLS-opt-in chart default | long | Accepted |
| [0106](0106-plugin-kind-autoregistration.md) | Plugin-kind auto-registration on ProviderConfig load | long | Accepted |

ADR-0103 was reserved for the conditional firecracker / VM-shaped runtime decision; its Phase-21 gate never tripped, so the number is an intentional gap — no file exists.
