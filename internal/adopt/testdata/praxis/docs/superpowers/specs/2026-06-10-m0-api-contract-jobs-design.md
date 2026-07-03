# M0 API-side — OpenAPI contract, generated clients, CLI rebuild, jobs seam

Date: 2026-06-10
Status: approved (brainstorming session, all three strategy forks settled by user)
Milestone: Phase 2 / M0 (docs/Phase2Pass.md) — API-side half. The console
fake-data swap (OIDC PKCE login, fetch layer, screen swaps) is the next
plan and consumes everything built here.

## Context: what already exists

Exploration findings that reframe the Phase2Pass M0 text:

- `internal/api/openapi.go` already builds an OpenAPI 3.0.3 document from
  the TypeDefinition registry at runtime and serves it at
  `/v1/openapi.json` (ADR-0010). It is good for live discovery, weak for
  codegen: no `operationId`s, untyped map construction, per-kind dynamic
  paths.
- ADR-0037 Operations are already a jobs API in all but name:
  `POST .../{name}/operations/{op}` mints an Operation resource, a
  Temporal workflow executes it (embedded in dev; `cmd/praxis-worker` is
  the standalone, stateless, queue-driven worker pool per ADR-0035),
  status writes ride the transactional outbox to JetStream (ADR-0048),
  and flat `GET /v1/operations/{id}` + `POST .../cancel` exist.
- What Operations lack vs the console's Job shape
  (`console/src/lib/fake/jobs.ts`): incremental progress (`progressPct`),
  an event log (`events[]`), and a list endpoint (only get-by-ID exists).
- The CLI (`internal/cli`, 66 files) rides a hand-rolled generic HTTP
  client (`internal/cli/client.go`). The resource tree is genuinely
  generic (`{kind}` chi param at `internal/api/server.go:712`), so a
  generated client can model it generically too.

## Decisions (user-settled)

1. **Spec strategy: hand spec + keep generator.** Hand-author
   `api/openapi.yaml` as THE contract. The runtime generator stays as the
   live, registry-aware doc behind the console's API explorer. A
   consistency test keeps the two agreeing where they overlap.
2. **Jobs scope: Operations ARE jobs + progress events.** No new Job
   kind, no parallel NATS execution lane. Temporal stays the engine; M0
   adds the progress seam and list surface.
3. **Codegen policy: commit generated code.** `pkg/client` (Go) and
   `clients/python/` are committed; CI regenerates and fails on diff.

## Design

### 1. The OpenAPI contract — `api/openapi.yaml`

- OpenAPI 3.0.3 (matches the generator), YAML, lives next to `api/proto/`.
- Covers the **entire** routed surface ("no hidden APIs"): health/version
  probes, `/v1/types`, audit verify + search, `/v1/watch` handshake,
  RBAC bootstrap, service-account token issuance, identity SVID
  (jwt/x509), admin endpoints (svid rotate, servicetoken revoke,
  marketplace intent, failover trigger), federation bundle, source
  sync/webhook, the generic resource tree modeled with a `{kind}` path
  parameter (list/create/get/put/patch/delete/start-operation), flat
  operation get/cancel, and the new `GET /v1/operations` (below).
- Every operation has an `operationId` and typed request/response
  schemas; the shared `Error` envelope
  (`code/message/details/remediation/requestId`) is defined once and
  referenced everywhere. Watch frame and audit search schemas carry over
  from the generator's components.
- New Phase 2 endpoints are spec-first from here on: the YAML diff is
  reviewed before the handler is written. `GET /v1/operations` is the
  first exercise of that rule.

### 2. Contract tests (the pin)

Plain Go tests against the existing `internal/api` httptest harness — no
containers, so they run in the normal `test` CI job. Three layers:

- **Route parity.** Walk the chi router (`chi.Walk`) and the spec paths;
  a route absent from the spec or a spec path absent from the router
  fails. This is the "spec drift fails CI" mechanism.
- **Response conformance.** Representative requests per endpoint
  validated with `kin-openapi`'s `openapi3filter` (request + response
  schema validation). Loading the spec through kin-openapi also
  validates the document itself — no extra lint tooling.
- **Generator consistency.** Where the hand spec and `buildOpenAPI`
  overlap (fixed endpoints, resource-tree shape, shared component
  schemas), assert agreement so the console explorer never contradicts
  the contract.

### 3. Generated clients, committed

- Go: `oapi-codegen` → `pkg/client` (importable as
  `github.com/praxis-io/praxis/pkg/client`). The generic `{kind}` tree
  yields generic resource CRUD — the CLI's kubectl-ish shape.
- Python: `openapi-python-client` → `clients/python/`.
- `make generate-api` regenerates both; CI regenerates and fails on
  `git diff --exit-code`, so spec/client drift is impossible.
- Toolchain versions pinned (Go tool dependency for oapi-codegen;
  pinned requirement for openapi-python-client).
- Python smoke test in CI: a pytest that imports the generated package,
  boots `praxis-api` (in-memory store, same harness the Go e2e uses),
  and round-trips `/healthz` + a resource list call — proves the client
  is usable, not just generated.

### 4. CLI rebuilt on the Go client

- `internal/cli/client.go`'s hand-rolled HTTP core delegates to
  `pkg/client`. The generated client is exercised continuously by the
  whole CLI test suite from then on (Phase2Pass principle 5).
- Behavior contract frozen: ADR-0011 noun/verb surface and §15 exit
  codes unchanged; existing CLI tests prove the rebuild changes
  plumbing, not behavior.
- Admin subcommands migrate in the same pass (the spec covers their
  endpoints). Anything that resists cleanly becomes an explicit
  follow-up item in the plan — nothing silently left hybrid.

### 5. Jobs seam = Operations + progress

- **Progress on the Operation record:** add `progressPct` (0–100,
  optional) and `events[]` (timestamped `{at, level, msg}` entries — the
  exact shape `console/src/lib/fake/jobs.ts` uses) to the Operation
  domain type and its spec schema. Workflows/activities report progress
  through a small helper; writes land in the store and ride the existing
  outbox to JetStream (ADR-0048) — "progress events on the bus" falls
  out of machinery already in place. Operations are stored resources, so
  `/v1/watch?kind=Operation` streams live job updates with no new
  transport.
- **`GET /v1/operations`:** tenant-scoped, filterable list (status,
  kind, limit, cursor). Spec-first. RBAC: synthesizes
  (verb=list, kind=Operation) analogous to the audit-search middleware.
- **CLI:** extend the existing `praxis operation` group (today:
  `start`, `get`) with `list`, `cancel`, and `watch`, all riding the
  generated client.
- **Demo job (M0 exit criterion):** the builtin echo operation reports
  staged progress; an e2e test drives
  submit → bus → worker → progress events → succeeded → audit entry.

### 6. Error handling & testing strategy

- Error envelope: modeled once in the spec; conformance layer validates
  real error responses against it.
- TDD per task (test-driven-development skill); contract tests +
  codegen-drift check in CI; echo-job e2e joins `test/e2e`.
- CI placement: contract tests in the `test` job (no containers);
  codegen drift as a step in `checks` (or its own quick job); e2e in the
  existing e2e lane.

## Out of scope (explicit)

- Console fake-data swap (next plan: OIDC PKCE, fetch layer, screens).
- Worker autoscale (M4-adjacent; the seam — stateless queue-driven
  workers — already exists).
- New resource kinds, tenant-portal retirement, audit-arc leftovers
  (WS-A2/WS-G), console CI job.

## Risks / notes

- The hand spec models `{kind}` generically; per-kind typed clients are
  deliberately NOT generated in M0 (they arrive with M1's spec-first
  compute endpoints).
- `buildOpenAPI` consistency test must tolerate the generator's
  registry-dynamic paths (compare the fixed-endpoint subset + tree
  shape, not the concrete plural paths).
- Temporal pin (server 1.32.0-157.1 prerelease) is untouched by this
  work; progress writes use the store, not new Temporal features.
- CLI rebuild is the largest single risk (66 files touch the client);
  mitigated by the frozen-behavior test gate and incremental task
  decomposition in the plan.
