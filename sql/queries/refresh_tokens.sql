-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at) values (
    $1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $2,
    $3
) RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT user_id FROM refresh_tokens 
WHERE expires_at > CURRENT_TIMESTAMP AND revoked_at IS NULL AND token = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE token = $1;
