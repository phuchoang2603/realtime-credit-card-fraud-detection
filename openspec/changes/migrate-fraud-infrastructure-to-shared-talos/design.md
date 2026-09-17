## Context

Deployment sources live in infra/argocd and infra/charts/fraud-service. The internal service exposes ports 8000/8010 and integrates with the shared Victoria platform.

The user confirmed both applications share Talos dev/prod clusters. The marketplace repository declares Victoria backends, collectors, operator, Grafana, and alerting. Its VMAgent selects scrape objects cluster-wide, while VLAgent filters logs to ecommerce. These are repository observations, not a claim that live resources were verified. Its Victoria chart archive is locally patched; this repo will neither copy it nor install that dependency.

This planning work uses read-only source inspection and previously fetched Context7 operator guidance. No local application tests, smoke checks, or live cluster checks are required for planning. Shell entry is avoided during artifact drafting because its formatting hooks can write unrelated files.

## Goals / Non-Goals

**Goals:** Separate application ownership from platform ownership; supply reproducible dev/prod GitOps configuration; preserve internal API access; integrate all three telemetry signals; document current deployment prerequisites.

**Non-Goals:** New payment microservices, public gateway routing, cluster/operator installation, platform ownership transfer, database adoption, Python or model changes, image-tag policy redesign, local tests/smoke checks, and automatic production rollout or destruction.

## Decisions

### 1. Application-owned infrastructure layout

Use infra/argocd/app-of-apps with dev/root.yaml and prod/root.yaml, plus infra/charts/fraud-service containing deployment, service, podscrape, rules, dashboards, values.yaml, and values-prod.yaml. Dashboard JSON lives under the app chart's dashboards/ directory. Keep infra/docker unchanged.

Choose payment-gateway as the application namespace in each workload cluster. Root names are payment-gateway-dev-root and payment-gateway-prod-root; child Application names are payment-gateway-dev-fraud-service and payment-gateway-prod-fraud-service. The Helm release is fraud-service in each separate cluster. Root Applications and their child Application CRs live in argo-cd on the management cluster; children use destination.name dev/prod and namespace payment-gateway. Use this repo's actual remote URL, not a copied marketplace URL.

Default roots track main, with an explicit configurable revision propagated to children for dev branch rollout. Preserve the existing image repository and version-tag policy: do not copy the marketplace's Git-SHA image injection, since this repo's release workflow publishes version tags. Namespace creation is owned by the application synchronization policy, without ownership of monitoring or shared resources. Configure resources and probes explicitly in values; retain existing endpoint/probe semantics in this change.

An umbrella chart containing the platform was rejected because ownership already exists elsewhere. A public Gateway/HTTPRoute is unnecessary for this internal service.

### 2. Shared telemetry contract

Create VMPodScrape in payment-gateway with pod label selection restricted to fraud-service and the http-metrics named port at /metrics. There is no need for all-namespace pod discovery. VMAgent's selection of scrape resources and the scrape's selection of target pods are separate controls; both must match.

Use the shared VictoriaTraces OTLP/gRPC address as environment-specific chart configuration. The marketplace source currently indicates vtsingle-vmks.monitoring.svc.cluster.local:4317; validate the live address during rollout preflight. Use a single configured service identity (default fraud-service), and preserve explicit export disablement for CI. Use gRPC endpoint syntax rather than appending an HTTP /v1/traces path.

Add service, trace_id, and span_id via structured logging processors. Only include trace fields when an active span is valid. Preserve request handling and model logic. Continue stdout collection; do not add a log push exporter or another collector.

Render application-owned dashboard ConfigMaps and VMRule resources in payment-gateway. Define stable fraud-specific dashboard UIDs and use shared data source UIDs as values. Query prediction count, latency histogram, and score distribution using actual exported names/labels; do not copy marketplace HTTP/gRPC RED expressions. Provide a LogsQL panel and trace links using trace_id. Start alerts with unavailable scrape targets scoped to the service and configurable duration; advanced business thresholds are deferred.

### 3. External prerequisite under the current platform owner

Document a concrete integration checklist for ../refurbished-marketplace/infra/charts/observability/values.yaml:

- Extend VLAgent's namespace filter to include payment-gateway while retaining ecommerce and existing exclusions.
- Confirm VMAgent resource selectors include payment-gateway.
- Configure VMAlert/operator rule selection to discover the new VMRule resources.
- Configure Grafana dashboard sidecar namespace/label discovery and RBAC to read application dashboard ConfigMaps; retain existing discovery.
- Confirm shared datasource UIDs, trace endpoint, required CRDs, and network access for scrape/export paths in each cluster.

This change produces the dependency checklist in this repo, not silent edits in the marketplace repo. Missing prerequisites block rollout acceptance and are not addressed by installing a fallback stack. Avoid hard-coded assumptions about selector labels: document and configure the agreed labels in the app chart.

### 4. Chart distribution and CI

The first-party fraud chart needs no upstream dependency. If an upstream dependency becomes necessary, pin it in Chart.yaml/Chart.lock and retain only its downloaded tgz under charts/; no unpacked vendor tree or locally patched archive.

Keep lint-test and helm as independent jobs in the existing CI file. Helm CI validates the app-of-apps chart, root manifests, and fraud chart for dev/default and prod overlays. Include checks for destination names, unique Application identities, matching scrape selectors/ports, valid dashboard JSON, and absence of forbidden platform resources. Where schema validation is used, supply schemas for Argo/Victoria CRs rather than claiming unknown custom resources were validated after skipping them. Telemetry-helper tests run only in CI; no local tests or smoke checks and no Docker builds in CI. CI rendering establishes manifest correctness, not live collector discovery or ingestion.

Update release.yml CHART_FILE and VALUE_FILE paths in the same change. Update formatter exclusions for the new infra Helm templates so the directory move does not expose them to unrelated formatters.

## Risks / Trade-offs

- Shared collector filters can silently omit telemetry → track platform integration as a prerequisite and require evidence from the deployed environment before rollout acceptance.
- Dashboard or rule discovery may be namespace-limited → coordinate selectors and RBAC with the shared owner, not another Grafana/operator installation.
- CI cannot establish live telemetry health → clearly distinguish CI results from rollout evidence; never mark external acceptance complete from rendered manifests alone.
- Existing inference/runtime issues are unrelated → preserve app/model behavior and report blockers separately rather than expanding this change.
