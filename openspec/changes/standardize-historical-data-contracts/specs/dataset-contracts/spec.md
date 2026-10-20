## Purpose

Provide one versioned dataset agreement that drives schemas, executable quality checks, documentation and future catalog ingestion.

## ADDED Requirements

### Requirement: Authoritative dataset contract

Historical payment events and separate fraud labels SHALL have an ODCS contract with stable identity, version, ownership, synthetic classification, field meanings and executable quality rules. Arrow schemas SHALL be derived from that contract, including the physical omission of fields introduced after a requested event schema version. Unsupported versions or mappings SHALL fail explicitly.

#### Scenario: Old event schema

- **WHEN** a producer requests the pre-cutover event schema
- **THEN** its Arrow schema omits the subsequently introduced user-agent field
- **AND** the current schema includes that field without a second maintained definition

### Requirement: Shared quality execution

One validation entry point SHALL execute the contract's schema and quality checks for source datasets. Identical repeated deliveries SHALL remain valid; conflicting duplicate identities, illegal payment transitions, invalid amounts, invalid label references and labels available before their occurrence SHALL fail. Unsupported or unexecuted declared rules SHALL prevent success.

#### Scenario: Duplicate delivery

- **WHEN** two identical copies of a payment event occur in a complete valid history
- **THEN** quality validation accepts the history without treating the copies as additional transitions

#### Scenario: Conflicting duplicate

- **WHEN** the same event identity has different content
- **THEN** validation fails with the declared duplicate-consistency rule

#### Scenario: Delayed label

- **WHEN** a chargeback label references an attempt without a successful payment
- **THEN** validation fails with the declared label-linkage rule

### Requirement: Traceable validation evidence

Validation SHALL produce structured results on success and failure containing contract identity, version, content hash and individual rule outcomes. Schema inspection SHALL reject unexpected fields and wrong physical types instead of hiding them through projection or coercion.

#### Scenario: Schema mismatch

- **WHEN** a source Parquet file has a wrong integer width or an unexpected field
- **THEN** the report identifies the file and schema failure
- **AND** the validation command exits unsuccessfully

### Requirement: Generated contract documentation

Dataset documentation SHALL be generated from the same ODCS source used by validation. CI SHALL reject invalid contracts and stale generated documentation and exercise representative valid and invalid datasets without requiring a running catalog.

#### Scenario: Contract changes

- **WHEN** a field meaning or quality rule changes without regenerating documentation
- **THEN** the documentation check fails
