-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $1,
    $2
) RETURNING *;


-- name: CheckUserExists :one
SELECT EXISTS(
    SELECT 1
    FROM users
    WHERE users.id = $1
) AS exists;


-- name: DeleteUsers :exec
DELETE FROM users;


-- name: GetUserByEmail :one
SELECT * FROM users
WHERE users.email = $1;


-- name: UpdateUser :one
UPDATE users
SET email = $2, hashed_password = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
