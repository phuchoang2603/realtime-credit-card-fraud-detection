## Purpose

Provide an isolated prod Kubeflow workspace for fraud-detection ML work while relying on the existing shared platform for authentication, controllers, ingress, and secret delivery.

## Requirements

### Requirement: Prod-only profile ownership

The deployment SHALL create a Kubeflow Profile in prod whose owner is the configured platform login identity. The Profile controller SHALL own creation of the workspace namespace; this repository SHALL NOT install a second Kubeflow control plane or pre-create that namespace.

#### Scenario: Reconcile the workspace

- **WHEN** the prod platform's Profile controller and CRDs are available and the consumer root reconciles
- **THEN** a Profile owned by the platform login identity is created and its namespace becomes available for consumer workloads

#### Scenario: Render dev

- **WHEN** the dev Argo CD root is rendered
- **THEN** it does not render a Kubeflow workspace or a Kubeflow secret-store application

### Requirement: Private workspace integrations

The prod workspace SHALL provide namespace-scoped credentials and configuration for pipeline artifacts and the Hub Registry without placing secret values in Git. It SHALL use the platform's shared workflow controller and artifact storage instead of installing duplicate cluster services.

#### Scenario: Prepare pipeline and registry resources

- **WHEN** the consumer-owned secret store can read its prod credentials and the Profile namespace exists
- **THEN** the namespace has the artifact repository, pipeline launcher, registry resources, and externally supplied secrets required to run those workloads

#### Scenario: Missing secret prerequisite

- **WHEN** the consumer's prod secret-store token or remote credentials are absent
- **THEN** the deployment does not embed replacement credential values in rendered manifests and the dependent workloads are not claimed ready

### Requirement: Shared-platform isolation

Synchronizing or removing the workspace SHALL NOT adopt or remove the platform-owned Kubeflow controllers, Dex, External Secrets operator, Argo Workflows controller, ingress, or platform namespace.

#### Scenario: Remove consumer workspace

- **WHEN** the consumer-owned workspace is removed
- **THEN** shared platform applications and their resources remain owned by the platform
