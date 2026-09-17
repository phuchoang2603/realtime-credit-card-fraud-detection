Migration CI evidence for `b5ade59` is recorded in [verification.md](verification.md): 14 tests and 92.50% coverage. Subsequent CI streamlining simplified the pipeline to focus purely on service linting and testing; live dev/prod acceptance remains separate. Local tests and smoke checks remain excluded.

## 1. Platform contract documentation

- [x] 1.1 Create docs/deployment/shared-observability.md with CRDs, configurable endpoints/data source UIDs, scrape/rule/dashboard selectors, and basic telemetry configuration.

## 2. Application chart and GitOps structure

- [x] 2.1 Move the first-party fraud chart to infra/charts/fraud-service with default/dev and prod values; ensure both render ClusterIP ports 8000/8010, matching pod selectors, existing model path, configured resources/probes, and no public ingress.
- [x] 2.2 Add infra/argocd/app-of-apps and dev/prod root manifests with payment-gateway-prefixed identities and revision propagation; ensure Application CRs stay on management and children target the correct registered workload cluster and payment-gateway namespace.
- [x] 2.3 Replace PodMonitor with application-scoped VMPodScrape targeting resource namespace, pod labels, and named metrics port.

## 3. Telemetry integration

- [x] 3.1 Configure direct VictoriaTraces OTLP/gRPC export and consistent configurable service identity; remove Alloy-specific defaults and descriptions.
- [x] 3.2 Add structured service/trace/span log fields while preserving request/model behavior; ensure active spans produce matching identifiers and startup logs do not fabricate trace identities.
- [x] 3.3 Add fraud dashboards as application-owned ConfigMaps with configurable discovery labels and shared data sources.
- [x] 3.4 Add VMRule for unavailable fraud scrape targets with configurable duration.

## 4. CI and release integration

- [x] 4.1 Update release.yml chart/value destinations to infra/charts/fraud-service; verify static workflow references resolve and image repository/version-tag behavior is preserved without publishing.
- [x] 4.2 Streamline CI to a single `lint-test` job running Ruff lint/format and pytest with coverage policy, removing custom Helm validation scripts.
- [x] 4.3 Keep lint and service tests together, retain coverage policy, and add focused telemetry-helper checks; verify workflow inspection shows no devenv, Docker build, cluster credential dependency, or new local test/smoke task.
- [x] 4.4 Update formatter exclusions for moved Helm templates and any future dependency archives; verify static configuration excludes those paths.

## 5. Legacy configuration cleanup

- [x] 5.1 Remove the old Argo catalog, vendored monitoring/ingress charts, and GKE Terraform sources after replacement files exist; verify git diff and reference searches show no active deployments/consumers and no change to shared marketplace resources.
- [x] 5.2 Remove any remaining Compose/client-container artifacts if present; verify file inventory and active documentation no longer require that workflow, retaining the service's manual client tool.
- [x] 5.3 Update README, docs/deployment/gitops.md, docs/deployment/ci.md, and deployment/GitOps access guidance; verify links and chart paths resolve and port-forward instructions target the new namespace/service while public ingress and GKE instructions are retired.

## 6. Validation evidence

- [x] 6.1 Review the final repo diff against both capability specs and verify service lint and test checks.
