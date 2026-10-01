## MODIFIED Requirements

### Requirement: Consistent operational identity

Both language examples SHALL emit structured logs with a consistent service identity and real trace/span identifiers when available. Request correlation SHALL be propagated or generated when absent. Metric dimensions SHALL avoid unbounded request, customer or transaction identifiers. Existing fraud metrics SHALL remain available from the application metrics endpoint. Trace export SHALL be disabled by default and SHALL require both explicit enablement and a configured OTLP endpoint; when disabled, service startup and requests SHALL NOT require a trace backend.

#### Scenario: Request without an incoming correlation identifier

- **WHEN** a request arrives without a request identifier
- **THEN** its response and structured request diagnostic share a generated identifier
- **AND** a startup log without an active span does not fabricate trace identifiers

#### Scenario: Unconfigured trace export

- **WHEN** a service starts with trace export disabled and no endpoint configured
- **THEN** it serves requests without attempting to export traces to a backend

#### Scenario: Explicit trace export

- **WHEN** tracing is enabled with an explicit OTLP endpoint
- **THEN** spans use the configured service identity and are exported to that endpoint

#### Scenario: Trace export enabled without endpoint

- **WHEN** tracing is explicitly enabled without configuring an endpoint
- **THEN** service startup rejects the invalid configuration rather than silently selecting a backend
