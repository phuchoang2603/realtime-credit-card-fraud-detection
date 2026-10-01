## Purpose

Own authoritative payment state as an append-only event stream, orchestrating fraud decisions and processor effects with durable idempotency and recovery.

## ADDED Requirements

### Requirement: Event-store append semantics

Payments SHALL persist each payment as the ordered `payments.v1` event stream and SHALL append only with the expected current version, rejecting stale writers. Each committed event SHALL receive a monotonically increasing global commit position. Status views SHALL be rebuildable from events alone, and rebuilding SHALL NOT call fraud or the processor.

#### Scenario: Concurrent writers

- **WHEN** two commands append to the same payment from the same version
- **THEN** exactly one commits and the other retries against the new state or fails

#### Scenario: Rebuild status

- **WHEN** the status view is discarded and rebuilt
- **THEN** it equals the view before discarding, and no external call occurs

### Requirement: Idempotent payment creation

Creating a payment SHALL be idempotent per `(integration_id, merchant_id, order_reference)`. A repeat with the same snapshot SHALL return the original payment, session, checkout token and expiry. A repeat with a different snapshot SHALL be a conflict. A new payment SHALL expire at a configured time after creation.

#### Scenario: Changed snapshot under same order

- **WHEN** a repeat creation changes the amount
- **THEN** it is rejected as a conflict and the original payment is unchanged

### Requirement: Scoped checkout access

Each session SHALL have a checkout token that authorizes attempts only for that session until expiry. The token SHALL be verifiable without storing it in plaintext.

#### Scenario: Token for another session

- **WHEN** an attempt presents a valid token issued for a different session
- **THEN** it is rejected

### Requirement: Idempotent attempts

Starting an attempt SHALL require an idempotency key scoped to the session. A repeated key with the same content SHALL return the recorded result without new events. A repeated key with different content SHALL be a conflict. An attempt SHALL be refused when the payment is not open or another attempt is active.

#### Scenario: Retried submission

- **WHEN** the same attempt request is sent twice
- **THEN** one attempt and one set of events exist, and both calls return the same result

### Requirement: Risk decision orchestration

Each attempt SHALL request a `fraud.v2` decision within a configured deadline and record it before any processor call. A `DECLINE` SHALL end the attempt with a risk decline. When fraud is unavailable, times out or errors, Payments SHALL fail closed: it SHALL record a `DECLINE` assessment with reason `FRAUD_UNAVAILABLE` and fallback policy version `payments-fallback-v1`, and SHALL NOT call the processor.

#### Scenario: Fraud unavailable

- **WHEN** the fraud call times out
- **THEN** the attempt is declined by risk with `FRAUD_UNAVAILABLE` and no processor operation is requested

### Requirement: Processor effect intent and unknown outcomes

Payments SHALL record `ProcessorOperationRequested` with a stable operation key before calling the processor and SHALL reuse that key on every retry. A confirmed result SHALL be recorded as success or processor decline. A timeout or lost response SHALL be recorded as `ProcessorOutcomeUnknown` and SHALL NOT be treated as a decline.

#### Scenario: Processor timeout

- **WHEN** the processor does not answer within the deadline
- **THEN** the attempt records an unknown outcome and the caller receives a pending status

### Requirement: Recovery and expiry workers

A recovery worker SHALL find attempts with unknown outcomes, and requested operations without a recorded outcome past a grace period. It SHALL query the processor by operation key and record the observed result. Where the processor has no record of the operation, it SHALL resubmit with the same key. An expiry worker SHALL record `PaymentExpired` for open payments past expiry with no attempt awaiting a processor outcome. Workers SHALL be safe to run concurrently and repeatedly.

#### Scenario: Crash after intent

- **WHEN** Payments stops after recording an operation request but before calling the processor
- **THEN** recovery submits that operation once with the same key and records its result

#### Scenario: Late success during pending expiry

- **WHEN** a payment is past expiry while its attempt outcome is unknown
- **THEN** it does not expire until recovery resolves the attempt, and a resolved success yields `PaymentSucceeded`

### Requirement: Payment status query

Payments SHALL return a payment's status, current version and attempt summaries only to its owning integration.

#### Scenario: Other integration reads status

- **WHEN** integration B requests integration A's payment
- **THEN** the payment is reported as not found
