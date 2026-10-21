# Historical synthetic payments

Provides reproducible, versioned synthetic source data for pipeline development and evaluation. Intended consumers are DP1 ingestion and downstream feature and training pipelines described in [data and ML flows](../architecture/data-ml-flows.md). This source stays outside bronze tables and production payment storage.

## Product interface

The product delivers complete payment histories as `payment_events` and separate attempt truth and delayed chargebacks as `fraud_labels`. Consumers read complete histories across date partitions and deduplicate identical event deliveries before interpreting transitions. Training uses labels only after `label_available_at`; scenario truth and future outcomes must not become decision inputs.

Ownership, agreement version, field meanings and quality rules live in the [ODCS agreement](../../contracts/data/historical-payments.odcs.yaml). Its [generated contract reference](historical-payments-contract.md) documents the declared schemas. Producer configuration and deployment details belong to their implementations; the agreement defines the interface consumers follow.

Production workflow execution and authenticated notebook evidence remain pending in the [implementation roadmap](../architecture/roadmap.md).

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
