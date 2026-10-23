## MODIFIED Requirements

### Requirement: Decision RPC

The fraud service SHALL expose `fraud.v1.FraudService/Decide` over internal gRPC. A request SHALL carry `integration_id`, `payment_id`, `attempt_id`, merchant and customer references, the checkout snapshot and the attempt signals. A response SHALL carry a `payments.v1.RiskAssessment` (an `APPROVE` or `DECLINE` outcome, zero or more reason codes and the policy version, plus model version, feature-set version and risk score only when a model or feature set contributed) and the evaluation time. A decline SHALL be a successful response, not an error status.

#### Scenario: Rule decline

- **WHEN** a valid request matches a decline rule
- **THEN** the RPC returns OK with outcome `DECLINE` and the matching reason code

#### Scenario: Rules-only versions

- **WHEN** a decision is produced without a model
- **THEN** the response carries policy version `rules-v1` and omits model version, feature-set version and risk score
