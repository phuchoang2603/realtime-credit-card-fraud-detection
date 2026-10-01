## Why

`talos-proxmox` removed VictoriaMetrics and Grafana and now owns a ClickStack/OTel platform. This repository still deploys Victoria CRDs and a Grafana dashboard and points fraud traces at the deleted VictoriaTraces service; remove those assumptions before implementing the pending payment contracts and minimum gateway.

## What Changes

- **BREAKING** Retire the application-owned `VMPodScrape`, `VMRule`, Grafana dashboard ConfigMap/JSON, their chart options, and the Victoria-specific rollout checklist. No replacement dashboard, alert, or scrape integration is installed in this phase.
- Remove the VictoriaTraces fallback from the fraud runtime and chart. Keep vendor-neutral structured logs, correlation, and the metrics listener, but make trace export opt-in with an explicitly configured endpoint. Until later integration, traces and application metrics are not promised in ClickStack; platform collection of container logs needs no application-specific collector.
- Update the current deployment and architecture guidance to state the interim observability limits and shared-platform ownership without referencing removed CRDs or services.
- Reconcile the unimplemented `define-payment-and-decision-contracts` and `build-minimum-payment-gateway` plans so neither reintroduces Victoria or requires a Grafana dashboard. Defer application ClickStack metrics/traces/alerts/dashboard work until the minimum gateway skeleton is in place.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `shared-victoria-observability`: Remove its obsolete platform-specific requirements; retain portable logging and metrics expectations in the existing service and deployment capabilities rather than promising a replacement integration now.
- `service-conventions`: Preserve structured logs and bounded telemetry lifecycle without requiring direct export to the former shared backend; traces require explicit opt-in configuration.
- `talos-gitops-deployment`: Keep application deployments independent of the shared platform, without referring to the old monitoring namespace or Victoria/Grafana components.

## Impact

- Fraud Helm chart templates, dashboard asset, values and schema; Python tracing configuration; existing architecture/deployment documentation and affected OpenSpec planning artifacts.
- No ClickHouse client, new collector, direct prod ingest credential, new service integration, or platform-repo change. Existing metrics and logs remain available to the application; there is a deliberate gap in application metric scraping, trace export and app-owned alerting until follow-up work.
