## Purpose

Publish every committed payment event from the authoritative Payments event store to Kafka, in commit order and without ever changing stored history.

## ADDED Requirements

### Requirement: Ordered publication of committed events

Each committed `payments.v1` event SHALL be published to `payments.events.v1` with the stored serialized `PaymentEvent` bytes as the value, keyed by `payment_id`. Events SHALL be published in commit order and only after their transaction commits. Events of one payment SHALL appear on the topic in `aggregate_version` order. Publication SHALL NOT modify, reorder or add events in the event store, and payment events SHALL NOT be routed to any other topic.

#### Scenario: Rolled-back append

- **WHEN** a transaction appending events rolls back
- **THEN** none of its events appear on `payments.events.v1`

#### Scenario: One payment's history

- **WHEN** a payment's events are read from `payments.events.v1`
- **THEN** they are in one partition, and after removing republished duplicates their versions increase by exactly 1 and each value equals the stored payload

### Requirement: At-least-once resumption

After a publisher restart, publication SHALL resume after the last recorded position. A restart MAY republish events with the same `event_id`, and SHALL NOT skip any committed event.

#### Scenario: Publisher restart

- **WHEN** the publisher stops while events are being committed and later restarts
- **THEN** every committed event appears on the topic at least once, in per-payment version order

### Requirement: Broker outage does not block payments

Payment commands SHALL NOT depend on broker or publisher availability. While either is unavailable, events SHALL remain committed, and publication SHALL catch up once both return.

#### Scenario: Kafka unavailable during checkout

- **WHEN** an attempt is submitted while Kafka is down
- **THEN** the attempt completes normally and its events are published after Kafka recovers
