## 1. Service implementation

- [x] 1.1 Implement versioned fraud protobuf contracts and reproducible service-local Go/Python bindings.
- [x] 1.2 Replace internal Python REST with gRPC, strict inputs, model loading, safe errors and correlated tracing.
- [x] 1.3 Add Go HTTP prediction translation with reused RPC channel, deadlines and cancellation.
- [x] 1.4 Preserve distinct liveness/readiness, bounded inference capacity and resource/process shutdown.
- [x] 1.5 Use internal-only fraud chart exposure and native gRPC health probes; retain metrics and image ownership.

## 2. Simplify verification and delivery

- [x] 2.1 Rewrite the suite around useful rules, validation, model, transport and lifecycle behavior; remove score-only tooling and redundant cases.
- [x] 2.2 Replace change detection and conditional CI jobs with independent Python, Go and all-chart Helm jobs.
- [x] 2.3 Reuse one test/build workflow template from explicit fraud/edge jobs; build on PRs and publish on main.
- [x] 2.4 Update existing guides and OpenSpec context; remove nested READMEs and obsolete generated evidence.
- [x] 2.5 Run lightweight local checks, push reviewable commits and open a PR.
- [x] 2.6 Review GitHub CI/image results and fix actionable failures; record the final result.

This task list replaces the earlier score/screenshot acceptance plan following the
user's 2026-09-23 simplification request. The previous mutation failure (69.17%,
38 unresolved) remains disclosed; removed requirements are not marked as passing.

## 3. Development and test cleanup

- [x] 3.1 Use the marketplace `languages.go` pattern with Delve and gopls.
- [x] 3.2 Remove Python service tools and dedicated readiness/process-probe tests; update CI and guides while retaining runtime health checks.
- [x] 3.3 Verify the cleanup, commit and push it to PR #68.
- [x] 3.4 Use the marketplace devenv codegen task pattern and remove the custom protobuf drift check.
- [x] 3.5 Remove Go and transport/config/observability Python tests; split CI into independent check jobs and a dedicated image workflow.
- [x] 3.6 Apply verification follow-ups: sanitized Python startup diagnostics, single logging identity, dead-config cleanup, schema-derived model columns, probe-free edge logs and pinned CI actions.
