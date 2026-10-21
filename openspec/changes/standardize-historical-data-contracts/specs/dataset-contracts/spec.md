## Purpose

Provide one versioned dataset agreement that downstream applications follow and existing tools use for validation, documentation and future catalog ingestion.

## ADDED Requirements

### Requirement: Authoritative dataset agreement

Historical payment events and separate fraud labels SHALL have an ODCS agreement with stable identity, version, ownership, synthetic classification, field meanings and quality rules. The agreement SHALL describe physical layout evolution without requiring a custom Python runtime or a second maintained schema agreement.

#### Scenario: Physical layout evolution

- **WHEN** a producer implements an event schema version
- **THEN** it follows the version metadata and field meanings declared in ODCS
- **AND** pipeline integration owns the physical mapping and verifies that it conforms

### Requirement: Declared data quality

The agreement SHALL declare executable SQL rules for payment histories and label integrity. Identical repeated deliveries SHALL be valid; conflicting identities, illegal transitions, invalid amounts, invalid label references and labels available before their occurrence SHALL violate the declared rules. Existing ODCS tooling SHALL execute these rules against a supported configured data source without a repository-owned validation wrapper.

#### Scenario: Duplicate delivery

- **WHEN** the native engine tests identical copies of an event in a complete valid history
- **THEN** the declared quality rules accept the history without treating copies as additional transitions

#### Scenario: Conflicting delivery

- **WHEN** the native engine tests the same event identity with different content
- **THEN** the duplicate-consistency rule fails

#### Scenario: Invalid label

- **WHEN** the native engine tests a chargeback label referencing an unsuccessful attempt
- **THEN** the label-linkage rule fails

### Requirement: Native agreement checks and documentation

CI SHALL lint the ODCS agreement using pinned existing tooling and reject drift in its native generated schema reference. CI in this foundation SHALL NOT claim to validate produced datasets. Full quality definitions SHALL remain in the agreement rather than a separately maintained rule suite.

#### Scenario: Invalid agreement

- **WHEN** the agreement violates the official ODCS schema
- **THEN** the native lint command exits unsuccessfully

#### Scenario: Documentation changes

- **WHEN** a field meaning changes without regenerating the schema reference
- **THEN** the documentation comparison fails

### Requirement: Downstream execution boundaries

Integration documentation SHALL distinguish agreement lint, data validation and catalog ingestion. It SHALL identify the current SQL dialect and leave Spark and Flink integration to their pipelines rather than claiming automatic compatibility. The existing engine SHALL provide native validation results for supported data sources.

#### Scenario: Dataset validation

- **WHEN** a job exposes the declared datasets in a configured DuckDB database
- **THEN** the documented native command executes schema and quality checks and writes a report
- **AND** the job owns data preparation, physical inspection and publication gating
