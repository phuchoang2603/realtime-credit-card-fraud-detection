## ADDED Requirements

### Requirement: Behavior-directed tests
Tests SHALL protect distinct business rules, boundaries, failure modes, resource
lifecycles or integration contracts. Duplicate assertions across layers SHALL have
a distinct integration purpose. Tests SHALL NOT be added solely to raise scores.

#### Scenario: Decision boundaries
- **WHEN** fraud decisions are tested
- **THEN** tests distinguish strict probability and independent amount/ratio boundaries, including zero historical average

#### Scenario: Transport failure
- **WHEN** input is invalid or a model fails
- **THEN** real RPC tests verify validation and sanitized status mapping with fixture-owned resources

### Requirement: Real-model repeatability
A bounded property test SHALL verify repeatable predictions and unchanged caller
inputs against the bundled model. It SHALL NOT claim accuracy or payment idempotency.

#### Scenario: Repeated valid prediction
- **WHEN** generated valid input is evaluated twice with the same model
- **THEN** decisions agree, probabilities match within a declared tolerance and input remains unchanged

### Requirement: Minimal CI
CI SHALL run one check job on PRs and main pushes without change-detection jobs or
conditional matrices. It SHALL run lint, useful Python/Go tests, generated-contract
drift and Helm validation. Custom numerical
coverage/mutation gates and screenshot generation SHALL NOT be required.

#### Scenario: Review a pull request
- **WHEN** a PR is opened or updated
- **THEN** CI verifies both service implementations and their shared protobuf contract
- **AND** existing reusable image tooling builds each service without publishing PR images

### Requirement: Honest evidence
The PR and CI logs SHALL record current verification. Historical failures SHALL
remain identified as historical rather than relabeled successful after a policy
change. Local checks SHALL be proportional; CI SHALL own full image verification.

#### Scenario: Prior mutation failure
- **WHEN** the simplified test policy replaces scoring gates
- **THEN** the previous mutation failure is disclosed and no current mutation certification is claimed
