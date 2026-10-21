## ADDED Requirements

### Requirement: Gateway workload deployment

The deployment SHALL target Accounts, Payments, vault simulator, processor simulator, Redis and Edge in `payment-gateway` through the configured prod root, each as its own Argo CD child application with ClusterIP services only. It SHALL NOT restore a dev app-of-apps root. A CloudNativePG cluster owned by this repository SHALL provide separate databases and owner roles for Accounts, Payments, vault and processor. No service's application credentials SHALL grant access to another service's database or logical replication. Application encryption keys, caller credentials and transport trust material SHALL come from namespace-scoped External Secrets, not from the repository or an assumed cross-namespace SecretStore. Network policy SHALL default-deny vault/database/Redis access except the required callers. Redis SHALL use a pinned upstream image, bounded memory, TTL-based disposable entries and Payments-only authenticated access; it SHALL NOT contain authoritative payment state. TLS endpoints SHALL use health checks that support their configured transport.

#### Scenario: Render gateway target

- **WHEN** the configured prod root is rendered
- **THEN** each workload and the database cluster have prod-qualified identities, no dev root is created, and no ingress or public route is created

#### Scenario: Cross-database access

- **WHEN** Payments' database role attempts to read the Accounts database
- **THEN** access is denied

#### Scenario: Vault database isolation

- **WHEN** Edge, Payments or processor credentials attempt to access the vault database directly
- **THEN** access is denied and only authenticated permitted vault API calls can obtain their scoped response

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
- **THEN** the release workflow publishes independently built images for fraud, edge, Accounts, Payments, vault simulator and processor simulator
- **AND** publication does not depend on manifest version comparison scripts
