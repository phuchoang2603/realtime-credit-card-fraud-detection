## ADDED Requirements

### Requirement: Golden lifecycle histories

The contracts SHALL include versioned golden payment histories encoded as protobuf JSON, each marked valid or invalid with the violated rule. They SHALL cover every legal transition and at least each prohibited transition named by the lifecycle requirement. The Payments state machine SHALL accept all valid and reject all invalid golden histories.

#### Scenario: State machine conformance

- **WHEN** the golden histories are stepped through the Payments state machine's transition rules
- **THEN** it accepts each valid history and rejects each invalid history for its recorded rule

## MODIFIED Requirements

### Requirement: Immutable checkout snapshot

A checkout snapshot SHALL contain the external order reference, total amount, line items with product category, condition, quantity and unit price, merchant and customer account creation times, and shipping country, region, postal code and an address fingerprint. Amounts SHALL be integer minor units with an ISO 4217 currency code. The snapshot SHALL NOT contain buyer names, emails, street lines or raw payment credentials. A snapshot SHALL NOT change after the payment is created.

#### Scenario: Marketplace session maps to a snapshot

- **WHEN** a marketplace session request with order, buyer, merchant, items and shipping address is converted
- **THEN** every snapshot field is derived from it and personal fields are reduced to the address fingerprint

#### Scenario: Line items disagree with total

- **WHEN** the sum of item quantity times unit price differs from the total amount or currencies differ
- **THEN** the snapshot is invalid

### Requirement: Payment event envelope

Every payment event SHALL carry a unique `event_id`, the `payment_id` aggregate key, a per-payment `aggregate_version` starting at 1 and increasing by exactly 1, `integration_id`, `occurred_at`, `recorded_at`, a payload `schema_version`, `correlation_id`, and exactly one lifecycle payload. Causation SHALL be implied by `aggregate_version` order rather than carried as a field. Delivery or ingestion times SHALL NOT be part of the envelope.

#### Scenario: Repeated delivery

- **WHEN** a consumer receives an event whose `event_id` it already processed
- **THEN** the event is a duplicate delivery and does not represent a new transition

#### Scenario: Version gap

- **WHEN** a payment history skips or repeats an `aggregate_version` under different event identifiers
- **THEN** the history is invalid

### Requirement: Delayed fraud labels

A fraud label SHALL reference `integration_id`, `payment_id` and `attempt_id`, state whether the attempt was fraudulent, record `occurred_at` and `label_available_at`, and identify its source. Labels SHALL be defined in the `payments.v1` package next to the events they reference, and SHALL be delivered on their own stream or files, separately from payment events. Chargeback labels SHALL reference only attempts that reached `PaymentSucceeded`. Simulation-truth labels, including the generating scenario name, SHALL be produced only by synthetic sources. Labels SHALL NOT appear in decision inputs.

#### Scenario: Label not yet available

- **WHEN** a training cutoff precedes a label's `label_available_at`
- **THEN** that label is not available for training at that cutoff

#### Scenario: Chargeback on declined attempt

- **WHEN** a chargeback label references an attempt that was declined
- **THEN** the label is invalid

### Requirement: Contract versioning

Contracts SHALL live under `contracts/<package>/vN/` with one package per owning boundary: `payments/v1` (shared payment types, risk assessment, lifecycle events, fraud labels and the Edge-to-Payments service), `fraud/v1` (the decision RPC), `vault/v1` and `processor/v1`. The risk decision SHALL have one message, `payments.v1.RiskAssessment`, used by both the fraud RPC response and `RiskDecisionRecorded`. Field numbers SHALL be assigned fresh in this change with no compatibility with earlier drafts; after it, additive changes SHALL keep existing field numbers and meaning, and changed semantics SHALL require a new package version. Package dependencies SHALL be acyclic so each package generates independently, and the protobuf fields SHALL match the historical-payments data contract.

#### Scenario: Field added mid-history

- **WHEN** a new optional field is added to a payload
- **THEN** earlier records remain valid with the field absent

#### Scenario: Protobuf matches the data contract

- **WHEN** a generated historical event row is parsed as `payments.v1.PaymentEvent` protobuf JSON
- **THEN** every field maps to a declared field and no `causation_id` or merchant category exists
