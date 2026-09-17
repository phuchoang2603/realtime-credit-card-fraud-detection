## Why

Fraud-service runs on the same Talos dev/prod clusters as refurbished-marketplace, where Victoria infrastructure has a shared owner. Application-scoped configuration keeps ownership clear and avoids duplicate platform resources.

## What Changes

- **BREAKING (deployment configuration):** Replace deployments/ with infra/argocd/ and infra/charts/fraud-service/, with distinct payment-gateway dev/prod Argo roots and an internal service in the payment-gateway namespace.
- Reuse platform-owned Victoria operators, collectors, backends, Grafana, and alerting. Add application-owned VMPodScrape, VMRule, and dashboard resources.
- Export traces directly to the shared VictoriaTraces endpoint and correlate structured logs with service, trace, and span identifiers.
- Track shared-stack namespace discovery changes in refurbished-marketplace as external rollout prerequisites; do not install or adopt shared resources in this repo.
- Remove legacy Alloy/k8s-monitoring, Loki, Tempo, kube-prometheus-stack, cert-manager, Traefik, GKE Terraform, and any remaining Compose configuration from this repository.
- Update Helm CI, release chart paths, and deployment documentation. Upstream dependencies, if introduced, use pinned downloaded tgz archives rather than expanded source trees.

## Capabilities

### New Capabilities

- `talos-gitops-deployment`: Environment-specific Argo deployment, application ownership, internal service access, and shared platform integration.
- `shared-victoria-observability`: Application telemetry, dashboards, and alerts integrated with the existing shared Victoria stack.

### Modified Capabilities

None; the main spec inventory is empty.

## Impact

Affects deployments/, infra/, .github/workflows/ci.yml and release.yml, service telemetry helpers, README, and deployment documentation. Application endpoints, inference logic, model bytes, Python/runtime choices, and image naming/version policy remain outside this change.

The shared stack owner must enable discovery for payment-gateway in both environments. Edits to that repository and live rollout are coordinated external work, not implied by implementing this repo's files. No local tests or smoke checks, CI Docker builds, cluster provisioning, public ingress, or automatic destruction of existing infrastructure are included.
