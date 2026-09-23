Current follow-up removes service tools and readiness/process probes and switches
Go devenv to `languages.go` with Delve and gopls. Runtime health checks remain.
The focused 13-test gRPC suite passes locally. The previous successful CI links
below refer to the earlier suite; current PR checks verify the cleanup. Nix syntax
parses. The refreshed shell resolves Go 1.26.7, gopls 0.23.0 and Delve 1.27.1.

# Verification

The rewritten Python suite passes 44 tests locally and in GitHub Actions. Ruff,
Actionlint and strict OpenSpec validation pass. Remote verification for commit
`4b51126` passed:

- [CI](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/35887863857): Python lint/tests, Go race tests, generated-contract drift, real Go-to-Python prediction, process lifecycle and Helm checks.
- [Service images](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/35887864303): independent fraud and edge image builds, without publishing PR images.
- [PR #68](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/pull/68): reviewable commits and check status.

Those links record historical results. The current workflow refactor leaves
runtime, contracts and behavior tests unchanged; current checks are on PR #68.

The suite protects rules, configuration, model compatibility, gRPC validation and
errors, request correlation, canceled inference capacity,
resource cleanup and real-model repeatability. CI now uses independent Python, Go and all-chart Helm jobs; image builds
use two explicit service jobs sharing the same `template.yml` workflow.

The previous mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved).
The user subsequently requested simpler tests/CI, superseding numerical gates and
screenshot requirements. No current coverage/mutation score is claimed. See the
[verification guide](../../../docs/development/verification.md) for current scope.

No merge, archive or deployment has been performed.
