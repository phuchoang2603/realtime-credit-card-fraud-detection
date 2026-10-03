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

The simulator SHALL create traffic only through the public checkout API of a synthetic integration: merchant provisioning, payment creation and attempt submission with simulated client context and synthetic tokens. Each attempt SHALL carry a unique `Idempotency-Key`. A request SHALL be retried only after a transport error or 503/504 response, and only with the same key. The simulator SHALL NOT write events, call internal services or inject delivery faults. Arrivals SHALL follow the configured rate, and purchase and scenario choices SHALL be seeded.

#### Scenario: Edge timeout

- **WHEN** an attempt request times out
- **THEN** the simulator retries it with the same `Idempotency-Key` and the gateway records at most one attempt

### Requirement: Live labels

The simulator SHALL publish a simulation-truth label for every attempt to `fraud.labels.v1` once its response is received. For fraudulent attempts and for `friendly_fraud`, it SHALL schedule a chargeback label with the configured delay multiplied by the configured time scale. When the chargeback is due, the simulator SHALL publish it only if the payment status shows that attempt succeeded. Labels SHALL use the `labels.v1` contract with `label_available_at` set to the publication time. Scenario names SHALL appear only in labels.

#### Scenario: Pending attempt at chargeback time

- **WHEN** a chargeback becomes due for an attempt whose payment is still pending
- **THEN** no chargeback label is published for it

#### Scenario: Compressed chargeback delay

- **WHEN** the configured delay is 7 days and the time scale is 1/1440
- **THEN** the chargeback label is published about 7 minutes after the attempt succeeded
