## Purpose

Keep verification small and honest: behavior tests that protect fraud decisions and minimal independent CI without score-only tooling.

## Requirements

### Requirement: Behavior-directed tests

Tests SHALL protect fraud decision rules, their thresholds and required decision inputs. Tests SHALL NOT cover configuration validation, observability or gRPC/HTTP transport. Tests SHALL NOT be added solely to raise scores.

#### Scenario: Decision boundaries

- **WHEN** fraud decisions are tested
- **THEN** tests distinguish the high-amount boundary, the account-age and amount boundaries of the geo-mismatch rule, and the absent-IP-country partition

#### Scenario: Required inputs

- **WHEN** a decision request lacks a required input
- **THEN** a test shows it is rejected instead of evaluated with an implicit default

### Requirement: Minimal CI

CI SHALL run independent Python, Go and Helm jobs on PRs and main pushes without
change-detection jobs or matrices. Checks live in the CI workflow; image builds
reuse one image workflow. Checks SHALL run Python lint/format and behavior tests,
Go static analysis and formatting through one pinned linter that type-checks the
module, and lint/render validation for every tracked Helm chart, including its
`values-*.yaml` overrides. Custom numerical coverage/mutation gates and screenshot
generation SHALL NOT be required.

#### Scenario: Review a pull request

- **WHEN** a PR is opened or updated
- **THEN** CI verifies each service implementation, using checked-in protobuf bindings where the service consumes contracts
- **AND** the reusable image workflow builds each service without publishing PR images

#### Scenario: Go defect or formatting drift

- **WHEN** the edge module fails to type-check, violates an enabled analyzer or is not gofumpt-formatted
- **THEN** the Go job fails with the reported findings

### Requirement: Honest evidence

The PR and CI logs SHALL record current verification. Historical failures SHALL
remain identified as historical rather than relabeled successful after a policy
change. Local checks SHALL be proportional; CI SHALL own full image verification.

#### Scenario: Prior mutation failure

- **WHEN** the simplified test policy replaces scoring gates
- **THEN** the previous mutation failure is disclosed and no current mutation certification is claimed
