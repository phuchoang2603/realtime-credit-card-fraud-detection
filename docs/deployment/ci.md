# CI and images

Every pull request and push to `main` runs three independent jobs in
[CI](../../.github/workflows/ci.yml):

1. Install locked Python dependencies, lint, check formatting and run behavior tests.
2. Format-check, vet and build the Go edge module independently.
3. Discover every tracked Helm chart and lint/render its defaults and `values-*.yaml` overrides.

There is no change-detection job, service matrix or numerical
coverage/mutation gate. A newer commit cancels stale CI work.

[Service images](../../.github/workflows/release.yml) uses two explicit jobs, `fraud` and `edge`, calling the
[image build workflow](../../.github/workflows/build-image.yml). Pull requests build both
images without publishing; pushes to `main` publish to GHCR with `latest` and short
commit-SHA tags. The edge reuses the existing Go Dockerfile template. Local Docker
builds are unnecessary; GitOps rollout remains separate from image publication.

The wiki workflow publishes `docs/` after documentation changes merge to `main`.
