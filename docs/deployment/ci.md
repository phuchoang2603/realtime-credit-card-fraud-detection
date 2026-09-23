# CI and images

Every pull request and push to `main` runs one `CI / check` job:

1. Install locked Python dependencies, lint/format and run behavior tests.
2. Test Go independently with the race detector and check generated protobuf bindings.
3. Run the Go-to-Python prediction and process-lifecycle check.
4. Lint/render both Helm environments and verify distinct gRPC health probes without public fraud ingress.

There is no change-detection job, conditional job graph, service matrix or numerical
coverage/mutation gate. A newer commit cancels stale CI work.

`Service images` uses two explicit jobs, `fraud` and `edge`, calling the same
[image workflow](../../.github/workflows/build-image.yml). Pull requests build both
images without publishing; pushes to `main` publish to GHCR with `latest` and short
commit-SHA tags. The edge reuses the existing Go Dockerfile template. Local Docker
builds are unnecessary; GitOps rollout remains separate from image publication.

The wiki workflow publishes `docs/` after documentation changes merge to `main`.
