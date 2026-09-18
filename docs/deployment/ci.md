# CI / CD and Release

Automation workflows are divided into two clear GitHub Actions pipelines:

## CI (`.github/workflows/ci.yml`)

Runs on pull requests and pushes to `main` affecting `src/fraud-service/**`.

- Runs a single `lint-test` job.
- Sets up Python 3.14 and `uv`.
- Installs locked service dependencies (`uv sync --locked --dev`).
- Runs Ruff linting (`ruff check`) and formatting checks (`ruff format --check`).
- Runs pytest with an 80% coverage threshold (`pytest --cov=app --cov-fail-under=80`).

## Release (`.github/workflows/release.yml`)

Runs on pushes to `main` affecting `src/fraud-service/**`.

- Runs a single `build-and-push` job.
- Builds the container image using `infra/docker/fraud-service.Dockerfile` with context `src/fraud-service`.
- Tags the image with the commit SHA and `latest`; dev deployments consume `latest`, while prod selects an explicit tag from `values-prod.yaml`.
- Publishes the container image to GitHub Container Registry (`ghcr.io`).
