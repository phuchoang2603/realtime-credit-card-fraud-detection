# Shared Victoria observability

Fraud-service reuses the VictoriaMetrics operator, VMAgent, VLAgent, VictoriaMetrics, VictoriaLogs, VictoriaTraces, Grafana, and alerting owned by refurbished-marketplace. This repository owns only application resources in `payment-gateway`; it never installs a fallback stack.

The source contract is `../refurbished-marketplace/infra/charts/observability/values.yaml`, plus that chart's production overlay. The defaults below were checked against repository configuration, **not live clusters**. Missing dependencies block rollout acceptance independently for dev and prod. CI rendering cannot prove discovery, ingestion, or network access.

## Application configuration

Configure each environment in `infra/charts/fraud-service/values.yaml` or `values-prod.yaml`:

| Value | Repository default | Contract |
|---|---|---|
| `telemetry.serviceName` | `fraud-service` | One `OTEL_SERVICE_NAME` for JSON `service`, trace resource identity, and metric resource identity |
| `telemetry.traceEndpoint` | `vtsingle-vmks.monitoring.svc.cluster.local:4317` | Direct plaintext OTLP/gRPC host:port; no HTTP `/v1/traces` suffix |
| `podScrape.labels` | `release: victoria` | Must match the platform VMAgent's resource selector; this label is not proof of discovery |
| `rules.labels` | `{}` | Set labels required by the shared VMAlert `ruleSelector` |
| `rules.unavailableFor` | `10m` | Alert on failed scrapes (`up == 0`) or missing series (`absent`), scoped to namespace and job |
| `dashboards.labels` | `grafana_dashboard: "1"` | Must match the Grafana dashboard sidecar label selector |
| `dashboards.datasources.metrics` | `VictoriaMetrics` | Existing Prometheus-compatible datasource UID |
| `dashboards.datasources.logs` | `VictoriaLogs` | Existing `victoriametrics-logs-datasource` UID |
| `dashboards.datasources.traces` | `VictoriaTraces` | Existing Tempo-compatible datasource UID |

Service identities and datasource UIDs are restricted by the chart values schema so they can be substituted safely into dashboard JSON, query strings, and links. Set telemetry through `telemetry.*`; do not duplicate those environment variables in `env`.

VMPodScrape discovers pods only in its own namespace using `app.kubernetes.io/name: fraud-service` and `payment-gateway/release: fraud-service`. It uses `jobLabel: app.kubernetes.io/name`, explicitly retains `namespace: payment-gateway` on samples, and scrapes `http-metrics` at `/metrics` on port 8010. VMAgent resource discovery and VMPodScrape pod discovery are separate selectors. Do not remove either application selector or rewrite its namespace/job labels in the shared collector.

Dashboard queries use `predictions_total` (with `is_fraud`), `prediction_latency_seconds_bucket`, and `fraud_prediction_score_bucket`, filtered by `namespace` and `job`. JSON is in the chart's `dashboards/` directory. The LogsQL panel filters `kubernetes.pod_namespace` and the structured `service` field. The request trace table links the selected `trace_id` to both VictoriaTraces and related VictoriaLogs records.

## Platform-owner integration checklist

The **platform owner** maintains refurbished-marketplace's observability values, operator configuration, shared collector/RBAC/network configuration, and Grafana datasources. The **application owner** maintains this chart, environment overrides, and application rollout evidence. Every Pending cell below must be replaced with an observed result, date, and evidence link for that environment before rollout acceptance.

| Dependency and concrete acceptance criterion | Owner | Dev | Prod |
|---|---|---|---|
| Management cluster has `applications.argoproj.io`; Argo registers destination names `dev` and `prod` with permission to create app resources and `payment-gateway`. Root and child CRs remain in management `argo-cd`. | Platform | Pending | Pending |
| Workload clusters have `vmpodscrapes.operator.victoriametrics.com` and `vmrules.operator.victoriametrics.com`, with `operator.victoriametrics.com/v1beta1` served. Existing VMAgent/VMAlert and their operator are healthy. This chart owns no CRDs. | Platform | Pending | Pending |
| Under `victoria-metrics-k8s-stack.vmagent.spec`, retain existing `selectAllByDefault: true`, `podScrapeNamespaceSelector`, and `podScrapeSelector` behavior or explicitly include payment-gateway and `podScrape.labels`. Operator/watch RBAC and VMAgent pod discovery can read the namespace. Confirm the fraud target appears and is `up=1`. | Platform + application | Pending | Pending |
| Under `victoria-metrics-k8s-stack.vlagent.spec.k8sCollector.excludeFilter`, include both ecommerce and payment-gateway while retaining current container exclusions. Confirm application JSON fields `service`, `trace_id`, `span_id`, `level`, and `event` are parsed as fields, and `kubernetes.pod_namespace` is retained. Use `event` as the message if the collector supports message-field selection. A request log must be queryable by its trace ID. | Platform | Pending | Pending |
| VMAlert `ruleNamespaceSelector` and `ruleSelector`, operator watch scope, and RBAC include the payment-gateway VMRule and configured `rules.labels`, while preserving existing rule discovery. Confirm `FraudServiceMetricsUnavailable` is loaded and evaluated. | Platform | Pending | Pending |
| Grafana `sidecar.dashboards.searchNamespace`, `label`, and `labelValue` include payment-gateway's configured dashboard labels while retaining existing namespaces. Its service account can get/list/watch those ConfigMaps. Confirm dashboard UID `fraud-service` loads. | Platform | Pending | Pending |
| Confirm the three datasource UIDs above or configure environment overrides. VictoriaLogs plugin is installed; VictoriaTraces uses Tempo at `/select/tempo` (repository source uses port 10428). Verify metric, LogsQL, and trace queries each return this application's data. | Platform + application | Pending | Pending |
| Preserve/configure VictoriaLogs derived field `trace_id` with `matcherType: label`, `matcherRegex: trace_id`, `url: '${__value.raw}'`, and `datasourceUid: VictoriaTraces` (or override UID). In the Tempo datasource, use `tracesToLogsV2.customQuery: true` and LogsQL `trace_id:="${__trace.traceId}"`, with automatic Loki trace/span filters disabled. Verify navigation matches the same trace ID in both directions. | Platform | Pending | Pending |
| Confirm `telemetry.traceEndpoint` resolves in each cluster, VTSingle listens on plaintext OTLP/gRPC 4317, and a request span arrives with the configured service name. | Platform + application | Pending | Pending |
| Allow VMAgent in monitoring to reach fraud pod TCP 8010; allow fraud pods to reach cluster DNS (UDP/TCP 53) and VictoriaTraces TCP 4317. Shared Grafana/VMAlert/collectors must retain access to their backend query/ingestion endpoints. Confirm Cilium/network policy permits these paths; no public ingress is needed. | Platform | Pending | Pending |
| Deploy the app, confirm internal HTTP 8000/model behavior, collect a prediction, observe the counter/histogram samples and matching log/span, and verify alert/ConfigMap discovery. Record per-environment acceptance evidence. | Application | Pending | Pending |

A replacement for the current log exclusion expression, to be reviewed and applied **in the platform repository by its owner**, is:

```text
not (kubernetes.pod_namespace:=ecommerce or kubernetes.pod_namespace:=payment-gateway) or kubernetes.container_name:=wait-for-db or kubernetes.container_name:=wait-for-mongo or kubernetes.container_name:=wait-for-meili
```

For explicit namespace selectors, include both existing namespaces and `payment-gateway`, rather than replacing the existing list. `rules.labels` and `dashboards.labels` configure discovery only; they do not transfer ownership of platform resources.

## Reference screenshots

These historical screenshots illustrate the views; they are not rollout evidence for this migration.

![Application metrics dashboard](images/app-metrics.png)
![Cluster metrics dashboard](images/cluster-metrics.png)
![Logs view](images/logs.png)
![Traces view](images/traces.png)
