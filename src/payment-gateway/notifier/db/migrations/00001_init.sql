-- +goose Up
CREATE TABLE payment_refs (
    payment_id text PRIMARY KEY,
    integration_id text NOT NULL,
    order_reference text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE webhook_deliveries (
    event_id text PRIMARY KEY,
    integration_id text NOT NULL,
    payment_id text NOT NULL,
    event_type text NOT NULL,
    body bytea NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'delivered', 'failed')),
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    last_status integer,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz
);

CREATE INDEX webhook_deliveries_due_idx ON webhook_deliveries (next_attempt_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE webhook_deliveries;
DROP TABLE payment_refs;
