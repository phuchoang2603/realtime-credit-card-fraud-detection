## Why

#44 needs live payment traffic that continues the historical dataset, plus configurable burst, late-arrival and duplicate delivery for the streaming work in #49. Strict streaming forbids fabricating payment events, so traffic must come through gateway commands and every delivery fault must be applied after Payments has committed events. Today committed events never leave Postgres, and nothing generates live traffic.

## What Changes

- Committed Payments events are published to a Kafka topic `payments.events.v1` in commit order, keyed by payment. This follows the refurbished-marketplace pattern: a Debezium outbox connector on Strimzi Kafka Connect reads the Payments `events` table from the WAL. Payments needs no Kafka code.
- Add a `live` simulator command. It rebuilds the profiles and scenario schedule of a historical dataset from its manifest, then drives clean gateway commands at a configured rate on wall-clock time. It publishes simulation-truth labels and compressed-delay chargeback labels to `fraud.labels.v1`. The client never injects faults on purpose.
- Add a `faults` command, a topic-to-topic fault injector from `payments.events.v1` to `payments.events.v1.faulty`. It applies burst holds, late delivery and duplicate delivery, each independently configured. Faults are selected by a seeded hash of the event ID, so the same events are affected on every run. The clean topic stays as ground truth.
- Deploy a `kafka` chart copied from refurbished-marketplace: a Strimzi KRaft cluster with one dual-role node pool, topics declared in values, Kafka Connect with a Debezium image, and the Payments outbox connector. Also add a simulator image and chart with the live simulator and the fault injector (dev only by default).
- Rewrite #44 around this scope. Drift moves to the final-coursework extension.

## Capabilities

### New Capabilities

- `payment-event-publication`: Ordered, at-least-once publication of committed payment events from the Payments event store to Kafka, without coupling checkout to the broker.
- `live-traffic-simulation`: Seeded live gateway traffic that continues a historical dataset's entities and scenarios, with live label delivery.
- `transport-fault-injection`: Deterministic burst, late and duplicate delivery between the clean and the faulty event topics, with observed rates.

### Modified Capabilities

- `talos-gitops-deployment`: Adds the Kafka cluster, Connect and outbox connector, topics, simulator workloads, and the Connect and simulator images.
- `focused-verification`: Adds fault-injection and dataset-continuation behavior tests.

## Impact

- `src/payments`: no code change. The gateway's CNPG cluster runs with `wal_level: logical`, and the `payments` role gets replication (amended in `build-minimum-payment-gateway`).
- `src/simulator`: `live` and `faults` commands (httpx, confluent-kafka), `configs/live-v1.yaml`.
- New `infra/charts/kafka` (`Kafka`, `KafkaNodePool`, `KafkaTopic`, `KafkaConnect`, `KafkaConnector`, secret-reader RBAC, `values-prod.yaml`), `infra/docker/connect-debezium.Dockerfile`, `infra/charts/simulator`, `infra/docker/simulator.Dockerfile`, `connect-debezium` and `simulator` image jobs, and app-of-apps children.
- Depends on `build-minimum-payment-gateway` (event store, logical replication, edge, deterministic identities, synthetic fingerprint key) and `generate-historical-synthetic-data` (reproducible profiles and manifest digest). Both changes are amended for this change.
- Platform prerequisite already owned by talos-proxmox: the Strimzi operator.
- Docs: `docs/verification/live-traffic.md`, roadmap #44 row, and the publication step in data-and-ML flows.
- Not included: Flink windows and deduplication (#49), drift scenarios (final coursework), Kafka authentication (#63), marketplace traffic (#41).
