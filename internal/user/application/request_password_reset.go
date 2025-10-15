package application

import (
	"context"
	"errors"
	"log"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
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
}

func NewRequestPasswordResetUseCase(
	userService service.UserService,
	tokenService service.TokenService,
	emailService email.Service,
	txManager db.TxManager,
) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{
		userService:  userService,
		tokenService: tokenService,
		emailService: emailService,
		txManager:    txManager,
	}
}

func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, input RequestPasswordResetInput) (*RequestPasswordResetOutput, error) {
	var resetTokenStr string
	var username string
	var userEmail string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Find user by email
		user, err := uc.userService.GetUserByEmail(txCtx, input.Email)
		if err != nil {
			// Only hide ErrUserNotFound for security - propagate other errors
			if errors.Is(err, usererrors.ErrUserNotFound) {
				return nil
			}
			return err
		}

		// Create password reset token
		resetToken, err := uc.tokenService.CreatePasswordResetToken(txCtx, user.UserID)
		if err != nil {
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
			log.Printf("[EMAIL_ERROR] Failed to send password reset email | email=%s | error=%v",
				userEmail, err)
			// Don't fail the request - user will need to try again
			// TODO: Consider adding metric/alert for email sending failures
		} else {
			log.Printf("[EMAIL_SUCCESS] Password reset email sent successfully | email=%s", userEmail)
		}
	}

	// Always return success message (security: don't reveal if email exists)
	return &RequestPasswordResetOutput{
		Message: "If your email is registered, you will receive a password reset link",
	}, nil
}
