## ADDED Requirements

### Requirement: Streaming workload deployment

The gateway's Kafka application SHALL additionally declare `payments.events.v1.faulty` and `fraud.labels.v1` with configured partitions and retention, without restoring a dev root.

#### Scenario: Complete streaming workload deployment contract

- **WHEN** the `Streaming workload deployment` capability is configured or exercised
- **THEN** The gateway's Kafka application SHALL additionally declare `payments.events.v1.faulty` and `fraud.labels.v1` with configured partitions and retention, without restoring a dev root. Live simulator and fault injector SHALL use one image as separate single-replica workloads, disabled by default and explicitly enabled for controlled synthetic runs. Only the live workload SHALL receive the synthetic integration's API key, from External Secrets, and network access to Edge; neither workload SHALL reach the vault. The fault injector SHALL NOT write to `payments.events.v1`. The simulator image SHALL reuse the shared image workflow and PR/main tag policy: same-repository PRs publish disposable PR tags, fork PRs build without publication, and main pushes publish main tags eligible for production selection.

#### Scenario: Render streaming target

- **WHEN** the configured prod target is rendered with simulation disabled and then explicitly enabled
- **THEN** the three topics remain configured, simulator workloads appear only when enabled, and no dev root, ingress or public route is created

#### Scenario: Fault injector isolation

- **WHEN** the fault injector workload is rendered
- **THEN** it has no gateway credentials or vault access, and its producer targets only the faulty topic
