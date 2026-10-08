## Why

The prod Talos platform now provides Kubeflow, Dex SSO, External Secrets, and a shared Argo Workflows controller, but this repository has not yet enabled its own Kubeflow Profile. Roll out the ML workspace first, without deploying fraud-service or a dev-only secret store before those applications are ready.

## What Changes

- Add a prod-only Kubeflow Profile and its namespace-scoped Hub Registry, pipeline artifact, and secret resources, using the platform's identity, controllers, and RustFS bucket instead of installing another platform stack.
- Configure the prod Argo CD root to enable the workspace and its required Doppler SecretStore. Keep the fraud-service child disabled during the initial rollout; do not add a dev Kubeflow or secret-store child.
- Keep the generic secret-store chart defaults disabled, retain only the prod override, and remove the unused dev override.
- Document the platform-first reconciliation, credential prerequisites, and checks needed before claiming working Kubeflow workloads.

## Capabilities

### New Capabilities

- `kubeflow-workspace`: A prod-only, consumer-owned Kubeflow Profile with private secrets, artifact configuration, and Hub Registry resources.

### Modified Capabilities

- `talos-gitops-deployment`: Initial Argo CD rendering enables only the prod Kubeflow workspace and its supporting secret store, leaving fraud-service deployment available but disabled until a later rollout.

## Impact

Changes are limited to the repo-owned Argo CD roots, Helm charts, and deployment guide. The platform repo retains ownership of shared Kubeflow, Dex, ESO, Argo Workflows, and ingress. No service APIs change. Enabling this root is an intentional change from the existing fraud-service-first GitOps default; no applications have been deployed yet.
