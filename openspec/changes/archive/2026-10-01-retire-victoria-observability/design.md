## Context

See `proposal.md` for the platform transition. The fraud chart currently renders `VMPodScrape`, `VMRule`, and a Grafana dashboard ConfigMap; its base and prod values supply a VictoriaTraces endpoint, and the Python exporter repeats that endpoint as a fallback. The platform now owns OTel agents in both environments and the prod ClickStack store. Its annotation-based metrics scraping and OTLP receiver are available, but application wiring is deliberately not part of this removal.

The `define-payment-and-decision-contracts` and `build-minimum-payment-gateway` changes are planned but unimplemented. They contain dashboard and VictoriaTraces assumptions that would restore the deleted integration if left unchanged.

## Goals / Non-Goals

**Goals:**

- Render the fraud chart without resources requiring absent observability CRDs or a Grafana sidecar.
- Keep logging and metrics instrumentation portable and preserve readiness independent of telemetry availability.
- Make disabled tracing safe by default; allow an explicitly configured exporter for opt-in local or future use without introducing a new destination now.
- Keep the in-flight plans aligned before their implementation begins.

**Non-Goals:**

- Do not ship pod scrape annotations, ClickHouse/HyperDX assets, new alert rules, or OTLP deployment configuration in this phase.
- Do not access ClickHouse or prod's authenticated ingest endpoint from an application; the platform agent owns cross-environment export.
- Do not remove the metrics listener, structured stdout logging, or bounded lifecycle support.

## Decisions

1. **Delete platform-specific chart resources outright.** Remove the three templates (`podscrape.yaml`, `rules.yaml`, `dashboards.yaml`) and the dashboard JSON; remove the `podScrape`, `rules`, and `dashboards` value/schema branches. Keep the `telemetry.serviceName` setting and port `8010`. This is preferable to shipping disabled Victoria templates, which would preserve obsolete CRD and Grafana configuration that later changes might accidentally enable.
2. **Use explicit trace-export opt-in.** Remove the Victoria endpoint from base/prod values and from Python exporter defaults. Default `TRACING_ENABLED` to false for the fraud runtime; when enabled, require `OTEL_EXPORTER_OTLP_ENDPOINT` before starting an exporter. Preserve the existing injectable exporter used by isolated instances and keep export/shutdown bounded. Removing the endpoint alone is insufficient: the Python OTLP gRPC exporter otherwise defaults to localhost:4317, which would produce another silent dead destination. An always-on export to `otel-agent.observability.svc:4317` is deferred by request.
3. **Preserve the right contracts, not the Victoria capability.** Retire all five requirements in `shared-victoria-observability`. Carry log correlation, service identity, endpoint-optional trace behavior and metrics availability into `service-conventions`; update `talos-gitops-deployment` to describe shared ownership without obsolete platform names. The legacy capability should not survive as a promise of active collection or dashboard availability.
4. **Rebase unimplemented plans rather than adding compatibility.** Remove the `shared-victoria-observability` delta from `define-payment-and-decision-contracts`, drop its dashboard work, and reconcile its proposal, design, tasks, and `service-conventions` operational-identity delta against the new baseline. Update `build-minimum-payment-gateway` design so it does not export directly to VictoriaTraces or require application ClickStack integration for the skeleton. Its vendor-neutral logs/metrics/tracing scaffolding can remain, with export deferred. Validate all affected changes after editing their planning artifacts.
5. **Replace current guidance, retain historical evidence only as historical.** Replace the Victoria-specific `shared-observability.md` checklist with a concise interim account of stdout logs, exposed but uncollected application metrics, and disabled trace export. Update GitOps and architecture text/diagram. Remove dead links and screenshots presented as a live Victoria/Grafana view; historical evidence can remain only where clearly labeled as such.

## Risks / Trade-offs

- [Fraud traces and app metrics temporarily absent from ClickStack] → Document the gap explicitly; platform container logs remain collected, and a separate post-skeleton integration will re-enable app signals.
- [Prior planned work recreates deleted resources] → Reconcile both in-flight changes in this removal change and revalidate their specs before implementing either.
- [Tracing configuration changes break existing local use] → Require an explicit endpoint when opted in and verify disabled/default and enabled/configured startup cases; keep readiness independent of backend reachability.
- [Deletion is disruptive to a previously deployed fraud chart] → Do not roll back to Victoria resources on rebuilt clusters; if rollback is required, restore only a compatible application image/chart without relying on removed platform CRDs.

## Migration Plan

1. Remove the Victoria chart resources, default endpoints and obsolete documentation; keep the fraud app's logs and metrics endpoint intact.
2. Reconcile the two pending change plans against the retired capability, validate chart rendering and OpenSpec, and land this change before either pending implementation.
3. Treat ClickStack logs as platform-collected, and record application metrics, traces, dashboards and alerting as not integrated yet. Introduce the app-to-agent integration in a separate change after the gateway skeleton exists.
