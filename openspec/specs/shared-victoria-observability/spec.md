## Purpose

Make fraud-service metrics, logs, traces, dashboards, and alerts available through the existing Victoria observability platform without duplicating that platform.

## Requirements

### Requirement: Application-scoped metrics discovery
Fraud-service SHALL expose its existing metrics at port 8010 and provide a scrape resource targeting only its own pods. Discovery SHALL use the existing platform collector.

#### Scenario: Platform collector discovers the application
- **WHEN** shared collector selection includes payment-gateway and fraud-service pods are available
- **THEN** their metrics endpoint is scraped using matching pod labels and the named metrics port
- **AND** unrelated workloads are not selected by the fraud scrape resource

### Requirement: Direct trace export
Fraud-service SHALL export OTLP traces to a configurable shared VictoriaTraces endpoint without an Alloy dependency and SHALL use a configurable, consistent service identity.

#### Scenario: A request produces spans
- **WHEN** tracing is enabled and the shared trace endpoint is configured
- **THEN** spans are exported directly to that endpoint with the configured service identity

#### Scenario: Telemetry export is disabled
- **WHEN** tracing is explicitly disabled for CI tests
- **THEN** request handling does not require a running trace backend

### Requirement: Correlated structured logs
Application logs SHALL remain structured JSON and include the service identity. Logs emitted with a valid active span SHALL include trace_id and span_id; logs without an active span SHALL NOT claim a fabricated trace identity.

#### Scenario: Request log correlation
- **WHEN** a log is emitted within an active request span
- **THEN** the JSON record includes the same trace and span identifiers as that span

#### Scenario: Startup logging
- **WHEN** a log is emitted before any request span exists
- **THEN** the JSON record includes the service identity without false trace or span identifiers

### Requirement: Fraud dashboards and alerts
Application-owned dashboards SHALL show prediction rate, prediction latency, fraud-score distribution, and logs using the existing shared data sources. Trace/log navigation SHALL use matching trace identifiers. Application-owned alert rules SHALL include an unavailable scrape target condition with configurable timing.

#### Scenario: Shared discovery is configured
- **WHEN** Grafana and alerting select the application's dashboard and rule resources
- **THEN** fraud panels and alert rules are available without deploying a new Grafana or alerting instance
- **AND** dashboard queries use the actual exported fraud metric names and application labels

### Requirement: Explicit platform integration prerequisites
Deployment documentation SHALL state the required CRDs, trace endpoint, namespace discovery, dashboard/rule selectors, and network access. Missing shared dependencies SHALL block rollout acceptance rather than trigger a fallback installation.

#### Scenario: Log collection excludes payment-gateway
- **WHEN** the shared log collector still accepts only marketplace namespaces
- **THEN** rollout acceptance remains pending until its existing owner includes payment-gateway
- **AND** this repository does not install a second collector
