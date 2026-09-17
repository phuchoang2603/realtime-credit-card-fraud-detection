## 1. Establish the service boundary

- [x] 1.1 Record the pre-existing validation baseline; keep tests in CI and omit model verification.
- [x] 1.2 Move app, tests, model, Python manifest/lock/runtime pin, and manual client into src/fraud-service/ as designed; verify model bytes are unchanged and no root compatibility copies remain.
- [x] 1.3 Add the service README and configure service-local pytest discovery/import paths; verify collection includes existing tests and excludes tools/test_client.py, and imports resolve from the service directory.
- [x] 1.4 Consolidate runtime/dev dependency groups in service-local pyproject.toml and uv.lock, including compatible lint tools; verify locked runtime and development installs succeed and a stale manifest is rejected.

## 2. Add devenv tooling

- [x] 2.1 Add devenv.nix, devenv.yaml, and devenv.lock with the service Python runtime, uv, Helm, kubectl, OpenSpec, and formatting hooks; verify shell entry succeeds without secrets, application generation, or service startup.
- [x] 2.2 Keep lint and test commands out of devenv tasks; verify CI invokes direct Ruff and pytest commands with the existing 80% coverage gate.
- [x] 2.3 Keep Helm checks out of devenv tasks; verify CI invokes direct Helm lint/render commands without cluster access.
- [x] 2.4 Update ignore rules and formatter exclusions for local devenv state, Python environments, Helm templates, and vendor content; verify lockfiles stay trackable and checks do not rewrite vendored sources.

## 3. Preserve image builds and release paths

- [x] 3.1 Move the application Dockerfile to infra/docker/fraud-service.Dockerfile, use src/fraud-service/ as context, and install locked production dependencies; verify the built image includes the model, preserves /app paths and ports, and excludes development dependencies.
- [x] 3.2 Add service-context Docker ignore rules; verify local environments and caches are excluded while required manifest, lock, README, app, and model files remain in the build context.
- [x] 3.3 Update release.yml trigger, version extraction/diff paths, and Dockerfile/context references; verify the new version lookup and build inputs locally without publishing and preserve existing image tags and Helm update destinations.
- [x] 3.4 Remove root requirements files after migrating consumers; verify active build, test, and release commands no longer reference them.

## 4. Introduce CI verification

- [x] 4.1 Replace lint-test.yml with ci.yml for PRs and main pushes using standard Python/uv/Ruff/pytest commands in one job and Helm lint/render in a separate job; verify workflow syntax and that relevant changes trigger validation without Docker or deployment credentials.
- [x] 4.2 Omit model verification as requested; verify this change adds no local smoke/test task or Docker build to CI.
- [x] 4.3 Verify the CI lint-test and Helm jobs use locked dependencies, retain the 80% coverage gate, and do not modify the lockfile.

## 5. Retire Compose and document the workflow

- [x] 5.1 Remove deployments/docker-compose/ and the client-only Dockerfile; verify no active workflow or developer command requires Compose and the relocated manual client still accepts API_URL.
- [x] 5.2 Update root README, service README, docs/development/local-setup.md, and docs/deployment/ci.md with the directory boundary and exact commands; verify documented paths exist and separate current legacy cluster deployment from this tooling migration.
- [x] 5.3 Audit active references to old app/test/model/manifest/client/Docker paths with rg and review the final diff; verify no accidental application behavior, model bytes, Kubernetes deployment, or telemetry backend changes are included.
