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

The deployment SHALL provide internal gRPC access on port 8000 and metrics on port 8010, preserve the configured model path and prediction rules, and SHALL NOT provision public ingress or TLS controllers. Liveness SHALL use the standard gRPC health service named `liveness`; readiness SHALL use `readiness` and SHALL fail when the model is unavailable.

#### Scenario: Internal deployment

- **WHEN** fraud-service is deployed with either environment configuration
- **THEN** it is exposed through a ClusterIP service and the existing model configuration is supplied
- **AND** no public ingress, cert-manager, or Traefik resources are created

#### Scenario: Missing model in a running process

- **WHEN** fraud-service starts without a usable model
- **THEN** its liveness probe succeeds and its readiness probe fails
- **AND** it does not qualify as a ready prediction endpoint

### Requirement: Shared platform ownership isolation

Synchronizing or deleting this repository's applications SHALL NOT create, adopt, or delete shared observability infrastructure, its CRDs, or its monitoring namespace.

#### Scenario: Remove a payment-gateway application

- **WHEN** its application-owned resources are removed
- **THEN** shared Victoria backends, collectors, operators, Grafana, and alerting remain under the existing platform owner's control

### Requirement: Service quality and automated image release

Every PR and main push SHALL run independent Python, Go and Helm check jobs
covering Python lint/format/tests, Go format/vet/build and Helm validation, using
the checked-in protobuf bindings. Two explicit service jobs SHALL reuse one image
workflow. PRs SHALL build without publishing; main pushes SHALL publish SHA/latest
images to GHCR.

#### Scenario: Pull request or branch push

- **WHEN** a PR is opened or updated
- **THEN** checks validate both services and image jobs build both without publication

#### Scenario: Merged service changes

- **WHEN** changes reach main
- **THEN** the release workflow publishes independently built fraud and edge images
- **AND** publication does not depend on manifest version comparison scripts
