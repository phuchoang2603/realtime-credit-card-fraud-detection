## Purpose

Apply configurable, reproducible delivery faults between the clean committed event topic and a faulty copy, so streaming jobs can be tested against burst, late and duplicate delivery while the authoritative history stays unchanged.

## ADDED Requirements

### Requirement: Faults only in the faulty copy

The fault injector SHALL consume `payments.events.v1` and publish to `payments.events.v1.faulty`. It SHALL forward each record's key and value byte-for-byte, so no event content, identity, `occurred_at` or `recorded_at` changes. It SHALL NOT write to the clean topic or to payment storage. With every fault disabled, the faulty topic SHALL contain each clean record exactly once.

#### Scenario: All faults disabled

- **WHEN** every fault rate and burst interval is zero
- **THEN** the faulty topic contains the same records as the clean topic, once each, in the same per-payment order

### Requirement: Configurable delivery faults

The injector SHALL support three independently configured faults:
- duplicate delivery publishes the record a second time after a delay within a configured range;
- late delivery holds the record for a delay within a configured range before publishing it;
- burst hold buffers all records during a configured hold period that repeats at a configured interval, then publishes them together.

Setting a rate or interval to zero SHALL disable that fault. Late delivery MAY reorder one payment's records in the faulty topic.

#### Scenario: Late event

- **WHEN** an event is selected for late delivery with a 10-minute delay
- **THEN** it appears in the faulty topic about 10 minutes after it was consumed, with its original `occurred_at`

#### Scenario: Burst hold

- **WHEN** the burst interval is 60 seconds and the hold is 15 seconds
- **THEN** records consumed during each hold appear in the faulty topic together when the hold ends

### Requirement: Deterministic selection

Selection for duplicate and late delivery, and the chosen delay, SHALL be derived from the configured fault seed, the fault name and the `event_id`. The same configuration SHALL affect the same events with the same delays on every run, regardless of arrival timing.

#### Scenario: Rerun on the same events

- **WHEN** the injector processes the same clean events twice with the same configuration
- **THEN** the same `event_id`s are duplicated and delayed by the same amounts

### Requirement: No loss on restart

The injector SHALL commit a consumer position only after every earlier record in that partition has been published, including held and delayed records. A restart MAY publish additional duplicates and SHALL NOT drop any record.

#### Scenario: Restart while records are delayed

- **WHEN** the injector stops while late records are held
- **THEN** after restart every held record still appears in the faulty topic

### Requirement: Configured and observed rates

The injector SHALL periodically report the configured settings with counts of records consumed, published, duplicated, delayed and held by bursts, the observed duplicate and late rates, and the maximum applied delay.

#### Scenario: Duplicate rate evidence

- **WHEN** the configured duplicate rate is 5% and 10,000 records have been consumed
- **THEN** the report shows about 500 duplicated records and an observed rate near 5%
