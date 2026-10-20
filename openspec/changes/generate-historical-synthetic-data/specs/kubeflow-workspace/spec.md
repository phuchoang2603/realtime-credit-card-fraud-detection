## ADDED Requirements

### Requirement: Historical dataset evidence with shared source access

The consumer SHALL provide a finite historical generator WorkflowTemplate in a prod data namespace watched by the existing shared Argo Workflows controller and a persistent Kubeflow Notebook in the fraud Profile for source validation. The workflow SHALL read its configuration without embedded credentials. The workflow and Notebook SHALL use the same `historical/*`-scoped RustFS identity, stored in Doppler `prd` and synchronized into each namespace as `historical-source`. The shared secret-store chart SHALL provide one ClusterSecretStore serving both namespaces through one Doppler token. The source identity SHALL grant list/read/write access without object deletion, bucket deletion or access to the pipeline artifact bucket. No second Argo controller SHALL be created.

#### Scenario: Generate and inspect a version

- **WHEN** the Profile, shared ClusterSecretStore and controller watching the data namespace are ready, and the historical workflow runs against a new dataset version
- **THEN** the workflow records its run and immutable manifest, and an authenticated prod Notebook reads and verifies the dataset using the same source identity

#### Scenario: Notebook attempts to delete source data

- **WHEN** the notebook identity tries to delete an object in `synthetic-source`
- **THEN** RustFS denies the request without affecting the generated dataset
