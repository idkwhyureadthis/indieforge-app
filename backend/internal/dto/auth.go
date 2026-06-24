package dto

import "indieforge/internal/middleware"

// RegisterRequest is the POST /auth/register request body.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the POST /auth/login request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserDTO is the public, camelCase representation sent to the frontend.
type UserDTO struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	IsDeveloper bool   `json:"isDeveloper"`
	CreatedAt   string `json:"createdAt"`
}

// AuthResponse is returned by both register and login.
type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

// NewUserDTO maps the authenticated principal to its wire representation.
func NewUserDTO(u middleware.User) UserDTO {
	return UserDTO{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Role:        string(u.Role),
		IsDeveloper: u.IsDeveloper,
		CreatedAt:   FormatTime(u.CreatedAt),
	}
}
