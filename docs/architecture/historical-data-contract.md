# urn:datacontract:fraud:historical-payments
## Info
*Complete synthetic payment histories and separately delivered fraud labels for reproducible pipeline development.*
- **name:** Historical synthetic payments
- **version:** 1.0.0
- **status:** active
- **team:** Fraud data platform

## Terms of Use
### Usage
Validate source datasets before publication and ingestion. Deduplicate identical event deliveries before interpreting lifecycle transitions.

### Purpose
Complete synthetic payment histories and separately delivered fraud labels for reproducible pipeline development.

### Limitations
Synthetic data only. Scenario truth and future outcomes must never become decision inputs. Labels may only be used after label_available_at. Empty configured schedules are valid.

## Servers
| Name | Type | Attributes |
| ---- | ---- | ---------- |
| validation | duckdb | *An ephemeral database of staged Parquet views supplied by the shared validator.*<br />• **environment:** local<br />• **database:** validation.duckdb |

## Schema
### payment_events
*One authoritative transition per event_id; repeated identical deliveries are permitted. Histories are complete across all date partitions.*

| Field | Type | Attributes |
| ----- | ---- | ---------- |
|  event_id | string | *Opaque event identity. Repeated deliveries retain this identity and identical content.*<br />• `required` |
|  payment_id | string | *Payment aggregate identity within integration_id.*<br />• `required` |
|  aggregate_version | integer | *Consecutive authoritative transition number starting at one; delivery duplicates do not increment it.*<br />• `required` |
|  integration_id | string | *Dedicated synthetic integration namespace; never share it with a real integration.*<br />• `required` |
|  occurred_at | timestamp | *Business occurrence time in UTC at microsecond precision.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
|  recorded_at | timestamp | *Source recording time, at or after occurred_at; not ingestion time.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
|  schema_version | integer | *Physical event schema. Version one omits user_agent_family; version two permits it.*<br />• `required` |
|  correlation_id | string | *Correlation identity shared across one payment history.*<br />• `required` |
|  causation_id | string | *Cause identity; every event after creation references the preceding authoritative event.*<br />• `required` |
|  payment_created | object | *Initial immutable checkout snapshot. Exactly one lifecycle payload is present per event.* |
| &#x2007;&#x21b3; merchant | object | *Integration-scoped merchant references.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; merchant_id | string | *Gateway merchant identity.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; external_seller_id | string | *Integration seller identity.*<br />• `required` |
| &#x2007;&#x21b3; customer | object | *Integration-scoped customer references.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; customer_id | string | *Gateway customer identity.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; external_buyer_id | string | *Integration buyer identity.*<br />• `required` |
| &#x2007;&#x21b3; snapshot | object | *Checkout facts captured at creation; raw personal and payment credentials are excluded.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; external_order_id | string | *Integration order reference.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; total | object | *Total in integer minor units; must equal the item sum.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; minor_units | integer | *Nonnegative integer amount in currency minor units.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; currency | string | *Three-letter uppercase currency code shared by all items.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; items | array | *Nonempty checkout line items.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; items | object | *No description.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; product_id | string | *Product reference.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; category | string | *Product category.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; condition | string | *Product condition.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; quantity | integer | *Positive item quantity.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; unit_price | object | *Price for one item in checkout currency.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; minor_units | integer | *Nonnegative integer amount in currency minor units.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x2007;&#x2007;&#x21b3; currency | string | *Three-letter uppercase currency code shared by all items.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; merchant_category | string | *Merchant category captured at checkout.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; merchant_account_created_at | timestamp | *Merchant account creation time.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
| &#x2007;&#x2007;&#x21b3; customer_account_created_at | timestamp | *Customer account creation time.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
| &#x2007;&#x2007;&#x21b3; shipping | object | *Coarse shipping geography and synthetic address fingerprint.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; country | string | *Shipping country code.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; region | string | *Shipping region.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; postal_code | string | *Shipping postal code.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; address_fingerprint | string | *Normalized keyed synthetic address fingerprint; no street lines.*<br />• `required` |
|  attempt_started | object | *Starts a new nonconcurrent attempt on an open payment.* |
| &#x2007;&#x21b3; attempt_id | string | *Attempt identity unique within this payment.*<br />• `required` |
| &#x2007;&#x21b3; signals | object | *Runtime observations; unavailable optional signals remain null.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; attempted_at | timestamp | *Attempt occurrence time.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
| &#x2007;&#x2007;&#x21b3; payment_method | object | *Method category and keyed synthetic token fingerprint.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; type | string | *Payment method category.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; fingerprint | string | *Synthetic method fingerprint; never raw credentials.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; device_fingerprint | string | *Observed synthetic device fingerprint.* |
| &#x2007;&#x2007;&#x21b3; ip_country | string | *Resolved IP country if available.* |
| &#x2007;&#x2007;&#x21b3; ip_network_fingerprint | string | *Keyed network fingerprint; never raw IP.* |
| &#x2007;&#x2007;&#x21b3; user_agent_family | string | *Optional browser family introduced in physical schema two.*<br />• **customProperties:** [{'property': 'sinceSchemaVersion', 'value': 2}] |
|  risk_decision_recorded | object | *Exactly one assessment for the active attempt before any processor request.* |
| &#x2007;&#x21b3; attempt_id | string | *Active attempt identity.*<br />• `required` |
| &#x2007;&#x21b3; assessment | object | *Decision evidence. Optional model and feature versions are absent for rules-only decisions.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; outcome | string | *RISK_OUTCOME_APPROVE or RISK_OUTCOME_DECLINE.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; reason_codes | array | *Nonempty policy reason codes.*<br />• `required` |
| &#x2007;&#x2007;&#x2007;&#x21b3; items | string | *No description.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; policy_version | string | *Evaluated policy version.*<br />• `required` |
| &#x2007;&#x2007;&#x21b3; model_version | string | *Evaluated model version if a model was used.* |
| &#x2007;&#x2007;&#x21b3; feature_set_version | string | *Evaluated feature-set version if features were used.* |
| &#x2007;&#x2007;&#x21b3; risk_score | number | *Optional model risk score.* |
|  processor_operation_requested | object | *Processor authorization request after risk approval.* |
| &#x2007;&#x21b3; attempt_id | string | *Active approved attempt identity.*<br />• `required` |
| &#x2007;&#x21b3; operation_key | string | *Idempotent processor operation identity.*<br />• `required` |
|  processor_outcome_unknown | object | *Pending processor result; only processor decline or success can resolve it.* |
| &#x2007;&#x21b3; attempt_id | string | *Active requested attempt identity.*<br />• `required` |
|  attempt_declined | object | *Risk or processor decline that permits a subsequent attempt.* |
| &#x2007;&#x21b3; attempt_id | string | *Active declined attempt identity.*<br />• `required` |
| &#x2007;&#x21b3; source | string | *DECLINE_SOURCE_RISK or DECLINE_SOURCE_PROCESSOR.*<br />• `required` |
| &#x2007;&#x21b3; reason | string | *Decline reason.*<br />• `required` |
|  payment_succeeded | object | *Terminal successful processor result.* |
| &#x2007;&#x21b3; attempt_id | string | *Successful requested attempt identity.*<br />• `required` |
| &#x2007;&#x21b3; processor_reference | string | *Synthetic processor result reference.*<br />• `required` |
|  payment_expired | object | *Terminal expiry with no pending processor result.* |
| &#x2007;&#x21b3; _present | boolean | *True marker for the otherwise empty expiry payload.*<br />• `required` |
### fraud_labels
*Separate synthetic attempt truth and delayed chargebacks. Never join future labels into features.*

| Field | Type | Attributes |
| ----- | ---- | ---------- |
|  integration_id | string | *Same dedicated synthetic namespace as the referenced payment.*<br />• `required` |
|  payment_id | string | *Referenced payment aggregate identity.*<br />• `required` |
|  attempt_id | string | *Referenced started attempt identity.*<br />• `required` |
|  is_fraud | boolean | *Fraud truth according to this source; friendly-fraud chargebacks may differ from simulation truth.*<br />• `required` |
|  occurred_at | timestamp | *Time of the labeled business occurrence.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
|  label_available_at | timestamp | *Earliest time this label may be used by training or evaluation.*<br />• **logicalTypeOptions:** {'timezone': True, 'defaultTimezone': 'UTC'}<br />• `required` |
|  source | string | *LABEL_SOURCE_SIMULATION_TRUTH or LABEL_SOURCE_CHARGEBACK.*<br />• `required` |
|  scenario | string | *Synthetic generating scenario on simulation truth only; excluded from features.* |

## SLA Properties


## Contract provenance

ODCS: v3.1.0; SHA-256: `dbb1632092910d61bddc1f285da2c4b170e285dd98fc0b20bb0bfb77a888ea0b`.

## Ownership and governance

```yaml
tags:
- synthetic
- source
- payments
team:
  name: Fraud data platform
  members:
  - username: phuchoang2603
    role: owner
customProperties:
- property: classification
  value: synthetic
- property: retentionPolicy
  value: Immutable versioned inputs; delete only through an explicit dataset cleanup
    operation.
- property: sqlDialect
  value: duckdb
```

## Physical schema versions

- `payment_events`: [1, 2].
- `fraud_labels`: [1].

Fields with `sinceSchemaVersion` are physically absent before that version.

## Quality rules

### payment_delivery_consistency

```yaml
id: payment_delivery_consistency
description: Repeated event identities must carry identical content.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM (
    SELECT event_id FROM {object} e
    GROUP BY event_id HAVING count(DISTINCT to_json(e)) > 1
  )
```

### payment_envelope

```yaml
id: payment_envelope
description: Envelope identities are nonempty, timestamps are ordered, schemas are
  supported and exactly one lifecycle payload is present.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM {object}
  WHERE event_id = '' OR payment_id = '' OR integration_id = ''
    OR correlation_id = '' OR causation_id = ''
    OR schema_version NOT IN (1, 2) OR aggregate_version < 1
    OR recorded_at < occurred_at
    OR (CAST(payment_created IS NOT NULL AS INTEGER)
      + CAST(attempt_started IS NOT NULL AS INTEGER)
      + CAST(risk_decision_recorded IS NOT NULL AS INTEGER)
      + CAST(processor_operation_requested IS NOT NULL AS INTEGER)
      + CAST(processor_outcome_unknown IS NOT NULL AS INTEGER)
      + CAST(attempt_declined IS NOT NULL AS INTEGER)
      + CAST(payment_succeeded IS NOT NULL AS INTEGER)
      + CAST(payment_expired IS NOT NULL AS INTEGER)) <> 1
    OR (payment_expired IS NOT NULL AND NOT payment_expired._present)
```

### payment_lifecycle

```yaml
id: payment_lifecycle
description: Complete canonical histories have consecutive versions, legal transitions,
  consistent attempt identities, correlation and causation.
type: sql
mustBe: 0
query: |
  WITH canonical AS (SELECT DISTINCT * FROM {object}),
  typed AS (
    SELECT *,
      CASE
        WHEN payment_created IS NOT NULL THEN 'created'
        WHEN attempt_started IS NOT NULL THEN 'started'
        WHEN risk_decision_recorded IS NOT NULL THEN 'decision'
        WHEN processor_operation_requested IS NOT NULL THEN 'requested'
        WHEN processor_outcome_unknown IS NOT NULL THEN 'unknown'
        WHEN attempt_declined IS NOT NULL THEN 'declined'
        WHEN payment_succeeded IS NOT NULL THEN 'succeeded'
        WHEN payment_expired IS NOT NULL THEN 'expired'
      END AS kind,
      coalesce(attempt_started.attempt_id, risk_decision_recorded.attempt_id,
        processor_operation_requested.attempt_id, processor_outcome_unknown.attempt_id,
        attempt_declined.attempt_id, payment_succeeded.attempt_id) AS attempt
    FROM canonical
  ),
  ordered AS (
    SELECT *,
      row_number() OVER history AS ordinal,
      count(*) OVER (PARTITION BY integration_id, payment_id) AS transitions,
      lag(kind) OVER history AS previous_kind,
      lag(event_id) OVER history AS previous_event,
      lag(occurred_at) OVER history AS previous_time,
      lag(correlation_id) OVER history AS previous_correlation,
      lag(risk_decision_recorded.assessment.outcome) OVER history AS previous_outcome,
      last_value(attempt_started.attempt_id IGNORE NULLS) OVER history AS active_attempt
    FROM typed
    WINDOW history AS (
      PARTITION BY integration_id, payment_id ORDER BY aggregate_version, event_id
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )
  )
  SELECT count(*) FROM ordered
  WHERE aggregate_version <> ordinal
    OR (ordinal = transitions AND kind NOT IN ('succeeded', 'expired'))
    OR (ordinal > 1 AND (causation_id <> previous_event
      OR correlation_id <> previous_correlation OR occurred_at < previous_time))
    OR (kind NOT IN ('created', 'expired') AND
      (attempt IS NULL OR attempt = '' OR attempt IS DISTINCT FROM active_attempt))
    OR NOT coalesce(CASE kind
      WHEN 'created' THEN ordinal = 1
      WHEN 'started' THEN previous_kind IN ('created', 'declined')
      WHEN 'decision' THEN previous_kind = 'started'
      WHEN 'requested' THEN previous_kind = 'decision'
        AND previous_outcome = 'RISK_OUTCOME_APPROVE'
      WHEN 'unknown' THEN previous_kind = 'requested'
      WHEN 'declined' THEN
        (previous_kind = 'decision' AND previous_outcome = 'RISK_OUTCOME_DECLINE'
          AND attempt_declined.source = 'DECLINE_SOURCE_RISK')
        OR (previous_kind IN ('requested', 'unknown')
          AND attempt_declined.source = 'DECLINE_SOURCE_PROCESSOR')
      WHEN 'succeeded' THEN previous_kind IN ('requested', 'unknown')
      WHEN 'expired' THEN previous_kind IN ('created', 'started', 'decision', 'declined')
      ELSE false END, false)
```

### payment_attempt_identity

```yaml
id: payment_attempt_identity
description: Each canonical attempt identity starts once within its scoped payment.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM (
    SELECT integration_id, payment_id, attempt_started.attempt_id
    FROM (SELECT DISTINCT * FROM {object})
    WHERE attempt_started IS NOT NULL
    GROUP BY ALL HAVING count(*) > 1
  )
```

### payment_risk_assessment

```yaml
id: payment_risk_assessment
description: Decisions declare an approve or decline outcome, policy version and nonempty
  reason codes.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM {object}
  WHERE risk_decision_recorded IS NOT NULL AND (
    risk_decision_recorded.assessment.outcome NOT IN ('RISK_OUTCOME_APPROVE', 'RISK_OUTCOME_DECLINE')
    OR risk_decision_recorded.assessment.policy_version = ''
    OR len(risk_decision_recorded.assessment.reason_codes) = 0
    OR list_contains(risk_decision_recorded.assessment.reason_codes, '')
  )
```

### payment_snapshot_amounts

```yaml
id: payment_snapshot_amounts
description: Checkout item quantities and amounts are valid and sum to the total in
  the same currency.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM {object} e
  WHERE payment_created IS NOT NULL AND (
    payment_created.snapshot.total.minor_units < 0
    OR NOT regexp_full_match(payment_created.snapshot.total.currency, '[A-Z]{3}')
    OR len(payment_created.snapshot.items) = 0
    OR payment_created.snapshot.total.minor_units IS DISTINCT FROM (
      SELECT sum(CAST(item.quantity AS HUGEINT) * item.unit_price.minor_units)
      FROM unnest(e.payment_created.snapshot.items) AS entries(item)
    )
    OR EXISTS (
      SELECT 1 FROM unnest(e.payment_created.snapshot.items) AS entries(item)
      WHERE item.quantity <= 0 OR item.unit_price.minor_units < 0
        OR item.unit_price.currency IS DISTINCT FROM e.payment_created.snapshot.total.currency
    )
  )
```

### label_availability

```yaml
id: label_availability
description: Labels cannot be available before their occurrence and carry a recognized
  source.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM {object}
  WHERE label_available_at < occurred_at
    OR source NOT IN ('LABEL_SOURCE_SIMULATION_TRUTH', 'LABEL_SOURCE_CHARGEBACK')
    OR (source = 'LABEL_SOURCE_SIMULATION_TRUTH' AND coalesce(scenario, '') = '')
    OR (source = 'LABEL_SOURCE_CHARGEBACK' AND (NOT is_fraud OR scenario IS NOT NULL))
```

### label_attempt_linkage

```yaml
id: label_attempt_linkage
description: Every label references a started attempt in the same integration and
  payment; chargebacks require that attempt to succeed.
type: sql
mustBe: 0
query: |
  SELECT count(*) FROM {object} label
  WHERE NOT EXISTS (
    SELECT 1 FROM payment_events event
    WHERE event.integration_id = label.integration_id AND event.payment_id = label.payment_id
      AND event.attempt_started.attempt_id = label.attempt_id
  ) OR (label.source = 'LABEL_SOURCE_CHARGEBACK' AND NOT EXISTS (
    SELECT 1 FROM payment_events event
    WHERE event.integration_id = label.integration_id AND event.payment_id = label.payment_id
      AND event.payment_succeeded.attempt_id = label.attempt_id
  ))
```
