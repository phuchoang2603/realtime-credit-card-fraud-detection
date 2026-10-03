## Purpose

Generate reproducible ecommerce payment histories with known fraud scenarios, configurable offline data problems and delayed labels, stored as immutable versioned source datasets outside bronze.

## ADDED Requirements

### Requirement: Validated generator configuration

The generator SHALL read one configuration file declaring dataset version, seed, synthetic integration identity, start date and duration, entity counts, purchase behavior, per-scenario rates and parameters, data-problem settings, label delays and output location. Invalid or inconsistent configuration SHALL fail before any output is written. Credentials SHALL NOT be accepted in the configuration file.

#### Scenario: Unknown scenario name

- **WHEN** the configuration enables a scenario that does not exist
- **THEN** the generator exits nonzero naming the invalid field and writes nothing

### Requirement: Reproducible generation

The same configuration, seed and generator revision SHALL produce the same records. Different seeds SHALL produce different histories with the same configured statistical shape.

#### Scenario: Regenerate a dataset

- **WHEN** a dataset is generated twice to separate locations with the same configuration and revision
- **THEN** both outputs contain identical records and identical manifest row counts

### Requirement: Continuable by live simulation

Profiles, and any single day's scenario schedule, SHALL be reproducible from the configuration and seed without generating purchases for earlier days. Merchant and customer IDs SHALL be derived from the configured synthetic `integration_id` and external seller and buyer IDs, and address, IP-network and payment-method fingerprints SHALL use the synthetic fingerprint key, both exactly as in the shared identity vectors. The manifest SHALL record the history end time, a digest of the generated profiles and the synthetic fingerprint key version.

#### Scenario: Rebuild profiles from a manifest

- **WHEN** profiles are rebuilt from a dataset manifest's configuration and seed with the same generator logic
- **THEN** their digest equals the manifest's profile digest

#### Scenario: Identity vectors

- **WHEN** the generator derives IDs and fingerprints for each case in the shared identity vectors
- **THEN** it produces exactly the listed values

### Requirement: Ecommerce behavior and fraud scenarios

Legitimate customers SHALL purchase mostly from familiar merchants, categories, devices, IP countries and shipping addresses, with amounts drawn per category. The generator SHALL support these fraud scenarios, each individually enabled and parameterized: `obvious_high_value`, `stolen_card_new_device`, `compromised_device_farm`, `account_takeover`, `reshipping_mule`, `card_testing`, `retry_attack`, and `friendly_fraud` label noise. Except for `obvious_high_value`, a scenario SHALL NOT be identifiable from a single attempt's fields alone; detecting it SHALL require history over customers, devices, addresses, payment methods or merchants.

#### Scenario: Card testing burst

- **WHEN** `card_testing` is enabled
- **THEN** its attempts arrive in bursts of low amounts from shared devices across many payment-method fingerprints, with elevated processor declines

#### Scenario: Scenario disabled

- **WHEN** a scenario rate is zero
- **THEN** no attempt carries that scenario in simulation truth

### Requirement: Contract-valid lifecycle histories

Every generated payment SHALL be a complete `payments.v1` history that satisfies the lifecycle transitions, including risk declines, processor declines, unknown processor outcomes that later resolve, retries and expiries. Risk decisions SHALL be recorded with the generator-owned policy version `sim-rules-v1`. The generator SHALL refuse to write a dataset containing an invalid history. Historical generation SHALL NOT call gateway APIs or write to payment event storage.

#### Scenario: Invalid history produced

- **WHEN** a generation defect yields a history that violates a transition
- **THEN** generation fails, identifies the payment and writes no dataset

### Requirement: Offline data problems

The configuration SHALL control merchant and category skew, the number of distinct device, IP-network, address and payment-method fingerprints, a schema-evolution cutover date before which a declared optional field is physically absent and the envelope schema version is lower, and a duplicate rate at which already-written events are exported again with the same `event_id`. The manifest SHALL record configured and observed values for each.

#### Scenario: Schema evolution cutover

- **WHEN** a dataset spans the configured cutover date
- **THEN** partitions before it lack the declared column, partitions after it contain it, and a merged read shows nulls only before the cutover

#### Scenario: Duplicate export

- **WHEN** the duplicate rate is 2%
- **THEN** about 2% of event rows repeat an earlier `event_id` with identical content, and the manifest reports the observed rate

### Requirement: Delayed labels and truth separation

The generator SHALL emit chargeback labels for fraudulent attempts that succeeded, with configured delays, plus simulation-truth labels for every attempt including its scenario. `friendly_fraud` SHALL produce chargeback labels on legitimate attempts. Labels SHALL be written to a dataset separate from payment events, and scenario names SHALL NOT appear in payment events.

#### Scenario: Declined fraudulent attempt

- **WHEN** a fraudulent attempt is declined
- **THEN** it has a simulation-truth label and no chargeback label

### Requirement: Immutable versioned output

Output SHALL be written under `historical/<dataset_version>/` as Parquet datasets for payment events and labels, partitioned by event date, with a manifest recording the configuration, its hash, the seed, generator revision, contract versions, row counts per partition and data-problem statistics. The manifest SHALL be written last. Generation SHALL refuse to write to a dataset version whose manifest already exists. The same command SHALL target S3-compatible object storage or a local directory.

#### Scenario: Existing dataset version

- **WHEN** the output already contains a manifest for the configured dataset version
- **THEN** the generator exits nonzero without modifying it

#### Scenario: Interrupted run

- **WHEN** generation stops before the manifest is written
- **THEN** the dataset version is incomplete, and consumers identify it by the missing manifest
