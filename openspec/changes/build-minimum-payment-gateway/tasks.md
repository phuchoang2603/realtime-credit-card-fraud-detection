## 1. Contracts and codegen

- [ ] 1.1 Add `contracts/accounts/v1/accounts.proto` (AuthenticateIntegration, ProvisionMerchant, ResolveCustomer), `contracts/payments/v1/payments.proto` (CreatePayment, StartAttempt, GetPayment), `contracts/vault/v1/vault.proto` (TokenizeSyntheticMethod, DescribeMethod, ResolveForProcessing) and `contracts/processor/v1/processor.proto` (Sale with integration scope, GetOperation); verify all compile with `protoc -I contracts` and existing payment/fraud event meanings are unchanged
- [ ] 1.2 Remove `go_package` options and extend `codegen:proto` with per-module Go generation using `M` import overrides for edge, accounts, payments, vault-simulator and processor-simulator; verify each module's `gen/` contains only the packages it imports and `GOWORK=off go build ./...` passes per module
- [ ] 1.3 Ensure golden histories exist under `contracts/payments/v1/testdata/` and `contracts/synthetic/identity-vectors.json` exists with the UUIDv5 namespace, merchant and customer ID cases, and synthetic-key address, network and payment-method fingerprint cases (add them if `generate-historical-synthetic-data` has not landed); verify they parse

## 2. Accounts

- [ ] 2.1 Scaffold `src/accounts` (composition root, config, gRPC health with DB readiness, bounded shutdown, slog, metrics and optional OTel without a configured exporter); verify readiness is NOT_SERVING with the database down
- [ ] 2.2 Add goose migrations and sqlc queries for integrations (hashed key, synthetic flag), merchants and customers, plus the `migrate` and `create-integration [--synthetic [--id <uuid>]]` subcommands; verify migrations apply to a fresh database, the key is printed once, and a duplicate `--id` is rejected
- [ ] 2.3 Implement authentication, idempotent merchant provisioning with conflict on changed data, and lazy customer mapping with UUIDv5-derived merchant and customer IDs; verify with tests for repeated provisioning, conflict and the identity vector cases

## 3. Vault and processor simulators

- [ ] 3.1 Scaffold `src/processor-simulator` with its own database, migrations and gRPC health; verify readiness follows the database
- [ ] 3.2 Implement idempotent Sale with processor-authenticated vault resolution, seeded latency and timeout injection, and GetOperation; verify same-key replay, different-request rejection, timeout-then-status, not-found, raw-card rejection and recorded-outcome recovery while vault is down
- [ ] 3.3 Scaffold `src/vault-simulator` with its own database/migrations, encrypted fixture payloads, token digests, vault-only encryption key, scoped tokenization CLI, metadata API and processor-only resolution over authenticated encrypted gRPC; verify foreign-token/unauthorized-call denial, raw-card rejection, opaque tokens, fingerprint-vector parity across token/behavior changes and no plaintext fixture payload in storage/logs

## 4. Payments

- [ ] 4.1 Implement the pure domain aggregate (`Apply`, command decisions) for the `payments.v1` lifecycle; verify it accepts all valid and rejects all invalid golden histories
- [ ] 4.2 Add migrations and sqlc queries for `events` (global sequence position, unique event ID and payment version), `payment_status`, `idempotency` and private encrypted `processor_requests`; commit request recovery data with its effect intent and clear the encrypted token with the terminal outcome; verify stale append, rollback atomicity, concurrent same-key requests, crash recovery and allocation-order/commit-order divergence against Postgres without relying on Redis
- [ ] 4.3 Implement CreatePayment with idempotency per integration, merchant and order reference, expiry, and HMAC checkout tokens; verify repeat-returns-same and changed-snapshot-conflict tests
- [ ] 4.4 Implement StartAttempt orchestration (attempt idempotency, active-attempt guard, fraud call with fail-closed fallback, committed effect intent, processor call through an application-owned port and simulator transport adapter, unknown outcome); verify tests for retried submission after restart, fraud timeout and processor timeout
- [ ] 4.5 Implement recovery and expiry workers with advisory-lock leadership; verify crash-after-intent, unknown-then-resolved and expiry-blocked-by-pending tests
- [ ] 4.6 Implement GetPayment scoped to the owning integration and status-view rebuild from events; verify the foreign-integration not-found and rebuild-equals-original tests
- [ ] 4.7 Add a Redis adapter for request-bound immutable completed-attempt results, populate only after commit, enforce configurable TTL/max memory and 50 ms operation deadlines, and fall back to Postgres; verify hit/miss, changed-request conflict, expiry/eviction/outage/restart and absence of tokens, pending state and sensitive payloads in cache values

## 5. Edge checkout API

- [ ] 5.1 Restructure the edge into transport/application packages with Accounts, Payments and restricted vault metadata gRPC clients, deadlines and correlation propagation; verify `golangci-lint` passes
- [ ] 5.2 Implement the four public routes, integration-key and checkout-token authentication, `Idempotency-Key` enforcement, return URL validation, synthetic-token-only payment input and error mapping; verify raw-card rejection and a local end-to-end run against all services
- [ ] 5.3 Implement address/IP-network fingerprints at Edge and use vault-derived synthetic method fingerprints, preserving the shared key/vectors and synthetic-only client context; verify formatting-insensitive address fingerprints, identity-vector parity, foreign/inactive token rejection and rejection of simulated context from a real integration

## 6. CI, images and deployment

- [ ] 6.1 Change the CI Go job to lint and test every Go module with Postgres and Redis service containers; verify CI passes on the PR
- [ ] 6.2 Add explicit accounts, payments, vault-simulator and processor-simulator jobs to `release.yml` using `build-image.yml`; verify PR builds do not publish
- [ ] 6.3 Add the CNPG chart with separate Accounts/Payments/vault/processor databases, owner roles, revoked public connect and `wal_level: logical`, without replication privileges on application roles; add application charts and a pinned, memory-bounded Redis cache chart (ClusterIP only, migrate init containers, namespace-scoped SecretStore/ExternalSecrets, authenticated encrypted service calls, TLS-aware health probes, default-deny vault/DB/Redis network policies), without application dashboards/alerts; verify `helm lint` and `helm template` for base and prod values and provision trust/caller/encryption secrets through the normal platform process
- [ ] 6.4 Register child applications in the configured prod app-of-apps root; verify rendered application names are prod-qualified, target only that cluster and do not restore a dev root

## 7. Evidence and documentation

- [ ] 7.1 Create a synthetic integration, tokenize fixtures through the scoped vault CLI and run the smoke flow (provision merchant, create payment, approve, risk decline, processor decline, timeout then recovery, expiry, Redis outage) locally/in integration; verify every resulting history passes the aggregate validator, then repeat after the normal GitOps rollout to the configured target
- [ ] 7.2 Verify that Payments cannot access Accounts/vault databases or logical replication, unauthorized callers cannot resolve vault fixtures, and Redis accepts only its scoped application identity in integration and after rollout; record allowed/denied access evidence
- [ ] 7.3 Update architecture current foundation, payment flows (fail closed, synthetic context, token format), gitops runtime/local run instructions and roadmap rows #34/#35/#37/#38/#39 with evidence links; distinguish implemented simulator behavior from later ISO 8583, VND/authentication, capture and settlement work; verify links resolve
- [ ] 7.4 Run `openspec validate build-minimum-payment-gateway --strict` and verify it passes
- [ ] 7.5 Record a nominal 50-attempt/s, 10-minute synthetic load run against the documented p95/error goals, plus separate deadline/lost-response and restart-recovery runs; include resource/latency settings and sensitive-data exclusion evidence, without presenting this as proof of the monthly availability SLO or PCI DSS compliance
