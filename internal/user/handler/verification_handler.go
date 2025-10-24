package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"go.uber.org/zap"
)

type VerificationHandler struct {
	verifyEmailUseCase        *application.VerifyEmailUseCase
	resendVerificationUseCase *application.ResendVerificationEmailUseCase
	logger                    logger.Logger
}

func NewVerificationHandler(
	verifyEmailUseCase *application.VerifyEmailUseCase,
	resendVerificationUseCase *application.ResendVerificationEmailUseCase,
	log logger.Logger,
) *VerificationHandler {
	return &VerificationHandler{
		verifyEmailUseCase:        verifyEmailUseCase,
		resendVerificationUseCase: resendVerificationUseCase,
		logger:                    log,
	}
}

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyEmail handles email verification
// GET /api/auth/verify-email?token={token}
func (h *VerificationHandler) VerifyEmail(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "verify email handler started",
		zap.String("handler", "VerifyEmail"),
	)

	token := c.Query("token")
	if token == "" {
		h.logger.Warn(ctx, "missing verification token",
			zap.String("handler", "VerifyEmail"),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	input := application.VerifyEmailInput{
		Token: token,
	}

	output, err := h.verifyEmailUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "verify email use case failed",
			zap.Error(err),
			zap.String("handler", "VerifyEmail"),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "verify email handler completed",
		zap.String("handler", "VerifyEmail"),
	)

	response.OK(c, output)
}

// ResendVerificationEmailRequest represents the resend verification email request
type ResendVerificationEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerificationEmail handles resending verification email
// POST /api/auth/resend-verification
func (h *VerificationHandler) ResendVerificationEmail(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "resend verification email handler started",
		zap.String("handler", "ResendVerificationEmail"),
	)

	var req ResendVerificationEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "ResendVerificationEmail"),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	input := application.ResendVerificationEmailInput{
		Email: req.Email,
	}

	output, err := h.resendVerificationUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "resend verification email use case failed",
			zap.Error(err),
			zap.String("handler", "ResendVerificationEmail"),
			zap.String("email", req.Email),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "resend verification email handler completed",
		zap.String("handler", "ResendVerificationEmail"),
		zap.String("email", req.Email),
	)

	response.OK(c, output)
}
