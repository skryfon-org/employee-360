package auth

// ResetPasswordRequest contains parameters for completing a password reset.
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
