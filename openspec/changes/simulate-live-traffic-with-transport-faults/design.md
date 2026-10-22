## Context

`build-minimum-payment-gateway` stores events in the Payments Postgres append-only `payment_events` ledger and publishes them through a Debezium outbox connector to `payments.events.v1` on a repository-owned Strimzi cluster in commit order, keyed by payment. It also gives Accounts deterministic customer and merchant IDs, and gives synthetic integrations a non-secret fingerprint key. refurbished-marketplace already runs the target Kafka pattern on the same Talos clusters: a `kafka` chart with a Strimzi KRaft cluster, topics declared in values, Kafka Connect built from a Debezium image, and one EventRouter outbox connector per service, including a BYTEA protobuf outbox published with `ByteArrayConverter`. `generate-historical-synthetic-data` derives its random streams per stage and per day, and records a profile digest and the history end in its manifest. talos-proxmox provides the Strimzi operator. The architecture already classifies burst, lateness and duplicate delivery as transport conditions (data-and-ML flows).

## Goals / Non-Goals

**Goals:**

- One clean source of truth (`payments.events.v1`) and one faulty copy, so streaming correctness can be judged by comparing outputs from the two topics.
- Fault choices that are reproducible without controlling arrival timing.
- Live traffic whose entities and scenarios join with the historical dataset.

**Non-Goals:**

- Client-side faults (deliberate duplicate requests, backdated timestamps, client-driven bursts). Bursts for streaming come from the injector.
- Faults on the labels topic. Label lateness comes only from chargeback delays.
- Durable chargeback schedules, multiple simulator replicas, Kafka authentication.

## Decisions

### Topics on the gateway's Kafka

`build-minimum-payment-gateway` owns `infra/charts/kafka`, the `payment-gateway-kafka` cluster, Connect and the outbox connector publishing `payments.events.v1`. This change only adds two connector-less entries to its `entities.<name>.topics` list:

| Topic | Key | Value | Partitions | Retention |
|---|---|---|---|---|
| `payments.events.v1` (existing) | `payment_id` | `PaymentEvent` protobuf | 3 | 7 days |
| `payments.events.v1.faulty` | `payment_id` | identical bytes | 3 | 7 days |
| `fraud.labels.v1` | `attempt_id` | `payments.v1.FraudLabel` protobuf | 1 | 30 days |

The faulty topic has the same partition count as the clean one, so keys map to the same partitions. The clean topic also feeds the gateway's webhook notifier; the injector uses its own consumer group and never writes to the clean topic, so webhooks are unaffected by injected faults.

### Topic-to-topic fault injector

The `faults` command consumes the clean topic in its own consumer group and runs a single scheduling loop:

```
consume record ──▶ parse event_id ──▶ schedule publications
                                        original: now, or now + late delay if selected
                                        copy:     now + duplicate delay if selected
due time inside a burst hold? ──▶ move it to the end of the hold
time-ordered queue ──▶ produce key/value unchanged ──▶ mark offset done
commit per partition = lowest offset not yet fully published
```

Selection uses `u(name) = first 8 bytes of SHA-256("{seed}:{name}:{event_id}") / 2^64`. A record is late when `u("late") < late.rate`, with delay `min + u("late-delay") × (max − min)`, and duplicate delivery works the same way. Hashing per event makes selection independent of arrival order and timing, so reruns affect the same events. Burst holds follow wall-clock time (`interval`, `hold`), since a burst describes time rather than particular events. Records keep their key, so they stay on the same partition. The producer stamps the publication time as the record timestamp, which is the delivery time streaming jobs observe.

Offset commits trail the oldest record not yet published, so a crash re-consumes and possibly republishes records but never loses one. When `max_pending` records are held, the consumer pauses until some are published, which bounds memory under long delays.

The injector logs one JSON summary per reporting interval: the configured settings, consumed, published, duplicated, delayed and burst-held counts, observed duplicate and late rates, and the maximum delay applied. Logs are the evidence channel until application metrics are collected.

Alternatives considered: fault transforms inside the outbox connector, which would put test behavior into the authoritative path and remove the clean ground truth; and a TCP proxy such as Toxiproxy, which cannot duplicate or delay individual records.

### Live simulator continues the historical dataset

`live` reads the dataset manifest from `synthetic-source` and rebuilds profiles and the scenario schedule with the generator's per-stage and per-day streams. It compares the profile digest and refuses to run on a mismatch, a start before history end, or a gap larger than `max_gap` (default 2 days). Evidence runs therefore generate a continuation dataset that ends on the run date.

Before arrivals start, the simulator tokenizes each fictional card it needs by posting its test card number to `POST /v1/payment-methods` and keeps the opaque tokens in a private run-local inventory. The gateway's allow-list maps the generator's card identity (`card-NNNNNN`) and behavior to one deterministic Luhn-valid number, so the vault's method fingerprint equals the historical fingerprint for that card. Do not publish or log token values or card numbers. On restart it may obtain new tokens for the same cards; fingerprints stay equal.

Arrivals follow a seeded Poisson process at `traffic.rate`. Each arrival samples a purchase from the rebuilt behavior model for the current simulated day, including active scenarios. The simulator then calls payment creation and one or more attempts. Payment creation carries a seeded idempotency key and the full snapshot; each attempt carries a seeded idempotency key, simulated client context and the token for its selected card. Keep the original token and request content unchanged on a transport retry. An asyncio loop with an httpx client and a concurrency limit keeps request latency from slowing the arrival process. When the gateway saturates, the shortfall shows up in the logged arrival lag instead of being silently dropped.

Labels: a truth label is produced after each attempt response. Chargebacks wait in an in-memory heap at `delay × label_time_scale`. When one is due, the simulator calls `GET /v1/payments/{id}` and publishes the chargeback only if that attempt succeeded.

### One configuration file

`configs/live-v1.yaml` has `dataset` (version, location), `traffic` (edge URL, rate, concurrency, `max_gap`, `label_time_scale`), `faults` (seed; `duplicate` and `late`, each with rate and delay range; `burst` with interval and hold; `max_pending`; report interval) and `kafka` (bootstrap servers, topic names). Pydantic validates it at startup. Integration and S3 credentials come only from runtime secret injection; never put token inventories or credentials in the dataset manifest.

### Deployment

One `simulator` image, with the `live` and `faults` commands as two Deployments in one chart. Each runs one replica with the `Recreate` strategy, so a rollout never runs two simulators at once. The chart is disabled by default and explicitly enabled for synthetic evidence runs through the configured prod root, then disabled after the run. Both connect to `payment-gateway-kafka-kafka-bootstrap:9092`. Only `live` receives the synthetic integration's API key and Edge network access; the fault injector does not.

The app-of-apps gains a `simulator` child next to the gateway's `kafka` application. Child sync retries absorb the topics and Secrets appearing in any order.

### Libraries

Python: confluent-kafka for consumer and producer with manual commits, httpx for async HTTP, and the existing protobuf bindings and pydantic.

## Risks / Trade-offs

- [Chargeback schedule lost on simulator restart] → Accepted for evidence runs. Truth labels are unaffected, and the restart time is logged.
- [Late delivery reorders a payment's events in the faulty topic] → Intended stress for #49. Consumers order by `aggregate_version`; the clean topic keeps commit order.
- [Restart duplicates add to injected duplicates] → The summary counts injected duplicates. Topic-level duplicate counts above that are attributed to restarts in the evidence.
- [Hash-based rates drift at small counts] → The summary reports observed rates next to configured ones.
- [Gateway throughput limits the live rate] → Streaming burst load comes from injector holds, not client rate.
- [Live traffic shares the clean topic with the notifier] → The synthetic integration has no webhook URL, so the notifier skips its events; live rate is bounded so it cannot starve the notifier on the single broker.
- [Generator changes invalidate old datasets for continuation] → The digest check fails fast. Generating a new continuation dataset takes minutes.

## Migration Plan

1. Add the two topics to the gateway's Kafka application through the configured prod root; no dev root is created.
2. Generate a continuation dataset ending on the run date and provision the synthetic integration's API key through Doppler.
3. Deploy the simulator chart. Record evidence with each fault enabled alone, then all together.

Rollback: disable the simulator chart. Publication to the clean topic is owned by the gateway and keeps running.

## Open Questions

- Default traffic rate, partition counts and fault parameters can be tuned after the first synthetic evidence run without changing behavior.
