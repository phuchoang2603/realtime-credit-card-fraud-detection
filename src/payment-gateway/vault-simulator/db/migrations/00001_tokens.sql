-- +goose Up
CREATE TABLE tokens (
    token_digest bytea PRIMARY KEY CHECK (octet_length(token_digest) = 32),
    integration_id uuid NOT NULL,
    method_type text NOT NULL,
    fingerprint text NOT NULL,
    fixture_ciphertext bytea NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tokens_integration_id_idx ON tokens (integration_id);

-- +goose Down
DROP TABLE tokens;
