## Why

The agreed payment gateway architecture and graduation roadmap currently live mainly in conversation and GitHub issues, while the README describes the existing terminal-oriented fraud service as a complete end-to-end system. PG-01 needs a reviewable repository documentation baseline that distinguishes current capabilities from the target architecture and makes the mini-coursework foundation traceable into final ML coursework.

## What Changes

- Document the current repository baseline and target Go-first microservices, Python fraud/ML boundary, gateway identities, and event-sourced payment aggregate.
- Add target deployment and numbered data-flow diagrams for checkout, asynchronous recovery/delivery, and historical/streaming data through ML serving.
- Document ownership boundaries with refurbished-marketplace and the shared platform, including synthetic processor scope and deferred tools.
- Map required mini-coursework and final ML capabilities to ticket links, expected artifacts, required evidence, and honest completion status.
- Specify the mini-to-final handoff: version and preserve generators, DP1-DP3, streaming jobs, schemas, benchmarks, and evidence; extend these assets for final ML rather than replace them.
- Update README navigation and claims to describe the existing system accurately and link the new documentation and GitHub roadmap.

## Capabilities

### New Capabilities

None. This change documents the agreed target architecture; it does not implement new runtime capabilities. `skip_specs: true` explicitly omits behavioral delta specs.

### Modified Capabilities

None. Existing Talos deployment and shared observability requirements remain unchanged. Future implementation tickets will introduce their own behavioral specifications.

## Impact

- Tracking: [PG-01 / issue #29](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/29), [project](https://github.com/users/phuchoang2603/projects/3), and [mini baseline acceptance #67](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/67).
- Planned implementation files: `README.md`, `docs/architecture/payment-gateway.md`, `docs/coursework/README.md`, `docs/coursework/mini-coursework.md`, and `docs/coursework/final-ml.md`.
- No application/API changes, migrations, infrastructure deployment, new dependencies, or edits to the marketplace repository.
- Completing this change proves documentation coverage only, not completion of any payment or coursework implementation.
