## Context

The original Python HTTP service mixed transport, inference and runtime resources.
The approved implementation moves public HTTP/JSON to a Go edge and uses gRPC
internally. The latest user direction favors useful tests, minimal CI and remote
verification over score tooling and repeating every check locally.

## Goals / Non-Goals

Provide a typed cross-language prediction path, independent builds, safe lifecycle
and compact checks. Do not implement Payments, choose persistence/event products,
maintain a parallel legacy REST API, or deploy to a cluster.

## Decisions

### Contracts and service ownership

`contracts/fraud/v1/fraud.proto` owns the internal API. A pinned generation script
writes service-local Go/Python bindings, permitting independent image builds.
Optional scalar presence distinguishes missing features from legitimate zeros.
The edge accepts protobuf JSON field names and maps gRPC errors to safe HTTP
responses. It reuses its channel and propagates deadlines/cancellation, request IDs
and W3C trace context. Future Payments owns payment orchestration; edge prediction
is only the existing scoring capability. Async events need their own durable
transport even if their payloads use protobuf.

### Python runtime

A composition root owns model, executor, metrics, tracing and gRPC resources.
Immutable validated inputs feed synchronous rules and inference. Worker capacity
remains occupied until native inference completes even if its RPC is canceled.
Instance tracing avoids global provider replacement. Telemetry cleanup and RPC
drain are bounded; a shutdown watchdog prevents stuck native calls from holding
interpreter exit indefinitely. The converted model targets the locked sklearn
runtime without private runtime patches; this is not retraining.

### Readiness and deployment

The edge keeps HTTP `/health` and `/ready`. Fraud replaces both HTTP endpoints with
the standard gRPC health API using `liveness` and `readiness` service names.
Liveness reports a live process; readiness additionally requires the model and
withdraws at shutdown. A missing model stops routing without a restart loop.
The chart uses Kubernetes 1.27+ numeric-port gRPC probes on 8000, metrics on 8010,
a ClusterIP service and no public fraud ingress. Image and chart protocol changes
must roll forward/back together; use immutable image tags for deployment.

### Tests and CI

Keep a small suite protecting decision boundaries and real-model repeatability.
Configuration, observability and transport tests are out of scope. Reuse existing
coverage instead of asserting framework internals or duplicating cases. Retain one
bounded real-model property. Service tools and dedicated readiness/process-probe
tests are removed at the user's request; runtime health checks remain configured.

Independent Python, Go and Helm jobs run lint, behavior/race tests and lint/render validation for every tracked chart and its values overrides.
There is no path-filter job or matrix. CI owns the check jobs. Two explicit image
jobs reuse one image workflow: PRs build without publication and main pushes
publish. Standard GitHub Actions are reused.

Custom coverage/mutation scoring and screenshot tools are removed per the latest
scope. The old 69.17% mutation failure remains documented; these changes do not
turn it into a pass. CI logs and the PR are the current review evidence.

## Risks / Trade-offs

- Protobuf JSON names intentionally break the prior uppercase payload convention.
- Native inference cannot be interrupted; retained worker slots bound concurrent work.
- Plaintext internal gRPC assumes cluster-private networking; transport identity/TLS policy remains future routing/security work.
- Three independent check jobs run on every PR; bindings regenerate through devenv instead of a custom CI drift gate.
- Coverage/mutation scores no longer gate changes; reviewers assess the distinct behaviors exercised by tests.

## Migration Plan

Commit runtime/contracts first, then test/CI simplification and coherent docs.
Run relevant lightweight checks locally, push a feature branch and open a PR.
Use GitHub-hosted checks for complete integration and image builds; fix actionable
failures in follow-up commits. Do not merge, archive or deploy automatically.

Go development uses `languages.go` with Delve and gopls, following the marketplace devenv pattern instead of adding `pkgs.go` directly.

Protobuf bindings regenerate through the devenv `codegen:proto` task before shell
entry when contracts or generation dependencies change. Nix supplies `protoc` and both languages' plugins; Python needs no compiler dependency. Commit generated sources alongside
contract edits. There is no custom generation script or CI drift gate.
