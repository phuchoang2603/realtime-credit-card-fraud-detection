-- name: InsertIdempotency :execrows
INSERT INTO idempotency (integration_id, scope, idempotency_key, request_digest, status, payment_id, attempt_id, created_at)
VALUES ($1, $2, $3, $4, 'in_flight', $5, $6, $7)
ON CONFLICT DO NOTHING;

-- name: GetIdempotency :one
SELECT * FROM idempotency WHERE integration_id = $1 AND scope = $2 AND idempotency_key = $3;

-- name: CompleteIdempotencyByAttempt :exec
UPDATE idempotency SET status = 'completed', completed_at = $2 WHERE attempt_id = $1 AND status <> 'completed';

-- name: CompleteIdempotencyByPayment :exec
UPDATE idempotency SET status = 'completed', completed_at = $2 WHERE payment_id = $1 AND attempt_id IS NULL AND status <> 'completed';
