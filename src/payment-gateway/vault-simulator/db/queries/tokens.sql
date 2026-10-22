-- name: InsertToken :exec
INSERT INTO tokens (token_digest, integration_id, method_type, fingerprint, fixture_ciphertext)
VALUES ($1, $2, $3, $4, $5);

-- name: GetActiveToken :one
SELECT method_type, fingerprint, fixture_ciphertext
FROM tokens
WHERE token_digest = $1 AND integration_id = $2 AND active;
