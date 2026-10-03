## ADDED Requirements

### Requirement: Internal fraud-service deployment

The deployment SHALL provide internal gRPC access on port 8000 and metrics on port 8010 and SHALL NOT provision public ingress or TLS controllers. Liveness SHALL use the standard gRPC health service named `liveness`; readiness SHALL use `readiness` and SHALL fail once shutdown begins. The deployment SHALL NOT supply model artifacts or model configuration.

#### Scenario: Internal deployment

- **WHEN** fraud-service is deployed with either environment configuration
- **THEN** it is exposed through a ClusterIP service without model path configuration
- **AND** no public ingress, cert-manager, or Traefik resources are created

#### Scenario: Pod termination

- **WHEN** a fraud-service pod is terminating
- **THEN** readiness is withdrawn before the gRPC listener closes, while liveness remains SERVING until health checks are no longer accepted
- **AND** it stops qualifying as a ready decision endpoint

## REMOVED Requirements

### Requirement: Internal fraud-service contract

**Reason**: The deployment no longer carries a model path, and readiness no longer depends on a model.
**Migration**: Replaced by "Internal fraud-service deployment" with the same ports, probe names and no-ingress rule.
