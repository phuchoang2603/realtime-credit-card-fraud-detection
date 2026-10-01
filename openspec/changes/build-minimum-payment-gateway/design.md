## Context

After `define-payment-and-decision-contracts`, the repository has `payments.v1` types and events, a rules-only `fraud.v2` service, and a health-only edge. The shared platform (talos-proxmox) owns CloudNativePG, External Secrets (Doppler), Strimzi and ClickStack/OTel. The marketplace uses goose migrations, sqlc and pgx, and this repository adopts the same tooling. The next change (live simulator, Kafka, Flink) will tail committed events. This change only has to make them durable and ordered.

## Goals / Non-Goals

**Goals:**

- An API-driven checkout path that produces authoritative, lifecycle-valid events for synthetic traffic.
- Correct idempotency, fail-closed risk handling and unknown-outcome recovery from the first version.
- Service boundaries matching the architecture: Accounts, Payments and processor as separate deployables.

**Non-Goals:**

- Kafka publication, outbox relay, live simulator (next change).
- Hosted checkout HTML page and browser redirect, webhooks (#40), marketplace integration (#41), public routing and rate limits (#63).
- Financial journal and postings (#36), capture/refund (#58), reconciliation reports (#59).
- Credential rotation, integration disablement flows beyond a flag, dashboards (#66).

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

The `events` table stores `global_position bigserial`, `event_id uuid unique`, `payment_id`, `aggregate_version` (unique with `payment_id`), `event_type`, `payload bytea` (serialized `PaymentEvent`), `occurred_at` and `recorded_at`. An append runs in one transaction: insert events at `expected_version + 1…`, update the `payment_status` view row, and write the idempotency result. A unique violation on `(payment_id, aggregate_version)` means a stale writer. The event table with its global position is the commit-log boundary. The next change's relay tails `global_position`, so no separate outbox table exists yet. Alternatives: EventStoreDB or Kafka as the store. Rejected because they add infrastructure, and a broker is not an authoritative store (payment-flows).

`bigserial` can commit out of order across concurrent transactions. The relay will read only below a safe watermark (for example, positions committed before the oldest in-flight transaction via `pg_snapshot_xmin`). That relay design is recorded here so this change's schema supports it.

### Domain aggregate replays golden histories

The aggregate is a pure Go state machine: `Apply(event)` for replay and `Decide(command) → events` for commands. Tests feed every golden history from `contracts/payments/v1/testdata/` through `Apply` and assert accept or reject. That is the same oracle the Python generator uses. If the generator change has not landed, this change adds the golden histories itself.

### Attempt orchestration as sequential commits

`StartAttempt` appends `AttemptStarted`, then calls fraud and appends `RiskDecisionRecorded`. On approve it appends `ProcessorOperationRequested` (the effect intent) and commits it before the processor call. It then calls `Sale` and appends the outcome. Each step is its own append with an expected version, so a crash leaves a state the recovery worker can see. The response returns the attempt's state after the last step. The idempotency record is written with the first append and completed with the final result, so a retry during processing gets the in-flight result, or `PENDING` after the unknown step.

Fail closed on fraud errors records a real `RiskDecisionRecorded` (`DECLINE`, `FRAUD_UNAVAILABLE`, `payments-fallback-v1`), so histories stay lifecycle-valid and the data shows fraud outages honestly.

### Checkout token without storage

`checkout_token = base64url(session_id ‖ expiry ‖ HMAC-SHA256(key, session_id ‖ expiry))`. Payments verifies by recomputing, and repeated creation returns the same token. The key comes from an ExternalSecret. Alternative considered: random tokens stored hashed. Rejected because idempotent creation would then have to return a token it cannot recover.

### Fingerprints at the edge

The edge normalizes (trim, case-fold, collapse whitespace) the full shipping address and computes `HMAC-SHA256(fingerprint_key, normalized)` for the address fingerprint. It does the same for the client IP's /24 (IPv4) or /48 (IPv6) network. The payment-method fingerprint is the HMAC of the synthetic token's card identity. Only fingerprints and country/region/postal code cross into internal services. For synthetic integrations, a `simulated_client` object supplies IP country and network directly. For real integrations, the network comes from the connection address, and the country is absent until a geolocation source is chosen.

### Synthetic token format

Tokens look like `tok_<card-id>_<behavior>` with behavior `ok`, `nsf`, `issuer` or `invalid`. The card identity drives the payment-method fingerprint, so the same card always has the same fingerprint regardless of behavior. The live simulator chooses behaviors per scenario, for example stolen cards that are often issuer-declined.

### Processor simulator store and faults

The simulator has its own database with a table of operations keyed by `operation_key`, request hash and outcome. Faults (latency distribution, timeout rate) are seeded configuration. A timeout records the outcome and then sleeps past the caller's deadline, which reproduces a lost response realistically. Transport is `processor.v1` gRPC for consistency with internal conventions, even though a real processor would be an external HTTP API. The adapter boundary in Payments keeps that swap local.

### Workers inside Payments

Recovery and expiry run as goroutines in Payments, and a Postgres advisory lock elects one active worker across replicas. Both are idempotent because they append with expected versions. A separate deployment is unnecessary at this scale. The architecture's reconciliation worker (#59) remains separate.

### Generated Go bindings per module

Each Go module keeps its own `gen/` with only the packages it uses. `codegen:proto` passes `--go_opt=M…` import-path overrides per consumer, and the protos drop `go_package`. This preserves `GOWORK=off` independent builds with the existing `go-service.Dockerfile`. Alternative: one shared generated module. Rejected because it would need `replace` directives and a multi-module Docker context.

### Database layout

One CNPG `Cluster` (`payment-gateway-pg`, one instance in dev, three in prod) with declarative `Database` resources and managed roles `accounts`, `payments`, `processor`. Each role owns only its database, with `CONNECT` revoked from `PUBLIC`. Each service's migrations (goose, embedded SQL) run as an init container invoking `<service> migrate`, so a service can only migrate its own schema. Alternative: one cluster per service. Rejected for homelab resource cost; the role separation gives the ownership guarantee.

### Libraries

pgx v5, sqlc for queries, goose v3, grpc-go with the health service, client_golang metrics, optional OTel SDK instrumentation without a configured exporter, and slog JSON logging. Application metrics, traces, dashboards and alerts are not integrated with ClickStack in this gateway skeleton; a later change will handle collection and export. This keeps service scaffolding portable without introducing a direct backend dependency.

## Risks / Trade-offs

- [Large change] → Tasks are grouped so contracts plus Accounts, then Payments plus processor, then edge plus deployment, can each land as a PR.
- [Synchronous attempt latency is the sum of fraud and processor] → Deadlines cap it. Unknown outcomes return `PENDING` rather than blocking.
- [Advisory-lock leader election] → If the leader dies its lock is released with the connection, and another replica takes over within one poll interval.
- [Global position gaps and out-of-order commits] → The relay reads below a snapshot watermark (see the event-store decision).
- [HMAC keys rotate] → Rotation changes fingerprints. Keys are versioned in secret names, and rotation handling belongs to #63.

## Migration Plan

1. Land contracts and Accounts; deploy the database cluster and Accounts in dev.
2. Land processor simulator and Payments; deploy in dev with fraud already running.
3. Land edge routes; create a synthetic integration with `accounts create-integration --synthetic` and store its key in Doppler; run a scripted smoke flow (provision merchant, create payment, approve, decline, timeout-then-recover) and record evidence.
4. Promote to prod after dev evidence. Rollback: revert applications; the database is retained, and nothing outside this repository reads it.

## Open Questions

- Payment expiry default (for example 30 minutes) and recovery grace period (for example 10 seconds) can be tuned with the live simulator without changing behavior.
