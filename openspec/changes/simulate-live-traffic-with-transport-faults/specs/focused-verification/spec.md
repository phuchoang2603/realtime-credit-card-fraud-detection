## ADDED Requirements

### Requirement: Streaming transport tests

Simulator tests SHALL cover deterministic fault selection, unchanged pass-through when all faults are disabled, no loss across an injector restart with held records against a CI Kafka broker, refusal to continue a dataset whose profile digest does not match, and stable method fingerprints after vault token reissuance. Outbox publication SHALL be verified in controlled synthetic evidence rather than tests of connector configuration. Evidence SHALL demonstrate that token values, vault data and private encrypted operation requests do not appear on Kafka. Tests SHALL NOT assert exact wall-clock timings.

#### Scenario: Injector restart in CI

- **WHEN** the restart test stops the injector while records are held and starts it again
- **THEN** every clean record appears in the faulty topic at least once
