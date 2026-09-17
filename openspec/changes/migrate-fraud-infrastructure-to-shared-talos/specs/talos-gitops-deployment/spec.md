## Purpose

Deploy fraud-service into the existing Talos environments with independent application ownership.

## ADDED Requirements

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

### Requirement: Environment validation in CI
CI SHALL validate the root configurations, application catalog, and fraud chart for both environments in a separate Helm job, while lint and application tests remain together. These checks SHALL NOT require devenv, Docker builds, or cluster credentials.

#### Scenario: Deployment configuration pull request
- **WHEN** a pull request changes deployment configuration
- **THEN** CI checks chart rendering and environment destinations, including application-specific custom resource structure
- **AND** invalid configuration fails the relevant CI job
