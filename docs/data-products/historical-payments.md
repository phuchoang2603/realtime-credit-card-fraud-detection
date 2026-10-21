# Historical synthetic payments

Provides reproducible, versioned synthetic source data for pipeline development and evaluation. Intended consumers are DP1 ingestion and downstream feature and training pipelines described in [data and ML flows](../architecture/data-ml-flows.md). This source stays outside bronze tables and production payment storage.

## Product interface

The product delivers complete payment histories as `payment_events` and separate attempt truth and delayed chargebacks as `fraud_labels`. Consumers read complete histories across date partitions and deduplicate identical event deliveries before interpreting transitions. Training uses labels only after `label_available_at`; scenario truth and future outcomes must not become decision inputs.

Ownership, agreement version, field meanings and quality rules live in the [ODCS agreement](../../contracts/data/historical-payments.odcs.yaml). Its [generated contract reference](historical-payments-contract.md) documents the declared schemas. Producer configuration and deployment details belong to their implementations; the agreement defines the interface consumers follow.

## Generation and publication

The [generator configuration](../../src/simulator/configs/historical-v1.yaml) controls the seed, date range, profiles, purchase behavior, scenarios, schema cutover, duplicate rate and storage layout. The generator builds deterministic profiles and purchase histories, then writes explicit [Arrow schemas](../../src/simulator/simulator/arrow.py) for events and separate labels.

The writer stages bounded batches, compacts and sorts each date partition, and publishes Zstandard-compressed Parquet with dictionaries and statistics. Row-group size and the approximate file-size target are configured independently; intentional duplicate exports and physical schema cutovers survive compaction. Published versions contain date-partitioned event and label Parquet, `validation.json` and `manifest.json` under `historical/<dataset_version>/`. The CLI supports configured S3 output and local output through `--output`.

Before publishing, the generator runs Data Contract CLI against the ODCS agreement over its compacted Parquet. It publishes the native `validation.json` report and records the agreement ID, version and hash in the last-written manifest; validation failure leaves the destination untouched. Parquet records remain deterministic while validation reports contain per-run identifiers and timestamps.

## Agreement and validation

Producers and consumers validate against the same agreement using supported engines. Current SQL rules target DuckDB; Spark and Flink integration is added with the relevant pipelines. Future DataHub ingestion uses the same ODCS source.

CI lints the agreement and checks the native schema reference for drift; the commands live in the [CI workflow](../../.github/workflows/ci.yml). After changing the agreement, use its export command with `docs/data-products/historical-payments-contract.md` as the output. Full quality rules remain in ODCS. The `schemaVersions` and `sinceSchemaVersion` custom properties describe layout evolution; engines do not automatically enforce them.

Agreement lint does not validate generated data. Producing and ingesting jobs own physical file checks, data preparation and publication gating. Once a job exposes complete histories as `payment_events` and `fraud_labels` tables or views in DuckDB, run from the repository root:

```bash
DATACONTRACT_DUCKDB_DATABASE=/path/to/staging.duckdb \
  uvx --from 'datacontract-cli[duckdb]==1.2.7' datacontract test \
  contracts/data/historical-payments.odcs.yaml --server validation \
  --output validation.json --output-format json
```

## Deployment and evidence

Historical evidence runs are finite jobs on the shared prod Argo Workflows controller in a separate data namespace; source sanity is inspected in a prod Kubeflow Notebook using the shared RustFS source identity stored in Doppler `prd`. See the [historical chart configuration](../../infra/charts/historical-generator/values.yaml) and [GitOps guide](../deployment/gitops.md).

Production workflow execution and authenticated Notebook results are recorded in [historical source verification](../verification/historical-data.md), with the implementation status in the [roadmap](../architecture/roadmap.md).
