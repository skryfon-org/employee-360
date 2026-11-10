package auth

// LogoutRequest contains parameters for revoking an active session.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
