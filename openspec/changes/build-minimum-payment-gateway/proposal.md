## Why

Strict streaming requires every live payment event to be produced by Payments, not fabricated by a simulator. That needs the smallest gateway that can take a checkout snapshot, run an attempt through the fraud decision and a processor, and durably record the lifecycle. This slice covers #34, #35, the API part of #37, #38 and the Payments side of #39. Kafka publication and the live simulator (`simulate-live-traffic-with-transport-faults`), Flink, webhooks and marketplace integration are left out.

## What Changes

- Add a Go **Accounts** service: hashed integration API keys, explicit idempotent merchant provisioning, lazy customer mapping with deterministically derived IDs, and a synthetic flag per integration. Synthetic integrations can be created with a chosen ID, so live simulation continues historical datasets.
- Add a Go **Payments** service with a Postgres event store. It handles idempotent payment creation, checkout sessions with scoped checkout tokens, and idempotent attempts. Attempts call `fraud.v2` Decide (failing closed when fraud is unavailable), persist a processor effect intent before calling the processor, and treat timeouts as unknown. A recovery worker resolves unknown and orphaned outcomes, and an expiry worker closes open payments. Every append uses an expected version and carries a global commit position for later publication.
- Add a Go **processor simulator** with its own store. It provides idempotent sale by operation key, outcomes set by synthetic token behavior, seeded latency and timeout injection, and an operation status query.
- Add public **checkout API** routes to the edge: merchant provisioning, payment creation, attempt submission and payment status. The edge computes address and IP-network fingerprints, so street lines and raw IPs never reach internal services. Synthetic integrations use a non-secret fingerprint key shared with the historical generator. Simulated client context is accepted only from synthetic integrations.
- Add gRPC contracts `accounts.v1`, `payments.v1` PaymentService and `processor.v1`, generated per consuming Go module.
- Deploy Accounts, Payments, processor simulator and edge (internal only) with a CloudNativePG cluster owned by this repository, per-service databases and roles, and migrations per service.
- The CI Go job lints and tests every Go module, and release adds one image job per new service.

## Capabilities

### New Capabilities

- `gateway-identity`: Integration authentication, merchant provisioning and customer mapping owned by Accounts.
- `payment-processing`: Payments commands, event-store append semantics, fraud and processor orchestration, unknown-outcome recovery and expiry.
- `processor-simulator`: Simulated external processor with idempotent sale, deterministic token behaviors, fault injection and status queries.
- `checkout-api`: Public edge HTTP routes, authentication, fingerprinting, simulated client context and error mapping.

### Modified Capabilities

- `service-conventions`: Adds health, lifecycle and persistence expectations for internal Go gRPC services.
- `focused-verification`: Adds Go service tests (aggregate against golden histories, idempotency, recovery, store integration) to CI.
- `talos-gitops-deployment`: Adds gateway workload and database deployment, and one image job per service.

## Impact

- New Go modules `src/accounts`, `src/payments`, `src/processor-simulator`; edge gains `internal/transport`/`application` packages for checkout routes and gRPC clients.
- New contracts `contracts/accounts/v1`, `contracts/payments/v1/payments.proto`, `contracts/processor/v1`, plus shared `contracts/synthetic/identity-vectors.json`; `codegen:proto` generates Go bindings per consumer module.
- New charts for each service and the database cluster; app-of-apps children for dev/prod; ExternalSecrets for fingerprint and token keys.
- Platform prerequisites already owned by talos-proxmox: CNPG operator, External Secrets. No new platform component.
- CI Go job gains `go test` for all modules with a Postgres service container; `release.yml` gains three image jobs.
- Docs: architecture current foundation, payment flows (fail-closed and synthetic context), gitops runtime table, CONTRIBUTING local run, roadmap rows #34/#35/#37/#38/#39.
- Depends on `define-payment-and-decision-contracts`; reuses golden histories from `generate-historical-synthetic-data` when present, otherwise adds them.
