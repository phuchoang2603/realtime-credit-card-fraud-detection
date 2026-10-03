## 1. Kafka chart and Connect image

- [ ] 1.1 Copy refurbished-marketplace's `infra/charts/kafka` (`_helpers.tpl`, `kafka-cluster.tpl`, `entities.tpl`, `connect.tpl`, `values.yaml`, `values-prod.yaml`), renamed to `payment-gateway-kafka`/`payment-gateway-connect` in `payment-gateway`, without the MongoDB connector branch or Connect tracing, with an optional per-entity `connector` and configurable event ID, key and fixed topic fields; verify `helm lint` and `helm template` for base and prod values render the node pool, cluster, three topics, Connect, one connector and one secret-reader Role
- [ ] 1.2 Add `infra/docker/connect-debezium.Dockerfile` (Strimzi `1.2.0-kafka-4.3.1` base, Debezium Postgres 3.7.0.Final only) and a `connect-debezium` job in `release.yml` using `build-image.yml`; verify the PR build does not publish
- [ ] 1.3 Register the `kafka` child application in the app-of-apps, with `values-prod.yaml` in the prod root; verify rendered names are environment-qualified and no operator or CRD resources are rendered

## 2. Outbox publication evidence

- [ ] 2.1 Deploy to dev and verify that the connector is `Ready`, that existing and new events of one payment appear on `payments.events.v1` in version order with values byte-equal to the stored payloads, and that the Connect service account cannot read the Accounts database Secret
- [ ] 2.2 Restart Connect during a burst of attempts, then scale Kafka to zero, submit an attempt and scale Kafka back; verify no committed event is missing from the topic, checkout succeeded during the outage, and slot lag returned to near zero afterwards

## 3. Simulator configuration and fault injector

- [ ] 3.1 Add httpx and confluent-kafka to `src/simulator`, the pydantic model for `configs/live-v1.yaml` (`dataset`, `traffic`, `faults`, `kafka`) and the `live`/`faults` CLI entry points; verify `uv sync --locked` and that an invalid delay range is rejected at startup
- [ ] 3.2 Implement hash-based selection and delays for duplicate and late delivery, plus the wall-clock burst hold; verify a test that two runs over the same event IDs select the same events with the same delays
- [ ] 3.3 Implement the consume, schedule and produce loop with unchanged key/value pass-through, per-partition commits trailing the oldest unpublished offset, the `max_pending` pause and the periodic JSON summary; verify the all-faults-disabled pass-through test and the restart-with-held-records test against a CI Kafka service container

## 4. Live simulator

- [ ] 4.1 Implement dataset continuation: read the manifest, rebuild profiles and the scenario schedule with the generator's per-stage and per-day streams, and refuse on digest mismatch, start before history end or gap above `max_gap`; verify the digest-mismatch refusal test
- [ ] 4.2 Implement merchant provisioning, the seeded Poisson arrival loop with bounded async concurrency, payment creation and attempts with seeded idempotency keys, simulated client context, synthetic tokens and retry with the same key only on transport errors or 503/504; verify against a local gateway that a forced edge timeout yields one recorded attempt
- [ ] 4.3 Implement truth labels and compressed chargebacks that check payment status when due, published to `fraud.labels.v1`; verify against a local gateway that a pending attempt gets no chargeback and a succeeded fraudulent attempt does

## 5. Image and deployment

- [ ] 5.1 Add `infra/docker/simulator.Dockerfile` and a `simulator` job in `release.yml` using `build-image.yml`, and add the Kafka service container to the CI Python job; verify the PR build does not publish and CI passes
- [ ] 5.2 Add `infra/charts/simulator` with `live` and `faults` Deployments (one replica, `Recreate`, ExternalSecret for the integration key), enabled in dev values and disabled in prod values; register it in the app-of-apps roots; verify `helm lint`/`helm template` show the workloads only in dev

## 6. Evidence and documentation

- [ ] 6.1 In dev, generate a continuation dataset ending on the run date, run the simulator, and verify a historical customer's live attempts carry the same `customer_id` and fingerprints as its historical events
- [ ] 6.2 Run the injector with duplicate, late and burst enabled one at a time, then together; record the configured and observed summaries, and verify that per-payment version order in the clean topic and the Postgres histories are unchanged
- [ ] 6.3 Add `docs/verification/live-traffic.md` with configuration, seeds, commands, summaries, outbox evidence and the label join by `attempt_id`; update the roadmap #44 row, the publication step in data-and-ML flows (Debezium outbox to Strimzi) and the gitops runtime table with the `kafka` and `simulator` applications; link the page from the sidebar; verify links resolve
- [ ] 6.4 Run `openspec validate simulate-live-traffic-with-transport-faults --strict` and verify it passes
