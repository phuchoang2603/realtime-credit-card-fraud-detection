## 1. Contracts

- [x] 1.1 Add `contracts/payments/v1/types.proto` (Money in minor units, merchant/customer refs, checkout snapshot, line items, shipping location with address fingerprint, attempt signals, payment method) with presence for optional signals; verify `protoc -I contracts` compiles it
- [x] 1.2 Add `contracts/payments/v1/events.proto` with the `PaymentEvent` envelope, lifecycle payloads and `RiskAssessment`, documenting legal transitions in comments; verify it compiles without importing `fraud`
- [x] 1.3 Add `contracts/fraud/v2/fraud.proto` with `FraudService.Decide`, outcome enum, reason codes and version fields; verify it compiles importing only `payments/v1/types.proto`
- [x] 1.4 Add `contracts/labels/v1/labels.proto` with `FraudLabel`, `source` enum and `label_available_at`; verify it compiles
- [x] 1.5 Delete `contracts/fraud/v1/`, and verify `rg "fraud.v1|fraudv1|fraud/v1"` finds no references outside archived OpenSpec changes and research docs

## 2. Code generation

- [x] 2.1 Change `codegen:proto` in `devenv.nix` to generate Python bindings for all contracts into `src/fraud-service/gen/` and no Go bindings; update treefmt exclusions; verify it produces `payments/`, `fraud/v2/` and `labels/` bindings under `gen/`
- [x] 2.2 Delete `src/fraud-service/fraud/v1/` and `src/edge/gen/`, include the regenerated bindings in the change, and verify the fraud service imports `fraud.v2` bindings from `gen/`

## 3. Fraud service

- [x] 3.1 Replace `app/schema.py` with validated decision-input and decision-result models mapped from `fraud.v2` requests, enforcing required presence and snapshot consistency; verify with the required-input tests in 3.4
- [x] 3.2 Implement `rules-v1` in `app/domain` (USD only, `HIGH_AMOUNT`, `NEW_ACCOUNT_GEO_MISMATCH`, all matching reasons, rules-only version fields); delete `pre_prediction_checks.py`, `data_preprocessing.py`, `model.py` and the rule error classes; verify with the boundary tests in 3.4
- [x] 3.3 Serve `Decide` through `app/transport/grpc.py`, evaluating rules directly; remove the executor, capacity semaphore, model loading and model readiness from `main.py`, `runtime.py` and `config.py` (drop `MODEL_PATH`, `INFERENCE_WORKERS`); verify the server starts and the gRPC health checks report readiness SERVING
- [x] 3.4 Rewrite tests: parametrized `HIGH_AMOUNT` boundary, account-age and amount boundaries, absent IP country, unsupported currency and a missing required input; delete `test_prediction_property.py` and the model fixtures; verify `uv run --locked pytest -q` passes
- [x] 3.5 Replace prediction metrics with `fraud_decisions_total{outcome,reason}` and `fraud_decision_latency_seconds`; verify the metrics endpoint exposes them after one decision
- [x] 3.6 Remove scikit-learn, pandas, numpy and hypothesis from `pyproject.toml`, relock, delete `models/`, stop copying models in `infra/docker/fraud-service.Dockerfile`; verify `uv lock --check` and CI image build pass

## 4. Edge

- [x] 4.1 Remove `POST /predict`, `prediction.go`, the fraud gRPC client and `FRAUD_GRPC_TARGET`/`PREDICTION_TIMEOUT` config; keep health, readiness and correlation; run `go mod tidy`; verify `GOWORK=off golangci-lint run ./...` passes and `/health` returns 200

## 5. Deployment and observability

- [x] 5.1 Remove `MODEL_PATH` from chart values/schema and keep gRPC probes; verify `helm lint` and `helm template` pass for base and prod values

## 6. Documentation and tracking

- [x] 6.1 Update CONTRIBUTING (contracts, codegen, running services), gitops runtime table, architecture current-foundation table and service-conventions layout for the new contracts and removed routes; verify no doc references `/predict`, `MODEL_PATH` or `fraud.v1` except historical research
- [x] 6.2 Update verification evidence and affected roadmap rows to record the handbook model retirement and the rules-only baseline without claiming model quality; verify links resolve
- [x] 6.3 Edit issue #70 to reference `fraud.v2` decisions instead of `fraud.v1` and unchanged model behavior; verify with `gh issue view 70`
- [x] 6.4 Run `openspec validate define-payment-and-decision-contracts --strict` and verify it passes
