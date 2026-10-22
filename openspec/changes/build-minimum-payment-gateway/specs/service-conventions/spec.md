## ADDED Requirements

### Requirement: Internal Go service operation

Payments, notifier, vault simulator and processor simulator SHALL follow the edge's composition-root shape with transport, application, domain and adapter packages. Each SHALL expose standard gRPC health checks with `liveness` and `readiness` service names. Readiness SHALL require its own database to be reachable with migrations applied, and SHALL NOT require any other service or Redis to be reachable. Each SHALL apply only its own migrations, bound shutdown like the edge, and emit structured logs, traces and metrics with a consistent service identity. Internal gRPC SHALL rely on the service mesh for transport encryption and caller identity rather than application TLS or bearer credentials; the vault SHALL separate tokenization/metadata from processing resolution on distinct ports authorized per workload identity. Probes SHALL use the native gRPC health check.

#### Scenario: Fraud unavailable at Payments startup

- **WHEN** Payments starts while fraud is down but its database is ready
- **THEN** Payments reports ready, and attempts fail closed as specified

#### Scenario: Database unreachable

- **WHEN** a service's database is unreachable
- **THEN** liveness stays SERVING and readiness reports NOT_SERVING

#### Scenario: Redis unavailable at Payments startup

- **WHEN** Payments starts with its database ready but Redis unavailable
- **THEN** it reports ready and uses durable database idempotency until Redis returns
