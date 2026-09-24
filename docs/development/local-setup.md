# Local setup

Use standard CPython 3.14 throughout local development, CI, and Docker. From the repository root:

```bash
devenv shell
cd src/fraud-service
uv sync --locked --dev
```

Devenv provides Python, uv, language tooling, Helm, kubectl, OpenSpec, and Python formatting via treefmt. Shell entry synchronizes the locked Python environment and runs configured treefmt tasks; tests, model checks and Helm checks are explicit. Devenv manages its Python environment under `.devenv/state/venv`.

For a Python edit, run the small suite from the service directory:

```bash
ruff check app tests
ruff format --check app tests
uv run --locked pytest -q
```

For a Go edit, run `(cd src/edge && GOWORK=off go vet ./... && GOWORK=off go build ./...)`
from the repository root; treefmt applies gofumpt. CI runs the chart checks and
builds both images; there is no need to repeat every check locally. On NixOS use
the shell-provided Ruff; CI uses `uv run --locked ruff`.

For an intentional dependency update, edit `src/fraud-service/pyproject.toml`, run `uv lock --project src/fraud-service`, and commit the manifest and lock together. Normal installs use `--locked` to reject drift.

Start fraud with `uv run --locked python -m app` (gRPC port 8000, metrics 8010).
From the repository root, run `go run ./src/edge/cmd/edge` (HTTP port 8080).
`FRAUD_GRPC_TARGET` defaults to `dns:///localhost:8000`; in a cluster use the fraud
Service DNS name. `PREDICTION_TIMEOUT` defaults to `3s` and inherits earlier client
cancellation. `GRACEFUL_SHUTDOWN_TIMEOUT` defaults to 30 seconds. The chart allows
40 seconds for drain, cleanup and forced exit if native inference is stuck.

The devenv `codegen:proto` task regenerates bindings on shell entry when contracts
or generation dependencies change. Run it explicitly with
`devenv tasks run codegen:proto`. Commit generated sources with contract changes
so each service builds independently; do not hand-edit them. Nix supplies `protoc`, Go plugins and the Python gRPC plugin. CI runs service tests without
a separate generated-code drift gate. Public `/predict` accepts protobuf JSON field
names (lower snake_case or lowerCamelCase).
Internal prediction calls use `fraud.v1.FraudService/Predict` only.

Requests reject unknown fields and non-finite numbers and are immutable once validated. The [model artifact](#model-artifact) is serialized for the current locked sklearn runtime; version drift requires an explicit artifact/runtime update.

Boolean environment switches accept only `true` or `false` (case-insensitive). The shared OTLP/gRPC collector uses an explicit `http://` endpoint; use `https://` for a TLS collector. The chart schema requires an explicit transport scheme.

## Model artifact

`src/fraud-service/models/model.pkl` is a trusted repository-owned pickle (protocol 5)
for locked scikit-learn 1.9.1. Loading rejects sklearn version drift and objects
without `predict_proba`; publish artifact/runtime updates together and run the
real-model property check.

On 2026-09-20 the original 1.0 artifact was converted once by adding the
zero-initialized `missing_go_to_left` tree field and normalizing leaf counts to
probabilities. This removed private sklearn runtime patches; it was not retraining
or evidence of improved accuracy. Future models need training provenance and
held-out evaluation independently of serving repeatability.

- Original SHA-256: `ac4a3e306fa7c552ac69537f14410f786b3637cb35c0efaf89c3754ba815b899`
- Current SHA-256: `7da9bec39ac963b5b15459ebc7fef62914e1dac1a5f2da2a7a34d35cddb5339f`
