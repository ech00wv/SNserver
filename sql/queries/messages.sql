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
SELECT * FROM messages;


-- name: GetMessage :one
SELECT * FROM messages
where messages.id = $1;



-- name: DeleteMessage :one
DELETE FROM messages 
WHERE id = $1 AND user_id = $2
RETURNING id;

-- name: GetAllMessagesForAuthor :many
SELECT * FROM messages
WHERE user_id = $1;