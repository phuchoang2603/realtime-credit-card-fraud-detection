## 1. Service implementation

- [x] 1.1 Implement versioned fraud protobuf contracts and reproducible service-local Go/Python bindings.
- [x] 1.2 Replace internal Python REST with gRPC, strict inputs, model loading, safe errors and correlated tracing.
- [x] 1.3 Add Go HTTP prediction translation with reused RPC channel, deadlines and cancellation.
- [x] 1.4 Preserve distinct liveness/readiness, bounded inference capacity and resource/process shutdown.
- [x] 1.5 Use internal-only fraud chart exposure and native gRPC health probes; retain metrics and image ownership.

## 2. Simplify verification and delivery

- [x] 2.1 Rewrite the suite around useful rules, validation, model, transport and lifecycle behavior; remove score-only tooling and redundant cases.
- [x] 2.2 Replace change detection and conditional CI jobs with one straightforward check job.
- [x] 2.3 Reuse one image workflow from explicit fraud/edge jobs; build on PRs and publish on main.
- [x] 2.4 Update existing guides and OpenSpec context; remove nested READMEs and obsolete generated evidence.
- [ ] 2.5 Run lightweight local checks, push reviewable commits and open a PR.
- [ ] 2.6 Review GitHub CI/image results and fix actionable failures; record the final result.

This task list replaces the earlier score/screenshot acceptance plan following the
user's 2026-09-23 simplification request. The previous mutation failure (69.17%,
38 unresolved) remains disclosed; removed requirements are not marked as passing.
