## MODIFIED Requirements

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
- **THEN** CI verifies both service implementations using their checked-in protobuf bindings
- **AND** the reusable image workflow builds each service without publishing PR images

#### Scenario: Go defect or formatting drift

- **WHEN** the edge module fails to type-check, violates an enabled analyzer or is not gofumpt-formatted
- **THEN** the Go job fails with the reported findings
