// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: refresh_tokens.sql

package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const createRefreshToken = `-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at) values (
    $1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    $2,
    $3
) RETURNING token, created_at, updated_at, user_id, expires_at, revoked_at
`

type CreateRefreshTokenParams struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (q *Queries) CreateRefreshToken(ctx context.Context, arg CreateRefreshTokenParams) error {
	_, err := q.db.ExecContext(ctx, createRefreshToken, arg.Token, arg.UserID, arg.ExpiresAt)
	return err
}

const getUserFromRefreshToken = `-- name: GetUserFromRefreshToken :one
SELECT user_id FROM refresh_tokens 
WHERE expires_at > CURRENT_TIMESTAMP AND revoked_at IS NULL AND token = $1
`

func (q *Queries) GetUserFromRefreshToken(ctx context.Context, token string) (uuid.UUID, error) {
	row := q.db.QueryRowContext(ctx, getUserFromRefreshToken, token)
	var user_id uuid.UUID
	err := row.Scan(&user_id)
	return user_id, err
}

const revokeRefreshToken = `-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE token = $1
`

func (q *Queries) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := q.db.ExecContext(ctx, revokeRefreshToken, token)
	return err
}
