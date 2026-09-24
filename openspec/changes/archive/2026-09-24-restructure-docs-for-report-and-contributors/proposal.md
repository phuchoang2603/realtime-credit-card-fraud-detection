## Why

`docs/` mixes contributor how-to with coursework evidence and architecture, and
the wiki publishes all of it. `service-conventions.md` restates behavior now owned
by `openspec/specs/service-conventions`, `local-setup.md` and `ci.md` are
contributor guides, and `payment-gateway.md` is 270 lines carrying three diagrams
and their contracts in one page.

## What Changes

- Add a root `CONTRIBUTING.md` holding devenv setup, local checks, code generation, running services and how CI/images work; remove `docs/development/local-setup.md` and `docs/deployment/ci.md`.
- Split `docs/architecture/payment-gateway.md` into a core page (foundation, ownership, deployment view, delivery sequence, limits), `payment-flows.md` (identity/checkout contract, payment authority, checkout sequence) and `data-ml-flows.md` (data/ML diagram and semantics).
- Slim `service-conventions.md` to layout, dependency direction, persistence/contract ownership and reference material; move it under `docs/architecture/` and link the spec for runtime behavior.
- Move verification evidence to `docs/verification/evidence.md` and include the model artifact provenance; move runtime configuration to `docs/deployment/gitops.md`.
- Correct stale statements encountered while relinking (FastAPI foundation, marketplace platform ownership, template/matrix CI wording, superseded coverage/mutation claims) and refresh `Home.md`, `_Sidebar.md`, the README tree and `openspec/config.yaml` context.

## Capabilities

No capability behavior changes; `skip_specs` is set.

## Impact

Documentation, README and OpenSpec context only. No runtime, contract, chart or
CI behavior changes. PR #68 links to moved guides need updating.
