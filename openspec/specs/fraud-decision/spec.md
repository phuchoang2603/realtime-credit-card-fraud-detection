## Purpose

Specify the internal fraud decision RPC, its input validation, the stateless rules baseline and the versions reported with every decision.

## Requirements

### Requirement: Decision RPC

The fraud service SHALL expose `fraud.v2.FraudService/Decide` over internal gRPC. A request SHALL carry `integration_id`, `payment_id`, `attempt_id`, merchant and customer references, the checkout snapshot and the attempt signals. A response SHALL carry an `APPROVE` or `DECLINE` outcome, zero or more reason codes, the evaluation time and the policy version, and SHALL carry model version, feature-set version and risk score only when a model or feature set contributed. A decline SHALL be a successful response, not an error status.

#### Scenario: Rule decline

- **WHEN** a valid request matches a decline rule
- **THEN** the RPC returns OK with outcome `DECLINE` and the matching reason code

#### Scenario: Rules-only versions

- **WHEN** a decision is produced without a model
- **THEN** the response carries policy version `rules-v1` and omits model version, feature-set version and risk score

### Requirement: Decision input validation

The service SHALL reject with `INVALID_ARGUMENT` a request missing any identifier, the attempt time, the snapshot amount, shipping country, customer account creation time, or payment-method fingerprint, and a request whose snapshot is invalid. Absent optional signals SHALL NOT be replaced by defaults that could match a rule. Error messages SHALL NOT echo request values.

#### Scenario: Missing amount

- **WHEN** a request omits the snapshot amount
- **THEN** the service returns `INVALID_ARGUMENT` rather than treating the amount as zero

#### Scenario: Unsupported currency

- **WHEN** a request uses a currency the active policy does not define thresholds for
- **THEN** the service returns `INVALID_ARGUMENT`

### Requirement: Rules baseline policy

Policy `rules-v1` SHALL support USD and SHALL decline with `HIGH_AMOUNT` when the amount exceeds 2,000.00 USD. It SHALL decline with `NEW_ACCOUNT_GEO_MISMATCH` when the IP country is present and differs from the shipping country, the customer account is younger than 7 days at attempt time, and the amount exceeds 200.00 USD. Otherwise it SHALL approve. All matching reason codes SHALL be returned.

#### Scenario: High-amount boundary

- **WHEN** the amount is exactly 2,000.00 USD with no other rule matching
- **THEN** the outcome is `APPROVE`, and 2,000.01 USD yields `DECLINE` with `HIGH_AMOUNT`

#### Scenario: Account age boundary

- **WHEN** a geo-mismatched attempt above 200.00 USD comes from an account exactly 7 days old
- **THEN** `NEW_ACCOUNT_GEO_MISMATCH` does not match

#### Scenario: IP country absent

- **WHEN** the IP country is absent
- **THEN** `NEW_ACCOUNT_GEO_MISMATCH` does not match

### Requirement: Stateless deterministic decisions

A decision SHALL depend only on the request and the policy version, not on earlier requests or the serving instance, so that replicas and revisions with the same policy version return the same outcome for the same request.

#### Scenario: Repeated request

- **WHEN** the same request is decided twice by different instances with the same policy version
- **THEN** outcome and reason codes are identical
