# GitOps deployment

The fraud service targets shared Talos `dev` and `prod` clusters through Argo CD. [talos-proxmox](https://github.com/phuchoang2603/talos-proxmox/blob/main/apps/README.md) owns cluster infrastructure, Argo CD and the Victoria operators/backends. This repository owns payment-gateway application resources.

Argo roots are [`infra/argocd/dev/root.yaml`](../../infra/argocd/dev/root.yaml) and [`infra/argocd/prod/root.yaml`](../../infra/argocd/prod/root.yaml). Both roots and their child Application CRs reside in the management cluster's `argo-cd` namespace. Children target registered cluster names `dev`/`prod` and namespace `payment-gateway`.

| Environment | Root                        | Child Application                    | Helm release    | Values                                       |
| ----------- | --------------------------- | ------------------------------------ | --------------- | -------------------------------------------- |
| Dev         | `payment-gateway-dev-root`  | `payment-gateway-dev-fraud-service`  | `fraud-service` | Chart defaults (`latest`, pulled always)     |
| Prod        | `payment-gateway-prod-root` | `payment-gateway-prod-fraud-service` | `fraud-service` | `values-prod.yaml` (currently also `latest`) |

The child template renders each `valuesFile` entry separately using its `$values/` repository reference. It explicitly sets the Helm release name, so the internal Service is `fraud-service` in both separate workload clusters. Pod and scrape selectors use the application-owned `payment-gateway/release` label instead of Argo's `app.kubernetes.io/instance` tracking label; Argo may change the latter without breaking selectors.

## Revisions and dev branches

Each root defines `targetRevision: &revision main` and reuses `*revision` in `spec.source.helm.valuesObject.global.targetRevision`. For a dev branch rollout, change the anchored value in `infra/argocd/dev/root.yaml` to the branch or commit SHA. YAML resolves the same revision for the root catalog and child chart/value sources. Commit the chart changes on that branch, then submit the updated root to management Argo CD using the normal GitOps process. Changing only the live root's `targetRevision` through the Argo UI does not update the embedded child revision: update both fields together or use the anchored source file. Restore the anchor to `main` when the branch rollout ends.

Dev follows the moving GHCR `latest` image and always pulls it. Production tracks `main` and should pin an immutable commit-SHA tag in `values-prod.yaml` once a promotion policy exists; the release workflow publishes both `latest` and short-SHA tags on every `main` push.

## Runtime configuration

Both services validate configuration at startup and exit non-zero with a sanitized diagnostic on invalid values. Boolean switches accept only `true` or `false` (case-insensitive).

| Service | Variable                           | Default                                                  | Purpose                                                          |
| ------- | ---------------------------------- | -------------------------------------------------------- | ---------------------------------------------------------------- |
| fraud   | `GRPC_PORT`                        | `8000`                                                   | Internal gRPC listener                                           |
| fraud   | `METRICS_PORT` / `METRICS_ENABLED` | `8010` / `true`                                          | Prometheus listener                                              |
| fraud   | `MODEL_PATH`                       | bundled `models/model.pkl`                               | Model artifact; a missing model fails readiness, not liveness    |
| fraud   | `INFERENCE_WORKERS`                | `4`                                                      | Bounded native inference capacity                                |
| fraud   | `GRACEFUL_SHUTDOWN_TIMEOUT`        | `30`                                                     | RPC drain seconds; a watchdog forces exit 5 s later              |
| fraud   | `TRACING_ENABLED`                  | `true`                                                   | Instance-owned OpenTelemetry provider                            |
| fraud   | `OTEL_SERVICE_NAME`                | `fraud-service`                                          | Identity for logs, metrics and traces                            |
| fraud   | `OTEL_EXPORTER_OTLP_ENDPOINT`      | `http://vtsingle-vmks.monitoring.svc.cluster.local:4317` | OTLP/gRPC collector; the scheme is explicit (`https://` for TLS) |
| edge    | `EDGE_HTTP_ADDR`                   | `:8080`                                                  | Public HTTP listener                                             |
| edge    | `FRAUD_GRPC_TARGET`                | `dns:///localhost:8000`                                  | Fraud service target; use the cluster Service DNS name           |
| edge    | `PREDICTION_TIMEOUT`               | `3s`                                                     | Per-request RPC deadline; also inherits client cancellation      |
| edge    | `EDGE_SHUTDOWN_TIMEOUT`            | `10s`                                                    | HTTP drain budget                                                |

The chart sets `terminationGracePeriodSeconds: 40` to cover the 30 s drain, bounded telemetry cleanup and the forced-exit margin.

## Internal access and rollout prerequisites

After choosing the intended workload cluster context, inspect the service with:

```bash
kubectl --context <workload-context> -n payment-gateway port-forward svc/fraud-service 8000:8000 8010:8010
```

The service is a `ClusterIP` exposing gRPC 8000 and HTTP metrics 8010. Kubernetes 1.27+ probes the standard gRPC health service using the `liveness` and `readiness` names. The fraud chart creates no Gateway, HTTPRoute or public ingress. This Deployment/ClusterIP layout is current; the planned target replaces it with a cluster-local Knative Service and a KServe predictor ([#70](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/70)). Public HTTP traffic belongs at the Go edge; edge deployment remains part of gateway delivery. Pair the gRPC image and chart in a rollout or rollback; the old HTTP image cannot satisfy gRPC probes. Use an immutable published image SHA for a real rollout, replacing the development `latest` default.

Complete the owner-specific [shared observability checklist](shared-observability.md) independently for dev and prod before accepting rollout. Missing discovery, CRDs, datasource configuration or network access blocks acceptance; this repo does not add fallback shared infrastructure. Image publication is described in [CONTRIBUTING](../../CONTRIBUTING.md#ci-and-images).

![Historical Argo CD application dashboard](images/argocd.png)
