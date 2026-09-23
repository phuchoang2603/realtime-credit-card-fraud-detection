## Purpose

Establish independently operable Go and Python service boundaries with predictable readiness, lifecycle, error and contract behavior.

## ADDED Requirements

### Requirement: Independent service ownership
Services SHALL build and run without starting another service or accessing its database. Go SHALL be primary for edge, Accounts, Payments and delivery; fraud serving and training SHALL retain Python ownership. Shared artifacts SHALL contain technical helpers or versioned wire contracts, not shared mutable domain models. The conventions SHALL document composition, transport/domain/persistence boundaries, service-owned migrations and independently deployable units.

#### Scenario: Build a service independently
- **WHEN** the edge or fraud service is built from a clean checkout using its documented command
- **THEN** its runtime artifact is produced without starting sibling services or relying on the marketplace checkout
- **AND** the artifact can be started with its own configuration

#### Scenario: Review a persistence integration
- **WHEN** a service adds persistence or a migration
- **THEN** its documented ownership restricts reads, writes and migrations to its own state
- **AND** communication with another service uses a versioned external contract

### Requirement: Distinct liveness and readiness
The edge SHALL expose HTTP `/health` for liveness and `/ready` for readiness (200 when ready, 503 otherwise). Fraud SHALL expose standard gRPC health checks with distinct `liveness` and `readiness` service names; readiness SHALL report SERVING only when required resources are initialized and requests can be accepted, and NOT_SERVING otherwise. Fraud readiness SHALL require a usable loaded model. An unavailable telemetry backend SHALL NOT alone make a service unready.

#### Scenario: Model unavailable
- **WHEN** fraud model loading fails
- **THEN** gRPC liveness reports SERVING
- **AND** gRPC readiness reports NOT_SERVING and Predict returns UNAVAILABLE

#### Scenario: Service ready
- **WHEN** required startup resources are available and shutdown has not started
- **THEN** the service readiness check reports ready

### Requirement: Bounded resource lifecycle
Service startup SHALL validate configuration before reporting ready. Shutdown SHALL withdraw readiness, stop accepting new work, allow active work a configured bounded drain period, and release resources with bounded telemetry flush. Imports and isolated application construction SHALL NOT open listeners or load model artifacts.

#### Scenario: Invalid configuration
- **WHEN** startup receives an invalid port or shutdown timeout
- **THEN** startup fails with a useful sanitized diagnostic and nonzero exit status
- **AND** the service does not report ready

#### Scenario: Termination with active work
- **WHEN** termination is requested during an active request
- **THEN** the service stops accepting new work and allows the request to finish within the drain budget
- **AND** work exceeding the budget cannot prevent bounded process exit

#### Scenario: Isolated test instances
- **WHEN** tests import the application and construct multiple isolated instances
- **THEN** no metrics-port collision or model load occurs until the respective lifecycle is started
- **AND** closing one instance does not clear another instance's model

### Requirement: Compatible contracts and safe errors
The supported fraud request/response schemas, decision thresholds, validation semantics and operational endpoints SHALL be explicit and verified by consumer tests. Legacy compatibility SHALL NOT constrain modernization; intentional changes SHALL update implementation, contracts and tests together. Internal synchronous business contracts SHALL use versioned protobuf services with generated Go/Python bindings; legacy internal JSON APIs SHALL be removed. Contract conventions SHALL state ownership, compatibility rules and breaking-version handling. Public errors SHALL NOT expose stack traces, credentials or storage internals; new contracts SHALL use stable error categories mapped at the transport boundary.

#### Scenario: Supported prediction caller
- **WHEN** a request matching the supported schema is submitted after a refactor or complete implementation rewrite
- **THEN** its status, response schema and risk decision match the documented current contract

#### Scenario: Unexpected application failure
- **WHEN** an unhandled adapter failure reaches an API boundary
- **THEN** the caller receives a sanitized failure response and operators can correlate its structured diagnostic

### Requirement: Consistent operational identity
Both language examples SHALL emit structured logs with a consistent service identity and real trace/span identifiers when available. Request correlation SHALL be propagated or generated when absent. Metric dimensions SHALL avoid unbounded request, customer or transaction identifiers. Existing fraud metrics and direct shared-platform trace export SHALL remain compatible.

#### Scenario: Request without an incoming correlation identifier
- **WHEN** a request arrives without a request identifier
- **THEN** its response and structured request diagnostic share a generated identifier
- **AND** a startup log without an active span does not fabricate trace identifiers

### Requirement: Edge translation and internal RPC
The public edge SHALL expose the existing prediction capability through JSON HTTP and call the fraud service using a reused gRPC channel, finite deadlines and propagated cancellation/correlation. Internal fraud SHALL NOT expose an HTTP business API.

#### Scenario: Public prediction
- **WHEN** a valid public prediction request reaches the edge
- **THEN** it invokes the generated fraud RPC and returns the prediction in public JSON
- **AND** invalid input, unavailable service and unexpected errors map to safe HTTP responses

#### Scenario: Missing protobuf field
- **WHEN** a prediction omits a required feature
- **THEN** fraud rejects it with INVALID_ARGUMENT rather than using an implicit scalar zero
