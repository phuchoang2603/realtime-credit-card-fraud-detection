## Purpose

Deploy fraud-service into the existing Talos environments with independent application ownership.

## Requirements

### Requirement: Environment-specific deployment ownership
The deployment configuration SHALL define distinct payment-gateway dev and prod roots in the management cluster, targeting the registered dev and prod workload clusters respectively. Workloads SHALL reside in the payment-gateway namespace, and application identities SHALL not collide with marketplace applications.

#### Scenario: Render both environments
- **WHEN** the dev and prod configurations are rendered
- **THEN** each root and child application has a distinct environment-qualified identity and its child targets only the corresponding workload cluster
- **AND** application resources remain within payment-gateway

### Requirement: Internal fraud-service contract
The deployment SHALL provide internal HTTP access on port 8000 and metrics on port 8010, preserve the configured model path and existing API behavior, and SHALL NOT provision public ingress or TLS controllers.

#### Scenario: Internal deployment
- **WHEN** fraud-service is deployed with either environment configuration
- **THEN** it is exposed through a ClusterIP service and the existing model configuration is supplied
- **AND** no public ingress, cert-manager, or Traefik resources are created

### Requirement: Shared platform ownership isolation
Synchronizing or deleting this repository's applications SHALL NOT create, adopt, or delete shared observability infrastructure, its CRDs, or its monitoring namespace.

#### Scenario: Remove a payment-gateway application
- **WHEN** its application-owned resources are removed
- **THEN** shared Victoria backends, collectors, operators, Grafana, and alerting remain under the existing platform owner's control

### Requirement: Lean service CI pipeline
CI SHALL run a focused `lint-test` job for the Python fraud service covering linting, formatting, and unit/integration tests with coverage gating. CI SHALL NOT require Helm validation jobs, custom manifest scripts, devenv, Docker builds, or cluster credentials.

#### Scenario: Service pull request or push
- **WHEN** changes are pushed to a branch or pull request
- **THEN** CI executes Ruff lint/format checks and pytest with code coverage validation
- **AND** failures in service quality checks fail the workflow
