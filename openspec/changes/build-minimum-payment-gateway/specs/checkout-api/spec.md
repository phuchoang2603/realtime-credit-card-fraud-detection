## Purpose

Expose the gateway's public HTTP/JSON checkout API at the Go edge and translate it into internal gRPC commands while keeping personal data out of internal services.

## ADDED Requirements

### Requirement: Public routes

The edge SHALL expose `PUT /v1/merchants/{external_seller_id}`, `POST /v1/payments` and `GET /v1/payments/{payment_id}` authenticated by an integration API key, and `POST /v1/checkout/sessions/{session_id}/attempts` authenticated by the session's checkout token and requiring an `Idempotency-Key` header. Request and response bodies SHALL be JSON with documented field names.

#### Scenario: Missing idempotency key

- **WHEN** an attempt is submitted without `Idempotency-Key`
- **THEN** the edge returns 400 without calling Payments

### Requirement: Payment creation input

Payment creation SHALL accept the marketplace session shape: order reference, external seller, buyer reference with account creation time, amount and currency, line items, shipping address and return URL. The edge SHALL validate that the return URL is absolute HTTPS.

#### Scenario: Marketplace-shaped request

- **WHEN** a request matching the documented shape is submitted for a provisioned merchant
- **THEN** the edge returns 201 with payment ID, session ID, checkout token and expiry

### Requirement: Personal data minimization

The edge SHALL keep shipping country, region and postal code, replace the recipient name, street lines and city with a keyed address fingerprint over the normalized full address, drop buyer email, and the client IP with a keyed network fingerprint, before calling internal services. It SHALL NOT log names, emails, street lines, raw IPs or tokens.

#### Scenario: Same address, different formatting

- **WHEN** two payments differ only in address letter case and surrounding whitespace
- **THEN** their address fingerprints are equal

### Requirement: Simulated client context

Attempt requests SHALL accept device fingerprint and user-agent family. For synthetic integrations only, they SHALL also accept simulated IP country and IP network. For real integrations, IP country SHALL be absent until a geolocation source exists.

#### Scenario: Synthetic attempt with simulated geography

- **WHEN** a synthetic integration submits an attempt with a simulated IP country
- **THEN** the recorded attempt signals carry that country

### Requirement: Safe error mapping

The edge SHALL map invalid input to 400, authentication failure to 401, unknown or foreign resources to 404, idempotency conflicts to 409, unavailable dependencies to 503 and deadline expiry to 504, without exposing internal details. Attempt responses SHALL report `SUCCEEDED`, `DECLINED` with reason, or `PENDING`.

#### Scenario: Idempotency conflict

- **WHEN** a repeated idempotency key carries different content
- **THEN** the edge returns 409
