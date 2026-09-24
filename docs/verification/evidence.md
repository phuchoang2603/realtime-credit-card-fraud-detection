# Verification evidence

Current verification scope for the fraud service and Go edge, with the honest record of superseded gates. Commands for running checks are in [CONTRIBUTING](../../CONTRIBUTING.md); the coursework rubric mapping is in the [roadmap](../architecture/roadmap.md).

## Current tests

The Python suite keeps tests that protect fraud decisions. Configuration validation, observability and gRPC/HTTP transport are not covered.

| Area                  | What the tests protect                                                                                                                                  |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Application and rules | Strict 0.5 fraud threshold, independent amount/ratio boundaries with the zero-average partition, customer and terminal blocklists, invalid model output |
| Real model            | Bounded property test: repeated predictions agree within 1e-12 and caller input is unchanged                                                            |

The Go edge has no unit tests. Its code is configuration, HTTP translation and gRPC client behavior. CI lints it with golangci-lint (type-check, default analyzers, gofumpt) and the image job compiles the binary.

CI runs Python lint/format/tests, Go lint and lint/render of every tracked Helm chart with its `values-*.yaml` overrides on each pull request and `main` push; two image jobs build both services and publish only from `main`. Run results are the verification record for each pull request.

## Model artifact

`src/fraud-service/models/model.pkl` is a trusted repository-owned pickle (protocol 5) for the locked scikit-learn runtime. Loading rejects sklearn version drift and objects without `predict_proba`; artifact and runtime updates are published together and covered by the real-model property test.

On 2026-09-20 the original 1.0 artifact was converted once by adding the zero-initialized `missing_go_to_left` tree field and normalizing leaf counts to probabilities. This removed private sklearn runtime patches; it was not retraining or evidence of improved accuracy. Future models need training provenance and held-out evaluation independently of serving repeatability.

- Original SHA-256: `ac4a3e306fa7c552ac69537f14410f786b3637cb35c0efaf89c3754ba815b899`
- Current SHA-256: `7da9bec39ac963b5b15459ebc7fef62914e1dac1a5f2da2a7a34d35cddb5339f`

## Superseded gates and limitations

The final ML rubric asks for greater-than-90% coverage and greater-than-80% mutation score. On 2026-09-23 the maintainer requested a simpler behavior-focused suite and minimal CI; the coverage/mutation gates, custom scoring tools and screenshot generation were removed as an explicit policy change. The prior mutation run failed at 69.17% (489 killed, 218 survived, 38 unresolved); it was not fixed or reclassified as passing, and no current coverage or mutation score is claimed.

Deployed load/SLO evidence remains future work under [#56](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/56) and [#62](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/62). A future report must identify workload, dataset, resource limits, duration, latency, errors and SLOs. Current tests make no claim about deployed load, model accuracy or payment idempotency.
