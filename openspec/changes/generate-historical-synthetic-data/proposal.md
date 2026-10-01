## Why

DP1–DP3, the analytical schemas and the offline rubric rows (#43) need reproducible ecommerce payment histories with known fraud, realistic data problems and delayed labels, stored outside bronze. The handbook simulator this project used before produces card-present terminal transactions that the new `payments.v1` contracts cannot represent. The new contracts carry devices, IP geography, shipping addresses and payment methods, so the generator can also simulate the ecommerce fraud patterns the gateway is meant to detect.

## What Changes

- Add a Python `src/simulator` project with a seeded `historical` command. It builds customer, merchant, device and payment-method profiles, generates legitimate purchases and eight fraud scenarios, and emits contract-valid `payments.v1` lifecycle events and `labels.v1` labels.
- Simulate the offline data problems the rubric names, through configuration: merchant/category skew, high-cardinality identifiers, schema evolution (a field absent before a cutover date), and duplicate exports of the same event.
- Validate every generated payment history against the lifecycle transitions before writing. Add shared golden valid/invalid histories under `contracts/payments/v1/testdata/` that Payments will reuse.
- Write immutable, versioned Parquet datasets with a manifest to the `synthetic-source` bucket on the TrueNAS-hosted MinIO at `minio.home.phuchoang.sbs`, partitioned by event date. Local filesystem output is supported for tests.
- Add a sanity notebook that reports distributions, approximate distinct counts, old-schema nulls, duplicate rates, fraud rate by scenario, and whether the scenarios are learnable but not trivial. Record its output as #43 evidence.
- Run simulator lint and tests in the existing Python CI job.

## Capabilities

### New Capabilities

- `historical-data-generation`: Seeded, configurable generation of ecommerce payment histories, fraud scenarios, offline data problems, delayed labels, and immutable versioned output to object storage.

### Modified Capabilities

- `payment-contracts`: Adds shared golden lifecycle histories that every producer's validator must accept or reject.
- `focused-verification`: Adds generator behavior tests (lifecycle conformance, determinism, data-problem rates) to the protected test scope.

## Impact

- New `src/simulator/` (own `pyproject.toml`/`uv.lock`), `contracts/payments/v1/testdata/`, and `src/simulator/configs/historical-v1.yaml`.
- `codegen:proto` also generates Python bindings into `src/simulator/`.
- CI Python job runs both Python projects; `.gitignore` drops the legacy `simulated-data-*` entries.
- External prerequisite: the operator creates the `synthetic-source` bucket and a scoped access key on the TrueNAS MinIO. Credentials come from environment or a gitignored `.env`, never from the repository.
- Docs: new `docs/verification/historical-data.md` with commands and explained captures, sidebar link, roadmap #43 row status, and data-and-ML flows storage wording.
- Depends on `define-payment-and-decision-contracts`. It does not depend on the gateway, Kafka or any orchestrator.
