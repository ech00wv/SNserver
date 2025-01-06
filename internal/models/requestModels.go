package models

type MessageRequest struct {
	Body string `json:"body"`
}

type UserRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}
