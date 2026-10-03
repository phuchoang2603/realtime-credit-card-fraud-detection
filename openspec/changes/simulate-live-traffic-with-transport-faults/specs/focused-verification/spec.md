## ADDED Requirements

### Requirement: Streaming transport tests

Simulator tests SHALL cover deterministic fault selection, unchanged pass-through when all faults are disabled, no loss across an injector restart with held records against a CI Kafka broker, and refusal to continue a dataset whose profile digest does not match. Outbox publication SHALL be verified in dev evidence rather than by tests of connector configuration. Tests SHALL NOT assert exact wall-clock timings.

#### Scenario: Injector restart in CI

- **WHEN** the restart test stops the injector while records are held and starts it again
- **THEN** every clean record appears in the faulty topic at least once
