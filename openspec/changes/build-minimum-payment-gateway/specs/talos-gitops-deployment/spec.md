## ADDED Requirements

### Requirement: Gateway workload deployment

The deployment SHALL target Edge, Payments, notifier, vault simulator, processor simulator, Redis and Kafka in `payment-gateway` through the configured prod root, with ClusterIP services.

#### Scenario: Complete gateway workload deployment contract

- **WHEN** the `Gateway workload deployment` capability is configured or exercised
- **THEN** The deployment SHALL target Edge, Payments, notifier, vault simulator, processor simulator, Redis and Kafka in `payment-gateway` through the configured prod root, with ClusterIP services. One application chart and Argo CD child SHALL own the five Go services through a services map plus gateway CNPG, Valkey, secrets, routing and security resources; Kafka SHALL retain its separate chart and child. Edge SHALL additionally have one Gateway API HTTPRoute on the cluster ingress gateway exposing only the hosted checkout paths `/checkout/*`, and the prod values SHALL publish that gateway host through the platform's Cloudflare ClusterTunnel `talos-proxmox` at `pay-gateway.phuchoang.sbs` so a marketplace buyer can reach the hosted page; the JSON API SHALL remain cluster-internal. It SHALL NOT restore a dev app-of-apps root. A CloudNativePG cluster owned by this repository SHALL provide separate databases and owner roles for Payments, notifier, vault and processor. Payments SHALL run migrations as its owner role and serve requests with a separate runtime role that has only `SELECT` and `INSERT` on the payment event ledger. No service's application credentials SHALL grant access to another service's database or logical replication. Application secrets SHALL come from External Secrets through the shared `fraud-doppler` ClusterSecretStore, whose namespace allow-list includes `payment-gateway` and whose bootstrap token lives in `operators`, not from the repository. Network policy SHALL default-deny vault/database/Redis access except the required callers, and SHALL allow Edge only from the ingress gateway and the namespaces of configured integrations. Redis SHALL be a standalone `Redis` resource of the platform's Redis operator (no sidecar), with bounded memory, `noeviction`, TTL-based idempotency records, a password from External Secrets and Payments-only access; Postgres remains the authoritative idempotency store. Internal gRPC SHALL be plaintext h2c secured by the Istio ambient mesh (STRICT mTLS and principal-based AuthorizationPolicy configured through the chart's `mesh.rules`), and workloads SHALL use native gRPC health probes. A repository-owned Kafka cluster, managed by the platform's Strimzi operator, SHALL run in `payment-gateway` with automatic topic creation disabled and `payments.events.v1` declared, and Kafka Connect SHALL publish only the Payments event table through dedicated CDC credentials granted by a Role scoped to that one Secret, excluding application credentials, private operation requests and vault data. Network policy SHALL admit Kafka clients only from the gateway namespace, and SHALL allow the notifier egress only to configured webhook destinations. Deleting the Kafka application SHALL NOT delete the Strimzi operator or its CRDs.

#### Scenario: Render gateway target

- **WHEN** the configured prod root is rendered
- **THEN** each workload, the database cluster, Kafka, Connect, the connector and the topic have prod-qualified identities, no dev root is created, and the only public surface is the Edge `/checkout/*` route, published through the Cloudflare tunnel

#### Scenario: Cross-database access

- **WHEN** Payments' database role attempts to read the notifier database
- **THEN** access is denied

#### Scenario: Vault database isolation

- **WHEN** Edge, Payments or processor credentials attempt to access the vault database directly
- **THEN** access is denied and only mesh-authorized vault API calls on their own port can obtain their scoped response

#### Scenario: Connector secret access

- **WHEN** the Connect service account requests a Secret other than its dedicated CDC Secret
- **THEN** access is denied

### Requirement: Separate PR and main image tags

Same-repository PRs SHALL publish every service image with `pr-<number>-<full-sha>` using the built merge commit. Fork PRs SHALL build without publication. Main pushes SHALL publish `main-<full-sha>` and `latest`; feature-branch pushes SHALL NOT publish images.

#### Scenario: Publish PR images

- **WHEN** a same-repository PR is opened or updated
- **THEN** all service images are published with that PR's tag and no production image pin changes

#### Scenario: Merged service changes

- **WHEN** changes reach main
- **THEN** fraud, edge, Payments, notifier, vault simulator, processor simulator and Debezium Connect images are built and published with the main commit tag and `latest`, without manifest version comparison scripts

### Requirement: Production image pinning

Only after every main service image has published successfully, one CI job SHALL pin the gateway, fraud-service and Kafka production chart tags to that commit's `main-<full-sha>`. PR and feature-branch runs SHALL NOT update production pins. The pin commit SHALL use `[skip ci]`, and CI SHALL ignore image-only production-values commits to prevent rebuild loops. No image-updater controller SHALL be required.

#### Scenario: New main image

- **WHEN** every service image for a main commit has published successfully
- **THEN** one CI job commits the same main tag to all three production charts and Argo CD reconciles them

#### Scenario: PR image is newer

- **WHEN** a PR image is newer than the latest main image
- **THEN** production pins remain unchanged

### Requirement: PR image cleanup

On PR closure, cleanup SHALL cancel that PR's active image build and delete repository-linked GHCR versions carrying that PR's `pr-<number>-<full-sha>` tag, including versions with additional tags. Versions without a matching tag SHALL be retained. Cleanup SHALL NOT inspect production chart pins or require a service list.

#### Scenario: Shared package version

- **WHEN** a version has a closing PR tag and also a main, latest or another PR tag
- **THEN** cleanup deletes the version and all its tags

## MODIFIED Requirements

### Requirement: Service quality and automated image release

Every PR and main push SHALL run independent Python, Go and Helm check jobs covering Python lint/format/tests, Go lint and tests for every module, and Helm validation, using the checked-in protobuf bindings. Each service image SHALL use the shared image workflow.

#### Scenario: Validate a pull request

- **WHEN** a PR is opened or updated
- **THEN** independent checks validate all services and the image matrix builds every service image
