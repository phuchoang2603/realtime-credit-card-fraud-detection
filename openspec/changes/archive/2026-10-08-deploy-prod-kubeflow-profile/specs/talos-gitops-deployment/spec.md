## MODIFIED Requirements

### Requirement: Environment-specific deployment ownership

The deployment configuration SHALL define distinct payment-gateway dev and prod roots in each environment's local Argo CD, targeting only that environment's cluster. When enabled, fraud-service workloads SHALL reside in the payment-gateway namespace; the prod Kubeflow workspace SHALL reside in its own Profile namespace. Application identities SHALL NOT collide with marketplace applications. During the initial rollout, only the prod Kubeflow workspace and its required prod secret store SHALL be enabled; fraud-service and dev-only children SHALL remain disabled until separately enabled.

#### Scenario: Render both environments

- **WHEN** the dev and prod configurations are rendered
- **THEN** each root and enabled child application has a distinct environment-qualified identity and its child targets only the corresponding workload cluster
- **AND** fraud-service resources remain in payment-gateway when enabled, while prod workspace resources target its Profile namespace

#### Scenario: Initial prod-only rollout

- **WHEN** the roots are rendered with their checked-in defaults
- **THEN** prod renders only the workspace and supporting prod secret-store children
- **AND** dev renders no child applications, and neither environment renders fraud-service
