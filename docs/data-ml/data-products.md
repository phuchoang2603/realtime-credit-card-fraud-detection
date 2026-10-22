# Data products

Each data product lives in one folder under [`contracts/data-products/`](../../contracts/data-products): an [ODPS](https://github.com/bitol-io/open-data-product-standard) descriptor `product.odps.yaml` and one [ODCS](https://github.com/bitol-io/open-data-contract-standard) contract per output port. The descriptor identifies the product, owner, access and ports; each contract defines its port's field meanings and quality rules. Product, contract, physical schema and dataset versions are independent, and dataset manifests record each identity, version and file hash.

Each contract has a generated schema reference under [`docs/data-ml/data-products/<product>/`](data-products). CI discovers every product folder, lints descriptor and contracts with pinned native tools, and diffs each reference against a fresh export (`bash scripts/ci-checks.sh contracts`). After changing a contract, regenerate its reference with the export command in [`scripts/ci-checks.sh`](../../scripts/ci-checks.sh).

Contract lint does not validate data. Producing and ingesting jobs own physical file checks, data preparation and publication gating. Once a job exposes a port's tables or views in DuckDB, run the contract's quality rules against them:

```bash
DATACONTRACT_DUCKDB_DATABASE=/path/to/staging.duckdb \
  uvx --from 'datacontract-cli[duckdb]==1.2.7' datacontract test \
  contracts/data-products/<product>/<port>.odcs.yaml --server validation \
  --output validation.json --output-format json
```

The SQL rules use DuckDB syntax. Spark and Flink execution, and DataHub catalog ingestion, are verified with those pipelines when they exist.

## Historical payments

[`historical-payments`](../../contracts/data-products/historical-payments) is the reproducible synthetic source for DP1 ingestion and downstream feature and training pipelines described in [data pipeline architecture](data-pipeline.md). It stays outside bronze tables and production payment storage.

| Port | Contract | Reference |
|---|---|---|
| `payment-events` | [`payment-events.odcs.yaml`](../../contracts/data-products/historical-payments/payment-events.odcs.yaml) | [schema](data-products/historical-payments/payment-events.md) |
| `fraud-labels` | [`fraud-labels.odcs.yaml`](../../contracts/data-products/historical-payments/fraud-labels.odcs.yaml) | [schema](data-products/historical-payments/fraud-labels.md) |

The fraud-labels linkage rule reads `payment_events`, so validate labels in a database that exposes both tables.

### Published version

`historical/v7` is the current accepted dataset, generated from [`src/simulator/configs/historical.yaml`](../../src/simulator/configs/historical.yaml) (seed 2603) and validated against both contracts before publication. Its `manifest.json` records the configuration hash, generator revision, contract hashes, per-partition row counts and storage metrics; [`historical-sanity.ipynb`](../../src/simulator/notebooks/historical-sanity.ipynb) holds the executed distribution, skew, cutover, duplicate and layout checks.

| Measure | Value |
|---|---|
| History | 2026-07-01 to 2026-09-28 (90 days); schema cutover 2026-08-15 |
| Catalogue | Refurbished electronics: phone, smartwatch, laptop, accessory, tablet, headphones, game console, camera; condition excellent, good or fair |
| Payments / unique events / exported event rows | 420,392 / 2,171,314 / 2,215,309 |
| Duplicate export rate | 1.99% (configured 2%) |
| Fraud attempts by scenario | stolen card 12,157; account takeover 3,360; card testing 1,225; obvious high value 510 |
| Labels / chargebacks / friendly-fraud payments | 442,817 / 12,309 / 1,175 |
| Distinct customers / devices / IP networks / addresses / payment methods | 30,000 / 53,845 / 9,202 / 20,123 / 26,724 |
| Layout | 91 event files (median 2.5 MiB, 237 MiB total) and 111 label files (median 0.24 MiB), one zstd file per day |

Generate a new version from the devenv shell; the command refuses a version that already has a manifest:

```bash
cd src/simulator
doppler run -p payment-gateway-fraud-detection -c prd -- sh -c \
  'AWS_ACCESS_KEY_ID="$HISTORICAL_RUSTFS_ACCESS_KEY" AWS_SECRET_ACCESS_KEY="$HISTORICAL_RUSTFS_SECRET_KEY" \
   uv run --locked simulator historical --config configs/historical.yaml'
```

Bump `dataset_version` in the config for every new dataset. Pass `--output <dir>` to generate locally without credentials.
