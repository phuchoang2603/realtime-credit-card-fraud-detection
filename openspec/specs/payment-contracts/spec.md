## Purpose

Define the versioned wire contracts for payment identities, checkout snapshots, attempt signals, lifecycle events and fraud labels shared by generators, gateway services and data pipelines.

## Requirements

### Requirement: Integration-scoped identity

Every payment record SHALL carry the owning `integration_id`. Merchant and customer references SHALL pair the gateway-issued identifier with the integration's external seller or buyer identifier. Identifier values SHALL be opaque strings. Synthetic sources SHALL use integration identities dedicated to synthetic data that are never shared with a real integration.

#### Scenario: Same external buyer under two integrations

- **WHEN** two integrations submit the same external buyer identifier
- **THEN** their records carry different `integration_id` values and are distinct customers

#### Scenario: Synthetic data isolation

- **WHEN** synthetic and real records exist in the same store
- **THEN** every synthetic record is selectable by its dedicated synthetic integration identity alone

### Requirement: Immutable checkout snapshot

A checkout snapshot SHALL contain the external order reference, total amount, line items with product category, condition, quantity and unit price, merchant category and account creation time, customer account creation time, and shipping country, region, postal code and an address fingerprint. Amounts SHALL be integer minor units with an ISO 4217 currency code. The snapshot SHALL NOT contain buyer names, emails, street lines or raw payment credentials. A snapshot SHALL NOT change after the payment is created.

#### Scenario: Marketplace session maps to a snapshot

- **WHEN** a marketplace session request with order, buyer, merchant, items and shipping address is converted
- **THEN** every snapshot field is derived from it and personal fields are reduced to the address fingerprint

#### Scenario: Line items disagree with total

- **WHEN** the sum of item quantity times unit price differs from the total amount or currencies differ
- **THEN** the snapshot is invalid

### Requirement: Attempt runtime signals

A payment attempt SHALL record its attempt time, payment-method type and fingerprint, and the device fingerprint, IP country, IP network fingerprint and user-agent family observed by the gateway. Signals that the gateway could not observe SHALL be absent rather than empty or zero. Raw IP addresses and card numbers SHALL NOT appear in the contract.

#### Scenario: Unknown IP geography

- **WHEN** the gateway cannot resolve the IP country
- **THEN** the attempt omits the IP country and remains valid

### Requirement: Payment event envelope

Every payment event SHALL carry a unique `event_id`, the `payment_id` aggregate key, a per-payment `aggregate_version` starting at 1 and increasing by exactly 1, `integration_id`, `occurred_at`, `recorded_at`, a payload `schema_version`, `correlation_id` and `causation_id`, and exactly one lifecycle payload. Delivery or ingestion times SHALL NOT be part of the envelope.

#### Scenario: Repeated delivery

- **WHEN** a consumer receives an event whose `event_id` it already processed
- **THEN** the event is a duplicate delivery and does not represent a new transition

#### Scenario: Version gap

- **WHEN** a payment history skips or repeats an `aggregate_version` under different event identifiers
- **THEN** the history is invalid

### Requirement: Payment lifecycle transitions

A payment SHALL begin with exactly one `PaymentCreated` event and be `OPEN` until `PaymentSucceeded` or `PaymentExpired`, after which no events SHALL follow. While open, at most one attempt SHALL be active. An attempt SHALL progress `AttemptStarted` → `RiskDecisionRecorded`, then either `AttemptDeclined` with source `RISK`, or `ProcessorOperationRequested` followed by `PaymentSucceeded`, `AttemptDeclined` with source `PROCESSOR`, or `ProcessorOutcomeUnknown`. An unknown outcome SHALL resolve only to `PaymentSucceeded` or `AttemptDeclined` with source `PROCESSOR`. A new attempt SHALL start only after the previous attempt is declined. `PaymentExpired` SHALL NOT occur while an attempt awaits a processor outcome. `RiskDecisionRecorded` SHALL carry outcome, reason codes and policy, model and feature versions.

#### Scenario: Retry after risk decline

- **WHEN** an attempt is declined by risk and the payment is still open
- **THEN** a new attempt may start under the same payment

#### Scenario: Processor call after risk decline

- **WHEN** `ProcessorOperationRequested` follows a risk decline for the same attempt
- **THEN** the history is invalid

#### Scenario: Expiry during unknown outcome

- **WHEN** a payment would expire while its attempt has an unknown processor outcome
- **THEN** expiry is not a legal transition until that attempt resolves

### Requirement: Delayed fraud labels

A fraud label SHALL reference `integration_id`, `payment_id` and `attempt_id`, state whether the attempt was fraudulent, record `occurred_at` and `label_available_at`, and identify its source. Chargeback labels SHALL reference only attempts that reached `PaymentSucceeded`. Simulation-truth labels, including the generating scenario name, SHALL be produced only by synthetic sources. Labels SHALL be delivered separately from payment events and SHALL NOT appear in decision inputs.

#### Scenario: Label not yet available

- **WHEN** a training cutoff precedes a label's `label_available_at`
- **THEN** that label is not available for training at that cutoff

#### Scenario: Chargeback on declined attempt

- **WHEN** a chargeback label references an attempt that was declined
- **THEN** the label is invalid

### Requirement: Contract versioning

Contracts SHALL live under `contracts/<capability>/vN/`. Additive changes SHALL keep existing field numbers and meaning; changed semantics SHALL require a new package version. Package dependencies SHALL be acyclic so each package generates independently.

#### Scenario: Field added mid-history

- **WHEN** a new optional field is added to a payload
- **THEN** earlier records remain valid with the field absent
