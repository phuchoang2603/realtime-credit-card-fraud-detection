# Verification Report: migrate-fraud-infrastructure-to-shared-talos

## Summary

| Dimension | Status |
|---|---|
| Completeness | 15/16 tasks checked; several checked tasks lack promised implementation or validation |
| Correctness | 9/9 requirements have implementation evidence; 4 fully supported by source inspection, 5 partially supported |
| Coherence | Ownership/layout decisions followed; release identity, revision propagation, dashboard configuration, and formatter decisions diverge |

**1 critical issue and 9 warnings. Not ready for archive.**

## Scope and evidence

Reviewed all proposal, design, task, and delta-spec artifacts using the repo-local `spec-driven` schema, plus the working-tree infrastructure, workflows, telemetry helpers, tests, and deployment documentation. This review includes uncommitted and untracked implementation files; HEAD was `01697a4963a19717a17bc494108c095e1e05875e`.

Entered the Nix devenv successfully. Its automatic treefmt hook reported zero changed files. No local application tests, Helm checks, smoke checks, live cluster checks, publishing, or deployment were performed, consistent with the change's validation scope. Completeness, correctness, and coherence were all reviewed; runtime and live discovery remain unverified.

CI evidence lookup: `gh run list --workflow ci.yml` returned HTTP 404 (workflow not found on the default branch). The general run list returned only [an unrelated Copilot code review](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/18903384231), for another commit. This is not lint/test/Helm evidence for this working tree. CI validation remains pending; no passing result is claimed.

Consulted current Context7 documentation for [Argo CD Helm behavior](https://github.com/argoproj/argo-cd/blob/master/docs/user-guide/helm.md), [external value files](https://github.com/argoproj/argo-cd/blob/master/docs/user-guide/multiple_sources.md), and [VictoriaMetrics pod discovery](https://github.com/VictoriaMetrics/operator/blob/master/docs/resources/vmpodscrape.md). Metric naming was also inspected in the locally installed Prometheus exporter source without importing or running the application.

## CRITICAL — must fix before archive

1. **Task 6.1 is incomplete and relevant CI evidence is unavailable.** `tasks.md:33` explicitly remains unchecked. Complete the fixes below, obtain lint/test/Helm CI results for the resulting committed revision, and record their actual URLs, SHA, and status. Keep this task pending until those results exist; historical or unrelated runs do not establish completion.

## WARNING — should fix

1. **Production value-file configuration is incompatible with the template.** `infra/argocd/prod/root.yaml:21` supplies a list, but `infra/argocd/app-of-apps/templates/loop.yaml:27` interpolates the entire list as one path. This produces a bracketed path such as `$values/[infra/charts/fraud-service/values-prod.yaml]`, rather than the intended file reference, preventing normal production manifest generation. Iterate over the list and emit each path separately. Add CI coverage that uses the actual production root values.

2. **Child Helm release identity and branch revision do not meet the design.** `infra/argocd/app-of-apps/templates/loop.yaml:23` defaults child revisions to `global.targetRevision: main`; roots do not pass their revision into that value. The child source also never sets `helm.releaseName: fraud-service`. Argo CD defaults the release name to the child Application name, so the Service will be environment-prefixed and `docs/deployment/gitops.md:12`'s `svc/fraud-service` command will not match. Set the child release name explicitly, account for Argo resource tracking labels when checking selectors, and provide/document a branch override that sets both root and child revisions. Verify both environments and a non-main revision in CI.

3. **Dashboard queries and navigation do not implement the telemetry contract.** `infra/charts/fraud-service/templates/dashboards.yaml:11` queries `fraud_predictions_total` and `fraud_prediction_latency_seconds_bucket`, while `src/fraud-service/app/utils/metrics_config.py:30` defines `predictions_total` and `prediction_latency_seconds`. The score instrument is a histogram, so querying bare `fraud_prediction_score` does not query its bucket series. There is no logs panel, no configurable datasource UID, no application label filtering, and the links do not carry an actual selected trace identifier. The dashboard discovery label is hard-coded at line 8, and JSON is inline rather than in the designed `dashboards/` directory. Supply the planned dashboard files, configurable discovery labels/datasources, exporter-compatible and application-scoped metric queries, a LogsQL panel, and trace-ID navigation. Add CI JSON/query checks before marking task 3.3 complete.

4. **Service identity is absent from logs and inconsistent when configured.** `src/fraud-service/app/utils/logging_config.py:7` adds valid trace/span identifiers but no service field. `src/fraud-service/app/utils/metrics_config.py:27` hard-codes `fraud-service`, whereas tracing reads `OTEL_SERVICE_NAME` in `app/main.py:62`. Use one configured service identity for logging, tracing, and metrics; ensure startup JSON includes it even without a span. Add CI assertions for default/custom identity and valid/absent span contexts.

5. **The unavailable-target alert misses failed scrapes.** `infra/charts/fraud-service/templates/rules.yaml:12` uses only `absent(up{job=...})`. A failed discovered target still has an `up` series with value zero, so absence is false. The expression also omits namespace scoping and VMRule discovery labels cannot be configured. Add a service/namespace-scoped failed-scrape condition, retain absence detection if intended, configure rule labels to the platform contract, and validate healthy, failed, and missing target cases in CI.

6. **CI lacks the promised environment and custom-resource assertions.** `.github/workflows/ci.yml:31` renders only the default fraud chart. The production catalog invocation merely changes name/destination and never consumes the production root's values-file list. Neither root is validated; there are no destination, identity, namespace, selector, port, forbidden-resource, dashboard JSON, or custom-resource structure assertions. Add root-driven dev/prod rendering, production chart lint/render, and explicit assertions/schema validation for Application, VMPodScrape, and VMRule resources. Ensure wrong destinations/selectors and malformed resources fail CI. Tasks 2.1–2.3 and 4.2 currently overstate validation.

7. **Promised telemetry-helper scenario tests are missing.** `src/fraud-service/tests/test_main.py:1` contains existing endpoint tests only; the only other test file is `conftest.py`. There are no enabled-export endpoint/service-name checks or active-span/startup JSON checks. Existing CI sets `TESTING_MODE=true`, providing disabled-backend request coverage in source, but this does not cover enabled tracing or log correlation. Add focused tests run by CI for exporter configuration, configured identity, matching trace/span IDs, and span-free startup records. Tasks 3.1, 3.2, and 4.3 should not claim those checks are present yet.

8. **Platform prerequisite documentation is only a summary.** `docs/deployment/shared-observability.md:5` correctly leaves live values pending, but lacks the designed per-dependency owner and dev/prod acceptance entries, explicit required CRD names, concrete selector/RBAC and network requirements, and the referenced platform values-file checklist. Expand it with those entries, preserving existing marketplace discovery and making missing dependencies an explicit rollout-acceptance blocker. Add the missing configurable rule/dashboard labels and datasource fields to the app chart. Do not mark external acceptance complete from source inspection.

9. **Formatter exclusions were not migrated.** `devenv.nix:24` still excludes only `deployments/**`, contrary to checked task 4.4. Replace the stale exclusion with appropriate `infra` Helm template and future dependency-archive paths. The current formatter only formats Python, so this is a design/configuration gap rather than evidence of damaged YAML.

## SUGGESTION

None beyond the actionable findings above.

## Requirement and scenario coverage

“Source-supported” means static evidence exists, not that CI or live acceptance passed.

| Requirement | Scenario coverage and evidence | Assessment |
|---|---|---|
| Environment-specific deployment ownership | “Render both environments”: distinct root/child identities, management namespace, destination names, and workload namespace are defined in `infra/argocd/{dev,prod}/root.yaml` and `app-of-apps/templates/loop.yaml:10`. Production value-file handling prevents the intended production configuration from resolving. | Partial; warnings 1, 2, 6 |
| Internal fraud-service contract | “Internal deployment”: `infra/charts/fraud-service/templates/service.yaml:11` defines ports 8000/8010; values retain `/app/models/model.pkl`; probes retain `/health`; inference behavior is unchanged in the application diff. No ingress/TLS controllers are present. | Source-supported; service-name documentation mismatch in warning 2 |
| Shared platform ownership isolation | “Remove a payment-gateway application”: catalog contains only the fraud chart, whose kinds are Deployment, Service, VMPodScrape, VMRule, and dashboard ConfigMap. No shared backend/operator/CRD/monitoring namespace manifests or dependencies exist. | Source-supported; deletion not exercised live |
| Environment validation in CI | “Deployment configuration pull request”: independent Helm and lint/test jobs exist without cluster credentials/devenv/Docker builds in `ci.yml`, but root, production overlay, and semantic validations are missing. | Partial; warning 6 |
| Application-scoped metrics discovery | “Platform collector discovers the application”: `podscrape.yaml:9` restricts namespace discovery and matches both pod labels, port `http-metrics`, and `/metrics`; metrics server listens on 8010. | Source-supported; platform selection/live scraping pending |
| Direct trace export | “A request produces spans”: `tracing_config.py:13` configures OTLP/gRPC endpoint and resource identity; `main.py:62` supplies the configurable name. “Telemetry export is disabled”: TESTING_MODE returns before exporter setup, and CI endpoint tests use it. | Source-supported for tracing; enabled-export tests absent, cross-signal identity warning 4 |
| Correlated structured logs | “Request log correlation”: valid context adds padded trace/span IDs in `logging_config.py:7`. “Startup logging”: invalid context adds no IDs, but the required service identity is absent. Neither scenario has a focused test. | Partial; warnings 4, 7 |
| Fraud dashboards and alerts | “Shared discovery is configured”: dashboard ConfigMap and VMRule exist, but metric queries, log panel, trace navigation, configurable discovery/datasources, and failed-target condition are incomplete. | Partial; warnings 3, 5, 6 |
| Explicit platform integration prerequisites | “Log collection excludes payment-gateway”: documentation calls for the existing owner to expand discovery, labels live values pending, and no fallback collector is installed. The concrete contract/acceptance checklist is incomplete. | Partial; warning 8 |

## Other coherence observations

The infrastructure move removes legacy vendored platform charts and GKE sources. No Compose configuration was found; the manual client remains at `src/fraud-service/tools/test_client.py`. Release chart/value paths resolve and the version-tag/image policy is preserved. Main application changes are confined to telemetry identity. The production resource overlay is explicit; default resources remain empty. Documentation and image assets have moved to the new hierarchy. These positives do not substitute for the missing CI evidence or negate the findings above.

## Final assessment

1 critical issue found. Fix before archiving. Resolve the 9 implementation/validation warnings, reconcile checked task claims with the resulting files, and obtain CI results for the final revision. Live platform discovery and ingestion acceptance remain a separate pending rollout prerequisite.
