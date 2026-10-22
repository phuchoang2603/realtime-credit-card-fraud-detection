## Purpose

Describe each data product with one ODPS descriptor and one ODCS contract per output port, kept together in one folder, so producers and consumers validate against the same rules with existing tools.

## Requirements

### Requirement: Data product folder

Each data product SHALL live in its own folder under `contracts/data-products/<product>/` containing `product.odps.yaml` and one `*.odcs.yaml` per output port. Every output port SHALL reference exactly one contract in the same folder by ID and version. Product, contract, physical schema and dataset versions SHALL be independent.

#### Scenario: Unresolved port contract

- **WHEN** an output port references a contract ID that no ODCS file in the product folder declares
- **THEN** the product lint fails

### Requirement: Historical payments product

The `historical-payments` product SHALL expose two output ports: `payment-events` and `fraud-labels`, each backed by its own ODCS contract with ownership, synthetic classification, field meanings, physical schema versions and SQL quality rules. The `fraud-labels` port SHALL declare the `payment-events` contract as an input contract. The descriptor SHALL record the storage location template, completion marker, access policy and consumption rules.

#### Scenario: Label leakage guidance

- **WHEN** a consumer reads the `fraud-labels` contract
- **THEN** it states that labels may be used only after `label_available_at` and that `scenario` is excluded from features

### Requirement: Declared data quality

The contracts SHALL declare executable SQL rules. Identical repeated event deliveries SHALL be valid; conflicting event identities, illegal lifecycle transitions, invalid amounts, invalid label sources, labels available before their occurrence and labels referencing missing or, for chargebacks, unsuccessful attempts SHALL fail. Data Contract CLI SHALL execute these rules against a configured DuckDB data source without a repository-owned validation wrapper or a second rule suite.

#### Scenario: Duplicate delivery

- **WHEN** identical copies of an event appear in a complete valid history
- **THEN** the payment-events rules pass

#### Scenario: Conflicting delivery

- **WHEN** the same `event_id` appears with different content
- **THEN** the delivery-consistency rule fails

#### Scenario: Invalid chargeback

- **WHEN** a chargeback label references an attempt that did not succeed
- **THEN** the fraud-labels linkage rule fails

### Requirement: Native lint and generated references

CI SHALL discover every product folder, lint each descriptor and contract with pinned Data Product CLI and Data Contract CLI, and reject drift between each contract and its generated Markdown reference under `docs/data-ml/data-products/<product>/`. CI lint SHALL NOT claim to validate produced datasets.

#### Scenario: Invalid contract

- **WHEN** a contract violates the official ODCS schema
- **THEN** CI fails

#### Scenario: Stale reference

- **WHEN** a field meaning changes without regenerating its reference
- **THEN** the documentation comparison fails

### Requirement: Execution boundaries

Documentation SHALL distinguish contract lint, data validation and catalog ingestion, name DuckDB as the current SQL dialect, and leave Spark, Flink and DataHub integration to their pipelines. Producing and ingesting jobs SHALL own data preparation, physical file checks and publication gating.

#### Scenario: Dataset validation

- **WHEN** a job exposes `payment_events` and `fraud_labels` in a DuckDB database
- **THEN** the documented `datacontract test` command checks each contract and writes a report
