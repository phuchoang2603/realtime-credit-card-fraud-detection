## Why

Strict streaming requires every live payment event to be produced by Payments, not fabricated by a simulator. That needs the smallest gateway that can take a checkout snapshot, run an attempt through the fraud decision and a processor, and durably record the lifecycle. This slice covers Accounts, payment event storage, the checkout API, the processor simulator and Payments-side fraud orchestration. Kafka publication and the live simulator (`simulate-live-traffic-with-transport-faults`), Flink, webhooks and marketplace integration are left out.

## What Changes

- Add a Go **Accounts** service: hashed integration API keys, explicit idempotent merchant provisioning, lazy customer mapping with deterministically derived IDs, and a synthetic flag per integration. Synthetic integrations can be created with a chosen ID, so live simulation continues historical datasets.
- Add a Go **Payments** service with a Postgres event store. It handles idempotent payment creation, checkout sessions with scoped checkout tokens, and idempotent attempts. Attempts call `fraud.v2` Decide (failing closed when fraud is unavailable), persist a processor effect intent before calling the processor, and treat timeouts as unknown. A recovery worker resolves unknown and orphaned outcomes, and an expiry worker closes open payments. Every append uses an expected version and atomically persists events, status and idempotency. A global sequence identifies rows; later WAL CDC supplies commit order, which the sequence does not guarantee.
- Add a Go **processor simulator** with its own store. It provides idempotent sale by operation key, outcomes set by synthetic token behavior, seeded latency and timeout injection, and an operation status query.
- Add an isolated Go **vault simulator** with its own database and credentials. An authenticated simulator CLI tokenizes allow-listed fictional card fixtures into opaque integration-scoped tokens. Edge receives only safe method metadata; only the processor identity may resolve encrypted processing fixtures. This demonstrates a PCI DSS-oriented boundary without accepting real card data or claiming compliance.
- Introduce a Payments-owned **Redis** cache for immutable completed-attempt idempotency results. Scope entries to integration/session and request content, populate only after commit, and fall back to Postgres on miss, eviction or outage. Keep pending state and credentials out of the cache.
- Keep provider wire protocols behind the Payments adapter boundary. The first slice uses `processor.v1` gRPC and synthetic tokens; the adopted bank-integration direction adds a synthetic ISO 8583 profile and then a selected acquirer's ISO 8583 or HTTP profile. Real-card custody, VND policy and issuer-authentication workflows require separate contract and acceptance work.
- Add public **checkout API** routes to the edge: merchant provisioning, payment creation, attempt submission and payment status. The edge computes address and IP-network fingerprints, so street lines and raw IPs never reach internal services. Synthetic integrations use a non-secret fingerprint key shared with the historical generator. Simulated client context is accepted only from synthetic integrations.
- Add gRPC contracts `accounts.v1`, `payments.v1` PaymentService, `vault.v1` and `processor.v1`, generated per consuming Go module. Replace behavior-encoded method tokens with vault-issued opaque tokens while retaining deterministic card fingerprints.
- Deploy Accounts, Payments, vault simulator, processor simulator, Redis and edge (internal only) with a CloudNativePG cluster owned by this repository, per-service databases and roles, and migrations per service. Authenticate service calls and restrict vault/database access with network policy.
- The CI Go job lints and tests every Go module, and release adds one image job per new service.
- Document functional/non-functional requirements and later merchant-based database sharding. Remove automated reconciliation and settlement-file ingestion from project scope; retain operation-status recovery.

## Capabilities

### New Capabilities

- `gateway-identity`: Integration authentication, merchant provisioning and customer mapping owned by Accounts.
- `payment-processing`: Payments commands, event-store append semantics, Redis result caching, fraud and processor orchestration, unknown-outcome recovery and expiry.
- `processor-simulator`: Simulated external processor with idempotent sale, deterministic token behaviors, fault injection and status queries.
- `checkout-api`: Public edge HTTP routes, authentication, fingerprinting, simulated client context, isolated synthetic vault/token access and error mapping.

### Modified Capabilities

- `service-conventions`: Adds health, lifecycle and persistence expectations for internal Go gRPC services.
- `focused-verification`: Adds Go service tests (aggregate against golden histories, idempotency, recovery, store integration) to CI.
- `talos-gitops-deployment`: Adds gateway workload and database deployment, and one image job per service.

## Impact

- New Go modules `src/accounts`, `src/payments`, `src/vault-simulator`, `src/processor-simulator`; edge gains `internal/transport`/`application` packages for checkout routes and gRPC clients.
- New contracts `contracts/accounts/v1`, `contracts/payments/v1/payments.proto`, `contracts/vault/v1`, `contracts/processor/v1`, plus shared `contracts/synthetic/identity-vectors.json`; `codegen:proto` generates Go bindings per consumer module. Vault resolution replaces parsing token behavior at Edge/processor; shared payment event schemas remain unchanged.
- New charts for each service and the database cluster; app-of-apps children for the configured prod root; ExternalSecrets for fingerprint and token keys. Local/integration validation precedes rollout; the retired dev root is not restored.
- Platform prerequisites already owned by talos-proxmox: CNPG operator, External Secrets. No new platform component.
- CI Go job gains `go test` for all modules with Postgres and Redis service containers; `release.yml` gains four image jobs, including vault-simulator. Redis uses an upstream pinned image, not a custom application image.
- Docs: architecture current foundation, payment flows (fail-closed and synthetic context), gitops runtime table and local run instructions, the gateway implementation roadmap.
- Depends on `define-payment-and-decision-contracts`; reuses golden histories from `generate-historical-synthetic-data` when present, otherwise adds them.
