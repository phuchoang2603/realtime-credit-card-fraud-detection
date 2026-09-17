# CI

`.github/workflows/ci.yml` runs for every pull request and every push to `main`, with two independent jobs:

- `lint-test` selects Python 3.14, runs `uv sync --locked --dev` in `src/fraud-service`, then runs locked Ruff lint/format commands and pytest with the existing 80% coverage gate. Ruff, pytest, coverage, and HTTP test tooling are locked development dependencies.
- `helm` runs lint/template validation for `infra/charts/fraud-service` and the Argo app-of-apps chart without cluster access.

CI does not invoke devenv, Docker builds, Docker Compose, or model verification. No deployment credentials or local telemetry services are needed. Exact check commands are in [local setup](../development/local-setup.md).

The release workflow watches pull requests that change `src/fraud-service/pyproject.toml`. Its version check controls the image build and publication. It builds `infra/docker/fraud-service.Dockerfile` with `src/fraud-service` as the context, publishes the existing GHCR version tag, and updates `infra/charts/fraud-service/Chart.yaml` and `infra/charts/fraud-service/values.yaml`.
