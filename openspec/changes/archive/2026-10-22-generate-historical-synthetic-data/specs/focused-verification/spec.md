## ADDED Requirements

### Requirement: Generator behavior tests

Simulator tests SHALL protect reproducibility for a small fixed configuration, identity vectors, Arrow schema parity with the ODCS contracts, schema-evolution and duplicate behavior, sorted file and row-group layout, truth separation between events and labels, and refusal to publish when a contract check fails. They SHALL use small local outputs and SHALL NOT require object storage or network access beyond the pinned contract CLI.

#### Scenario: Offline test run

- **WHEN** simulator tests run in CI without storage credentials
- **THEN** they generate to a temporary directory, validate it against both contracts and verify these behaviors
