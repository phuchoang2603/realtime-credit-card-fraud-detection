# Verification

The rewritten Python suite passes 44 tests locally and in GitHub Actions. Ruff,
Actionlint and strict OpenSpec validation pass. Remote verification for commit
`4b51126` passed:

- [CI](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/35887863857): Python lint/tests, Go race tests, generated-contract drift, real Go-to-Python prediction, process lifecycle and Helm checks.
- [Service images](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/35887864303): independent fraud and edge image builds, without publishing PR images.
- [PR #68](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/pull/68): reviewable commits and check status.

This documentation follow-up records those results; it does not change the tested
runtime, contracts, tests or workflows.

The suite protects rules, configuration, model compatibility, gRPC validation and
errors, request correlation, model-aware readiness, canceled inference capacity,
resource cleanup and real-model repeatability. CI is one check job; image builds
use two explicit service jobs sharing one workflow.

The previous mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved).
The user subsequently requested simpler tests/CI, superseding numerical gates and
screenshot requirements. No current coverage/mutation score is claimed. See the
[verification guide](../../../docs/development/verification.md) for current scope.

No merge, archive or deployment has been performed.
