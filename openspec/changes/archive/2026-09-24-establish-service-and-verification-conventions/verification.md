# Verification

## Current state

The Python suite has 16 tests in `test_rules.py`, `test_application.py` and
`test_prediction_property.py`. They protect the blocklists, the independent
amount/ratio boundaries including the zero-average partition, the strict 0.5
probability threshold, invalid model output rejection and bounded real-model
repeatability. Configuration, observability and gRPC/HTTP transport tests were
removed at the user's request on 2026-09-23. The Go edge has no unit tests; CI
formats, vets and builds the module.

Local checks inside devenv pass: Ruff lint and format, pytest (16 passed),
`go vet`, `go build`, gofumpt and strict OpenSpec validation. Protobuf bindings
regenerate through the devenv `codegen:proto` task; there is no CI drift gate.

CI runs three independent jobs (`python`, `go`, `helm`) in `ci.yml` and two
explicit image jobs (`fraud`, `edge`) in `release.yml` through `build-image.yml`.
Pull requests build without publishing; `main` pushes publish to GHCR. Current
results are on [PR #68](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/pull/68).

Runtime health checks remain configured: the edge serves `/health` and `/ready`;
fraud serves gRPC health `liveness` and `readiness`, probed by the chart.

No merge, archive or deployment has been performed.

## History

Commit `4b51126` carried a 44-test suite including transport and lifecycle
coverage; its [CI](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/35887863857)
and [image](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/35887864303)
runs passed. Those links record that earlier suite, not the current one.

The previous mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved).
The user subsequently requested simpler tests/CI, superseding numerical gates and
screenshot requirements. No current coverage/mutation score is claimed. See the
[verification guide](../../../docs/development/verification.md) for current scope.
