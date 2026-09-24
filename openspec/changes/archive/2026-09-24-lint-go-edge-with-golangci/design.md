## Context

The edge module has no unit tests; its CI job only formats, vets and builds.
The `edge / build` image job compiles the binary on every PR, so a check-job
`go build` duplicates it.

## Decisions

Use golangci-lint v2 with the default linter set (`errcheck`, `govet`,
`ineffassign`, `staticcheck`, `unused`) and the `gofumpt` formatter with
`extra-rules` so CI enforces the same formatting treefmt applies locally. Keep
`GOWORK=off` so the module resolves independently of any workspace. Pin the
action and the linter version; nixpkgs supplies the same linter version in devenv.

Type-checking inside golangci-lint fails the job on any compile error, so no
separate `go vet` or `go build` step remains. Do not add tests or extra linters
speculatively; enable more only when a concrete defect motivates it.

## Risks / Trade-offs

- A nixpkgs/action version skew could report different findings; both are pinned to 2.13.2 now and should move together.
- golangci-lint is slower than `go vet` on a cold cache; the action caches analysis results.
