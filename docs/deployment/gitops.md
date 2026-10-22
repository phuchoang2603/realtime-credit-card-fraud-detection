# GitOps deployment

GitOps manages the prod-only Kubeflow workspace and the minimum payment gateway. [talos-proxmox](https://github.com/phuchoang2603/talos-proxmox/blob/main/docs/operations/kubeflow.md) owns the shared Talos clusters, Argo CD, Kubeflow controllers, Dex SSO, External Secrets, Argo Workflows, and ingress. This repository owns its Profile, gateway workloads and service policies.

The Argo root is [`infra/argocd/prod/root.yaml`](../../infra/argocd/prod/root.yaml). Its children target the prod cluster. Gateway workloads use `payment-gateway`; Profile resources use `argo-cd` as their Argo destination so Argo does not create the Profile-owned `fraud-ml-prod` namespace first.

| Environment | Root                        | Enabled child Application                   | Helm release         | Configuration                                       |
| ----------- | --------------------------- | ------------------------------------------- | -------------------- | --------------------------------------------------- |
| Prod        | `payment-gateway-prod-root` | `payment-gateway-prod-secret-store`         | `secret-store`       | `values-prod.yaml`: Doppler `prd` in `fraud-ml-prod` |
| Prod        | `payment-gateway-prod-root` | `payment-gateway-prod-kubeflow-workspace`   | `kubeflow-workspace` | Profile and namespace resources in `fraud-ml-prod`  |

Gateway children are `payment-gateway-prod-{gateway-platform,gateway-database,gateway-redis,kafka,fraud-service,edge,payments,notifier,vault-simulator,processor-simulator}`. The CNPG cluster owns four databases with separate roles; Payments uses an owner for migrations, `payments_app` at runtime, and a separate replication principal restricted to the event ledger. Redis is one operator-managed standalone Valkey CR with noeviction and a password; only Payments is admitted. Kafka/Connect publish ledger bytes to the internal `payments.events.v1` topic. Only Edge `/checkout/*` is publicly routed; JSON APIs stay ClusterIP-only.

The child template renders each `valuesFile` entry separately using its `$values/` repository reference and explicitly sets the Helm release name. The fraud-service internal Service is `fraud-service` in the target cluster's `payment-gateway` namespace. Its Pod and Service selectors use the application-owned `payment-gateway/release` label instead of Argo's `app.kubernetes.io/instance` tracking label.

The workspace chart owns the `fraud-ml-prod` Kubeflow Profile (with the same email identity as platform Dex), the namespace-scoped Hub Registry and its database ExternalSecret, RustFS artifact credentials via ExternalSecret, a KFP launcher, and an Argo artifact repository. The generic `secret-store` chart is disabled by default and enabled only through its prod values file. Its token has read-only access to the `payment-gateway-fraud-detection` Doppler project's `prd` config, which holds `KUBEFLOW_REGISTRY_DB_PASSWORD`, `KUBEFLOW_RUSTFS_ACCESS_KEY`, and `KUBEFLOW_RUSTFS_SECRET_KEY`. No secret values are committed to Git. Reconcile the platform first: its Profile controller creates the namespace with ambient mesh enabled. The prod store and workspace Applications retry while the platform CRDs and Profile namespace become available. Kubernetes-native Pipelines are experimental, and the Hub Registry database uses local storage without HA; do not use this workspace with valuable production data until end-to-end validation, backup, and restore are in place.

Once the Profile namespace exists, provision Kubernetes Secret `fraud-doppler-token` with key `dopplerToken` in `fraud-ml-prod`, using `ESO_DOPPLER_TOKEN` from the consumer Doppler `prd` config. Do not commit or print the token. The prod SecretStore and workspace ExternalSecrets can then reconcile; re-provision after namespace recreation or token rotation. The RustFS access key currently matches the shared platform's bucket-scoped identity, so separate Doppler ownership does not yet imply separate S3 access; provision a distinct RustFS identity before requiring storage isolation. Verify Profile ownership, ExternalSecret readiness, authenticated Dashboard access, and a real pipeline artifact upload and download before calling the workspace operational.

## Revisions and branch rollouts

Each root uses an anchored `targetRevision` (normally `main`) and reuses `*revision` in `spec.source.helm.valuesObject.global.targetRevision`. For a branch rollout, change the anchored value in the root for that environment to the branch or commit SHA. YAML resolves the same revision for the root catalog and child chart/value sources. Commit the chart changes on that branch, then submit the updated root to that environment's Argo CD using the normal GitOps process. Changing only the live root's `targetRevision` through the Argo UI does not update the embedded child revision: update both fields together or use the anchored source file. The gateway root temporarily targets `feat/build-minimum-payment-gateway` for verification and must stay there during verification; restore it to `main` before merge.

Production should pin an immutable commit-SHA tag in `values-prod.yaml` once a promotion policy exists; the release workflow publishes both `latest` and short-SHA tags on every `main` push. A supported `workflow_dispatch` with `service=all` publishes short-SHA images from the selected branch; it does not update `latest` on a feature branch. Verify the matching SHA images exist and pin every service and Connect before applying the root. PR image builds never publish.

## Runtime configuration

All services validate configuration at startup and exit non-zero with a sanitized diagnostic on invalid values. Boolean switches accept only `true` or `false` (case-insensitive).

| Service | Variable                           | Default                                                  | Purpose                                                          |
| ------- | ---------------------------------- | -------------------------------------------------------- | ---------------------------------------------------------------- |
| fraud   | `GRPC_PORT`                        | `8000`                                                   | Internal gRPC listener                                           |
| fraud   | `METRICS_PORT` / `METRICS_ENABLED` | `8010` / `true`                                          | Prometheus listener                                              |
| fraud   | `GRACEFUL_SHUTDOWN_TIMEOUT`        | `30`                                                     | RPC drain seconds; shutdown watchdog bounds process exit              |
| fraud   | `TRACING_ENABLED`                  | `false`                                                  | Opt-in instance-owned OpenTelemetry provider                     |
| fraud   | `OTEL_SERVICE_NAME`                | `fraud-service`                                          | Identity for logs, metrics and traces                            |
| fraud   | `OTEL_EXPORTER_OTLP_ENDPOINT`      | unset                                                    | Required only when tracing is enabled; explicit OTLP/gRPC target |
| edge    | `EDGE_HTTP_ADDR`                   | `:8080`                                                  | Public HTTP listener                                             |
| edge    | `EDGE_SHUTDOWN_TIMEOUT`            | `10s`                                                    | HTTP drain budget                                                |
| edge | `EDGE_INTEGRATIONS_FILE`, `EDGE_PAYMENTS_ADDR`, `EDGE_VAULT_ADDRESS` | required / chart values | Hashed integration config and internal dependencies |
| payments | `PAYMENTS_DATABASE_URL`, `PAYMENTS_MIGRATE_DATABASE_URL` | required / owner fallback | Separate runtime and migration identities |
| payments | `PAYMENTS_REDIS_ADDR`, `PAYMENTS_REDIS_PASSWORD_FILE`, `PAYMENTS_IDEMPOTENCY_TTL` | chart values / `24h` | First idempotency check; DB fallback within a 50 ms Redis deadline |
| payments | `PAYMENTS_ATTEMPT_BUDGET`, `PAYMENTS_FRAUD_TIMEOUT`, `PAYMENTS_PROCESSOR_TIMEOUT` | `5s`, `1s`, `3s` | Bounded attempt and dependency budgets |
| notifier | `NOTIFIER_DATABASE_URL`, `NOTIFIER_INTEGRATIONS_FILE`, `NOTIFIER_KAFKA_BROKERS` | required / chart values | Durable delivery state, webhook subscriptions and internal event feed |
| vault | `VAULT_DATABASE_URL`, `VAULT_ENCRYPTION_KEY_FILE`, `VAULT_CALLERS_FILE` | required | Vault-owned storage/key and role-specific caller credentials |
| processor | `PROCESSOR_DATABASE_URL`, `VAULT_ADDRESS`, `VAULT_CA_FILE`, `VAULT_CREDENTIAL_FILE` | required | Processor-owned store and authenticated vault resolution |
| processor | `PROCESSOR_LATENCY_MIN`, `PROCESSOR_LATENCY_MAX`, `PROCESSOR_TIMEOUT_RATE` | prod `20ms`, `150ms`, `0` | Nominal simulator latency and optional injected lost responses |

The fraud chart sets `terminationGracePeriodSeconds: 40` to cover the 30 s drain, bounded telemetry cleanup and the forced-exit margin.

## Internal access and rollout prerequisites

Choose the intended workload cluster context and inspect the service with:

```bash
kubectl --context <workload-context> -n payment-gateway port-forward svc/fraud-service 8000:8000 8010:8010
```

The fraud-service chart creates a `ClusterIP` exposing gRPC 8000 and HTTP metrics 8010. Kubernetes 1.27+ probes the standard gRPC health service using the `liveness` and `readiness` names. The chart creates no Gateway, HTTPRoute or public ingress. Production rollout acceptance remains outstanding; a later serving increment replaces this layout with a cluster-local Knative Service and a KServe predictor (Knative and KServe serving). Public HTTP traffic belongs at Edge. Gateway Go/fraud pods enroll individually in the existing Istio ambient mesh with strict mTLS and service-account caller policies; vault role permissions additionally use application TLS and separate bearer credentials. Pair the gRPC image and chart in a rollout or rollback; the old HTTP image cannot satisfy gRPC probes. Use matching published short-SHA image tags for a real rollout, replacing the development `latest` defaults.

The [shared observability guide](shared-observability.md) describes the interim limits: platform-collected stdout logs, an exposed but uncollected fraud metrics endpoint, and disabled trace export. No application dashboard or alert is installed; telemetry integration is not a rollout prerequisite. Image publication is defined by the [release workflow](../../.github/workflows/release.yml).

## Gateway secrets and local checks

Gateway ExternalSecrets use the namespace-scoped `gateway-doppler` SecretStore in `payment-gateway`, reading Doppler `payment-gateway-fraud-detection/prd`. Its bootstrap Secret is `gateway-doppler-token`, key `dopplerToken`, sourced through the platform secret-provisioning process from `ESO_DOPPLER_TOKEN`; the workspace's namespace-scoped store does not cover it. Do not hand-create application secrets. Required Doppler keys are:

- `GATEWAY_DB_PAYMENTS_PASSWORD`, `GATEWAY_DB_PAYMENTS_APP_PASSWORD`, `GATEWAY_DB_PAYMENTS_CDC_PASSWORD`, `GATEWAY_DB_NOTIFIER_PASSWORD`, `GATEWAY_DB_VAULT_PASSWORD`, `GATEWAY_DB_PROCESSOR_PASSWORD`.
- `GATEWAY_REDIS_PASSWORD`, `GATEWAY_PAYMENTS_ENCRYPTION_KEY`, `GATEWAY_CHECKOUT_TOKEN_KEY`, `GATEWAY_VAULT_ENCRYPTION_KEY`, `GATEWAY_FINGERPRINT_KEY`, `GATEWAY_CHECKOUT_COOKIE_KEY`.
- `GATEWAY_VAULT_EDGE_CREDENTIAL`, `GATEWAY_VAULT_PROCESSOR_CREDENTIAL`, `GATEWAY_INTEGRATIONS_JSON`.
- `GATEWAY_INTERNAL_CA_CERT`, `GATEWAY_VAULT_TLS_CERT`, `GATEWAY_VAULT_TLS_KEY`, `GATEWAY_PAYMENTS_TLS_CERT`, `GATEWAY_PAYMENTS_TLS_KEY`. Certificates must cover their service's `*.payment-gateway.svc.cluster.local` DNS name; keys and PEM material stay in Doppler/ExternalSecrets.

`GATEWAY_INTEGRATIONS_JSON` is an array containing `integration_id` (UUID), `name`, `api_key_sha256` (hex SHA-256), `synthetic`, and optional `webhook: {url, secret, events}`. The synthetic ID must match the historical dataset; `EDGE_SYNTHETIC_FINGERPRINT_KEY=synthetic-fingerprint-v1` is non-secret. Real integrations use the secret fingerprint key. The marketplace receiver is `http://web.ecommerce.svc.cluster.local:8080/webhooks/payment-gateway`, with subscriptions `attempt.declined`, `payment.succeeded`, `payment.expired`. Keep notifier egress destinations in its chart values aligned with configured webhook URLs; HTTPS endpoints require corresponding explicit destination rules.

Enter `devenv shell`, which generates bindings and sqlc queries. For a relevant module, use `cd src/<module>; GOWORK=off go test -short ./...; GOWORK=off golangci-lint run ./...`. Full `go test -race -count=1 ./...` starts module-local testcontainers-go dependencies for PostgreSQL, Valkey and Kafka as needed, both locally and in CI. Docker is required unless `-short` is set; there are no `TEST_*` overrides, silent integration skips, shared test module or CI service containers. `bash scripts/ci-checks.sh go` discovers and checks every module; CI owns the full check/image sweep. End-to-end smoke, access, outage and load checks run only on the prod Kubernetes cluster after GitOps rollout, without manually running services or databases in Docker.

## Gateway verification

On 2026-10-22, fraud lint/format and 13 tests, simulator lint/format and 24 tests, protobuf compilation with `protoc -I contracts`, strict OpenSpec validation and the chart lint/render sweep passed. All five Go modules build and pass short tests and lint. Payments' serial race/integration suite passes, including PostgreSQL ledger/role checks, idempotency, recovery and readiness; a separate test verifies Redis restart loss falls back to Postgres without another effect. All 166 generated events in the historical sample, including expiry, parse as protobuf JSON after aligning its presence marker. The prod root renders 12 prod-qualified children on the feature revision. All 64 local documentation links resolve; the shared-platform links were corrected, while the existing ISO standards reference returns HTTP 403 to automated requests. Notifier and simulator integration checks are recorded below once complete.

The [draft gateway PR](https://github.com/phuchoang2603/payment-gateway-fraud-detection/pull/82) remains unmerged. A supported `service=all` [branch image run](https://github.com/phuchoang2603/payment-gateway-fraud-detection/actions/runs/39735415369) succeeded for `fac4b51` before deployment was deferred. Later runtime changes have not been published. The user deferred Kubernetes deployment on 2026-10-22: do not apply the root, sync Argo, or run cluster checks until reauthorized.

A read-only Doppler key-name check found all 20 `GATEWAY_*` prerequisites above absent from `prd`; `ESO_DOPPLER_TOKEN` exists. The `payment-gateway` namespace and gateway Argo root were absent at inspection. No application secret was created and no secret value was printed. Production deployment is deferred by the user; no root/manifests were applied and no cluster resources were changed. Smoke, access, CDC restart/outage and nominal 50-attempt/s for 10 minutes plus failure/recovery load evidence remain unverified. Unit/integration results do not establish the monthly availability SLO or PCI DSS compliance. The [task checklist](../../openspec/changes/build-minimum-payment-gateway/tasks.md) tracks the remaining acceptance gates.
