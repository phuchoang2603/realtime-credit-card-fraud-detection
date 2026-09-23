## Why

The Go edge should own public HTTP/JSON while Python fraud serves a typed internal
API. Service conventions need executable lifecycle and error handling without a
large verification framework. The user approved gRPC migration and subsequently
requested a rewritten useful test suite, minimal CI, frequent commits and a PR.

## What Changes

- Implement a public Go prediction adapter backed by versioned protobuf/gRPC.
- Replace Python REST, private sklearn patches and import-time runtime resources.
- Keep distinct liveness/readiness, bounded shutdown, safe errors and correlation.
- Make fraud internal-only with native gRPC Kubernetes health probes.
- Replace score-focused verification with behavior tests and independent Python, Go and Helm CI jobs.
- Reuse one test/build workflow template from two explicit service jobs; build on PRs and publish on main.
- Remove scattered READMEs and consolidate essential guidance in existing docs.

## Capabilities

### New Capabilities

- `service-conventions`: Independent service ownership, typed contracts and runtime lifecycle.
- `focused-verification`: Small behavioral tests, contract checks and minimal CI.

### Modified Capabilities

- `talos-gitops-deployment`: Internal gRPC access/probes and simplified quality/image workflows.

## Impact

Touches Go/Python services, generated contracts, chart probes/exposure, CI and
existing development guides. Public prediction input uses protobuf JSON names;
internal REST is intentionally removed. No database, broker, Payments orchestration
or cluster deployment is introduced.

The latest simplicity request supersedes earlier >90% coverage, >80% mutation and
screenshot acceptance requirements for this change. The previous mutation failure
remains historical; removal of the gate is not a claim that it passed.
