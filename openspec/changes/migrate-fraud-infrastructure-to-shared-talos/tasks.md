## 1. Platform contract documentation

- [x] 1.1 Create docs/deployment/shared-observability.md with CRDs, configurable endpoints/data source UIDs, scrape/rule/dashboard selectors, log namespace filter, and network prerequisites; verify each dependency has an owner and dev/prod acceptance entry, with unverified live values labeled pending.

## 2. Application chart and GitOps structure

- [x] 2.1 Move the first-party fraud chart to infra/charts/fraud-service with default/dev and prod values; verify in Helm CI that both render ClusterIP ports 8000/8010, matching pod selectors, existing model path, configured resources/probes, and no public ingress.
- [x] 2.2 Add infra/argocd/app-of-apps and dev/prod root manifests with payment-gateway-prefixed identities and revision propagation; verify in CI that Application CRs stay on management and children target the correct registered workload cluster and payment-gateway namespace.
- [x] 2.3 Replace PodMonitor with application-scoped VMPodScrape; verify CI assertions cover resource namespace, pod labels, named metrics port, and absence of shared platform resources or CRDs.

## 3. Telemetry integration

- [x] 3.1 Configure direct VictoriaTraces OTLP/gRPC export and consistent configurable service identity; remove Alloy-specific defaults and descriptions, and verify CI covers endpoint configuration plus export-disabled request handling.
- [x] 3.2 Add structured service/trace/span log fields while preserving request/model behavior; verify in CI that active spans produce matching identifiers and startup logs do not fabricate trace identities.
- [x] 3.3 Add fraud dashboards as application-owned ConfigMaps with configurable discovery labels and shared data sources; verify in CI that JSON parses, UIDs are unique, and prediction/latency/score queries match the exporter output and trace/log links use trace_id.
- [x] 3.4 Add VMRule for unavailable fraud scrape targets with configurable duration; verify CI validates its schema/expression and application scoping without depending on a second alerting stack.

## 4. CI and release integration

- [x] 4.1 Update release.yml chart/value destinations to infra/charts/fraud-service; verify static workflow references resolve and image repository/version-tag behavior is preserved without publishing.
- [x] 4.2 Expand the existing Helm job to validate both environments, app-of-apps, roots, application custom resources, and dashboard JSON; verify CI fails on malformed manifests and incorrect destinations/selectors rather than silently skipping all custom resource checks.
- [x] 4.3 Keep lint and service tests together, retain coverage policy, and add focused telemetry-helper checks; verify workflow inspection shows no devenv, Docker build, cluster credential dependency, or new local test/smoke task.
- [x] 4.4 Update formatter exclusions for moved Helm templates and any future dependency archives; verify static configuration excludes those paths and the app chart has no unnecessary upstream dependency or unpacked vendor tree.

## 5. Legacy configuration cleanup

- [x] 5.1 Remove the old Argo catalog, vendored monitoring/ingress charts, and GKE Terraform sources after replacement files exist; verify git diff and reference searches show no active deployments/ consumers and no change to shared marketplace resources.
- [x] 5.2 Remove any remaining Compose/client-container artifacts if present; verify file inventory and active documentation no longer require that workflow, retaining the service's manual client tool.
- [x] 5.3 Update README, docs/deployment/gitops.md, docs/deployment/ci.md, and deployment/GitOps access guidance; verify links and chart paths resolve and port-forward instructions target the new namespace/service while public ingress and GKE instructions are retired.

## 6. Validation evidence

- [ ] 6.1 Review the final repo diff against both capability specs and obtain CI lint/test/Helm results; record actual result links/status, leaving validation pending if CI has not run, without running local tests or smoke checks.
