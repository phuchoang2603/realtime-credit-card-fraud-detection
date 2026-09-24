## ADDED Requirements

### Requirement: Behavior-directed tests

Tests SHALL protect fraud decision rules, the probability threshold and real-model
repeatability. Tests SHALL NOT cover configuration validation, observability or
gRPC/HTTP transport. Tests SHALL NOT be added solely to raise scores.

#### Scenario: Decision boundaries

- **WHEN** fraud decisions are tested
- **THEN** tests distinguish strict probability and independent amount/ratio boundaries, including zero historical average

### Requirement: Real-model repeatability

A bounded property test SHALL verify repeatable predictions and unchanged caller
inputs against the bundled model. It SHALL NOT claim accuracy or payment idempotency.

#### Scenario: Repeated valid prediction

- **WHEN** generated valid input is evaluated twice with the same model
- **THEN** decisions agree, probabilities match within a declared tolerance and input remains unchanged

### Requirement: Minimal CI

CI SHALL run independent Python, Go and Helm jobs on PRs and main pushes without
change-detection jobs or matrices. Checks live in the CI workflow; image builds
reuse one image workflow. Checks SHALL run Python lint/format and behavior tests,
Go format/vet/build, and lint/render validation for every tracked Helm chart,
including its `values-*.yaml` overrides. Custom numerical coverage/mutation gates
and screenshot generation SHALL NOT be required.

#### Scenario: Review a pull request

- **WHEN** a PR is opened or updated
- **THEN** CI verifies both service implementations using their checked-in protobuf bindings
- **AND** the reusable image workflow builds each service without publishing PR images

### Requirement: Honest evidence

The PR and CI logs SHALL record current verification. Historical failures SHALL
remain identified as historical rather than relabeled successful after a policy
change. Local checks SHALL be proportional; CI SHALL own full image verification.

#### Scenario: Prior mutation failure

- **WHEN** the simplified test policy replaces scoring gates
- **THEN** the previous mutation failure is disclosed and no current mutation certification is claimed
