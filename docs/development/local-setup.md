# Local setup

Use standard CPython 3.14 throughout local development, CI, and Docker. From the repository root:

```bash
devenv shell
cd src/fraud-service
uv sync --locked --dev
```

Devenv provides Python, uv, language tooling, Helm, kubectl, OpenSpec, and Python formatting via treefmt. Dependency installation is explicit. Shell entry does not run lint, tests, model checks, Helm checks, or Nix formatting. The service environment is `src/fraud-service/.venv`.

Run checks from the service directory (using Nix-provided Ruff locally):

```bash
ruff check app tests tools
ruff format --check app tests tools
TESTING_MODE=true uv run --locked pytest --cov=app --cov-report=term-missing --cov-fail-under=80 tests
```

From the repository root, validate the existing chart without cluster access:

```bash
helm lint infra/charts/fraud-service
helm template test infra/charts/fraud-service --namespace payment-gateway
```

For an intentional dependency update, edit `src/fraud-service/pyproject.toml`, run `uv lock --project src/fraud-service`, and commit the manifest and lock together. Normal installs use `--locked` to reject drift.

The manual client is `src/fraud-service/tools/test_client.py`; run it against an explicitly chosen endpoint using `API_URL` (see the service README). Deployment instructions target the shared Talos clusters described in `docs/deployment/gitops.md`.

On NixOS, use the shell-provided `ruff` executable; PyPI binaries require a compatible Linux loader. CI uses `uv run --locked ruff` from the service lockfile.
