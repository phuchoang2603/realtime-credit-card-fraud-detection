-- name: InsertOperation :one
INSERT INTO operations (
    operation_key, request_hash, integration_id, amount_minor_units, currency,
    outcome, decline_reason, processor_reference
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (operation_key) DO NOTHING
RETURNING operation_key, request_hash, outcome, decline_reason, processor_reference, recorded_at;

-- name: GetOperation :one
SELECT operation_key, request_hash, outcome, decline_reason, processor_reference, recorded_at
FROM operations
WHERE operation_key = $1;
