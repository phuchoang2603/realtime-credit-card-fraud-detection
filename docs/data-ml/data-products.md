# Data products

Data products are published through paired [ODPS](https://github.com/bitol-io/open-data-product-standard) descriptors and [ODCS](https://github.com/bitol-io/open-data-contract-standard) agreements under [`contracts/`](../../contracts). The descriptor identifies the product, owner, access and output port; the agreement defines field meanings and quality rules. Product, agreement, physical schema and dataset versions are independent, so new manifests record both descriptor identities, versions and hashes.

Each agreement has a generated, human-readable schema reference checked into [`docs/data-ml/contracts/`](contracts). CI lints ODCS/ODPS with pinned native tools and diffs the generated reference against a fresh export; see the commands in the [CI workflow](../../.github/workflows/ci.yml). After changing an agreement, regenerate its reference with the same export command used in CI.

Producers and consumers validate against the same agreement using supported engines. Agreement lint does not validate generated data; producing and ingesting jobs own physical file checks, data preparation and publication gating. Once a job exposes the product's tables or views, run the agreement's `datacontract test` command against them, for example:

```bash
DATACONTRACT_DUCKDB_DATABASE=/path/to/staging.duckdb \
  uvx --from 'datacontract-cli[duckdb]==1.2.7' datacontract test \
  contracts/data/<agreement>.odcs.yaml --server validation \
  --output validation.json --output-format json
```

The current example is [historical synthetic payments](../../contracts/data/historical-payments.odcs.yaml), the reproducible source for DP1 ingestion and downstream feature/training pipelines described in [data pipeline architecture](data-pipeline.md). It stays outside bronze tables and production payment storage; see its [generated reference](contracts/historical-payments-contract.md) for the declared schema.
