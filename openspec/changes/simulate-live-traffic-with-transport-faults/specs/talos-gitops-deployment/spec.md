## ADDED Requirements

### Requirement: Streaming workload deployment

The deployment SHALL provide, in dev and prod, a Kafka cluster owned by this repository and managed by the platform's Strimzi operator. It SHALL disable automatic topic creation and declare the topics `payments.events.v1`, `payments.events.v1.faulty` and `fraud.labels.v1` with configured partitions and retention. A Kafka Connect cluster SHALL run a Debezium outbox connector that publishes the Payments `events` table to `payments.events.v1`. The connector SHALL read only the Payments database credentials, granted through a Role scoped to that one Secret. The live simulator and fault injector SHALL run from one simulator image as separate single-replica workloads, enabled in dev and disabled in prod by default. The synthetic integration key SHALL come from External Secrets. The Connect and simulator images SHALL be built by the shared image workflow, built without publishing on PRs and published on main pushes. Synchronizing or deleting these applications SHALL NOT create, adopt or delete the Strimzi operator.

#### Scenario: Render streaming environments

- **WHEN** dev and prod configurations are rendered
- **THEN** both contain the Kafka cluster, Connect, the outbox connector and the three topics, only dev contains the simulator workloads, and no ingress or public route is created

#### Scenario: Connector secret access

- **WHEN** the Connect service account requests a Secret other than the Payments database Secret
- **THEN** access is denied

#### Scenario: Remove the Kafka application

- **WHEN** the repository's Kafka application is deleted
- **THEN** the Strimzi operator and its CRDs remain under the platform owner's control
