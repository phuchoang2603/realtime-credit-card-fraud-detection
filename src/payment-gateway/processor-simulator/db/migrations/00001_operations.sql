-- +goose Up
CREATE TABLE operations (
    operation_key text PRIMARY KEY,
    request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
    integration_id uuid NOT NULL,
    amount_minor_units bigint NOT NULL CHECK (amount_minor_units > 0),
    currency text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('APPROVED', 'DECLINED')),
    decline_reason text,
    processor_reference text NOT NULL UNIQUE,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((outcome = 'DECLINED') = (decline_reason IS NOT NULL))
);

-- +goose Down
DROP TABLE operations;
