-- +goose Up
CREATE TABLE payments (
    payment_id uuid PRIMARY KEY,
    integration_id uuid NOT NULL,
    session_id uuid NOT NULL UNIQUE,
    order_reference text NOT NULL,
    status text NOT NULL CHECK (status IN ('open', 'succeeded', 'expired')),
    version bigint NOT NULL CHECK (version >= 1),
    merchant_id uuid NOT NULL,
    external_seller_id text NOT NULL,
    customer_id uuid NOT NULL,
    external_buyer_id text NOT NULL,
    snapshot bytea NOT NULL,
    return_url text NOT NULL,
    correlation_id text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (integration_id, order_reference)
);

CREATE INDEX payments_open_expiry ON payments (expires_at) WHERE status = 'open';

CREATE TABLE attempts (
    attempt_id uuid PRIMARY KEY,
    payment_id uuid NOT NULL REFERENCES payments (payment_id),
    ordinal integer NOT NULL,
    status text NOT NULL CHECK (status IN (
        'started', 'risk_approved', 'risk_declined', 'requested', 'unknown', 'declined', 'succeeded')),
    version bigint NOT NULL CHECK (version >= 1),
    decline_source integer NOT NULL DEFAULT 0,
    decline_reason text NOT NULL DEFAULT '',
    operation_key text NOT NULL DEFAULT '',
    processor_reference text NOT NULL DEFAULT '',
    started_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (payment_id, ordinal)
);

CREATE INDEX attempts_awaiting_processor ON attempts (updated_at) WHERE status IN ('requested', 'unknown');

REVOKE ALL ON payments, attempts FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON payments, attempts TO payments_app;

-- +goose Down
DROP TABLE attempts;
DROP TABLE payments;
