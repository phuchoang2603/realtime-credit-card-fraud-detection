# Verification

The rewritten Python suite passes 44 tests locally. Ruff and Actionlint pass.
The previous migration also passed real Go-to-Python prediction, process drain,
Go race tests and Helm checks; the current branch delegates complete integration
and image builds to GitHub CI. Remote results will be linked after PR creation.

The suite protects rules, configuration, model compatibility, gRPC validation and
errors, request correlation, model-aware readiness, canceled inference capacity,
resource cleanup and real-model repeatability. CI is one check job; image builds
use two explicit service jobs sharing one workflow.

The previous mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved).
The user subsequently requested simpler tests/CI, superseding numerical gates and
screenshot requirements. No current coverage/mutation score is claimed. See the
[verification guide](../../../docs/development/verification.md) for current scope.

No merge, archive or deployment has been performed.
