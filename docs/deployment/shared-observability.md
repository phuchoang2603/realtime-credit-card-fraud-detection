# Shared observability (interim)

The shared platform owns its telemetry collectors, store and UI; this repository deploys only application workloads. Platform collection of container stdout logs requires no application-specific collector. Fraud emits structured logs with service and request correlation; trace/span IDs appear only when an active span exists.

The fraud Service still exposes Prometheus metrics at `:8010/metrics`, but this chart does not configure collection, application dashboards or alerts. Do not interpret an available metrics endpoint as proof that application metrics appear in ClickStack.

Trace export is disabled by default. To opt in outside the chart, set `TRACING_ENABLED=true` and an explicit `OTEL_EXPORTER_OTLP_ENDPOINT` for an OTLP/gRPC receiver; without both, startup rejects tracing rather than selecting a fallback. The chart sets tracing disabled and does not configure an endpoint. Fraud readiness does not depend on telemetry backend availability. Application metrics, traces, dashboards and alerts are deferred until the minimum gateway skeleton is in place and a separately scoped integration is implemented.
