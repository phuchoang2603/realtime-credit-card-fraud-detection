# Shared Victoria observability

Fraud-service reuses the VictoriaMetrics operator, VMAgent, VLAgent, VictoriaMetrics, VictoriaLogs, VictoriaTraces, Grafana, and alerting owned by refurbished-marketplace. This repository deploys only application resources in `payment-gateway`.

Before dev or prod rollout, the platform owner must include `payment-gateway` in VLAgent's namespace filter, confirm VMAgent selects `VMPodScrape` objects, allow VMRule and dashboard ConfigMap discovery, and verify shared datasource UIDs, Victoria CRDs, OTLP endpoint, and network policy. Current endpoint and selectors remain pending until checked on each cluster.

## Application metrics

![Application metrics dashboard](images/app-metrics.png)

## Cluster metrics

![Cluster metrics dashboard](images/cluster-metrics.png)

## Logs

![Logs view](images/logs.png)

## Traces

![Traces view](images/traces.png)
