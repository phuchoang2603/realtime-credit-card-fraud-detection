-- +goose Up
CREATE TABLE payment_events (
    global_position bigserial PRIMARY KEY,
    event_id uuid NOT NULL UNIQUE,
    payment_id uuid NOT NULL,
    aggregate_version bigint NOT NULL CHECK (aggregate_version >= 1),
    event_type text NOT NULL,
    payload bytea NOT NULL,
    occurred_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL,
    UNIQUE (payment_id, aggregate_version)
);

-- +goose StatementBegin
CREATE FUNCTION payment_events_reject_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'payment_events is append-only (% rejected)', TG_OP
        USING ERRCODE = 'integrity_constraint_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER payment_events_immutable_rows
    BEFORE UPDATE OR DELETE ON payment_events
    FOR EACH ROW EXECUTE FUNCTION payment_events_reject_change();

CREATE TRIGGER payment_events_immutable_truncate
    BEFORE TRUNCATE ON payment_events
    FOR EACH STATEMENT EXECUTE FUNCTION payment_events_reject_change();

REVOKE ALL ON payment_events FROM PUBLIC;
GRANT SELECT, INSERT ON payment_events TO payments_app;
GRANT USAGE ON SEQUENCE payment_events_global_position_seq TO payments_app;
GRANT SELECT ON goose_db_version TO payments_app;

CREATE PUBLICATION dbz_publication FOR TABLE payment_events WITH (publish = 'insert');

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'payments_cdc') THEN
        GRANT SELECT ON payment_events TO payments_cdc;
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP PUBLICATION dbz_publication;
DROP TABLE payment_events;
DROP FUNCTION payment_events_reject_change();
