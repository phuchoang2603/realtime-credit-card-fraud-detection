-- name: InsertProcessorRequest :exec
INSERT INTO processor_requests (operation_key, payment_id, attempt_id, integration_id, request_hash, encrypted_token, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetProcessorRequest :one
SELECT * FROM processor_requests WHERE attempt_id = $1;

-- name: ClearProcessorRequestToken :exec
UPDATE processor_requests SET encrypted_token = NULL, token_cleared_at = $2
WHERE attempt_id = $1 AND encrypted_token IS NOT NULL;
