# Payment gateway architecture

This is the **planned target** for the payment gateway and its fraud decisioning path. The repository currently supplies a Go HTTP edge and a Python gRPC fraud service; the target takes synthetic payment-method tokens through hosted checkout, evaluates risk, executes simulated processor operations, and delivers durable payment outcomes.

## How the gateway is organized

- [Payment flows](payment-flows.md): identity, checkout and payment-authority contracts, with the checkout and recovery sequence.
- [Service conventions](service-conventions.md): code layout, dependency direction and per-service ownership rules.
- [Platform architecture](../overview/architecture.md): deployable units, ownership boundaries and the full deployment diagram.

## Gateway design

These are target decisions; the current health-only edge and fraud service do not yet implement checkout.

| Component | Design |
| --- | --- |
| Edge, Accounts and Payments | Go services split along merchant/checkout API, integration identity, and payment commands; synchronous fraud and provider calls use bounded deadlines. |
| Payment events | Payments' append-only Postgres event streams are authoritative; current-status projections are rebuildable. Commands atomically persist events, status and idempotency using expected versions. |
| Financial journal | A later Payments-owned balanced financial journal, separate from workflow events. An authorization hold, captured payment and settled funds are different facts. |
| Redis idempotency cache | A bounded Redis cache for completed-attempt results; durable database constraints still arbitrate retries. Cache loss falls back to Postgres. Distributed rate limiting is a later use with its own failure policy. |
| Event publication | The committed event table supplies the publication boundary. Later WAL CDC publishes stored payloads to Kafka without a second outbox table. Sequence allocation is not commit order; checkpoint CDC offsets and preserve per-payment versions. |
| Webhook delivery | A separate delivery service owns durable inbox/retry state and signed merchant webhooks; consumers deduplicate stable event IDs. Neither Kafka nor merchant webhook availability blocks the synchronous processor path. |
| Processor adapter | An adapter maps canonical operations to the processor simulator now and a synthetic bank profile later. ISO 8583 and provider HTTP APIs remain outside payment-domain and fraud contracts. Live connectivity needs separate production scope. |
| Card vault | A separately deployed synthetic vault with its own database, encrypted fictional fixtures, opaque tokens and restricted APIs. It models credential isolation without accepting real card data or establishing PCI DSS compliance. |
| Bank files and reconciliation | Excluded for simplicity. Keep status-based processor recovery and explicit pending outcomes; do not claim settlement verification. |

The initial simulator exposes a combined `Sale`; `PaymentSucceeded` records that provider result, not bank settlement. Separate authorization/capture and refunds require their own lifecycle and journal work. The initial availability objective is NFR-03. Database sharding is an explicit later scaling increment under NFR-06, not part of the first deployment.

### Bank adapter and ISO 8583

Yes, the target must transmit and parse ISO 8583 payloads **when the selected bank/acquirer requires that interface**. The bank-facing adapter owns this translation; browsers, merchant APIs, fraud RPCs, event streams and analytical tables keep their own versioned contracts. [ISO 8583:2023](https://www.iso.org/standard/79451.html) specifies message structure/format and explicitly excludes transport and settlement. A standard number alone is insufficient to connect to a bank.

Before implementing a real bank connection, obtain its integration profile and certification requirements:

- Pin the ISO edition, MTIs, required/conditional fields, bitmap and character/binary encoding, length prefixes/framing, amount/currency mapping and response-code semantics.
- Specify connection lifecycle, network management/echo, deadlines, authentication/encryption, and MAC/HSM/key-management responsibilities where the profile requires them. Do not assume a universal TCP, TLS or HTTP transport.
- Persist a mapping from gateway operation identity to provider context and transaction identifiers. STAN/RRN are correlation fields within a profile, not globally unique replacements for gateway IDs.
- Define duplicate, late, unmatched and malformed response handling; timeout recovery uses the profile's supported query or reversal/advice. If no safe resolution is available, retain pending status and surface it for inspection. A missing response must not trigger an unqualified resend of a financial request. Automated settlement reconciliation is outside scope.
- Keep raw card credentials and sensitive ISO wire fields out of ordinary logs, event payloads, Kafka and the ML lake. Real credential retrieval/encoding must stay within the deliberately scoped cardholder-data environment.

For a concrete example of profile differences, [Worldpay's Lync host format](https://docs.worldpay.com/core/lync-iso-8583-message-specification/host-message-format) uses `0100/0110` for authorization, `0220/0230` for completion/capture, `0400/0410` for reversal, and ASCII hexadecimal bitmaps. Its timeout reversal behavior is part of that provider's interface. This is evidence for profile-specific design, not a Vietnamese-bank field map to copy.

Delivery proceeds from the gRPC operation simulator to a **named synthetic ISO profile with an actual encoded request/response exchange**. The synthetic ISO increment must prove byte fixtures, response correlation, duplicates, late responses and uncertain-outcome recovery without live cards. A selected bank's certified profile would require a separate production scope, including how to verify financial outcomes; live connectivity is not a deliverable of this simplified project. The current gRPC simulator is not an ISO 8583 implementation. The [minimum gateway plan](../../openspec/changes/build-minimum-payment-gateway/design.md#adopted-bank-integration-and-policy-direction) records these follow-up gates.

### Vietnam policy adoption

Use the [Vietnam payment-policy research](../research/vietnam-payment-policy.md) as the source-backed baseline, preserving regulation, provider policy and fictional scenarios as separately versioned material. Introduce a VND profile with coherent units, validation and generator changes; the current fraud input is USD-only. Add issuer-authentication outcomes and pending/expiry/cancellation contracts separately from `fraud.v2` APPROVE/DECLINE. Issuer-wide daily totals must come from issuer fixtures or provider evidence. No real PAN, OTP, PIN, biometric or identity images are needed for this work.

### Delivery sequence

The first working slice is hosted checkout → rules-based fraud decision → simulated capture → durable financial outcome → signed webhook delivery → marketplace integration. Demonstrate duplicates, timeouts and recovery before layering in learned risk models. The fraud contract leaves feature retrieval and model inference independently deployable as ML capabilities arrive.

The API-driven first slice includes the isolated vault simulator and Redis result cache. Then accept the mini data baseline, extend it into feature publication/training/serving, and add authorization/capture and refunds. Database sharding follows measured load and migration evidence. Security, infrastructure reproducibility, tests, telemetry and CI/CD begin with the first service and expand with each workload.

## Planning and verification

- [Requirements](../overview/requirements.md): functional and non-functional requirements by delivery stage.
- [Implementation roadmap](../overview/roadmap.md): build order, acceptance stages and deferred decisions.
