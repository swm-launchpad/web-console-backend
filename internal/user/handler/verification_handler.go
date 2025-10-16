package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
)

type VerificationHandler struct {
	verifyEmailUseCase        *application.VerifyEmailUseCase
	resendVerificationUseCase *application.ResendVerificationEmailUseCase
}

func NewVerificationHandler(
	verifyEmailUseCase *application.VerifyEmailUseCase,
	resendVerificationUseCase *application.ResendVerificationEmailUseCase,
) *VerificationHandler {
	return &VerificationHandler{
		verifyEmailUseCase:        verifyEmailUseCase,
		resendVerificationUseCase: resendVerificationUseCase,
	}
}

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyEmail handles email verification
// GET /api/auth/verify-email?token={token}
func (h *VerificationHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	input := application.VerifyEmailInput{
		Token: token,
	}

	output, err := h.verifyEmailUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}

// ResendVerificationEmailRequest represents the resend verification email request
type ResendVerificationEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerificationEmail handles resending verification email
// POST /api/auth/resend-verification
func (h *VerificationHandler) ResendVerificationEmail(c *gin.Context) {
	var req ResendVerificationEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	input := application.ResendVerificationEmailInput{
		Email: req.Email,
	}

	output, err := h.resendVerificationUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}
