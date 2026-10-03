# Shared observability (interim)

The shared platform owns its telemetry collectors, store and UI; this repository deploys only application workloads. Platform collection of container stdout logs requires no application-specific collector. Fraud emits structured logs with service and request correlation; trace/span IDs appear only when an active span exists.

The fraud service exposes Prometheus metrics at `:8010/metrics`: `fraud_decisions_total{outcome,reason}` counts decisions by outcome and reason (`NONE` when no rule matches), and `fraud_decision_latency_seconds` measures rule evaluation time. These replace the retired prediction metrics. The chart does not configure collection, application dashboards or alerts; an available endpoint does not prove that ClickStack receives the metrics.

Trace export is disabled by default. To opt in outside the chart, set `TRACING_ENABLED=true` and an explicit `OTEL_EXPORTER_OTLP_ENDPOINT` for an OTLP/gRPC receiver; without both, startup rejects tracing rather than selecting a fallback. The chart sets tracing disabled and does not configure an endpoint. Fraud readiness does not depend on telemetry backend availability. Metrics collection, trace export, dashboards and alerts are deferred until the minimum gateway skeleton is in place and a separately scoped integration is implemented.
