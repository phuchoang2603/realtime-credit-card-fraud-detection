## ADDED Requirements

### Requirement: Go service tests

The Go CI job SHALL lint and test every Go module. Payments tests SHALL run the domain aggregate against the golden lifecycle histories, and SHALL cover creation and attempt idempotency conflicts, concurrent same-key requests, replay after process restart, fail-closed fraud handling, unknown-outcome recovery including crash after intent, and expiry blocked by pending outcomes. Event-store append and idempotency persistence SHALL be tested against a real Postgres provided by CI, including transactions whose global sequence allocation order differs from commit order. Processor simulator tests SHALL cover sale idempotency and timeout-then-status. Checkout and simulator boundary tests SHALL verify rejection of raw card credentials. Tests SHALL NOT exercise HTTP or gRPC transport except through these behaviors.

Vault tests SHALL cover integration/role authorization, opaque token issuance, encrypted fixture persistence and stable fingerprint vectors. Payments recovery tests SHALL prove that private encrypted operation requests permit recovery after a crash without leaking tokens into events. Redis integration tests SHALL cover completed-result hits, content conflicts and expiry/eviction/outage fallback without another processor effect. Recorded processor status SHALL remain available during a vault outage. Acceptance SHALL report the documented [functional](../../../../../docs/overview/requirements.md#functional-requirements) and [non-functional](../../../../../docs/overview/requirements.md#non-functional-requirements) targets with nominal and injected-fault evidence separately; bank-file reconciliation is outside scope.

#### Scenario: Stale append in CI

- **WHEN** the store integration test appends with a stale expected version against CI Postgres
- **THEN** the append is rejected and the test passes
