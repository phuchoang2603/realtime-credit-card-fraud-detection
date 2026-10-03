# Verification evidence

Current verification scope for the fraud service and Go edge, with the honest record of superseded gates. Commands for running checks are in [CONTRIBUTING](../../CONTRIBUTING.md); the coursework rubric mapping is in the [roadmap](../architecture/roadmap.md).

## Current tests

The Python behavior suite checks the `rules-v1` high-amount and geo-mismatch boundaries, absent signals, required fields, USD-only policy and snapshot totals. It does not test configuration validation, metrics or transport. The Go edge has no unit tests; it currently exposes only health and readiness and is type-checked and linted in CI. CI keeps independent Python, Go and Helm jobs and two image builds.

On 2026-10-03, local verification passed the 13 Python tests, Ruff, `go test ./...`, Go lint, Helm lint/render (base and overrides), `uv lock --check` and strict OpenSpec validation. A live gRPC smoke check also confirmed decision metrics on an ephemeral `/metrics` listener, readiness NOT_SERVING while liveness remained SERVING during a controlled drain, and decision RPCs returning UNAVAILABLE while draining. This smoke check is not part of the behavior test suite or deployed-load evidence; PR #73 image checks cover its published head, not uncommitted workspace edits.

## Retired handbook model

The bundled credit-card handbook model and its terminal-feature contract were removed because the ecommerce checkout does not supply their terminal-window features. The current fraud service applies request-only `rules-v1` and reports `policy_version=rules-v1`; it produces no risk score, model version or feature-set version. These boundary tests make no claim about model quality or deployed load. Training provenance and held-out evaluation are future work.

The retired artifact was converted on 2026-09-20 without retraining or demonstrating improved accuracy. Its last recorded SHA-256 was `7da9bec39ac963b5b15459ebc7fef62914e1dac1a5f2da2a7a34d35cddb5339f`.

## Superseded gates and limitations

The final ML rubric asks for greater-than-90% coverage and greater-than-80% mutation score. On 2026-09-23 the maintainer requested a simpler behavior-focused suite and minimal CI; the coverage/mutation gates, custom scoring tools and screenshot generation were removed as an explicit policy change. The prior mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved); it was not fixed or reclassified as passing, and no current coverage or mutation score is claimed.

Deployed load/SLO evidence remains future work under [#56](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/56) and [#62](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/62). A future report must identify workload, dataset, resource limits, duration, latency, errors and SLOs. Current tests make no claim about deployed load, model accuracy or payment idempotency.
