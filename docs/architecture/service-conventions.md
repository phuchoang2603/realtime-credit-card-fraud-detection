# Service conventions

Each service is an independently buildable and deployable unit that owns its runtime, contracts and dependencies. Runtime behavior (liveness/readiness, bounded lifecycle, safe errors, operational identity, edge translation) is specified in [`openspec/specs/service-conventions`](../../openspec/specs/service-conventions/spec.md); this page records layout, dependency direction and ownership. The [refurbished marketplace](https://github.com/phuchoang2603/refurbished-marketplace/blob/main/docs/architecture.md) is the reference for composition roots, Go modules, versioned contracts and small shared runtime helpers.

## Layout

```text
contracts/
  payments/v1/                   # payment types and lifecycle events
  fraud/v2/                      # internal decision RPC
  labels/v1/                     # delayed fraud labels
src/
  edge/                          # Go module; cmd/edge is the composition root
    cmd/edge/main.go
    internal/config/             # validated environment configuration
    internal/transport/          # HTTP health endpoints and correlation middleware
  fraud-service/                 # Python package; app.main:FraudServer is the composition root
    app/main.py                  # gRPC composition and lifecycle
    app/domain/                  # validated decision values and pure policy rules
    app/transport/               # gRPC adapter, request mapping and correlation
    app/telemetry/               # logging, metrics and tracing setup
    gen/fraud/v2/                # generated decision bindings
    gen/payments/v1/             # generated shared payment types
    gen/labels/v1/               # generated label types
    tests/
shared/                          # only technical helpers or wire contracts (none yet)
```

Both languages keep business rules separate from transport. The edge has no business domain yet, so it needs only HTTP transport and configuration. Python fraud serving evaluates a single stateless policy in `app/domain`, maps protobuf at `app/transport` and keeps telemetry in `app/telemetry`. Add an application/use-case layer only when a service coordinates multiple domain or I/O operations, and add storage adapters only when it persists data. Do not create empty layers for symmetry.

## Dependency direction

```text
transport -> domain
transport -> telemetry
```

Composition roots validate configuration and assemble the service; they contain no business decisions. Transport handlers map protocol input and errors. Domain code owns decisions and stable errors without importing gRPC, HTTP frameworks, SQL drivers or broker clients. When persistence or multi-step use cases arrive, add a storage adapter and a use-case coordinator around the domain rather than putting I/O in the policy. A service never imports another service's `internal` package or reads another service's database.

## Persistence and contracts

Each service owns its schema, migrations and persistence adapter; a deployment or migration job for one service cannot modify another service's state. Domain events, integration messages and projections remain separate concepts.

Internal synchronous communication uses versioned protobuf services under `contracts/<capability>/vN/` with generated, committed Go and Python bindings; no parallel internal REST/JSON API is maintained. Additive changes require compatible readers; breaking semantics require a new version and an announced migration window. Public HTTP/JSON belongs only at the Go edge. Asynchronous events may use protobuf payloads but need their own durable transport with an outbox or equivalent boundary.

The versioned protobufs in `contracts/` are the shared wire contracts. Extract technical helpers only when multiple services in the same language actually reuse them; Go and Python cannot share runtime code directly. Do not share mutable domain models or cross-service repositories. No database, broker, event store or application framework is chosen by these conventions; Payments will add event storage under [#33](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/33) and [#35](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/35).

## Images and reference material

The edge image is built from its own module with `GOWORK=off` using the reusable [`infra/docker/go-service.Dockerfile`](../../infra/docker/go-service.Dockerfile); the fraud image keeps its `src/fraud-service` context. Two explicit image jobs share one workflow (see [CONTRIBUTING](../../CONTRIBUTING.md)).

In the marketplace, `services/orders/cmd/orders/main.go` demonstrates a composition root, `shared/runtime/http.go` bounded HTTP shutdown, `shared/err/grpcerr/map.go` transport error mapping and `docs/development/code-generation.md` workspace/module conventions. This repository adopts those patterns selectively and keeps gateway domain ownership separate.
