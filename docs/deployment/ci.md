# CI and images

Every pull request and push to `main` runs three independent jobs through
[the reusable template](../../.github/workflows/template.yml):

1. Install locked Python dependencies, lint/format and run behavior tests.
2. Test Go independently with the race detector.
3. Discover every tracked Helm chart and lint/render its defaults and `values-*.yaml` overrides.

There is no change-detection job, service matrix or numerical
coverage/mutation gate. A newer commit cancels stale CI work.

`Service images` uses two explicit jobs, `fraud` and `edge`, calling the same
[template](../../.github/workflows/template.yml). Pull requests build both
images without publishing; pushes to `main` publish to GHCR with `latest` and short
commit-SHA tags. The edge reuses the existing Go Dockerfile template. Local Docker
builds are unnecessary; GitOps rollout remains separate from image publication.

The wiki workflow publishes `docs/` after documentation changes merge to `main`.
