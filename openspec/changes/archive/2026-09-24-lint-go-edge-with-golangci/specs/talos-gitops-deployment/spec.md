## MODIFIED Requirements

### Requirement: Service quality and automated image release

Every PR and main push SHALL run independent Python, Go and Helm check jobs
covering Python lint/format/tests, Go lint and Helm validation, using the
checked-in protobuf bindings. Two explicit service jobs SHALL reuse one image
workflow. PRs SHALL build without publishing; main pushes SHALL publish SHA/latest
images to GHCR.

#### Scenario: Pull request or branch push

- **WHEN** a PR is opened or updated
- **THEN** checks validate both services and image jobs build both without publication

#### Scenario: Merged service changes

- **WHEN** changes reach main
- **THEN** the release workflow publishes independently built fraud and edge images
- **AND** publication does not depend on manifest version comparison scripts
