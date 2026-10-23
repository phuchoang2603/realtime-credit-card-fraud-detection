## Purpose

Generate seeded live checkout traffic through the gateway that continues a historical dataset's customers, merchants, devices, payment methods and fraud scenarios, and deliver its labels separately.

## ADDED Requirements

### Requirement: Continue a historical dataset

The live simulator SHALL take a historical dataset version and rebuild that dataset's profiles and scenario schedule from its manifest configuration and seed, without reading its event files. It SHALL refuse to start when the rebuilt profiles do not match the manifest's profile digest, when the current time is before the dataset's history end, or when the gap after history end exceeds the configured maximum. Simulated days SHALL continue the historical calendar on wall-clock time.

#### Scenario: Generator logic changed since the dataset

- **WHEN** the rebuilt profile digest differs from the manifest
- **THEN** the simulator exits nonzero before sending any request

#### Scenario: Same customer across the handoff

- **WHEN** a customer from the historical dataset purchases live
- **THEN** the gateway records the same `customer_id`, device fingerprint, address fingerprint and payment-method fingerprints as in the historical events

### Requirement: Clean gateway commands only

The simulator SHALL create payment traffic only through the checkout API of a synthetic integration: test-card tokenization, payment creation with the full snapshot, and attempt submission with simulated client context and the returned opaque tokens.

#### Scenario: Complete clean gateway commands only contract

- **WHEN** the `Clean gateway commands only` capability is configured or exercised
- **THEN** The simulator SHALL create payment traffic only through the checkout API of a synthetic integration: test-card tokenization, payment creation with the full snapshot, and attempt submission with simulated client context and the returned opaque tokens. It SHALL NOT call the vault, Payments, Fraud or processor internal APIs directly. Token values SHALL NOT be seeded, logged, published or placed in dataset manifests. Each payment creation and attempt SHALL carry a unique `Idempotency-Key`; transport or 503/504 retries SHALL reuse the same key, token and request content. The simulator SHALL NOT write events or inject delivery faults. Arrivals SHALL follow the configured rate, and purchase and scenario choices SHALL be seeded.

#### Scenario: Edge timeout

- **WHEN** an attempt request times out
- **THEN** the simulator retries it with the same `Idempotency-Key` and the gateway records at most one attempt

#### Scenario: Opaque tokens preserve dataset continuation

- **WHEN** the simulator tokenizes a historical card's test card number and receives a new opaque token
- **THEN** its live method fingerprint equals the historical fingerprint, without exposing token behavior to Fraud

### Requirement: Live labels

The simulator SHALL publish a simulation-truth label for every attempt to `fraud.labels.v1` once its response is received.

#### Scenario: Complete live labels contract

- **WHEN** the `Live labels` capability is configured or exercised
- **THEN** The simulator SHALL publish a simulation-truth label for every attempt to `fraud.labels.v1` once its response is received. For fraudulent attempts and for `friendly_fraud`, it SHALL schedule a chargeback label with the configured delay multiplied by the configured time scale. When the chargeback is due, the simulator SHALL publish it only if the payment status shows that attempt succeeded. Labels SHALL use the `payments.v1.FraudLabel` contract with `label_available_at` set to the publication time. Scenario names SHALL appear only in labels.

#### Scenario: Pending attempt at chargeback time

- **WHEN** a chargeback becomes due for an attempt whose payment is still pending
- **THEN** no chargeback label is published for it

#### Scenario: Compressed chargeback delay

- **WHEN** the configured delay is 7 days and the time scale is 1/1440
- **THEN** the chargeback label is published about 7 minutes after the attempt succeeded
