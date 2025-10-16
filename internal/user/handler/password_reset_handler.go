package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
)

type PasswordResetHandler struct {
	requestPasswordResetUseCase *application.RequestPasswordResetUseCase
	resetPasswordUseCase        *application.ResetPasswordUseCase
}

func NewPasswordResetHandler(
	requestPasswordResetUseCase *application.RequestPasswordResetUseCase,
	resetPasswordUseCase *application.ResetPasswordUseCase,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		requestPasswordResetUseCase: requestPasswordResetUseCase,
		resetPasswordUseCase:        resetPasswordUseCase,
	}
}

// RequestPasswordResetRequest represents the password reset request
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestPasswordReset handles password reset request
// POST /api/auth/request-password-reset
func (h *PasswordResetHandler) RequestPasswordReset(c *gin.Context) {
	var req RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	input := application.RequestPasswordResetInput{
		Email: req.Email,
	}

	output, err := h.requestPasswordResetUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}

// ResetPasswordRequest represents the password reset execution request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPassword handles password reset execution
// POST /api/auth/reset-password
func (h *PasswordResetHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	input := application.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}

	output, err := h.resetPasswordUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}
