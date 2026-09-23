# Verification

Keep tests that protect distinct behavior. The current Python suite has 44 cases:

| Area | What the tests protect |
| --- | --- |
| Application and rules | Strict fraud threshold, independent amount/ratio boundaries, blocklists, invalid model output |
| Configuration and model | Invalid startup settings, model type/version compatibility |
| gRPC | Required feature presence, validation, safe error mapping, correlation and tracing |
| Lifecycle | Canceled inference capacity, resource cleanup and bounded drain |
| Real model | Generated prediction repeatability and unchanged caller inputs |

Go tests cover HTTP translation, RPC deadlines, safe public errors and bounded
request drain. Service tools, process probes and dedicated readiness checks were
removed at the user's request; runtime Kubernetes health checks remain configured.

Run the relevant tests locally inside devenv; CI owns the full checks and image
builds. See [local setup](local-setup.md) and [CI](../deployment/ci.md). Reuse existing
coverage instead of repeating assertions across layers. Avoid getter/no-op tests,
framework implementation assertions and parameter combinations with no distinct risk.

On 2026-09-23 the user requested a simpler behavior-focused suite and minimal CI.
The former coverage/mutation gates, custom scoring tools and screenshot generation
were removed as an explicit policy change. The prior mutation run failed at 69.17%
(489 killed, 218 survived, 38 unresolved); it was not fixed or reclassified as passing.
That historical score does not certify the rewritten tests.

Deployed load/SLO evidence remains future work under #56/#62. A future report must
identify workload, dataset, resource limits, duration, latency, errors and SLOs.
Current tests make no claim about deployed load, model accuracy or payment idempotency.
