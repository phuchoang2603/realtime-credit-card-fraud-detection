## MODIFIED Requirements

### Requirement: Shared platform ownership isolation

Synchronizing or deleting this repository's applications SHALL NOT create, adopt, or delete shared observability infrastructure, its operators, or its namespace. The application deployment SHALL NOT require CRDs or data sources from the removed observability stack and SHALL NOT install a replacement collector, dashboard, or alerting resource before a separately scoped integration change.

#### Scenario: Remove a payment-gateway application

- **WHEN** its application-owned resources are removed
- **THEN** the platform's telemetry collectors, store, and UI remain under the platform owner's control

#### Scenario: Render fraud deployment during the transition

- **WHEN** the dev and prod application configurations are rendered
- **THEN** neither requires resources from the removed observability stack
- **AND** the existing fraud metrics listener remains available without claiming application metrics are being collected
