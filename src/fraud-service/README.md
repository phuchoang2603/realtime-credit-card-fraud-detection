# Fraud service

This directory owns the application, model, tests, manual client, Python 3.14 runtime pin, manifest, and dependency lock. Enter `devenv shell` from the repository root, then:

```bash
cd src/fraud-service
uv sync --locked --dev
ruff check app tests tools
ruff format --check app tests tools
TESTING_MODE=true uv run --locked pytest --cov=app --cov-report=term-missing --cov-fail-under=80 tests
```

The API listens on port 8000 and Prometheus metrics on 8010. Start it explicitly with:

```bash
.venv/bin/uvicorn app.main:app --host 0.0.0.0 --port 8000 --no-access-log
```

Run the manual traffic generator against your selected endpoint:

```bash
API_URL=http://localhost:8000/predict uv run --locked python tools/test_client.py
```

Build the production image from the repository root:

```bash
docker build -f infra/docker/fraud-service.Dockerfile -t fraud-service:local src/fraud-service
```

The image installs locked runtime dependencies only and starts the installed executable directly, without dependency synchronization. `MODEL_PATH` can override `/app/models/model.pkl`. The model file is preserved; model loading and inference compatibility verification are outside this change's scope.

On NixOS, use the shell-provided `ruff` executable; PyPI binaries require a compatible Linux loader. CI uses `uv run --locked ruff` from the service lockfile.
