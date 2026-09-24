# Verification

Keep tests that protect fraud decisions. Configuration validation, observability and gRPC/HTTP transport are not covered.

| Area                  | What the tests protect                                                                        |
| --------------------- | --------------------------------------------------------------------------------------------- |
| Application and rules | Strict fraud threshold, independent amount/ratio boundaries, blocklists, invalid model output |
| Real model            | Generated prediction repeatability and unchanged caller inputs                                |

The Go edge has no unit tests. Its code is configuration, HTTP translation and gRPC client behavior, which this suite does not cover. CI still compiles the module with `go test -race ./...`.

Run the relevant tests locally inside devenv; CI owns the full checks and image builds. See [local setup](local-setup.md) and [CI](../deployment/ci.md).

The mini-coursework covers data generation, pipelines, schemas and benchmarks. The final ML rubric later asks for greater-than-90% coverage and greater-than-80% mutation score. Neither requires these configuration, observability or transport tests. On 2026-09-23 the user requested a simpler behavior-focused suite and minimal CI. The former coverage/mutation gates, custom scoring tools and screenshot generation were removed as an explicit policy change. The prior mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved); it was not fixed or reclassified as passing. That historical score does not certify the current tests.

Deployed load/SLO evidence remains future work under #56/#62. A future report must identify workload, dataset, resource limits, duration, latency, errors and SLOs. Current tests make no claim about deployed load, model accuracy or payment idempotency.
