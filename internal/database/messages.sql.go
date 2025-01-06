// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: messages.sql

package database

import (
	"context"

	"github.com/google/uuid"
)

const createMessage = `-- name: CreateMessage :one
INSERT INTO messages(id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $1,
    $2
) RETURNING id, created_at, updated_at, body, user_id
`

type CreateMessageParams struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func (q *Queries) CreateMessage(ctx context.Context, arg CreateMessageParams) (Message, error) {
	row := q.db.QueryRowContext(ctx, createMessage, arg.Body, arg.UserID)
	var i Message
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Body,
		&i.UserID,
	)
	return i, err
}

const deleteMessage = `-- name: DeleteMessage :one
DELETE FROM messages 
WHERE id = $1 AND user_id = $2
RETURNING id
`

type DeleteMessageParams struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
}

func (q *Queries) DeleteMessage(ctx context.Context, arg DeleteMessageParams) (uuid.UUID, error) {
	row := q.db.QueryRowContext(ctx, deleteMessage, arg.ID, arg.UserID)
	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}

const getAllMessages = `-- name: GetAllMessages :many
SELECT id, created_at, updated_at, body, user_id FROM messages
ORDER BY created_at ASC
`

func (q *Queries) GetAllMessages(ctx context.Context) ([]Message, error) {
	rows, err := q.db.QueryContext(ctx, getAllMessages)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Message
	for rows.Next() {
		var i Message
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Body,
			&i.UserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMessage = `-- name: GetMessage :one
SELECT id, created_at, updated_at, body, user_id FROM messages
where messages.id = $1
`

func (q *Queries) GetMessage(ctx context.Context, id uuid.UUID) (Message, error) {
	row := q.db.QueryRowContext(ctx, getMessage, id)
	var i Message
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Body,
		&i.UserID,
	)
	return i, err
}
