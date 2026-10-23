-- name: InsertAttempt :exec
INSERT INTO attempts (
    attempt_id, payment_id, ordinal, status, version, decline_source, decline_reason,
    operation_key, processor_reference, started_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateAttempt :execrows
UPDATE attempts
SET status = $2, version = $3, decline_source = $4, decline_reason = $5,
    operation_key = $6, processor_reference = $7, updated_at = $8
WHERE attempt_id = $1 AND version = $9;

-- name: ListAttempts :many
SELECT * FROM attempts WHERE payment_id = $1 ORDER BY ordinal;

-- name: GetAttempt :one
SELECT * FROM attempts WHERE attempt_id = $1;

-- name: ListRecoveryAttempts :many
SELECT attempt_id, payment_id, status, updated_at FROM attempts
WHERE status IN ('requested', 'unknown') AND updated_at <= $1
ORDER BY updated_at
LIMIT $2;

-- name: OldestAwaitingProcessor :one
SELECT coalesce(extract(epoch FROM now() - min(updated_at)), 0)::float8 AS age_seconds FROM attempts WHERE status IN ('requested', 'unknown');
