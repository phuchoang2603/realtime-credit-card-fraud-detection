## ADDED Requirements

### Requirement: Streaming workload deployment

The deployment SHALL provide a Kafka cluster through the configured prod root, owned by this repository and managed by the platform's Strimzi operator, without restoring a dev root. It SHALL disable automatic topic creation and declare the topics `payments.events.v1`, `payments.events.v1.faulty` and `fraud.labels.v1` with configured partitions and retention. Kafka Connect SHALL publish only the Payments event table through dedicated CDC credentials, granted through a Role scoped to that one Secret; application credentials, private operation requests and vault data SHALL be excluded. Live simulator and fault injector SHALL use one image as separate single-replica workloads, disabled by default and explicitly enabled for controlled synthetic runs. Only the live workload SHALL receive integration-scoped vault-tokenization credentials and authenticated encrypted vault access; integration/caller credentials SHALL come from External Secrets. The fault injector SHALL NOT receive vault access. Connect/simulator images SHALL reuse the image workflow, without publication on PRs and with publication on main pushes. Synchronizing or deleting these applications SHALL NOT create, adopt or delete the Strimzi operator.

#### Scenario: Render streaming target

- **WHEN** the configured prod target is rendered with simulation disabled and then explicitly enabled
- **THEN** Kafka, Connect, the connector and topics remain configured, simulator workloads appear only when enabled, and no dev root, ingress or public route is created

#### Scenario: Connector secret access

- **WHEN** the Connect service account requests a Secret other than its dedicated CDC Secret
- **THEN** access is denied

#### Scenario: Remove the Kafka application

- **WHEN** the repository's Kafka application is deleted
- **THEN** the Strimzi operator and its CRDs remain under the platform owner's control
