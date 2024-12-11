-- name: CreateMessage :one
INSERT INTO messages(id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $1,
    $2
) RETURNING *;

-- name: GetAllMessages :many
SELECT * FROM messages
ORDER BY created_at ASC;


-- name: GetMessage :one
SELECT * FROM messages
where messages.id = $1;