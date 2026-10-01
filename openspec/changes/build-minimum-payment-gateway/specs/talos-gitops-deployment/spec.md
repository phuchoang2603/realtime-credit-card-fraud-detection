## ADDED Requirements

### Requirement: Gateway workload deployment

The deployment SHALL run Accounts, Payments, the processor simulator and the edge in `payment-gateway` for dev and prod, each as its own Argo CD child application with ClusterIP services only. A CloudNativePG cluster owned by this repository SHALL provide separate databases and owner roles for Accounts, Payments and the processor simulator. No service's credentials SHALL grant access to another service's database. Application secrets SHALL come from External Secrets, not from the repository.

#### Scenario: Render gateway environments

- **WHEN** dev and prod configurations are rendered
- **THEN** each workload and the database cluster have environment-qualified identities, and no ingress or public route is created

#### Scenario: Cross-database access

- **WHEN** Payments' database role attempts to read the Accounts database
- **THEN** access is denied

## MODIFIED Requirements

### Requirement: Service quality and automated image release

Every PR and main push SHALL run independent Python, Go and Helm check jobs
covering Python lint/format/tests, Go lint and tests for every module, and Helm
validation, using the checked-in protobuf bindings. One explicit job per service
image SHALL reuse one image workflow. PRs SHALL build without publishing; main
pushes SHALL publish SHA/latest images to GHCR.

#### Scenario: Pull request or branch push

- **WHEN** a PR is opened or updated
- **THEN** checks validate all services and image jobs build every service image without publication

#### Scenario: Merged service changes

- **WHEN** changes reach main
- **THEN** the release workflow publishes independently built images for fraud, edge, Accounts, Payments and the processor simulator
- **AND** publication does not depend on manifest version comparison scripts
