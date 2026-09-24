# Service conventions

Each service is an independently buildable and deployable unit that owns its runtime, contracts and dependencies. Runtime behavior (liveness/readiness, bounded lifecycle, safe errors, operational identity, edge translation) is specified in [`openspec/specs/service-conventions`](../../openspec/specs/service-conventions/spec.md); this page records layout, dependency direction and ownership. The [refurbished marketplace](https://github.com/phuchoang2603/refurbished-marketplace/blob/main/docs/architecture.md) is the reference for composition roots, Go modules, versioned contracts and small shared runtime helpers.

## Layout

```text
contracts/
  fraud/v1/                      # versioned protobuf contracts
src/
  edge/                          # Go module; cmd/edge is the composition root
    cmd/edge/main.go
    gen/fraud/v1/                # generated client bindings, committed
    internal/config/             # validated environment configuration
    internal/server/             # HTTP adapters and gRPC client translation
  fraud-service/                 # Python package; app.main:FraudServer is the composition root
    app/                         # composition, application service, rules, schema, runtime
    fraud/v1/                    # generated server bindings, committed
    tests/
shared/                          # only technical helpers or wire contracts (none yet)
```

Future Go services (Accounts, Payments, webhook delivery) follow the edge shape with `internal/transport`, `internal/application`, `internal/domain` and `internal/adapters`. Fraud serving, feature work and training remain Python.

## Dependency direction

```text
transport -> application -> domain
                ^             ^
                |             |
           adapters implement ports
```

Composition roots validate configuration, construct adapters and assemble the application; they contain no business decisions. Transport handlers map protocol input and errors. Application services coordinate use cases. Domain code owns decisions and stable errors without importing gRPC, HTTP frameworks, SQL drivers or broker clients. Adapters implement interfaces owned by the consuming application. A service never imports another service's `internal` package or reads another service's database.

## Persistence and contracts

Each service owns its schema, migrations and persistence adapter; a deployment or migration job for one service cannot modify another service's state. Domain events, integration messages and projections remain separate concepts.

Internal synchronous communication uses versioned protobuf services under `contracts/<capability>/vN/` with generated, committed Go and Python bindings; no parallel internal REST/JSON API is maintained. Additive changes require compatible readers; breaking semantics require a new version and an announced migration window. Public HTTP/JSON belongs only at the Go edge. Asynchronous events may use protobuf payloads but need their own durable transport with an outbox or equivalent boundary.

A shared package may provide logging, tracing, metrics, lifecycle or wire-contract support; it must not provide mutable domain models or cross-service repositories. No database, broker, event store or application framework is chosen by these conventions; Payments will add event storage under [#33](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/33) and [#35](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/35).

## Images and reference material

The edge image is built from its own module with `GOWORK=off` using the reusable [`infra/docker/go-service.Dockerfile`](../../infra/docker/go-service.Dockerfile); the fraud image keeps its `src/fraud-service` context. Two explicit image jobs share one workflow (see [CONTRIBUTING](../../CONTRIBUTING.md)).

In the marketplace, `services/orders/cmd/orders/main.go` demonstrates a composition root, `shared/runtime/http.go` bounded HTTP shutdown, `shared/err/grpcerr/map.go` transport error mapping and `docs/development/code-generation.md` workspace/module conventions. This repository adopts those patterns selectively and keeps gateway domain ownership separate.
