## 1. Remove obsolete deployment resources

- [x] 1.1 Delete `podscrape.yaml`, `rules.yaml`, `dashboards.yaml` and the fraud Grafana dashboard JSON; remove their `podScrape`, `rules` and `dashboards` keys from chart defaults/schema; verify `helm lint` and `helm template` pass for base and prod values and neither output contains Victoria CRDs or a Grafana dashboard ConfigMap.
- [x] 1.2 Remove `traceEndpoint` from base/prod chart values and schema, keep the service identity and metrics port, and render `TRACING_ENABLED=false` for the interim fraud deployment; verify the base/prod rendered environments contain no Victoria endpoint and still expose metrics on port 8010.

## 2. Make tracing opt-in

- [x] 2.1 Remove the VictoriaTraces exporter fallback and default tracing to disabled in the Python fraud runtime; reject explicit tracing enablement without an OTLP endpoint while preserving injected exporters and bounded shutdown; verify focused tests cover disabled startup, missing-endpoint rejection and enabled export with an explicit endpoint.
- [x] 2.2 Run the relevant fraud Python checks and render both chart variants to verify application readiness/metrics remain independent of telemetry availability and no implicit localhost/old-backend export is introduced.

## 3. Update guidance and planned work

- [x] 3.1 Replace the Victoria checklist in `docs/deployment/shared-observability.md` with the short interim contract, update `docs/deployment/gitops.md` and `docs/architecture/payment-gateway.md`, and remove or clearly label stale Grafana/Victoria screenshots; verify current deployment docs describe logs, uncollected application metrics and opt-in traces without claiming app dashboards or alerts.
- [x] 3.2 Reconcile `define-payment-and-decision-contracts` proposal, design, specs and tasks: remove the now-invalid `shared-victoria-observability` delta and dashboard task, retain documented new fraud metric names, and merge the revised operational-identity requirement into its `service-conventions` delta; verify `openspec validate define-payment-and-decision-contracts --strict` passes.
- [x] 3.3 Reconcile `build-minimum-payment-gateway` design and any affected tasks so service scaffolding retains portable logging/metrics without direct VictoriaTraces export or premature ClickStack dashboard/alert integration; verify `openspec validate build-minimum-payment-gateway --strict` passes and its deployment plan remains limited to gateway workloads.

## 4. Validate the removal boundary

- [x] 4.1 Run `openspec validate retire-victoria-observability --strict`, fraud chart lint/template for both overlays, and focused Python checks; inspect active (non-archived) charts, docs and in-flight plans for obsolete Victoria/Grafana references, allowing only historical references clearly marked as such. Confirm ClickStack app integration remains deferred.
