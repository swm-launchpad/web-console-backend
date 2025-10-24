package application

import (
	"context"
	"errors"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type RequestPasswordResetInput struct {
	Email string
}

type RequestPasswordResetOutput struct {
	Message string
}

type RequestPasswordResetUseCase struct {
	userService  service.UserService
	tokenService service.TokenService
	emailService email.Service
	txManager    db.TxManager
	logger       logger.Logger
}

func NewRequestPasswordResetUseCase(
	userService service.UserService,
	tokenService service.TokenService,
	emailService email.Service,
	txManager db.TxManager,
	log logger.Logger,
) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{
		userService:  userService,
		tokenService: tokenService,
		emailService: emailService,
		txManager:    txManager,
		logger:       log,
	}
}

func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, input RequestPasswordResetInput) (*RequestPasswordResetOutput, error) {
	uc.logger.Info(ctx, "request password reset started",
		zap.String("email", input.Email),
	)

	var resetTokenStr string
	var username string
	var userEmail string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Find user by email
		user, err := uc.userService.GetUserByEmail(txCtx, input.Email)
		if err != nil {
			uc.logger.Error(ctx, "failed to get user by email",
				zap.Error(err),
				zap.String("email", input.Email),
			)
			// Only hide ErrUserNotFound for security - propagate other errors
			if errors.Is(err, usererrors.ErrUserNotFound) {
				return nil
			}
			return err
		}

		// Create password reset token
		resetToken, err := uc.tokenService.CreatePasswordResetToken(txCtx, user.UserID)
		if err != nil {
			uc.logger.Error(ctx, "failed to create password reset token",
				zap.Error(err),
				zap.Uint("user_id", user.UserID),
			)
			return err
		}

		// Store for email sending (outside transaction)
		resetTokenStr = resetToken.Token
		username = user.Username
		userEmail = user.Email

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Send password reset email (outside transaction)
	// Only send if we have a token (i.e., user was found)
	if resetTokenStr != "" {
		if err := uc.emailService.SendPasswordResetEmail(userEmail, username, resetTokenStr); err != nil {
			uc.logger.Error(ctx, "failed to send password reset email",
				zap.Error(err),
				zap.String("email", userEmail),
			)
			// Don't fail the request - user will need to try again
		} else {
			uc.logger.Info(ctx, "password reset email sent successfully",
				zap.String("email", userEmail),
			)
		}
	}

	// Always return success message (security: don't reveal if email exists)
	return &RequestPasswordResetOutput{
		Message: "If your email is registered, you will receive a password reset link",
	}, nil
}
