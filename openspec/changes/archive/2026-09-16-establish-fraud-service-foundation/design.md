## Context

See proposal.md for motivation. The service imports `app.*`, resolves its default model relative to `app/main.py`, and exposes HTTP on 8000 and metrics on 8010. Tests patch `app.main.model`; they do not establish that the real pickle loads. The service uses supported Python 3.14 and uv. Release automation reads the service pyproject and builds the service Dockerfile.

The reference marketplace supplies the devenv and service ownership pattern, but its Go generators, Colima settings, and cluster dependencies do not apply here. This repository currently has no devenv setup, so planning used available host tools.

## Goals / Non-Goals

**Goals:** Preserve service behavior while giving each future service its own dependency/build boundary; provide reproducible commands from the repository root; catch path and model packaging regressions before publication.

**Non-Goals:** No API redesign, shared Python package, payment implementation, telemetry rewrite, cluster changes, or broad formatting of vendored Helm sources. Infrastructure migration and image-tag policy changes remain separate.

## Decisions

### 1. Keep the existing Python package within a service directory

Move `app/`, `tests/`, `models/`, `pyproject.toml`, `uv.lock`, and `.python-version` to `src/fraud-service/`. Move `client/test_client.py` to `src/fraud-service/tools/test_client.py`; it is a manual traffic generator, not a pytest test. Add a short service README for the manifest's readme reference and service-specific commands. Keep repository overview documentation at root.

Retain `app.*` imports and the adjacent app/models layout, avoiding a simultaneous package rename. Configure pytest discovery to `tests/` and import resolution from the service directory rather than relying on repository-root PYTHONPATH. Do not leave root compatibility copies or symlinks. A root uv workspace was considered but adds coupling before a second Python service exists.

### 2. One dependency lock and Python 3.14

Use service-local uv locking for local commands, CI, and Docker. Put pytest and coverage in a development dependency group. Update runtime packages and their transitive lock entries for Python 3.14 compatibility; keep model bytes unchanged. Remove requirements.txt and requirements-dev.txt after all build consumers use the lock.

Use locked installation that fails on manifest/lock drift; production installs omit development dependencies. The service runtime pin is supported Python 3.14. The requested `3.14t` free-threaded identifier is rejected by devenv's Python provider, so it cannot be the active devenv version until that provider supports it. Dependency installation uses explicit locked commands. Tests run in CI and may be invoked manually. Model verification is omitted by user decision.

Python 3.14 and compatible scikit-learn/dependency updates are authorized for this change. The preserved pickle is not verified or retrained.

### 3. Root devenv owns developer entry points

Add devenv.nix, devenv.yaml, and devenv.lock. Provide uv, the service Python runtime, Helm, kubectl, OpenSpec, and Python formatting support. Lock Python lint tools in the service's dev dependencies for CI; use Nix-provided Ruff locally on NixOS. Ignore devenv/direnv state and generated environments while tracking lockfiles.

Devenv provides the Python 3.14 environment, uv, language tooling, and Python-only treefmt. Explicit uv sync creates the service-local virtualenv. It does not define automatic lint, test, or Helm tasks; those checks run explicitly in CI and can be invoked manually with the same direct commands. Only Python formatting is enabled; Nix checks/formatting are removed. Formatter hooks exclude Helm templates and vendored trees.

Shell entry provides tooling and hooks but does not deploy, start services, regenerate application files, or require secrets. Explicit sync installs service dependencies. Do not copy marketplace shell-entry code generation or Docker socket assumptions. Remove Compose and its configuration, plus the client Dockerfile, because the user has retired that workflow. Docker remains a build/verification tool.

### 4. CI validates the boundary and preserves release compatibility

Use `.github/workflows/ci.yml` for PR and main-push validation. CI uses standard `setup-python`, `setup-uv`, and Helm actions directly; devenv remains a local development shell. Install with `uv sync --locked --dev`, then run lint and pytest/coverage together in one job, with Helm lint/template in a separate job. CI needs no Docker daemon, kubeconfig, deployment credentials, or running telemetry backend.

Run the workflow for every PR/main push initially; this avoids omitted test, lockfile, workflow, Dockerfile, or Nix changes and skipped required checks. Service-specific filtering can be added when more services exist. Validate the first-party fraud chart through the shared command.

Move the application Dockerfile to `infra/docker/fraud-service.Dockerfile` with build context `src/fraud-service/`; keep `/app/app`, `/app/models`, ports, command, and MODEL_PATH compatibility. Add a service-context dockerignore excluding environments, caches, and test artifacts while including the model and lockfile. Adapt the existing release workflow's trigger, version lookup/diff paths, and Docker context/file paths; preserve the existing GHCR image identity, version tags, and Helm update destinations. A commit-tag release redesign is deferred.

### 5. Verification belongs in CI

Retain existing endpoint/rule tests and the 80% gate. CI owns endpoint/rule tests. Model compatibility verification is explicitly omitted. Devenv does not execute smoke checks or tests automatically. Do not change `/health` semantics as part of this migration.

## Risks / Trade-offs

- Python 3.14 requires newer scientific and web dependencies → verify clean locked installs; model compatibility is not verified in this change.
- A manifest move can invalidate readme/import/build paths → service README, explicit working directories, locked installs, and production-image packaging verification.
- Metrics binds port 8010 at import → run service tests in one process without adding parallel pytest workers; avoid parallel pytest workers.
- Dependency installation has startup/cache cost → use caching where available and keep the toolset small.
- Compose removal removes the old local telemetry environment → document tests and manual client usage against an explicitly supplied endpoint; cluster observability replacement remains a later change.
- Existing release logic has policy limitations → update path assumptions only; verify version extraction and build inputs without publishing during migration validation.

## Migration Plan

1. Record the install/tooling baseline; update dependencies for Python 3.14 and omit model verification.
2. Move the service files and Dockerfile, consolidate dependency consumers, and update release paths in the same change.
3. Add devenv environment configuration, CI checks, and ignore rules.
4. Remove Compose/client container artifacts and update README plus docs/development/local-setup.md and docs/deployment/ci.md. Identify still-existing deployment instructions as the legacy infrastructure workflow.
5. Validate shell entry, locked sync, CI lint/test/Helm jobs, and remaining references to moved paths.

No deployment is required. Rollback is a repository revert restoring file locations and workflow/build references together; retain the previous published image and model bytes.
