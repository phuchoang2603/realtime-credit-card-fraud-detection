## Context

`build-minimum-payment-gateway` stores events in the Payments Postgres `events` table (`event_id`, `payment_id`, `event_type`, `payload bytea`), on a CNPG cluster with logical WAL. It also gives Accounts deterministic customer and merchant IDs, and gives synthetic integrations a non-secret fingerprint key. refurbished-marketplace already runs the target Kafka pattern on the same Talos clusters: a `kafka` chart with a Strimzi KRaft cluster, topics declared in values, Kafka Connect built from a Debezium image, and one EventRouter outbox connector per service, including a BYTEA protobuf outbox published with `ByteArrayConverter`. `generate-historical-synthetic-data` derives its random streams per stage and per day, and records a profile digest and the history end in its manifest. talos-proxmox provides the Strimzi operator. The architecture already classifies burst, lateness and duplicate delivery as transport conditions (data-and-ML flows).

## Goals / Non-Goals

**Goals:**

- One clean source of truth (`payments.events.v1`) and one faulty copy, so streaming correctness can be judged by comparing outputs from the two topics.
- Fault choices that are reproducible without controlling arrival timing.
- Live traffic whose entities and scenarios join with the historical dataset.

**Non-Goals:**

- Client-side faults (deliberate duplicate requests, backdated timestamps, client-driven bursts). Bursts for streaming come from the injector.
- Faults on the labels topic. Label lateness comes only from chargeback delays.
- Durable chargeback schedules, multiple simulator replicas, Kafka authentication, Connect tracing (application telemetry export is a later change).

## Decisions

### Kafka chart copied from refurbished-marketplace

`infra/charts/kafka` copies the marketplace chart's structure and its Strimzi settings, which have already been tuned for these clusters:

- `KafkaNodePool` `<cluster>-dual-role` (controller and broker), one replica, JBOD persistent claim with shared KRaft metadata, plus the marketplace's JVM sizes and observed-usage resource requests.
- `Kafka` in KRaft mode with node pools, Kafka 4.3.1 / metadata 4.3-IV0, internal `plain` (9092) and `tls` (9093) listeners, `auto.create.topics.enable: false` and replication factor 1. The topic operator runs with explicit resources and a non-empty spec, which keeps Argo CD server-side apply valid.
- `KafkaTopic` resources rendered from an `entities.<name>.topics` list in values. An entity's `connector` is optional, so topics without a source (faulty, labels) use the same list.
- `KafkaConnect` with `use-connector-resources`, the `KubernetesSecretConfigProvider`, replication factor −1 for internal topics and `metadata.recovery.strategy: none`. The marketplace's OTel tracing block is omitted until application telemetry is exported.
- A `Role` and `RoleBinding` letting the Connect service account `get` only the Payments database Secret.
- `values-prod.yaml` overrides only storage size and topic retention or partitions, like the marketplace's.

Deviations: the marketplace deploys to a dedicated `kafka` namespace so its Gateway policies don't intercept Kafka TLS. This repository has no such policies, and its spec keeps workloads in `payment-gateway`, so the chart deploys there. Cluster and Connect names are `payment-gateway-kafka` and `payment-gateway-connect`, so nothing collides with `ecommerce-kafka-cluster`. The Connect image (`infra/docker/connect-debezium.Dockerfile`) uses the same Strimzi base and Debezium version, with only the Postgres plugin.

| Topic | Key | Value | Partitions | Retention |
|---|---|---|---|---|
| `payments.events.v1` | `payment_id` | `PaymentEvent` protobuf | 3 | 7 days |
| `payments.events.v1.faulty` | `payment_id` | identical bytes | 3 | 7 days |
| `fraud.labels.v1` | `attempt_id` | `labels.v1` protobuf | 1 | 30 days |

The faulty topic has the same partition count as the clean one, so keys map to the same partitions. A 7-day retention on the clean topic allows replaying it into streaming jobs to compare results.

### Events table as the Debezium outbox

The `payments` entity's connector uses the marketplace's Postgres outbox template on the existing `events` table: `pgoutput`, slot `debezium_payments_slot`, `publication.autocreate.mode: filtered`, and `table.include.list: public.events`. EventRouter maps `table.field.event.id: event_id`, `table.field.event.key: payment_id` and `table.field.event.payload: payload`. `route.topic.replacement` is the constant `payments.events.v1`, so every event type lands in one topic. It uses `StringConverter` for keys and `ByteArrayConverter` for values, so the stored protobuf bytes are published unchanged. Snapshot mode stays at the default, so events committed before the connector existed are published too. `autoRestart` recovers from startup failures, for example the database not being ready yet.

The WAL gives commit order and never contains rolled-back appends. A payment's appends commit in version order because of the expected-version check, so per-payment order holds without the watermark logic a polling relay would need. Connect stores its source offsets, which gives at-least-once resumption.

Alternative considered: a publication worker inside Payments that polls by global position under a `pg_snapshot_xmin` watermark and keeps its own checkpoint. Rejected because it adds Kafka code, a checkpoint table and a leader election to Payments, while the marketplace already operates the Debezium path on these clusters. The template's extra columns (`publish_attempts`, `published_at`) belong to the marketplace's own outbox tables and aren't needed here.

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

`live` reads the dataset manifest from `synthetic-source` and rebuilds profiles and the scenario schedule with the generator's per-stage and per-day streams. It compares the profile digest and refuses to run on a mismatch, a start before history end, or a gap larger than `max_gap` (default 2 days). Evidence runs therefore generate a continuation dataset that ends on the run date. On start it provisions every profile merchant with the idempotent `PUT /v1/merchants`, so retries are harmless.

Arrivals follow a seeded Poisson process at `traffic.rate`. Each arrival samples a purchase from the rebuilt behavior model for the current simulated day, including active scenarios. The simulator then calls payment creation and one or more attempts. Each attempt carries a seeded idempotency key, simulated client context, and `tok_<card-id>_<behavior>` tokens. An asyncio loop with an httpx client and a concurrency limit keeps request latency from slowing the arrival process. When the gateway saturates, the shortfall shows up in the logged arrival lag instead of being silently dropped.

Labels: a truth label is produced after each attempt response. Chargebacks wait in an in-memory heap at `delay × label_time_scale`. When one is due, the simulator calls `GET /v1/payments/{id}` and publishes the chargeback only if that attempt succeeded.

### One configuration file

`configs/live-v1.yaml` has `dataset` (version, location), `traffic` (edge URL, rate, concurrency, `max_gap`, `label_time_scale`), `faults` (seed; `duplicate` and `late`, each with rate and delay range; `burst` with interval and hold; `max_pending`; report interval) and `kafka` (bootstrap servers, topic names). Pydantic validates it at startup. The integration key and S3 credentials come only from the environment.

### Deployment

One `simulator` image, with the `live` and `faults` commands as two Deployments in one chart. Each runs one replica with the `Recreate` strategy, so a rollout never runs two simulators at once. The chart is enabled in dev values and disabled in prod values. Both connect to `payment-gateway-kafka-kafka-bootstrap:9092`.

The app-of-apps gains `kafka` and `simulator` children next to the gateway workloads. Prod uses `infra/charts/kafka/values-prod.yaml`. Child sync retries absorb Strimzi CRDs, the database and the Payments Secret appearing in any order, as in the marketplace.

### Libraries

Python: confluent-kafka for consumer and producer with manual commits, httpx for async HTTP, and the existing protobuf bindings and pydantic. Kafka Connect: Strimzi `kafka:1.2.0-kafka-4.3.1` with the Debezium Postgres connector 3.7.0.Final, matching the marketplace.

## Risks / Trade-offs

- [Chargeback schedule lost on simulator restart] → Accepted for evidence runs. Truth labels are unaffected, and the restart time is logged.
- [Late delivery reorders a payment's events in the faulty topic] → Intended stress for #49. Consumers order by `aggregate_version`; the clean topic keeps commit order.
- [Restart duplicates add to injected duplicates] → The summary counts injected duplicates. Topic-level duplicate counts above that are attributed to restarts in the evidence.
- [Hash-based rates drift at small counts] → The summary reports observed rates next to configured ones.
- [Gateway throughput limits the live rate] → Streaming burst load comes from injector holds, not client rate.
- [Unauthenticated internal Kafka listener] → ClusterIP only, inside `payment-gateway`; authentication belongs to #63.
- [The replication slot retains WAL while Connect or Kafka is down] → Same exposure the marketplace accepts. Payments' CNPG volume sizing and slot lag are checked in the outage evidence, and a long outage is resolved by dropping and recreating the connector.
- [Single broker with replication factor 1] → Matches the marketplace's prod choice to save RAM. Kafka is a delivery copy, and Postgres keeps the authoritative events.
- [Generator changes invalidate old datasets for continuation] → The digest check fails fast. Generating a new continuation dataset takes minutes.

## Migration Plan

1. Deploy the Kafka application in dev. The connector snapshots existing events, then streams new ones.
2. Generate a continuation dataset ending on the run date, and store the synthetic integration key in Doppler.
3. Deploy the simulator chart. Record evidence with each fault enabled alone, then all together.

Rollback: disable the simulator chart, and delete the connector to stop publication. Committed events stay in Postgres. A recreated connector with a new slot snapshots again, so consumers see duplicates, not gaps.

## Open Questions

- Default traffic rate, partition counts and fault parameters can be tuned after the first dev run without changing behavior.
