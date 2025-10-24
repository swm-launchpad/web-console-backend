package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"go.uber.org/zap"
)

type PasswordResetHandler struct {
	requestPasswordResetUseCase *application.RequestPasswordResetUseCase
	resetPasswordUseCase        *application.ResetPasswordUseCase
	logger                      logger.Logger
}

func NewPasswordResetHandler(
	requestPasswordResetUseCase *application.RequestPasswordResetUseCase,
	resetPasswordUseCase *application.ResetPasswordUseCase,
	log logger.Logger,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		requestPasswordResetUseCase: requestPasswordResetUseCase,
		resetPasswordUseCase:        resetPasswordUseCase,
		logger:                      log,
	}
}

// RequestPasswordResetRequest represents the password reset request
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestPasswordReset handles password reset request
// POST /api/auth/request-password-reset
func (h *PasswordResetHandler) RequestPasswordReset(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "request password reset handler started",
		zap.String("handler", "RequestPasswordReset"),
	)

	var req RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "RequestPasswordReset"),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	input := application.RequestPasswordResetInput{
		Email: req.Email,
	}

	output, err := h.requestPasswordResetUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "request password reset use case failed",
			zap.Error(err),
			zap.String("handler", "RequestPasswordReset"),
			zap.String("email", req.Email),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "request password reset handler completed",
		zap.String("handler", "RequestPasswordReset"),
		zap.String("email", req.Email),
	)

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
	ctx := c.Request.Context()
	h.logger.Info(ctx, "reset password handler started",
		zap.String("handler", "ResetPassword"),
	)

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "ResetPassword"),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	input := application.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}

	output, err := h.resetPasswordUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "reset password use case failed",
			zap.Error(err),
			zap.String("handler", "ResetPassword"),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "reset password handler completed",
		zap.String("handler", "ResetPassword"),
	)

	response.OK(c, output)
}
