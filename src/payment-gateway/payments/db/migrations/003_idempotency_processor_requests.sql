-- +goose Up
CREATE TABLE idempotency (
    integration_id uuid NOT NULL,
    scope text NOT NULL,
    idempotency_key text NOT NULL,
    request_digest bytea NOT NULL,
    status text NOT NULL CHECK (status IN ('in_flight', 'completed')),
    payment_id uuid NOT NULL,
    attempt_id uuid,
    created_at timestamptz NOT NULL,
    completed_at timestamptz,
    PRIMARY KEY (integration_id, scope, idempotency_key)
);

CREATE UNIQUE INDEX idempotency_attempt ON idempotency (attempt_id) WHERE attempt_id IS NOT NULL;

CREATE TABLE processor_requests (
    operation_key text PRIMARY KEY,
    payment_id uuid NOT NULL,
    attempt_id uuid NOT NULL UNIQUE REFERENCES attempts (attempt_id),
    integration_id uuid NOT NULL,
    request_hash bytea NOT NULL,
    encrypted_token bytea,
    created_at timestamptz NOT NULL,
    token_cleared_at timestamptz
);

REVOKE ALL ON idempotency, processor_requests FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON idempotency, processor_requests TO payments_app;

-- +goose Down
DROP TABLE processor_requests;
DROP TABLE idempotency;
