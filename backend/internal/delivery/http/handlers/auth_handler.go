package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	authtypes "github.com/skryfon-org/employee-360/backend/internal/types/auth"
	authusecase "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auth"
)

// AuthHandler handles HTTP requests for user authentication and password recovery.
// In accordance with Clean Architecture, handlers perform only request serialization,
// validation, and response formatting, delegating all business logic to usecases.
type AuthHandler struct {
	loginUseCase          authusecase.LoginUseCase
	tokenRefreshUseCase   authusecase.TokenRefreshUseCase
	logoutUseCase         authusecase.LogoutUseCase
	forgotPasswordUseCase authusecase.ForgotPasswordUseCase
	resetPasswordUseCase  authusecase.ResetPasswordUseCase
}

// NewAuthHandler constructs a new AuthHandler with its required usecase dependencies.
func NewAuthHandler(
	loginUseCase authusecase.LoginUseCase,
	tokenRefreshUseCase authusecase.TokenRefreshUseCase,
	logoutUseCase authusecase.LogoutUseCase,
	forgotPasswordUseCase authusecase.ForgotPasswordUseCase,
	resetPasswordUseCase authusecase.ResetPasswordUseCase,
) *AuthHandler {
	return &AuthHandler{
		loginUseCase:          loginUseCase,
		tokenRefreshUseCase:   tokenRefreshUseCase,
		logoutUseCase:         logoutUseCase,
		forgotPasswordUseCase: forgotPasswordUseCase,
		resetPasswordUseCase:  resetPasswordUseCase,
	}
}

// Login authenticates a user across all roles (super_admin, admin, employee) using email and password.
//
// @Summary      Authenticate user
// @Description  Authenticates a user by email and password, issuing an access token (stateless JWT) and a rotating refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      authtypes.LoginRequest  true  "Login credentials"
// @Success      200      {object}  response.Envelope{data=authtypes.LoginResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      500      {object}  response.Envelope
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req authtypes.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	clientInfo := GetClientInfo(c)

	res, err := h.loginUseCase.Execute(c.Request.Context(), req, clientInfo.IPAddress, clientInfo.UserAgent)
	if err != nil {
		if errors.Is(err, domainerrors.ErrInvalidCredentials) {
			response.Unauthorized(c, "invalid email or password")
			return
		}
		if errors.Is(err, domainerrors.ErrUserInactive) {
			response.Forbidden(c, "user account is inactive")
			return
		}
		response.Internal(c, "an unexpected error occurred")
		return
	}

	response.Success(c, res)
}

// Refresh rotates an existing refresh token and issues a new access/refresh token pair.
//
// @Summary      Refresh tokens
// @Description  Rotates a valid refresh token and issues a new access token and refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      authtypes.TokenRefreshRequest  true  "Refresh token payload"
// @Success      200      {object}  response.Envelope{data=authtypes.TokenRefreshResponse}
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      500      {object}  response.Envelope
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req authtypes.TokenRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	clientInfo := GetClientInfo(c)

	res, err := h.tokenRefreshUseCase.Execute(c.Request.Context(), req, clientInfo.IPAddress, clientInfo.UserAgent)
	if err != nil {
		if errors.Is(err, domainerrors.ErrInvalidToken) ||
			errors.Is(err, domainerrors.ErrTokenExpired) ||
			errors.Is(err, domainerrors.ErrTokenRevoked) {
			response.Unauthorized(c, "invalid or expired refresh token")
			return
		}
		if errors.Is(err, domainerrors.ErrUserNotFound) ||
			errors.Is(err, domainerrors.ErrUserInactive) {
			response.Unauthorized(c, "invalid token session")
			return
		}
		response.Internal(c, "an unexpected error occurred")
		return
	}

	response.Success(c, res)
}

// Logout revokes the active session and refresh token.
//
// @Summary      Log out user
// @Description  Revokes the presented refresh token and invalidates the session. Requires a valid access token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      authtypes.LogoutRequest  true  "Logout payload containing the refresh token"
// @Success      200      {object}  response.Envelope
// @Failure      400      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      500      {object}  response.Envelope
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req authtypes.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	tenantID, userID, ok := tenantAndUser(c)
	if !ok {
		return
	}

	if err := h.logoutUseCase.Execute(c.Request.Context(), tenantID, userID, req); err != nil {
		if errors.Is(err, domainerrors.ErrInvalidToken) {
			response.BadRequest(c, "invalid token")
			return
		}
		response.Internal(c, "an unexpected error occurred")
		return
	}

	response.Success(c, gin.H{"message": "logged out successfully"})
}

// ForgotPassword initiates an enumeration-safe password reset flow.
//
// @Summary      Request password reset
// @Description  Requests a password reset token to be sent to the given email address. Always returns 200 OK to prevent user enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      authtypes.ForgotPasswordRequest  true  "Email for password recovery"
// @Success      200      {object}  response.Envelope
// @Failure      400      {object}  response.Envelope
// @Failure      500      {object}  response.Envelope
// @Router       /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req authtypes.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.forgotPasswordUseCase.Execute(c.Request.Context(), req); err != nil {
		response.Internal(c, "an unexpected error occurred")
		return
	}

	response.Success(c, gin.H{"message": "if that email exists in the organization, password reset instructions have been sent"})
}

// ResetPassword completes a password reset by verifying a token and updating the user's password.
//
// @Summary      Reset password
// @Description  Resets a user's password using a valid, unexpired reset token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      authtypes.ResetPasswordRequest  true  "Reset token and new password"
// @Success      200      {object}  response.Envelope
// @Failure      400      {object}  response.Envelope
// @Failure      500      {object}  response.Envelope
// @Router       /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req authtypes.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.resetPasswordUseCase.Execute(c.Request.Context(), req); err != nil {
		if errors.Is(err, domainerrors.ErrInvalidToken) ||
			errors.Is(err, domainerrors.ErrUserNotFound) ||
			errors.Is(err, domainerrors.ErrUserInactive) {
			response.BadRequest(c, "invalid or expired reset token")
			return
		}
		if errors.Is(err, domainerrors.ErrInvalidPassword) {
			response.BadRequest(c, err.Error())
			return
		}
		response.Internal(c, "an unexpected error occurred")
		return
	}

	response.Success(c, gin.H{"message": "password reset successfully"})
}
