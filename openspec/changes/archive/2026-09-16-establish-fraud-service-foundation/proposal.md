## Why

The fraud application, model, tests, and Python dependency files currently occupy the repository root, making it difficult to introduce payment gateway sibling services. Local tooling and CI also use inconsistent dependency installation and path assumptions.

## What Changes

- **BREAKING (developer paths):** Move app, tests, model, Python manifests/lock/runtime pin, and test client under `src/fraud-service/`; preserve the HTTP contract and inference behavior.
- Establish service-local `pyproject.toml` and `uv.lock` as the dependency source for development, CI, and image builds; remove duplicate requirements files after migrating their consumers.
- Add root devenv configuration and lockfile with standard Python 3.14 and compatible updated dependencies, uv, language tooling, Helm, kubectl, OpenSpec, and treefmt for local development.
- Introduce CI that validates the service with standard Python/uv/Ruff/pytest commands and validates Helm separately, without invoking devenv, Docker, Compose, or cluster credentials.
- Move the application Dockerfile to `infra/docker/fraud-service.Dockerfile` and adapt the existing release workflow to the new paths while retaining its current image naming/version policy.
- Retire the Compose development configuration and client-only Dockerfile, and document the replacement tooling/test workflow.

## Capabilities

### New Capabilities

None. This change establishes repository and tooling conventions without introducing product behavior.

### Modified Capabilities

None. The spec inventory is empty; `.openspec.yaml` declares `skip_specs: true` for this refactor/tooling change. Design and tasks contain its verification criteria.

## Impact

Affected areas: root Python files, `app/`, `tests/`, `models/`, `client/`, Docker build configuration, `.github/workflows/`, `.gitignore`, developer documentation, and `deployments/docker-compose/`.

No live infrastructure actions, Argo/Helm migration, Victoria stack migration, ingress changes, new payment services, model retraining, or new image promotion strategy are included. Existing Kubernetes manifests and GKE provisioning remain for a later change.
