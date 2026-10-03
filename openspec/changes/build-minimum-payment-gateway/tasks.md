## 1. Contracts and codegen

- [ ] 1.1 Add `contracts/accounts/v1/accounts.proto` (AuthenticateIntegration, ProvisionMerchant, ResolveCustomer), `contracts/payments/v1/payments.proto` (CreatePayment, StartAttempt, GetPayment) and `contracts/processor/v1/processor.proto` (Sale, GetOperation); verify all compile with `protoc -I contracts`
- [ ] 1.2 Remove `go_package` options and extend `codegen:proto` with per-module Go generation using `M` import overrides for edge, accounts, payments and processor-simulator; verify each module's `gen/` contains only the packages it imports and `GOWORK=off go build ./...` passes per module
- [ ] 1.3 Ensure golden histories exist under `contracts/payments/v1/testdata/` and `contracts/synthetic/identity-vectors.json` exists with the UUIDv5 namespace, merchant and customer ID cases, and synthetic-key address, network and payment-method fingerprint cases (add them if `generate-historical-synthetic-data` has not landed); verify they parse

## 2. Accounts

- [ ] 2.1 Scaffold `src/accounts` (composition root, config, gRPC health with DB readiness, bounded shutdown, slog, metrics and optional OTel without a configured exporter); verify readiness is NOT_SERVING with the database down
- [ ] 2.2 Add goose migrations and sqlc queries for integrations (hashed key, synthetic flag), merchants and customers, plus the `migrate` and `create-integration [--synthetic [--id <uuid>]]` subcommands; verify migrations apply to a fresh database, the key is printed once, and a duplicate `--id` is rejected
- [ ] 2.3 Implement authentication, idempotent merchant provisioning with conflict on changed data, and lazy customer mapping with UUIDv5-derived merchant and customer IDs; verify with tests for repeated provisioning, conflict and the identity vector cases

## 3. Processor simulator

- [ ] 3.1 Scaffold `src/processor-simulator` with its own database, migrations and gRPC health; verify readiness follows the database
- [ ] 3.2 Implement idempotent Sale with token behaviors, seeded latency and timeout injection, and GetOperation; verify tests for same-key replay, different-request rejection, timeout-then-status and not-found

## 4. Payments

- [ ] 4.1 Implement the pure domain aggregate (`Apply`, command decisions) for the `payments.v1` lifecycle; verify it accepts all valid and rejects all invalid golden histories
- [ ] 4.2 Add migrations and sqlc queries for `events` (global position, unique event ID and payment version), `payment_status` and `idempotency`; implement append-with-expected-version in one transaction; verify the stale-append integration test against Postgres
- [ ] 4.3 Implement CreatePayment with idempotency per integration, merchant and order reference, expiry, and HMAC checkout tokens; verify repeat-returns-same and changed-snapshot-conflict tests
- [ ] 4.4 Implement StartAttempt orchestration (attempt idempotency, active-attempt guard, fraud call with fail-closed fallback, committed effect intent, processor call, unknown outcome); verify tests for retried submission, fraud timeout and processor timeout
- [ ] 4.5 Implement recovery and expiry workers with advisory-lock leadership; verify crash-after-intent, unknown-then-resolved and expiry-blocked-by-pending tests
- [ ] 4.6 Implement GetPayment scoped to the owning integration and status-view rebuild from events; verify the foreign-integration not-found and rebuild-equals-original tests

## 5. Edge checkout API

- [ ] 5.1 Restructure the edge into transport/application packages with Accounts and Payments gRPC clients, deadlines and correlation propagation; verify `golangci-lint` passes
- [ ] 5.2 Implement the four public routes, integration-key and checkout-token authentication, `Idempotency-Key` enforcement, return URL validation and error mapping; verify with a local end-to-end run against all services
- [ ] 5.3 Implement address, IP-network and payment-method fingerprints with the secret key for real integrations and the non-secret synthetic key for synthetic ones, and the synthetic-only simulated client context; verify formatting-insensitive address fingerprints, the identity vector fingerprint cases, and rejection of simulated context from a real integration

## 6. CI, images and deployment

- [ ] 6.1 Change the CI Go job to lint and test every Go module with a Postgres service container; verify CI passes on the PR
- [ ] 6.2 Add explicit accounts, payments and processor-simulator jobs to `release.yml` using `build-image.yml`; verify PR builds do not publish
- [ ] 6.3 Add the CNPG cluster chart with per-service databases, owner roles, revoked public connect, `wal_level: logical` and replication for the `payments` role; add charts for accounts, payments, processor-simulator and edge (ClusterIP, gRPC or HTTP probes, migrate init containers, ExternalSecrets for keys), without application telemetry dashboards, alerts or trace endpoints; verify `helm lint` and `helm template` for base and prod values
- [ ] 6.4 Register child applications in the dev and prod app-of-apps roots; verify rendered application names are environment-qualified and target only their cluster

## 7. Evidence and documentation

- [ ] 7.1 Deploy to dev, create a synthetic integration, run the smoke flow (provision merchant, create payment, approve, risk decline, processor decline, timeout then recovery, expiry) and verify every resulting history passes the aggregate validator
- [ ] 7.2 Verify that the Payments role is denied access to the Accounts database in dev and record the command output
- [ ] 7.3 Update architecture current foundation, payment flows (fail closed, synthetic context, token format), gitops runtime table, CONTRIBUTING local run instructions and roadmap rows #34/#35/#37/#38/#39 with evidence links; verify links resolve
- [ ] 7.4 Run `openspec validate build-minimum-payment-gateway --strict` and verify it passes
