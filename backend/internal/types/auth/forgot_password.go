package auth

// ForgotPasswordRequest contains parameters for initiating a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}
