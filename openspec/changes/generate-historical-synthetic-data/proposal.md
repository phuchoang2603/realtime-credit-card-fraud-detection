## Why

DP1–DP3 and the offline rubric rows (#43) need reproducible ecommerce payment histories with known fraud, realistic data problems and delayed labels, stored outside bronze. Main already has an ODCS agreement and ODPS descriptor for this data, but no generator, and the files are scattered across `contracts/data/` and `contracts/products/` with one contract covering two different outputs. The earlier generator plan also maintained its own protobuf mapping and Python lifecycle validator next to the ODCS rules, so the same rules would live in two places. This change replaces the separate generator and contract plans with one: a data product folder that owns its contracts, and a generator that publishes only data those contracts accept.

## What Changes

- **BREAKING (repo layout):** Move the historical product to `contracts/data-products/historical-payments/` with `product.odps.yaml`, `payment-events.odcs.yaml` and `fraud-labels.odcs.yaml`. Remove `contracts/data/` and `contracts/products/`.
- Split the single ODCS agreement into two contracts, one per ODPS output port: `payment-events` and `fraud-labels`. The labels port declares the events contract as an input contract, because label linkage is checked against payment events.
- Add a Python `src/simulator` project with a seeded `historical` command. It builds customer, merchant, device and payment-method profiles, generates legitimate purchases and five fraud scenarios, and writes events and labels as Parquet whose shape follows the ODCS schemas.
- Write compact, query-friendly Parquet: daily partitions with one compacted file per day, sorted by time, Zstandard, bounded row groups and an approximate file-size target, produced by DuckDB with a memory cap and disk spill.
- Run the generator locally from devenv. No container image, Helm chart or Argo Workflow.
- Default history: 90 days, long enough for 30-day customer baselines, label delays and drift-shaped scenario windows.
- Simulate the rubric's offline data problems through configuration: merchant/category skew, high-cardinality fingerprints, a schema-evolution cutover and duplicate exports.
- Gate publication with the ODCS rules through Data Contract CLI. No repository-owned lifecycle validator, protobuf-to-Arrow mapper or second rule suite.
- Keep datasets continuable by the live simulator: seed-derived profiles and daily schedules, identities matching the shared identity vectors, and a manifest with history end and profile digest.
- Regenerate the historical data: publish a fresh dataset version to `s3://synthetic-source/historical/` under the new contracts. Older prefixes are removed only after the new version is accepted.
- Add a short sanity notebook (distributions, skew, distinct counts, cutover nulls, duplicate rate, fraud rate by scenario).
- CI lints every product folder with the pinned CLIs and runs simulator lint and tests.

## Capabilities

### New Capabilities

- `historical-data-generation`: Seeded generation of ecommerce payment histories, fraud scenarios, offline data problems and delayed labels, published as immutable versioned datasets after contract validation.
- `dataset-contracts`: Data product folders with an ODPS descriptor and one ODCS contract per output port, native lint, generated schema references and native data validation.

### Modified Capabilities

- `focused-verification`: Adds generator behavior tests to the protected test scope.

## Impact

- Moved: `contracts/data/historical-payments.odcs.yaml` and `contracts/products/historical-payments.odps.yaml` → `contracts/data-products/historical-payments/`. Contract IDs change to `urn:datacontract:fraud:historical-payment-events` and `urn:datacontract:fraud:historical-fraud-labels`; the product ID stays `urn:dataproduct:fraud:historical-payments` with version 2.0.0.
- Docs: `docs/data-ml/contracts/historical-payments-contract.md` is replaced by one generated reference per contract; `data-products.md` gains the historical product section and evidence.
- New `src/simulator/` with its own `pyproject.toml`/`uv.lock`; CI contract step moves into `scripts/ci-checks.sh`.
- Golden `payments.v1` lifecycle histories move to `build-minimum-payment-gateway`, which owns the Go aggregate that replays them. This change adds `contracts/synthetic/identity-vectors.json` if the gateway has not.
- Storage: the existing `synthetic-source` bucket on TrueNAS MinIO and the shared historical-source identity from Doppler `prd`. No credentials in the repository.
- Depends on `define-payment-and-decision-contracts` for field meanings. `simulate-live-traffic-with-transport-faults` depends on this change. No gateway, Kafka or orchestrator dependency.
