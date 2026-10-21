## Purpose

Generate reproducible ecommerce payment histories with known fraud scenarios, configurable offline data problems and delayed labels, published as immutable versioned datasets that satisfy the historical-payments data product contracts.

## ADDED Requirements

### Requirement: Validated generator configuration

The generator SHALL read one configuration file declaring dataset version, seed, synthetic integration identity, start date and duration (default 90 days), entity counts, purchase behavior, per-scenario rates and parameters, data-problem settings, label delays and output location. Invalid or inconsistent configuration SHALL fail before any output is written. Credentials SHALL NOT be accepted in the configuration file.

#### Scenario: Unknown scenario name

- **WHEN** the configuration enables a scenario that does not exist
- **THEN** the generator exits nonzero naming the invalid field and writes nothing

### Requirement: Reproducible generation

The same configuration, seed and generator revision SHALL produce the same records. Different seeds SHALL produce different histories with the same configured statistical shape.

#### Scenario: Regenerate a dataset

- **WHEN** a dataset is generated twice to separate locations with the same configuration and revision
- **THEN** both outputs contain identical records and identical manifest row counts

### Requirement: Continuable by live simulation

Profiles, and any single day's scenario schedule, SHALL be reproducible from the configuration and seed without generating earlier days. Merchant and customer IDs and address, IP-network and payment-method fingerprints SHALL match the shared identity vectors. The manifest SHALL record the history end time, a digest of the generated profiles and the synthetic fingerprint key version.

#### Scenario: Rebuild profiles from a manifest

- **WHEN** profiles are rebuilt from a manifest's configuration and seed with the same generator logic
- **THEN** their digest equals the manifest's profile digest

#### Scenario: Identity vectors

- **WHEN** the generator derives IDs and fingerprints for each case in the shared identity vectors
- **THEN** it produces exactly the listed values

### Requirement: Ecommerce behavior and fraud scenarios

Legitimate customers SHALL purchase mostly from familiar merchants, categories, devices, IP countries and shipping addresses, with amounts drawn per category. The generator SHALL support these individually enabled and parameterized scenarios: `obvious_high_value`, `stolen_card_new_device`, `card_testing`, `account_takeover` and `friendly_fraud` label noise. Except for `obvious_high_value`, a scenario SHALL NOT be identifiable from a single attempt's fields alone.

#### Scenario: Card testing burst

- **WHEN** `card_testing` is enabled
- **THEN** its attempts arrive in bursts of low amounts from shared devices across many payment-method fingerprints, with elevated processor declines

#### Scenario: Scenario disabled

- **WHEN** a scenario rate is zero
- **THEN** no label carries that scenario

### Requirement: Lifecycle histories

Every generated payment SHALL be a complete history covering risk declines, processor declines, unknown processor outcomes that later resolve, retries and expiries. Risk decisions SHALL record the generator-owned policy version `sim-rules-v1`. Historical generation SHALL NOT call gateway APIs or write to payment event storage.

#### Scenario: Retried payment

- **WHEN** an attempt is declined and the customer retries
- **THEN** the next attempt starts on the same payment after the decline and the history ends in success or expiry

### Requirement: Offline data problems

The configuration SHALL control merchant and category skew, the number of distinct device, IP-network, address and payment-method fingerprints, a schema-evolution cutover date before which `user_agent_family` is physically absent and `schema_version` is 1, and a duplicate rate at which events are exported again with identical content. The manifest SHALL record configured and observed values for each.

#### Scenario: Schema evolution cutover

- **WHEN** a dataset spans the configured cutover date
- **THEN** partitions before it lack the column, partitions after it contain it, and a merged read shows nulls only before the cutover

#### Scenario: Duplicate export

- **WHEN** the duplicate rate is 2%
- **THEN** about 2% of event rows repeat an earlier `event_id` with identical content, and the manifest reports the observed rate

### Requirement: Delayed labels and truth separation

The generator SHALL emit chargeback labels for fraudulent attempts that succeeded, with configured delays, and simulation-truth labels with their scenario for every attempt. `friendly_fraud` SHALL produce chargeback labels on legitimate attempts. Labels SHALL be written to the `fraud_labels` dataset, and scenario names SHALL NOT appear in payment events.

#### Scenario: Declined fraudulent attempt

- **WHEN** a fraudulent attempt is declined
- **THEN** it has a simulation-truth label and no chargeback label

### Requirement: Contract-gated immutable output

Output SHALL be written under `historical/<dataset_version>/` as Parquet partitioned by day (events by occurrence date, labels by availability date) for `payment_events` and `fraud_labels`, following the schemas of the historical-payments contracts. Before publishing, the generator SHALL run both contracts' native quality checks against the staged data and SHALL NOT publish if any check fails. The manifest SHALL be written last and record configuration and hash, seed, generator revision, product and contract identities, versions and hashes, row counts per partition, data-problem statistics and validation results. Generation SHALL refuse a dataset version whose manifest already exists. The same command SHALL target S3-compatible storage or a local directory.

#### Scenario: Validated bytes are published

- **WHEN** validation passes
- **THEN** the uploaded Parquet files are the same files the contract checks read

#### Scenario: Contract violation

- **WHEN** a generation defect yields an illegal lifecycle transition
- **THEN** the payment-events lifecycle check fails, the generator exits nonzero with the native report and no manifest is written

#### Scenario: Existing dataset version

- **WHEN** the output already contains a manifest for the configured dataset version
- **THEN** the generator exits nonzero without modifying it

#### Scenario: Interrupted run

- **WHEN** generation stops before the manifest is written
- **THEN** consumers identify the dataset version as incomplete by the missing manifest

### Requirement: Sorted and compacted Parquet layout

Each daily partition SHALL be written as a single file unless it exceeds the target file size, preserving every row including intentional duplicates. Events SHALL be sorted by occurrence time and labels by availability time, each with deterministic identity tie-breakers. Sorting SHALL use a configurable memory budget with disk spill rather than holding a whole partition in Python memory. Files SHALL use Zstandard compression and row-group statistics. Row-group rows and an approximate target file size SHALL be configured independently, and files SHALL rotate only at row-group boundaries. Each partition SHALL contain a single physical schema version.

#### Scenario: Duplicates stay with their originals

- **WHEN** a partition contains original events and duplicate exports
- **THEN** its sorted files hold exactly the same rows, and no file contains only duplicates

#### Scenario: Time-filtered read

- **WHEN** a reader filters a sorted partition by a time range
- **THEN** the result equals a filtered full scan, and row groups outside the range can be skipped from their statistics

### Requirement: Local execution

Generation SHALL run as a local CLI from the repository's development environment, with credentials taken only from the environment. It SHALL NOT require a container image, Kubernetes resource or workflow engine.

#### Scenario: Workstation run

- **WHEN** an operator runs the `historical` command with storage credentials in the environment
- **THEN** it generates, validates and publishes the dataset without any cluster access
