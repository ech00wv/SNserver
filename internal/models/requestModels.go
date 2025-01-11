package models

type MessageRequest struct {
	Body string `json:"body"`
}

type UserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PaymentProviderWebhook struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}
