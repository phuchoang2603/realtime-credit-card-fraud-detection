# Service conventions

The gateway follows the ownership and deployment boundaries in [the architecture guide](../architecture/payment-gateway.md). The refurbished marketplace is a reference for composition roots, Go modules, versioned contracts and small shared runtime helpers. Its service-owned database code is not copied into Payments: domain and application code own interfaces, while persistence adapters remain private to each service.

## Layout

Each service is an independently buildable and deployable unit:

```text
src/
  edge/
    go.mod
    cmd/edge/main.go             # configuration and composition root
    internal/
      transport/http/            # protocol adapters
      application/               # use-case orchestration
      domain/                    # business rules and ports
      adapters/                  # persistence and external clients
  fraud-service/
    pyproject.toml
    app/                          # Python composition and application packages
    tests/
contracts/
  fraud/v1/                       # versioned protobuf contracts
shared/                           # only technical helpers or wire contracts
```

The edge image is packaged independently from its own module with `GOWORK=off` (only `SERVICE_PATH=src/edge` is copied) using the reusable [`infra/docker/go-service.Dockerfile`](../../infra/docker/go-service.Dockerfile); the fraud image keeps its existing `src/fraud-service` context and publication tags. Two explicit service jobs reuse one image workflow while retaining the fraud image package name and tags.

The exact Python package layout may be rewritten as the service modernizes. Internal names, globals and test imports are not public contracts. The container command, test discovery, test configuration and documentation must move together with a rewrite.

## Composition and dependency direction

Composition roots validate configuration, construct adapters and assemble the application. They do not contain business decisions. Dependencies point inward:

```text
transport -> application -> domain
                ^             ^
                |             |
           adapters implement ports
```

Transport handlers map protocol input and errors. Application services coordinate use cases. Domain code owns decisions and stable errors without importing FastAPI, gRPC, SQL drivers or broker clients. Adapters implement interfaces owned by the consuming application. A service never imports another service's `internal` package or reads another service's database.

Go is primary for edge, Accounts, Payments and webhook delivery. Fraud serving, feature work and training remain Python workloads. A shared package may provide logging, tracing, metrics, lifecycle or versioned wire-contract support; it must not provide mutable domain models or cross-service repositories.

## Persistence and contracts

Each service owns its schema, migrations and persistence adapter. A deployment or migration job for one service cannot modify another service's state. Domain events, integration messages and projections remain separate concepts. Cross-service communication uses a versioned contract under `contracts/<capability>/vN/` or an equivalent explicitly owned module. Additive changes require compatible readers; breaking semantics require a new supported version and an announced migration window.

No database, broker, event-store implementation or application framework is selected by these conventions. Those decisions belong to the capability that needs them. Payments will add event storage and recovery rules under #33/#35; this change only makes the boundary testable.

## Runtime behavior

The public edge provides `/health` for process liveness and `/ready` for capability readiness. Readiness is 200 only after required resources are initialized and before shutdown starts. Health remains available when a recoverable dependency, such as the fraud model, is unavailable. Shutdown withdraws readiness, drains active work for a bounded period, flushes owned telemetry, then closes resources. Telemetry backend availability alone does not determine readiness.

Logs are structured, use a configured service identity and carry a propagated or generated request ID. Trace and span IDs are included only when an active span supplies them. Secrets, credentials and stack traces never appear in public errors. Metric labels do not contain unbounded request, customer or transaction identifiers.

## Verification

The [verification strategy](verification.md) owns the behavior checks and CI scope. New services add contract, lifecycle and behavior tests at their owning layer. A complete Python rewrite is acceptable when it preserves the external fraud contract and produces stronger isolation, lifecycle handling and evidence.

## Reference implementation

The marketplace's `services/orders/cmd/orders/main.go` demonstrates a composition root; `shared/runtime/http.go` demonstrates bounded HTTP shutdown; `shared/err/grpcerr/map.go` demonstrates transport error mapping; and `docs/development/code-generation.md` documents workspace/module conventions. This repository adopts those patterns selectively and keeps gateway domain ownership separate.

## Implemented Python example

The composition root is `app.main:FraudServer`; `python -m app` owns validated gRPC server configuration. The application service and local rules are synchronous, and grpc.aio dispatches inference to a bounded worker pool. Frozen Pydantic request objects reject extra/non-finite data. Transport code owns gRPC statuses; rules raise application errors.

A gRPC interceptor provides generated/propagated request IDs, sanitized unexpected
failures and instance-owned OpenTelemetry server spans with W3C parent propagation.
Provider creation belongs to startup; no global provider is replaced and cleanup
is bounded. The current model artifact replaces runtime sklearn monkeypatches.

Internal synchronous communication uses versioned protobuf services under
`contracts/<capability>/vN/`, with generated Go/Python bindings. The edge alone
translates public HTTP/JSON; it reuses its gRPC connection and propagates deadlines,
cancellation and correlation. Its `/predict` exposes existing scoring, while future
Payments owns payment orchestration. Events may use protobuf with a broker/outbox;
gRPC does not replace durable asynchronous delivery.

Fraud implements standard gRPC health checks: `liveness` reports a running process,
`readiness` requires a loaded model, and `fraud.v1.FraudService` follows readiness.
A missing model must stop traffic without triggering a liveness restart loop.
The edge keeps `/health` and `/ready`: the latter withdraws before shutdown and
reflects local serving capability, not the health of every downstream service.

Tests should protect fraud decision boundaries and real-model repeatability. Configuration validation, observability and gRPC/HTTP transport do not need tests. Coverage and mutation results remain honest measurements, not reasons to add low-value tests. Boolean configuration uses `true`/`false` (case-insensitive); aliases such as `yes`, `on` and `1` are intentionally unsupported.
