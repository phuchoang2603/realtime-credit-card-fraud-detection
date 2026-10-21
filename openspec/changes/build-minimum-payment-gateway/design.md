## Context

After `define-payment-and-decision-contracts`, the repository has `payments.v1` types and events, a rules-only `fraud.v2` service, and a health-only edge. The shared platform (talos-proxmox) owns CloudNativePG, External Secrets (Doppler), Strimzi and ClickStack/OTel. The marketplace uses goose migrations, sqlc and pgx, and this repository adopts the same tooling. The next change (live simulator, Kafka, Flink) will tail committed events. This change only has to make them durable and ordered.

## Goals / Non-Goals

**Goals:**

- An API-driven checkout path that produces authoritative, lifecycle-valid events for synthetic traffic.
- Correct idempotency, fail-closed risk handling and unknown-outcome recovery from the first version.
- Service boundaries matching the architecture: Accounts, Payments and processor as separate deployables.

**Non-Goals:**

- Kafka publication, live simulator and transport faults (`simulate-live-traffic-with-transport-faults`).
- Hosted checkout HTML page and browser redirect, webhooks (#40), marketplace integration (#41), public routing and rate limits (#63).
- Financial journal and postings (#36), capture/refund (#58), reconciliation reports (#59).
- Credential rotation, integration disablement flows beyond a flag, dashboards (#66).
- Real bank connectivity, ISO 8583 wire exchange, raw-card vault/CDE, VND-specific policy and issuer-authentication challenges. These are adopted follow-up directions, not implemented by the synthetic sale interface in this slice.

## Decisions

### Request flow

```
simulator / marketplace
      │ HTTPS JSON (integration key | checkout token)
      ▼
   edge ──gRPC──▶ Accounts (authenticate, merchant, customer)
      │
      └──gRPC──▶ Payments ──gRPC──▶ fraud.v2 Decide      (deadline 1 s, fail closed)
                    │     ──gRPC──▶ processor.v1 Sale    (deadline 3 s → unknown)
                    ▼
                 Postgres: events, idempotency, status view
                    ▲
          recovery + expiry workers (same process, leader via advisory lock)
```

The edge resolves identity through Accounts and passes gateway IDs to Payments. Payments never calls Accounts, so its only synchronous dependencies are fraud and the processor.

### Event store in Postgres

The `events` table stores `global_position bigserial`, `event_id uuid unique`, `payment_id`, `aggregate_version` (unique with `payment_id`), `event_type`, `payload bytea` (serialized `PaymentEvent`), `occurred_at` and `recorded_at`. An append runs in one transaction: insert events at `expected_version + 1…`, update the `payment_status` view row, and write the idempotency result. A unique violation on `(payment_id, aggregate_version)` means a stale writer. The event table is the commit-log boundary. In `simulate-live-traffic-with-transport-faults`, Debezium reads committed inserts from the WAL and maps the stored event payload to Kafka, so no separate outbox table or publisher code exists in Payments. `bigserial` positions can commit out of order across concurrent transactions; consumers rely on WAL commit order and per-payment versions, not on positions. Alternatives: EventStoreDB or Kafka as the store. Rejected because they add infrastructure; the chosen authority is the Payments-owned Postgres event stream.

Idempotency scope, request-content binding and in-flight/final results survive process restarts in Postgres. Database uniqueness and transaction semantics arbitrate concurrent requests; a Redis pre-check is not the authority and is unnecessary for the first slice. Later caches or rate limits must not make cache eviction, expiry or outage permit another payment effect.

### Domain aggregate replays golden histories

The aggregate is a pure Go state machine: `Apply(event)` for replay and `Decide(command) → events` for commands. Tests feed every golden history from `contracts/payments/v1/testdata/` through `Apply` and assert accept or reject. That is the same oracle the Python generator uses. If the generator change has not landed, this change adds the golden histories itself.

### Attempt orchestration as sequential commits

`StartAttempt` appends `AttemptStarted`, then calls fraud and appends `RiskDecisionRecorded`. On approve it appends `ProcessorOperationRequested` (the effect intent) and commits it before the processor call. It then calls `Sale` and appends the outcome. Each step is its own append with an expected version, so a crash leaves a state the recovery worker can see. The response returns the attempt's state after the last step. The idempotency record is written with the first append and completed with the final result, so a retry during processing gets the in-flight result, or `PENDING` after the unknown step.

Fail closed on fraud errors records a real `RiskDecisionRecorded` (`DECLINE`, `FRAUD_UNAVAILABLE`, `payments-fallback-v1`), so histories stay lifecycle-valid and the data shows fraud outages honestly.

### Checkout token without storage

`checkout_token = base64url(session_id ‖ expiry ‖ HMAC-SHA256(key, session_id ‖ expiry))`. Payments verifies by recomputing, and repeated creation returns the same token. The key comes from an ExternalSecret. Alternative considered: random tokens stored hashed. Rejected because idempotent creation would then have to return a token it cannot recover.

### Fingerprints at the edge

The edge normalizes (trim, case-fold, collapse whitespace) the full shipping address and computes `HMAC-SHA256(fingerprint_key, normalized)` for the address fingerprint. It does the same for the client IP's /24 (IPv4) or /48 (IPv6) network. The payment-method fingerprint is the HMAC of the synthetic token's card identity. Only fingerprints and country/region/postal code cross into internal services. For synthetic integrations, a `simulated_client` object supplies IP country and network directly. For real integrations, the network comes from the connection address, and the country is absent until a geolocation source is chosen.

Real integrations use the secret key from the ExternalSecret. Synthetic integrations use a separate, non-secret synthetic key in chart values, the same key the historical generator uses. Historical and live fingerprints of one synthetic entity are therefore equal, the generator needs no secret, and synthetic and real fingerprints can never collide. Alternative considered: accepting precomputed fingerprints from synthetic integrations. Rejected because the edge's normalization would then go unexercised for all simulated traffic. Normalization and HMAC cases live in `contracts/synthetic/identity-vectors.json`, which the Go edge and the Python generator both test against.

### Deterministic gateway identities

Accounts derives `merchant_id` as UUIDv5 of `"{integration_id}:merchant:{external_seller_id}"` and `customer_id` as UUIDv5 of `"{integration_id}:customer:{external_buyer_id}"`, under a fixed namespace recorded in the identity vectors. Rows still store creation times and categories. `create-integration --synthetic --id <uuid>` creates a synthetic integration with the ID that the historical dataset's configuration declares. A live simulator continuing that dataset then produces the same gateway IDs without importing mappings. Alternative considered: random IDs plus a mapping import from the dataset. Rejected because it adds an import path and ties Accounts state to a dataset version.

### Synthetic token format

Tokens look like `tok_<card-id>_<behavior>` with behavior `ok`, `nsf`, `issuer` or `invalid`. The card identity drives the payment-method fingerprint, so the same card always has the same fingerprint regardless of behavior. The live simulator chooses behaviors per scenario, for example stolen cards that are often issuer-declined.

### Processor simulator store and faults

The simulator has its own database with a table of operations keyed by `operation_key`, request hash and outcome. Faults (latency distribution, timeout rate) are seeded configuration. A timeout records the outcome and then sleeps past the caller's deadline, which reproduces a lost response realistically. Transport is `processor.v1` gRPC for this synthetic slice. Payments depends on application-owned provider operations, not generated transport types; a transport adapter maps them to the simulator. `Sale` and `GetOperation` describe this simulator's capabilities, not a guarantee that every bank exposes an idempotency key or status query.

### Adopted bank integration and policy direction

The [architecture comparison and bank boundary](../../../docs/architecture/payment-gateway.md#reference-payment-gateway-alignment) adopt the reference gateway's separation of synchronous processing, asynchronous delivery and reconciliation, preserving event-sourced authority. Follow-up increments must meet these gates:

- First prove a synthetic ISO 8583 request/response exchange through a bank adapter and a bank simulator, with a named, versioned fictional profile, byte-level fixtures and no live bank or real card data. The current gRPC simulator remains the minimum slice; it is not an ISO wire test. Keep ISO codecs and connection/session state outside the payment aggregate and fraud API.
- Before real connectivity, obtain the chosen bank/acquirer's integration specification and certification requirements. Pin its ISO edition/profile, MTIs, required/conditional fields, encoding/bitmap rules, framing, transport/session management, security/MAC/key responsibilities, response codes and amount/currency rules. ISO 8583 standardizes messages; it does not prescribe transport or settlement. HTTP-based acquiring interfaces remain a provider-specific alternative.
- Durably correlate the gateway operation key with provider context and transaction identifiers. STAN/RRN alone are not global gateway identities. Specify handling for duplicate, late, malformed and unmatched responses. Recovery must use the profile's supported query, reversal/advice or reconciliation process; never infer that a timed-out financial request is safe to resend because the synthetic `GetOperation` API exists.
- A separate lifecycle increment must distinguish issuer authentication, authorization holds, capture, reversal, refund and settlement before adapting such operations to ISO messages. The current `Sale`/`PaymentSucceeded` contract does not prove settlement, and `fraud.v2` APPROVE does not authenticate a cardholder. Add the required events, API states and caller/generator scenarios together rather than reinterpreting existing payloads.
- Adopt the [Vietnam research](../../../docs/research/vietnam-payment-policy.md) as a source-backed policy-design baseline. Version regulation, provider terms and synthetic policy separately with role/channel/effective dates. Add VND units/validation and authentication success, failure, pending, expiry and cancellation only with coordinated contract changes. Issuer-wide daily totals belong to issuer fixtures/provider evidence, not gateway-only fraud features. Do not ingest real PAN, PIN, OTP or biometric data for these simulations.

Real card collection requires a separately designed vault or hosted tokenization boundary and a cardholder-data environment for components that handle credentials, including a bank connector when applicable. Keep raw credentials and wire dumps out of Payments events, fraud inputs, Kafka and analytical stores. An isolated vault alone does not establish PCI DSS compliance; the synthetic slice makes no compliance or five-nines availability claim.

### Workers inside Payments

Recovery and expiry run as goroutines in Payments, and a Postgres advisory lock elects one active worker across replicas. Both are idempotent because they append with expected versions. A separate deployment is unnecessary at this scale. The architecture's reconciliation worker (#59) remains separate.

### Generated Go bindings per module

Each Go module keeps its own `gen/` with only the packages it uses. `codegen:proto` passes `--go_opt=M…` import-path overrides per consumer, and the protos drop `go_package`. This preserves `GOWORK=off` independent builds with the existing `go-service.Dockerfile`. Alternative: one shared generated module. Rejected because it would need `replace` directives and a multi-module Docker context.

### Database layout

One CNPG `Cluster` (`payment-gateway-pg`, three instances in the configured prod target) with declarative `Database` resources and managed roles `accounts`, `payments`, `processor`. Each role owns only its database, with `CONNECT` revoked from `PUBLIC`. The cluster enables `wal_level: logical` for later CDC; the later publication change must scope replication credentials separately from normal application credentials. Each service's migrations (goose, embedded SQL) run as an init container invoking `<service> migrate`, so a service can only migrate its own schema. Local/integration Postgres supports validation without restoring the retired dev root. Alternative: one cluster per service. Rejected for homelab resource cost; database roles isolate ordinary application access, while cluster administrators and replication credentials remain privileged.

### Libraries

pgx v5, sqlc for queries, goose v3, grpc-go with the health service, client_golang metrics, optional OTel SDK instrumentation without a configured exporter, and slog JSON logging. Application metrics, traces, dashboards and alerts are not integrated with ClickStack in this gateway skeleton; a later change will handle collection and export. This keeps service scaffolding portable without introducing a direct backend dependency.

## Risks / Trade-offs

- [Large change] → Tasks are grouped so contracts plus Accounts, then Payments plus processor, then edge plus deployment, can each land as a PR.
- [Synchronous attempt latency is the sum of fraud and processor] → Deadlines cap it. Unknown outcomes return `PENDING` rather than blocking.
- [Advisory-lock leader election] → If the leader dies its lock is released with the connection, and another replica takes over within one poll interval.
- [Global position gaps and out-of-order commits] → Later publication uses WAL commit order and durable connector offsets, not a `global_position > last_seen` scan.
- [HMAC keys rotate] → Rotation changes fingerprints. Keys are versioned in secret names, and rotation handling belongs to #63.

## Migration Plan

1. Land contracts and Accounts; validate migrations and database-role isolation against local/integration Postgres.
2. Land processor simulator and Payments; validate idempotency, replay and recovery with fraud in the integration environment.
3. Land edge routes; create a synthetic integration with `accounts create-integration --synthetic` and run a scripted local smoke flow (provision merchant, create payment, approve, decline, timeout-then-recover) and record evidence.
4. Render the configured prod root and review namespace-scoped secret prerequisites for `payment-gateway`; the existing workspace SecretStore is in `fraud-ml-prod` and does not cover gateway workloads automatically. Roll out through the normal GitOps process after validation, provision the synthetic key through Doppler, and repeat the smoke flow against the rollout. Rollback: revert applications; retain the database and use schema-compatible images. No dev root is introduced.

## Open Questions

- Payment expiry default (for example 30 minutes) and recovery grace period (for example 10 seconds) can be tuned with the live simulator without changing behavior.
