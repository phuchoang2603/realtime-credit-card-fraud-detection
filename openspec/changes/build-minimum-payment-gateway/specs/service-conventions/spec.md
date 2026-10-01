## ADDED Requirements

### Requirement: Internal Go service operation

Accounts, Payments and the processor simulator SHALL follow the edge's composition-root shape with transport, application, domain and adapter packages. Each SHALL expose standard gRPC health checks with `liveness` and `readiness` service names. Readiness SHALL require its own database to be reachable with migrations applied, and SHALL NOT require any other service to be reachable. Each SHALL apply only its own migrations, bound shutdown like the edge, and emit structured logs, traces and metrics with a consistent service identity.

#### Scenario: Fraud unavailable at Payments startup

- **WHEN** Payments starts while fraud is down but its database is ready
- **THEN** Payments reports ready, and attempts fail closed as specified

#### Scenario: Database unreachable

- **WHEN** a service's database is unreachable
- **THEN** liveness stays SERVING and readiness reports NOT_SERVING
