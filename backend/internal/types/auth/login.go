package auth

import (
	"time"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// LoginRequest carries exactly what a client submits to authenticate: email
// and password. Client IP/User-Agent for refresh-token audit metadata are
// server-derived, not client-supplied, so they are threaded into
// LoginUseCase.Execute as separate explicit parameters rather than living on
// this request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse contains the result of a successful login.
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	TokenType    string       `json:"token_type"`
	User         *entity.User `json:"user,omitempty"`
}
