## Context

See proposal.md for motivation and scope. The current service exposes `/predict` and `/health`, loads a bundled model, and requires caller-computed customer/terminal window features (`src/fraud-service/app/main.py`, `schema.py`). CI currently gates coverage at 80%; it does not establish the final ML rubric's greater-than-90% target. Deployment manifests and telemetry provide a reusable foundation, but their presence alone is not evidence of a live deployment.

The existing specifications cover shared Talos deployment and Victoria observability. The marketplace gateway note assigns commerce to marketplace and payment execution/risk to this repository. Its current payment service treats failed sessions as terminal; the target allows attempt retries under an order, so this contract mismatch must remain visible for PG-04 (#32), not be silently resolved by documentation.

The GitHub project contains the accepted roadmap, including mini baseline acceptance #67. This change captures that agreement in versioned repository documentation. A design artifact is warranted because the documentation spans service boundaries, identity, payment persistence, and data/ML lineage; no behavioral delta specification is warranted for documentation alone.

## Goals / Non-Goals

**Goals:**
- Make architectural ownership and data movement understandable without reading the conversation.
- Map coursework capabilities to planned work and concrete evidence, preserving mini-coursework as the final ML baseline.
- Separate observed repository facts, accepted target decisions, deferred implementation choices, and unverified operational claims.

**Non-Goals:**
- Implement services, select frameworks/stores/brokers, change deployment or CI behavior, or complete later tickets.
- Define full API schemas or the final payment event catalog; those belong to PG-03 through PG-05.
- Claim PCI certification, five-nines availability, live processor connectivity, or production model quality.

## Decisions

### 1. Keep documentation in a small navigable set

Use `docs/architecture/payment-gateway.md` for current/target architecture, service ownership, flows, accepted decisions, limitations, and roadmap links. Use `docs/coursework/README.md` for evidence conventions and the mini-to-final handoff. Keep one capability coverage plan per coursework in `mini-coursework.md` and `final-ml.md`. README remains a short entry point with an accurate repo map, table of contents, and links.

Use embedded Mermaid source for maintainable diagrams, since this is documentation rather than a diagram-tool selection exercise. Preserve the existing Excalidraw artifacts and identify them as historical where applicable. The alternative of one large README obscures detailed rubric evidence; duplicating the existing diagram as the target would misrepresent current scope.

### 2. Document target service ownership without prematurely selecting tools

The target service table describes deployable Go edge/hosted checkout, Go Accounts, Go Payments, Go webhook delivery, Python fraud decision, independently deployable model inference, Python drift workloads, and data/training jobs. Processor adapters and the financial journal initially remain owned by Payments; reconciliation is a Payments-owned worker rather than an independent source of truth.

Go module/composition-root, transport/domain/persistence separation, service-owned data, inbox/outbox and telemetry patterns follow `../refurbished-marketplace`. Existing infrastructure technologies may be named as observed facts; target event-store, database, broker, orchestrator and feature-platform products remain undecided. Document shared platform ownership separately from application ownership and avoid inventing database-per-class microservices.

### 3. Record the identity and payment guarantees at architectural depth

Accounts owns integration credentials, authorized merchant provisioning and scoped external references. One marketplace integration manages separate merchant accounts for sellers. `(integration_id, external_seller_id)` and `(integration_id, external_buyer_id)` map to gateway IDs. Customer mapping may be lazy; seller provisioning is authorized. Buyers retain marketplace authentication and receive short-lived checkout access. Merchant records do not imply wallets, payouts or bank accounts.

Payments owns an authoritative ordered event stream per payment aggregate, expected-version concurrency, durable command idempotency, and rebuildable status projections. Financial postings are distinct from workflow and integration events. Document durable external-effect intents, stable processor operation keys, unknown outcomes, reconciliation, atomic integration publication, and consumer deduplication. State reconstruction never calls processors, reruns models, or sends webhooks. Record risk decisions with model/policy/feature versions. Do not imply that a broker alone is the event store or that at-least-once delivery is exactly-once transport.

The alternative of current-state tables as the payment authority was considered during exploration and rejected by the user's event-sourcing decision; do not reopen that decision in PG-01.

### 4. Use separate deployment and flow diagrams

Provide a target deployable-unit view, a checkout/recovery sequence, and a data/ML flow. Major nodes represent deployable workloads or infrastructure, with external marketplace, processor simulator and shared-platform boundaries clearly labeled. SDKs and domain aggregates are not drawn as separately deployed services. Storage is labeled by role and owning service, without selecting a product.

Use numbered labeled arrows indicating the data carried; restart numbering for distinct flows and distinguish synchronous calls from asynchronous delivery in the legend. Show the browser redirect separately from the authoritative webhook/status outcome. Include failure/unknown-result resolution and replay boundaries in explanatory text. Label all target diagrams as planned and give the current baseline its own inventory rather than implying all target nodes exist.

### 5. Map coursework capabilities to work and evidence

Use the mini-coursework and final ML rubrics to identify required capabilities and proof. Organize coverage plans by capability, with issue links, expected implementation artifacts, required proof, and status. Related requirements may be grouped for clarity; preserving individual rubric rows, original points, source-row IDs, CSV snapshots, and source provenance is not required.

Expected future artifacts must be explicitly marked planned; evidence links are not fabricated. Include documentation and diagram requirements alongside the relevant capabilities.

Preserve greater-than-90% coverage, greater-than-80% changed-code mutation score, distributed-training proof, incremental data versioning, both feature writers, and per-workload CI/CD evidence. Treat named course tools as examples per user instruction, while retaining the underlying capability and proof. Fetch the reference repo's mini-coursework docs and use them as evidence-format guidance, not as proof of this repository's completion.

### 6. Make the mini-to-final handoff testable as documentation

The coursework guide maps baseline assets to extensions:

| Mini baseline | Final ML extension |
| --- | --- |
| Configurable historical/streaming generators | Drift scenarios and delayed labels |
| DP1/DP2 and governed schemas | Versioned training inputs and extended lineage |
| DP3 historical features | Point-in-time training and incremental materialization |
| Streaming windows | Offline and online feature writers with TTL/freshness |
| Baselines, benchmarks and evidence | Regression checks and versioned handoff |

Baseline acceptance #67 records source/data/schema/configuration versions, seeds, manifests, run IDs and evidence. Training acceptance #53 depends on that baseline; exploratory work may overlap. Keep miniature-coursework novelty evidence achievable at baseline (data-quality monitoring and deterministic batch/stream replay parity); later online parity and payment reconciliation extend it.

Historical simulation writes valid histories to isolated source storage; live simulation submits gateway commands. Both share event/scenario contracts. Training rows are per scored attempt/decision, not per event. Label truth stays separate, with availability timestamps; historical features respect what was known at decision time. Delivery duplicates and late arrivals are transport test conditions, not corruption of authoritative aggregate history.

### 7. Validate documentation rather than introduce runtime tests

Check relative links, referenced local paths, issue mappings, diagram syntax/rendering, coursework capability coverage, and consistency with source files and accepted decisions. Keep evidence status as planned, existing-but-unverified, or verified-with-proof. Run OpenSpec validation for planning artifacts. This change does not require application tests or new CI gates because it changes no runtime behavior.

## Risks / Trade-offs

- [Target diagrams mistaken for deployed reality] -> Label target views, inventory existing capabilities with source links, and avoid operational assertions without proof.
- [Documentation becomes premature implementation design] -> Record agreed boundaries/invariants; link detailed contracts and tool decisions to later tickets.
- [Two rubrics duplicate work or hide prerequisites] -> Share asset references and explicitly document baseline acceptance, extension and regression evidence.
- [Existing spec/deployment documentation drift] -> Flag the current internal-service spec versus configured Gateway/HTTPRoute exposure for a follow-up; do not change ingress behavior or rewrite existing requirements in PG-01.
- [Unknown repository ownership details] -> Describe shared infrastructure as externally owned and cite owning-repo documentation; resolve documentation discrepancies with evidence rather than assuming ownership transfer.

## Migration Plan

Implement the documentation, validate them, then update README links. No deployment, data migration, or runtime rollback is involved. A documentation revert restores the previous presentation without changing services. Keep PG-01 open until the documentation deliverables and rubric coverage checks exist; creation of this proposal alone does not complete it.
