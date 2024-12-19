package models

import "github.com/google/uuid"

type MessageRequest struct {
	Body   string    `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}

type UserRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}
