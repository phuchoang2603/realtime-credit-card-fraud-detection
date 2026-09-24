## Context

The wiki workflow publishes `docs/` from `main`, so every page there is report
material. Contributor instructions belong where GitHub surfaces them: a root
`CONTRIBUTING.md`. Main specs now hold the normative service behavior, so prose
duplicating them drifts.

## Decisions

Audience decides location. `CONTRIBUTING.md` is the single contributor guide;
`docs/architecture` explains design and conventions; `docs/deployment` records
operational configuration and shared-platform integration; `docs/verification`
records honest evidence; `docs/research` stays as is.

Split `payment-gateway.md` by concern rather than shortening the diagrams: the core
page keeps the deployment view and ownership tables, payment contracts and the
sequence diagram move to `payment-flows.md`, and the data/ML flow moves to
`data-ml-flows.md`. Each page links its neighbours; numbering remains local to a
diagram.

Keep `service-conventions.md` to what only it can show (layout tree, dependency
direction, persistence/contract ownership, marketplace reference files) and link
`openspec/specs/service-conventions/spec.md` for runtime behavior instead of
repeating it. Fix stale factual statements met while relinking; do not rewrite
unaffected research or roadmap content. Leave archived OpenSpec changes untouched
as historical records.

## Risks / Trade-offs

- Wiki page names come from file basenames; the new names are unique so no collisions arise.
- Roadmap rows that cited superseded coverage/mutation numbers are corrected to the current honest state, which reads as a regression in status; that is accurate.
