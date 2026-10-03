## Why

Generators, data pipelines and the gateway services cannot start until they share one definition of payment identities, checkout snapshots, attempt signals, lifecycle events, risk decisions and labels. Today the only contract is `fraud.v1.Predict`, which carries 21 precomputed credit-card handbook features (terminal risk windows) that no ecommerce gateway produces, and a model trained on that data. The contract must describe ecommerce decision-time inputs now, and leave room for the fraud service to later run on Knative with KServe inference ([#70](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/70)) without another breaking change.

## What Changes

- Add `payments.v1` wire types: integration-scoped merchant/customer references, immutable checkout snapshot shaped after the marketplace session request, attempt runtime signals and payment-method fingerprints, money in integer minor units.
- Add `payments.v1` payment event envelope and a thin lifecycle catalog (payment created, attempt started, risk decision recorded, processor requested/unknown, attempt declined, payment succeeded/expired) with legal transitions and ordering rules shared by the historical generator and Payments.
- Add `labels.v1` fraud label record with `label_available_at`, separating delayed chargeback labels from synthetic simulation truth.
- **BREAKING**: Replace `fraud.v1.FraudService/Predict` with `fraud.v2.FraudService/Decide`. The request carries raw attempt context; the response returns an approve/decline outcome, reason codes and policy/model/feature versions. Rule declines become normal outcomes instead of `PERMISSION_DENIED` errors. `contracts/fraud/v1` and its bindings are deleted.
- **BREAKING**: Fraud service becomes a stateless rules-only baseline (`rules-v1`) on the new contract. The bundled handbook model, pickle loader, sklearn/pandas/numpy dependencies, handbook blocklists and the real-model property test are removed. Readiness no longer depends on a model.
- **BREAKING**: Remove the edge's public `POST /predict` and its fraud gRPC client. Risk decisions are requested by Payments, not by public callers; the edge keeps health endpoints until checkout routes arrive.
- Replace prediction metrics with decision outcome, reason and latency metrics; defer dashboard work until after the minimum gateway skeleton.

## Capabilities

### New Capabilities

- `payment-contracts`: Versioned payment identity, checkout snapshot, attempt signal, lifecycle event and label contracts, including legal transitions and synthetic-tenant isolation.
- `fraud-decision`: The `fraud.v2` decision RPC, input validation, the `rules-v1` baseline policy and version reporting.

### Modified Capabilities

- `service-conventions`: Fraud readiness and lifecycle no longer involve a model; contract scenarios target the decision RPC; the edge prediction translation requirement is removed; fraud metric compatibility is replaced by documented metric names.
- `focused-verification`: Behavior tests protect `rules-v1` boundaries and required inputs; the real-model repeatability requirement is removed; CI wording no longer assumes the edge has bindings.
- `talos-gitops-deployment`: The internal fraud deployment no longer supplies a model path and readiness no longer depends on a model.

## Impact

- Contracts: new `contracts/payments/v1`, `contracts/fraud/v2`, `contracts/labels/v1`; delete `contracts/fraud/v1`.
- Fraud service: `app/` rewritten around the decision contract; `models/`, preprocessing and model code deleted; `pyproject.toml`/`uv.lock` lose scikit-learn, pandas, numpy and hypothesis; Dockerfile stops copying models.
- Edge: `/predict`, fraud client configuration (`FRAUD_GRPC_TARGET`, `PREDICTION_TIMEOUT`) and `gen/` removed; Go dependencies tidied.
- Tooling: `codegen:proto` generates Python bindings for every contract into the fraud service and no Go bindings until a Go consumer exists; treefmt exclusions follow the generated paths.
- Deployment and observability: chart env, probes documentation and metrics; no dashboard or alert resources in this change.
- Docs: CONTRIBUTING, gitops runtime table, architecture current-foundation table, service conventions layout, verification evidence and affected roadmap rows; issue #70 text references `fraud.v2`.
- No external caller uses the fraud service or the edge prediction route; dev/prod roll forward without a migration window.
