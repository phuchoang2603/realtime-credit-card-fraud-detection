# Contributing

Design, deployment and coursework evidence live in [`docs/`](docs/Home.md) and publish to the wiki from `main`. This page covers working in the repository.

## Environment

```bash
devenv shell
```

Devenv provides CPython 3.14 with uv, Go with gopls, Delve and golangci-lint, `protoc` with the Go and Python gRPC plugins, Helm, kubectl, OpenSpec and treefmt. Shell entry synchronizes the locked Python environment under `.devenv/state/venv`, regenerates protobuf bindings when contracts change, and installs the git hook that runs treefmt (Ruff format, gofumpt, oxfmt) on commit.

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

`contracts/fraud/v1/fraud.proto` owns the internal fraud API. The devenv task `codegen:proto` regenerates `src/edge/gen/` and `src/fraud-service/fraud/` on shell entry when contracts or generation dependencies change; run it explicitly with `devenv tasks run codegen:proto`. Commit generated sources together with contract edits and never hand-edit them; there is no CI drift gate because each service builds from its checked-in bindings.

## Running the services

```bash
(cd src/fraud-service && uv run --locked python -m app)   # gRPC 8000, metrics 8010
go run ./src/edge/cmd/edge                               # HTTP 8080
```

The edge calls `dns:///localhost:8000` by default. `POST /predict` accepts protobuf JSON field names (`tx_amount` or `txAmount`); `GET /health` and `GET /ready` report liveness and readiness. Environment variables and their defaults are listed in the [deployment guide](docs/deployment/gitops.md#runtime-configuration).

## Dependencies

Edit `src/fraud-service/pyproject.toml`, run `uv lock --project src/fraud-service`, and commit the manifest and lock together; installs use `--locked` and reject drift. Go dependencies follow the usual `go get` / `go mod tidy` flow inside `src/edge` with `GOWORK=off`.

## CI and images

Every pull request and push to `main` runs three independent jobs in [`ci.yml`](.github/workflows/ci.yml): Python lint, format check and behavior tests; golangci-lint on the Go edge (type-check, default analyzers, gofumpt); and lint/render of every tracked Helm chart with each of its `values-*.yaml` overrides. There is no change detection, matrix or numerical coverage gate. A newer commit cancels stale runs.

[`release.yml`](.github/workflows/release.yml) runs two explicit jobs, `fraud` and `edge`, through [`build-image.yml`](.github/workflows/build-image.yml). Pull requests build both images without publishing; pushes to `main` publish to GHCR with `latest` and short commit-SHA tags. Local Docker builds are unnecessary. GitOps rollout is separate from publication; see the [deployment guide](docs/deployment/gitops.md).

## Changes and specifications

Behavior is specified under `openspec/specs/`. Non-trivial work goes through an OpenSpec change in `openspec/changes/` (proposal, design, delta specs, tasks), is synced into the main specs and archived when done. Keep commits small and reviewable, open a pull request, and let GitHub-hosted checks provide the verification record.
