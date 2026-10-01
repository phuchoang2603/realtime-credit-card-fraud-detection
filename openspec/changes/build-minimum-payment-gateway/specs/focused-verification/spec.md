## ADDED Requirements

### Requirement: Go service tests

The Go CI job SHALL lint and test every Go module. Payments tests SHALL run the domain aggregate against the golden lifecycle histories, and SHALL cover creation and attempt idempotency conflicts, fail-closed fraud handling, unknown-outcome recovery including crash after intent, and expiry blocked by pending outcomes. Event-store append and idempotency persistence SHALL be tested against a real Postgres provided by CI. Processor simulator tests SHALL cover sale idempotency and timeout-then-status. Tests SHALL NOT exercise HTTP or gRPC transport except through these behaviors.

#### Scenario: Stale append in CI

- **WHEN** the store integration test appends with a stale expected version against CI Postgres
- **THEN** the append is rejected and the test passes
