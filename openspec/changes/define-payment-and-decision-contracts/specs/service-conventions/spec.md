## ADDED Requirements

### Requirement: Liveness and readiness probes

The edge SHALL expose HTTP `/health` for liveness and `/ready` for readiness (200 when ready, 503 otherwise). Fraud SHALL expose standard gRPC health checks with distinct `liveness` and `readiness` service names; readiness SHALL report SERVING only when startup has completed, the decision policy is active and shutdown has not started, and NOT_SERVING otherwise. An unavailable telemetry backend SHALL NOT alone make a service unready.

#### Scenario: Shutdown started

- **WHEN** fraud shutdown begins
- **THEN** gRPC readiness reports NOT_SERVING while liveness remains SERVING until the process exits

#### Scenario: Service ready

- **WHEN** required startup resources are available and shutdown has not started
- **THEN** the service readiness check reports ready

### Requirement: Versioned contracts and safe errors

The supported fraud decision request/response schemas, policy thresholds, validation semantics and operational endpoints SHALL be explicit in the versioned contract and the documentation; decision rules and thresholds SHALL be protected by behavior tests. Legacy compatibility SHALL NOT constrain modernization; intentional changes SHALL update implementation, contracts and tests together. Internal synchronous business contracts SHALL use versioned protobuf services with generated bindings for each consuming language; legacy internal JSON APIs SHALL be removed. Contract conventions SHALL state ownership, compatibility rules and breaking-version handling. Public errors SHALL NOT expose stack traces, credentials or storage internals; new contracts SHALL use stable error categories mapped at the transport boundary.

#### Scenario: Supported decision caller

- **WHEN** a request matching the supported decision schema is submitted after a refactor or complete implementation rewrite
- **THEN** its status, response schema, outcome and reason codes match the documented current contract

#### Scenario: Unexpected application failure

- **WHEN** an unhandled adapter failure reaches an API boundary
- **THEN** the caller receives a sanitized failure response and operators can correlate its structured diagnostic

## MODIFIED Requirements

### Requirement: Bounded resource lifecycle

Service startup SHALL validate configuration before reporting ready. Shutdown SHALL withdraw readiness, stop accepting new work, allow active work a configured bounded drain period, and release resources with bounded telemetry flush. Imports and isolated application construction SHALL NOT open listeners or start telemetry.

#### Scenario: Invalid configuration

- **WHEN** startup receives an invalid port or shutdown timeout
- **THEN** startup fails with a useful sanitized diagnostic and nonzero exit status
- **AND** the service does not report ready

#### Scenario: Termination with active work

- **WHEN** termination is requested during an active request
- **THEN** the service stops accepting new work and allows the request to finish within the drain budget
- **AND** work exceeding the budget cannot prevent bounded process exit

#### Scenario: Isolated instances

- **WHEN** the application is imported and multiple isolated instances are constructed
- **THEN** no metrics-port collision occurs until the respective lifecycle is started
- **AND** closing one instance does not change another instance's readiness

### Requirement: Consistent operational identity

Both language examples SHALL emit structured logs with a consistent service identity and real trace/span identifiers when available. Request correlation SHALL be propagated or generated when absent. Metric dimensions SHALL avoid unbounded request, customer, payment or transaction identifiers. Fraud metric names and labels SHALL be documented, and existing fraud metrics SHALL remain available from the application metrics endpoint. Trace export SHALL be disabled by default and SHALL require explicit enablement and a configured OTLP endpoint; disabled tracing SHALL NOT require a trace backend for service startup or requests. Application dashboards and alerts are deferred.

#### Scenario: Request without an incoming correlation identifier

- **WHEN** a request arrives without a request identifier
- **THEN** its response and structured request diagnostic share a generated identifier
- **AND** a startup log without an active span does not fabricate trace identifiers

#### Scenario: Unconfigured trace export

- **WHEN** the fraud service starts without trace export enabled or an endpoint configured
- **THEN** requests succeed without attempting to contact a trace backend

#### Scenario: Explicit trace export

- **WHEN** trace export is enabled with an explicit OTLP endpoint
- **THEN** spans use the configured service identity and export to that endpoint

#### Scenario: Trace export enabled without endpoint

- **WHEN** tracing is explicitly enabled without an OTLP endpoint
- **THEN** startup rejects the invalid configuration rather than selecting a backend

## REMOVED Requirements

### Requirement: Distinct liveness and readiness

**Reason**: Fraud readiness no longer depends on a loaded model.
**Migration**: Replaced by "Liveness and readiness probes", which keeps the endpoint and health-service names.

### Requirement: Compatible contracts and safe errors

**Reason**: The supported contract is now the `fraud.v2` decision RPC rather than prediction.
**Migration**: Replaced by "Versioned contracts and safe errors" with the same compatibility and error rules.

### Requirement: Edge translation and internal RPC

**Reason**: Risk decisions are requested by Payments on the internal path; a public prediction route has no caller in the target architecture and would be a parallel path to retire later.
**Migration**: Internal callers use `fraud.v2.FraudService/Decide` over gRPC. The edge gains public checkout routes in `build-minimum-payment-gateway`; missing-field rejection is now specified in `fraud-decision`.
