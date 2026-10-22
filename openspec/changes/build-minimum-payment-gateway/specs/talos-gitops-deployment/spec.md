## ADDED Requirements

### Requirement: Gateway workload deployment

The deployment SHALL target Edge, Payments, notifier, vault simulator, processor simulator, Redis and Kafka in `payment-gateway` through the configured prod root, with ClusterIP services. One application chart and Argo CD child SHALL own the five Go services through a services map plus gateway CNPG, Valkey, secrets, routing and security resources; Kafka SHALL retain its separate chart and child. Edge SHALL additionally have one Gateway API HTTPRoute on the cluster ingress gateway exposing only the hosted checkout paths `/checkout/*`. It SHALL NOT restore a dev app-of-apps root. A CloudNativePG cluster owned by this repository SHALL provide separate databases and owner roles for Payments, notifier, vault and processor. Payments SHALL run migrations as its owner role and serve requests with a separate runtime role that has only `SELECT` and `INSERT` on the payment event ledger. No service's application credentials SHALL grant access to another service's database or logical replication. Application encryption keys, caller credentials and transport trust material SHALL come from namespace-scoped External Secrets, not from the repository or an assumed cross-namespace SecretStore. Network policy SHALL default-deny vault/database/Redis access except the required callers, and SHALL allow Edge only from the ingress gateway and the namespaces of configured integrations. Redis SHALL be a standalone `Redis` resource of the platform's Redis operator (no sidecar), with bounded memory, `noeviction`, TTL-based idempotency records, a password from External Secrets and Payments-only access; Postgres remains the authoritative idempotency store. Integration keys and webhook secrets SHALL come from External Secrets. TLS endpoints SHALL use health checks that support their configured transport. A repository-owned Kafka cluster, managed by the platform's Strimzi operator, SHALL run in `payment-gateway` with automatic topic creation disabled and `payments.events.v1` declared, and Kafka Connect SHALL publish only the Payments event table through dedicated CDC credentials granted by a Role scoped to that one Secret, excluding application credentials, private operation requests and vault data. Network policy SHALL admit Kafka clients only from the gateway namespace, and SHALL allow the notifier egress only to configured webhook destinations. Deleting the Kafka application SHALL NOT delete the Strimzi operator or its CRDs.

#### Scenario: Render gateway target

- **WHEN** the configured prod root is rendered
- **THEN** each workload, the database cluster, Kafka, Connect, the connector and the topic have prod-qualified identities, no dev root is created, and the only public route is the Edge HTTPRoute for `/checkout/*`

#### Scenario: Cross-database access

- **WHEN** Payments' database role attempts to read the notifier database
- **THEN** access is denied

#### Scenario: Vault database isolation

- **WHEN** Edge, Payments or processor credentials attempt to access the vault database directly
- **THEN** access is denied and only authenticated permitted vault API calls can obtain their scoped response

#### Scenario: Connector secret access

- **WHEN** the Connect service account requests a Secret other than its dedicated CDC Secret
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
- **THEN** the release workflow publishes independently built images for fraud, edge, Payments, notifier, vault simulator, processor simulator and the Debezium Connect image
- **AND** publication does not depend on manifest version comparison scripts
