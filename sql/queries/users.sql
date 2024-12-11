-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1
) RETURNING *;

-- name: CheckUserExists :one
SELECT EXISTS(
    SELECT 1
    FROM users
    WHERE users.id = $1
) AS exists;

-- name: DeleteUsers :exec
DELETE FROM users;
