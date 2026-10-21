# Contributing

Design, deployment and coursework evidence live in [`docs/`](docs/Home.md) and publish to the wiki from `main`. This page covers working in the repository.

## Environment

```bash
devenv shell
```

Devenv provides CPython 3.14 with uv, Go with gopls, Delve and golangci-lint, `protoc` with the Go and Python gRPC plugins, Helm, kubectl, OpenSpec and treefmt. Shell entry synchronizes the locked Python environment under `.devenv/state/venv`, regenerates protobuf bindings when contracts change, and installs the git hook that runs treefmt (Ruff format, gofumpt) on commit.

## Local checks

Run only the checks relevant to your edit; CI owns the full set and the image builds.

| Edit     | Command (from the repository root)                                                    |
| -------- | ------------------------------------------------------------------------------------- |
| Python   | `cd src/fraud-service && ruff check app tests && uv run --locked pytest -q`           |
| Go       | `cd src/edge && GOWORK=off golangci-lint run ./...`                                   |
| Helm     | `helm lint infra/charts/fraud-service -f infra/charts/fraud-service/values-prod.yaml` |
| OpenSpec | `openspec validate --all --strict`                                                    |

On NixOS use the shell-provided Ruff; CI uses `uv run --locked ruff`. The Python suite is intentionally small; see the [verification evidence](docs/verification/evidence.md) for its scope.

## Contracts and generated code

The historical dataset agreement is [ODCS](contracts/data/historical-payments.odcs.yaml). It owns schemas, field meanings, ownership and SQL quality rules. Run `uvx --from datacontract-cli==1.2.7 datacontract lint contracts/data/historical-payments.odcs.yaml --all-errors` from the repository root. Regenerate its [schema reference](docs/architecture/historical-data-contract.md) with `uvx --from datacontract-cli==1.2.7 datacontract export markdown contracts/data/historical-payments.odcs.yaml --output docs/architecture/historical-data-contract.md`. CI lints the agreement and checks this native export for drift; the full quality rules remain in ODCS.

Producing and ingesting jobs must validate actual data against the agreement before publication. Once a job exposes complete histories as `payment_events` and `fraud_labels` tables or views in a local DuckDB database, run `DATACONTRACT_DUCKDB_DATABASE=/path/to/staging.duckdb uvx --from 'datacontract-cli[duckdb]==1.2.7' datacontract test contracts/data/historical-payments.odcs.yaml --server validation --output validation.json --output-format json`. This uses the existing engine directly; data preparation and physical file checks belong with the job. CI in this foundation does not validate generated datasets.

The `schemaVersions` and `sinceSchemaVersion` custom properties describe physical layout evolution; engines do not automatically enforce them. Identical source deliveries are allowed. Current SQL rules use DuckDB syntax; Spark and Flink integration must verify engine support and rule execution when those pipelines are implemented. Do not maintain separate handwritten rule suites. Future DataHub ingestion consumes the same ODCS document; catalog ingestion and executing quality rules are separate operations. Backend Protobuf continues to own wire messages.

`contracts/payments/v1/` defines checkout snapshots and lifecycle events, `contracts/fraud/v2/` defines the internal decision RPC, and `contracts/labels/v1/` defines delayed labels. The devenv task `codegen:proto` regenerates the Python bindings under `src/fraud-service/gen/` on shell entry when contracts or generation dependencies change; run it explicitly with `devenv tasks run codegen:proto`. The generated modules retain their `fraud.v2`, `payments.v1` and `labels.v1` import paths; `gen/` must be on `PYTHONPATH` (set by devenv, pytest and the container). Keep generated sources with contract edits and never hand-edit them; Go bindings are added when a Go service consumes the contracts.

## Running the services

```bash
(cd src/fraud-service && PYTHONPATH=gen uv run --locked python -m app)   # gRPC 8000, metrics 8010
go run ./src/edge/cmd/edge                               # HTTP 8080
```

The fraud service exposes internal `fraud.v2.FraudService/Decide` on port 8000 and gRPC health services `liveness` and `readiness`. The edge currently provides only `GET /health` and `GET /ready`; checkout routes arrive with the minimum gateway. Environment variables and their defaults are listed in the [deployment guide](docs/deployment/gitops.md#runtime-configuration).

## Dependencies

Edit `src/fraud-service/pyproject.toml`, run `uv lock --project src/fraud-service`, and commit the manifest and lock together; installs use `--locked` and reject drift. Go dependencies follow the usual `go get` / `go mod tidy` flow inside `src/edge` with `GOWORK=off`.

## CI and images

Every pull request and push to `main` runs four independent jobs in [`ci.yml`](.github/workflows/ci.yml): ODCS agreement lint and documentation drift; Python lint, format check and behavior tests; golangci-lint on the Go edge (type-check, default analyzers, gofumpt); and lint/render of every tracked Helm chart with each of its `values-*.yaml` overrides. There is no change detection, matrix or numerical coverage gate. A newer commit cancels stale runs.

[`release.yml`](.github/workflows/release.yml) runs two explicit jobs, `fraud` and `edge`, through [`build-image.yml`](.github/workflows/build-image.yml). Pull requests build both images without publishing; pushes to `main` publish to GHCR with `latest` and short commit-SHA tags. Local Docker builds are unnecessary. GitOps rollout is separate from publication; see the [deployment guide](docs/deployment/gitops.md).

## Changes and specifications

Behavior is specified under `openspec/specs/`. Non-trivial work goes through an OpenSpec change in `openspec/changes/` (proposal, design, delta specs, tasks), is synced into the main specs and archived when done. Keep commits small and reviewable, open a pull request, and let GitHub-hosted checks provide the verification record.
