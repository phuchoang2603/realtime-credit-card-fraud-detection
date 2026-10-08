## Context

The sibling platform repo already manages prod Kubeflow, Dex/OAuth2 Proxy, ESO, shared Argo Workflows, and ingress. It expects the consumer to create `fraud-ml-prod` via a Profile, then provide its registry, RustFS, and KFP namespace resources. This repository already has uncommitted candidate Helm charts and root changes; implementation must refine that work rather than replace it. See proposal.md for the rollout motivation.

## Goals / Non-Goals

**Goals:** Keep Profile and sensitive workspace dependencies consumer-owned, make the prod root render only the minimum required children, and keep undeployed dev and fraud-service configuration available for later rollout.

**Non-Goals:** Deploy the shared platform, introduce Kubeflow on dev, ship fraud-service or edge workloads now, or declare end-to-end ML readiness from a successful Helm render alone.

## Decisions

- Use the existing `infra/charts/kubeflow-workspace` candidate as a prod-only release for the cluster-scoped Profile plus explicit `fraud-ml-prod` resources. Use the platform-served `kubeflow.org/v1` Profile API. Its owner matches the platform Dex identity; the Profile controller creates the namespace, never Argo CD `CreateNamespace` at that destination. Avoid copying platform Kubeflow resources or deploying a second controller.
- Retain the generic secret-store chart with `values.yaml` disabled by default and only `values-prod.yaml` enabling `fraud-doppler` in the Profile namespace. Keep it a separate child from the workspace because it needs a separately supplied, namespace-local Doppler token, and workspace ExternalSecrets reference this store. Remove the dev-only values file and dev store child rather than preparing an unused environment.
- Override `fraud-service.enabled: false` in the dev and prod roots; leave the default fraud-service chart and the root's default catalog intact for later explicit rollout. The dev root consequently produces no child Applications. Prod enables only secret-store and workspace. Keep shared app-of-apps retry for the two prod children because the Profile namespace is created asynchronously after platform CRDs and controller become available.
- Target prod's local Kubernetes server because the prod Argo CD has no registered `prod` cluster alias. Keep the workspace Helm release's Argo destination in existing `argo-cd` while the resources specify `fraud-ml-prod`; this avoids an Argo-managed namespace appearing before the Profile controller runs. Namespace-scoped resources may need retries until the namespace exists. The AppProject permits local namespaces and the cluster-scoped Profile.
- Keep public SSO and ingress entirely platform-owned. Workspace ExternalSecrets fetch the consumer project's registry password and RustFS credentials, and its artifact ConfigMaps use the platform bucket and shared Argo controller. Update the existing GitOps guide to reflect the environment-local Argo CD roots and this limited initial rollout rather than implying fraud-service or dev secret-store is deployed.

## Risks / Trade-offs

- [Platform CRDs or Profile namespace not yet ready] -> Reconcile platform first; use child sync retries and verify Profile namespace before assessing workspace health.
- [Missing consumer Doppler token or credentials] -> Document the namespace-local `fraud-doppler-token` prerequisite and verify ExternalSecrets; never embed secrets in Git.
- [Hub Registry database uses an ephemeral upstream storage configuration] -> Treat the workspace as experimental and require durable storage and backups before valuable model metadata is stored.
- [Disable flag mistaken for stopping an already deployed child] -> Nothing has been deployed yet; if the rollout state changes, check existing Argo applications before relying on a disabled catalog entry.

## Migration Plan

No deployed consumer resources require migration. Finish the pending repo-owned manifests, reconcile the prod platform first, supply the consumer's prod Doppler token and credentials, then submit the consumer prod root. Validate Helm render, Profile and namespace creation, ExternalSecrets, authenticated access, and a real pipeline artifact round trip. To defer rollout before submission, do not apply the prod root; if deployed later, use an explicit consumer-owned cleanup plan rather than deleting platform resources.
