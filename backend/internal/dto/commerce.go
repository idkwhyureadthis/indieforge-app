package dto

// PaymentDTO mirrors the frontend Payment type.
type PaymentDTO struct {
	ID              string `json:"id"`
	GameID          string `json:"gameId"`
	UserID          string `json:"userId"`
	Kind            string `json:"kind"`
	Amount          int    `json:"amount"`
	Status          string `json:"status"`
	FriendUsername  string `json:"friendUsername,omitempty"`
	ConfirmationURL string `json:"confirmationUrl"`
	CreatedAt       string `json:"createdAt"`
}

// CreatePaymentRequest is the POST /payments request body.
type CreatePaymentRequest struct {
	GameID         string `json:"gameId"`
	Kind           string `json:"kind"`
	FriendUsername string `json:"friendUsername"`
	PlanID         string `json:"planId"` // non-empty when subscribing to a developer plan
}

// UserSubscriptionDTO carries subscription metadata alongside the game in library responses.
type UserSubscriptionDTO struct {
	ID        string  `json:"id"`
	Game      GameDTO `json:"game"`
	ExpiresAt *string `json:"expiresAt"` // ISO 8601, nil for legacy subscriptions
	Active    bool    `json:"active"`
}
