# Validation

The initial verification found Python 3.14 installation failures from legacy dependency pins, invalid Ruff selectors, Docker context path errors, test packages in production dependencies, and retired Compose/client artifacts. These were corrected following the requested Python 3.14 migration.

- Regenerated the service lock, including transitive dependencies, and successfully installed locked development dependencies on Python 3.14.
- Built `fraud-service:verify-python314` with the service directory as Docker context. Confirmed Python 3.14, packaged model file, and absence of pytest, pytest-cov, and Ruff in the production environment.
- Ran locked Ruff lint/format checks and the existing endpoint/rule tests in an isolated Python 3.14 Linux container: 8 tests passed, 87.91% coverage (80% gate). Existing Pydantic deprecation warnings remain.
- Docker starts `/app/.venv/bin/uvicorn` directly, with no startup dependency synchronization.
- Model bytes remain unchanged. Model loading/inference verification was omitted by explicit user request.
- Removed Compose configuration and the client Dockerfile. Restored the vendored Loki Nix file and removed Nix formatting from devenv.
- Local NixOS uses Nix-provided Ruff; CI uses the Ruff version in the service lockfile.
- OpenSpec strict validation and whitespace checks passed.

## Verification report — 2026-09-16

Change: `establish-fraud-service-foundation` (schema: `spec-driven`).

| Dimension | Status |
| --- | --- |
| Completeness | 18/18 tasks complete; implementation evidence found for all task groups |
| Correctness | Task criteria checked; 8/8 tests passed, 87.91% coverage against an 80% gate |
| Coherence | All five design decisions followed; no material migration divergence found |

### Evidence

- Service boundary: reviewed `src/fraud-service/pyproject.toml:1`, pytest configuration at line 37, service README, relocated app/tests/client, and removal of root compatibility files. Default pytest discovery collected only the eight existing tests, excluding the manual client.
- Compared moved Python sources against HEAD. Differences were formatting/import cleanup and the equivalent `datetime.UTC` alias in the manual client. Model bytes match HEAD exactly. Existing Helm, Argo, and Terraform files have no tracked changes.
- Tooling: successfully entered `devenv shell`; Python-only formatting changed no files. Nix-provided Ruff lint and format checks passed. Lockfiles remain trackable; local environments and caches are ignored.
- Dependencies and CI: clean locked development installation succeeded in an isolated Python 3.14.2 container. Locked Ruff checks and pytest passed; the lockfile remained byte-identical. Changing the project version in the disposable container caused `uv sync --locked --dev` to reject the stale lock as expected.
- Packaging: rebuilt `infra/docker/fraud-service.Dockerfile:1` using the service context. Image inspection confirmed the direct uvicorn command and ports 8000/8010. Production inspection confirmed Python 3.14, unchanged packaged model bytes, and absence of pytest, pytest-cov, and Ruff. The image build successfully used the locked production install.
- CI/release: reviewed `.github/workflows/ci.yml:1` and the release diff. CI retains separate Python and Helm jobs, all PR/main triggers, and the coverage gate. Release changes only relocate manifest/build paths; image tags and Helm destinations remain unchanged.
- Helm lint and template succeeded without cluster access. Strict OpenSpec validation and `git diff --check` passed. Active documentation/workflow searches found no stale consumers of the removed requirements, Compose, or client paths.
- Consulted current uv documentation through Context7 for locked sync and development-group exclusion semantics.

### Issues by priority

**CRITICAL:** None.

**WARNING:** No change-specific implementation or design divergences found.

**SUGGESTION:** In a separate release-workflow fix, update `.github/workflows/release.yml:15` to fetch the history required by the version comparison and test version-changed/version-unchanged cases. The existing shallow checkout can leave `HEAD^` unavailable; the fallback at line 23 produces plain manifest text, while line 28 expects a diff addition. This limitation predates this change, whose design explicitly defers release-logic redesign.

### Scope and limitations

- Delta requirement/scenario mapping was skipped because `.openspec.yaml` explicitly sets `skip_specs: true`; design and tasks supplied the verification criteria.
- Real model loading/inference was deliberately not verified, as required by the design. Endpoint tests use mocked model behavior.
- Test output includes 25 dependency/API deprecation warnings; these did not fail the tests or coverage gate.
- GitHub-hosted workflow execution, publishing, and live deployment were not performed. Release validation covers local paths and preserved configuration, not successful publication.

All in-scope checks passed. Ready for archive, with the separate release-workflow suggestion noted above.
