## REMOVED Requirements

### Requirement: Application-scoped metrics discovery

**Reason**: The shared Victoria collector and its scrape CRD no longer exist; replacement application discovery is deferred.
**Migration**: Keep the application metrics listener and port under `talos-gitops-deployment`; a later change can arrange collection through the platform's telemetry agent.

### Requirement: Direct trace export

**Reason**: VictoriaTraces no longer exists, and direct export to it cannot be a default or rollout prerequisite.
**Migration**: Keep optional OTLP trace export with an explicit endpoint under `service-conventions`; do not configure an endpoint until the follow-up integration.

### Requirement: Correlated structured logs

**Reason**: Log identity and correlation remain useful but are not a Victoria-specific capability.
**Migration**: Retain the behavior under `service-conventions` without requiring a Victoria collector or new application log exporter.

### Requirement: Fraud dashboards and alerts

**Reason**: The Grafana discovery path and Victoria alert-rule CRD no longer exist.
**Migration**: Remove these resources; defer application dashboards and alerts until the gateway skeleton and later ClickStack integration.

### Requirement: Explicit platform integration prerequisites

**Reason**: The listed CRDs, trace backend, selectors, and network prerequisites describe the removed platform.
**Migration**: Document the interim telemetry limitations and retain platform ownership isolation under `talos-gitops-deployment`.
