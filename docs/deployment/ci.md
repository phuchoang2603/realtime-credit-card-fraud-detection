# CI and release

`.github/workflows/ci.yml` runs on pull requests and pushes to `main`. It runs a single `lint-test` job for the Python fraud service:

- Sets up Python 3.14 and `uv`.
- Installs locked service dependencies with `uv sync --locked --dev`.
- Runs Ruff linting (`ruff check`) and formatting checks (`ruff format --check`).
- Runs pytest with an 80% coverage threshold (`pytest --cov=app --cov-fail-under=80`).

Docker builds, container publishing, and deployment workflows remain separate.
