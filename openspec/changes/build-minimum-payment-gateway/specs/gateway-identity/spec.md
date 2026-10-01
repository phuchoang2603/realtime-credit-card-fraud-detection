## Purpose

Own integration authentication and the mapping of external sellers and buyers to integration-scoped gateway merchants and customers.

## ADDED Requirements

### Requirement: Integration authentication

Accounts SHALL authenticate an integration from its API key and return its `integration_id` and synthetic flag. Keys SHALL be stored only as hashes and SHALL be shown once when created. Unknown or disabled keys SHALL be rejected without revealing which condition applied. Integration identity SHALL come only from the credential, never from caller-supplied identifiers.

#### Scenario: Caller claims another integration

- **WHEN** a request authenticated as integration A names integration B's identifiers
- **THEN** it acts only within integration A

#### Scenario: Disabled key

- **WHEN** a disabled key is presented
- **THEN** authentication fails exactly as for an unknown key

### Requirement: Explicit merchant provisioning

Accounts SHALL create a merchant for `(integration_id, external_seller_id)` with category and account creation time. Repeating the same request SHALL return the same `merchant_id`. A repeated request with a different category or creation time SHALL be rejected as a conflict. Payments SHALL NOT be created for unprovisioned merchants.

#### Scenario: Repeated provisioning

- **WHEN** the same seller is provisioned twice with identical data
- **THEN** both calls return the same `merchant_id`

### Requirement: Lazy customer mapping

Accounts SHALL map `(integration_id, external_buyer_id)` to a stable `customer_id` on first use, recording the buyer's account creation time. Later lookups SHALL return the same `customer_id`.

#### Scenario: First payment for a buyer

- **WHEN** a payment names a buyer never seen for that integration
- **THEN** a customer is created and later payments reuse it

### Requirement: Synthetic integrations

Integrations SHALL be created as synthetic or real, and the flag SHALL NOT change afterwards. Synthetic integrations SHALL be the only ones permitted to supply simulated client context.

#### Scenario: Real integration sends simulated context

- **WHEN** a real integration submits simulated client context
- **THEN** the request is rejected
