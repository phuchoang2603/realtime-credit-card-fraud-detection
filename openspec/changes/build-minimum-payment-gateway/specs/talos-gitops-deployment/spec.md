## ADDED Requirements

### Requirement: Gateway workload deployment

The deployment SHALL target Edge, Payments, notifier, vault simulator, processor simulator, Redis and Kafka in `payment-gateway` through the configured prod root, with ClusterIP services. One application chart and Argo CD child SHALL own the five Go services through a services map plus gateway CNPG, Valkey, secrets, routing and security resources; Kafka SHALL retain its separate chart and child. Edge SHALL additionally have one Gateway API HTTPRoute on the cluster ingress gateway exposing only the hosted checkout paths `/checkout/*`, and the prod values SHALL publish that gateway host through the platform's Cloudflare ClusterTunnel `talos-proxmox` at `pay-gateway.phuchoang.sbs` so a marketplace buyer can reach the hosted page; the JSON API SHALL remain cluster-internal. It SHALL NOT restore a dev app-of-apps root. A CloudNativePG cluster owned by this repository SHALL provide separate databases and owner roles for Payments, notifier, vault and processor. Payments SHALL run migrations as its owner role and serve requests with a separate runtime role that has only `SELECT` and `INSERT` on the payment event ledger. No service's application credentials SHALL grant access to another service's database or logical replication. Application secrets SHALL come from External Secrets through the shared `fraud-doppler` ClusterSecretStore, whose namespace allow-list includes `payment-gateway` and whose bootstrap token lives in `operators`, not from the repository. Network policy SHALL default-deny vault/database/Redis access except the required callers, and SHALL allow Edge only from the ingress gateway and the namespaces of configured integrations. Redis SHALL be a standalone `Redis` resource of the platform's Redis operator (no sidecar), with bounded memory, `noeviction`, TTL-based idempotency records, a password from External Secrets and Payments-only access; Postgres remains the authoritative idempotency store. Internal gRPC SHALL be plaintext h2c secured by the Istio ambient mesh (STRICT mTLS and principal-based AuthorizationPolicy configured through the chart's `mesh.rules`), and workloads SHALL use native gRPC health probes. A repository-owned Kafka cluster, managed by the platform's Strimzi operator, SHALL run in `payment-gateway` with automatic topic creation disabled and `payments.events.v1` declared, and Kafka Connect SHALL publish only the Payments event table through dedicated CDC credentials granted by a Role scoped to that one Secret, excluding application credentials, private operation requests and vault data. Network policy SHALL admit Kafka clients only from the gateway namespace, and SHALL allow the notifier egress only to configured webhook destinations. Deleting the Kafka application SHALL NOT delete the Strimzi operator or its CRDs.

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

## MODIFIED Requirements

### Requirement: Service quality and automated image release

Every PR and branch push SHALL run independent Python, Go and Helm check jobs
covering Python lint/format/tests, Go lint and tests for every module, and Helm
validation, using the checked-in protobuf bindings. One explicit job per service
image SHALL reuse one image workflow. Pull-request runs SHALL build without
publishing; every branch push SHALL publish SHA-tagged images to GHCR, and a
default-branch push SHALL also tag `latest`. After a branch push publishes, one
release job SHALL rewrite the prod charts' single image tag (gateway, fraud
service and Kafka Connect) to that commit's short SHA and push it with
`[skip ci]`, so Argo CD deploys the exact images the branch produced without
anyone editing tags.

#### Scenario: Pull request or branch push

- **WHEN** a PR is opened or updated
- **THEN** checks validate all services and image jobs build every service image without publication

#### Scenario: Merged service changes

- **WHEN** changes reach main
- **THEN** the release workflow publishes independently built images for fraud, edge, Payments, notifier, vault simulator, processor simulator and the Debezium Connect image, tagged with the commit SHA and `latest`
- **AND** publication does not depend on manifest version comparison scripts

#### Scenario: Non-default branch push

- **WHEN** changes are pushed to a non-default branch
- **THEN** the release workflow publishes SHA-tagged images for those services without moving `latest`

#### Scenario: Pinned image tag

- **WHEN** a branch push finishes publishing its images
- **THEN** the prod chart tag equals that commit's short SHA and Argo CD rolls each workload to that image without a manual tagging commit, and the pin commit's `[skip ci]` marker prevents a second release run
