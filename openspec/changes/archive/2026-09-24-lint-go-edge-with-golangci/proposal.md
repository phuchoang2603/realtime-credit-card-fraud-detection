## Why

The Go CI job runs `gofmt -l`, `go vet` and `go build` as three separate steps.
golangci-lint type-checks the module, includes `govet` by default and can enforce
gofumpt formatting, while the image job already compiles the edge binary on every PR.
One linter step replaces three overlapping ones and adds static analysis the
current job lacks.

## What Changes

- Add a `.golangci.yml` for `src/edge` using the v2 schema with the gofumpt formatter matching the treefmt rules.
- Replace the CI Go steps with the pinned golangci-lint action.
- Provide `golangci-lint` in devenv so the same check runs locally.
- Update the CI, local setup and verification guides.

## Capabilities

### Modified Capabilities

- `focused-verification`: the Go check is golangci-lint instead of format/vet/build.
- `talos-gitops-deployment`: the check-job description names Go lint.

## Impact

CI workflow, one lint configuration file, devenv packages and three guides. No
runtime, contract or chart behavior changes.
