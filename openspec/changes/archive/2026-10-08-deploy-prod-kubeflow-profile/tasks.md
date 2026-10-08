## 1. Prod-Only Argo Catalog

- [x] 1.1 Remove the pending dev secret-store child and its unused `values-dev.yaml`, explicitly disable fraud-service in both roots, and verify rendered dev has no children while prod has only the workspace and prod secret-store children.
- [x] 1.2 Retain disabled generic secret-store defaults and the prod Doppler `prd` override; verify `helm lint` succeeds and `helm template` produces no SecretStore with defaults but one in `fraud-ml-prod` with the prod values file.

## 2. Consumer Kubeflow Workspace

- [x] 2.1 Finish the pending workspace chart, use the platform-served Profile API and matching Dex owner without a pre-created namespace, and verify `helm template` renders one Profile and no Namespace or shared controllers.
- [x] 2.2 Verify the workspace renders the Hub Registry, prod artifact and KFP configuration, and ExternalSecrets referring to the prod-only consumer SecretStore without secret literals; run chart lint and inspect the rendered manifests against the platform's contracts.
- [x] 2.3 Keep retry support for the prod child Applications while the platform Profile namespace appears asynchronously; verify the rendered children contain the expected sync retry and prod destinations.

## 3. Guide And Validation

- [x] 3.1 Update the existing GitOps deployment guide for prod-only Kubeflow, local Argo CD roots, Doppler prerequisites, experimental storage, and platform-first reconciliation; verify it no longer claims dev secret-store or fraud-service is enabled.
- [x] 3.2 Run OpenSpec strict validation and the repo's Helm checks for the changed charts and rendered Argo catalogs; verify only the intended prod children are enabled and report any checks that require a live cluster separately.
