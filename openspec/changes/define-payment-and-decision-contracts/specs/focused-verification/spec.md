## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Real-model repeatability

**Reason**: The handbook model is retired with the `fraud.v1` feature contract; the rules-only baseline has no model to check.
**Migration**: Stateless determinism is specified in `fraud-decision`. Model repeatability returns with the trained ecommerce model and its serving work (#53, #56, #70).
