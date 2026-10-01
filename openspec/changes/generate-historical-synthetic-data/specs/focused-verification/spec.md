## ADDED Requirements

### Requirement: Generator behavior tests

Simulator tests SHALL protect lifecycle validation against the golden histories, reproducibility for a small fixed configuration, schema-evolution and duplicate behavior, and truth separation between events and labels. They SHALL use small local outputs and SHALL NOT require object storage or network access.

#### Scenario: Offline test run

- **WHEN** simulator tests run in CI without credentials
- **THEN** they generate to a temporary directory and verify these behaviors
