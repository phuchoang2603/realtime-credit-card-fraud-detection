## Purpose

Stand in for an external card processor so Payments can exercise approvals, declines, timeouts and status-based recovery with reproducible behavior.

## Requirements

### Requirement: Synthetic provider boundary

The simulator SHALL use `processor.v1` gRPC and synthetic payment-method tokens. Payments SHALL access it through a transport adapter outside its domain aggregate. The simulator SHALL NOT accept real card credentials. Its operation-key idempotency and status query SHALL be treated as simulator capabilities, not universal bank-protocol guarantees.

#### Scenario: Raw card credentials

- **WHEN** a sale supplies raw card credentials instead of a supported synthetic token
- **THEN** the simulator rejects the request without recording a sale

### Requirement: Idempotent sale

The simulator SHALL execute a sale for an operation key, integration identity, vault-issued synthetic payment-method token and amount. It SHALL durably record the outcome so that a repeated key with the same request returns the original outcome, and a repeated key with a different request is rejected. Recorded operations SHALL survive restarts. Returning a recorded outcome SHALL NOT require vault availability.

#### Scenario: Retry after timeout

- **WHEN** a sale is resubmitted with the same key after a timeout
- **THEN** the original outcome is returned and no second sale is recorded

### Requirement: Token behaviors

The processor SHALL resolve new operations' opaque tokens through the vault on the processor-only vault port for the owning integration.

#### Scenario: Complete token behaviors contract

- **WHEN** the `Token behaviors` capability is configured or exercised
- **THEN** The processor SHALL resolve new operations' opaque tokens through the vault on the processor-only vault port for the owning integration. The fictional fixture SHALL determine approve, insufficient funds, issuer decline or invalid-token behavior; the token value SHALL NOT encode that behavior. An unknown or inactive token SHALL be rejected without a sale. A vault outage SHALL return unavailable without a fabricated outcome or charge. Neither fixture contents nor token values SHALL be logged or placed in payment events.

#### Scenario: Insufficient-funds card

- **WHEN** a token with insufficient-funds behavior is charged
- **THEN** the sale is declined with that reason every time

#### Scenario: Vault outage during recovery

- **WHEN** the vault is unavailable after the simulator recorded a sale and Payments queries its operation key
- **THEN** the simulator returns the recorded outcome without resolving the token again

### Requirement: Fault injection

The simulator SHALL support seeded, configurable response latency and a timeout rate at which the outcome is recorded but the response is withheld past the caller's deadline.

#### Scenario: Injected timeout

- **WHEN** a sale is selected for timeout
- **THEN** its outcome is recorded, the caller receives no timely response, and a later status query returns the recorded outcome

### Requirement: Operation status query

The simulator SHALL return the recorded outcome for an operation key, or not-found when it never received the operation.

#### Scenario: Never received

- **WHEN** status is queried for a key the simulator never received
- **THEN** it returns not-found
