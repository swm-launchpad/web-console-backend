package application

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type ResendVerificationEmailInput struct {
	Email string
}

type ResendVerificationEmailOutput struct {
	Message string
}

type ResendVerificationEmailUseCase struct {
	userService  service.UserService
	tokenService service.TokenService
	emailService email.Service
	txManager    db.TxManager
}

func NewResendVerificationEmailUseCase(
	userService service.UserService,
	tokenService service.TokenService,
	emailService email.Service,
	txManager db.TxManager,
) *ResendVerificationEmailUseCase {
	return &ResendVerificationEmailUseCase{
		userService:  userService,
		tokenService: tokenService,
		emailService: emailService,
		txManager:    txManager,
	}
}

func (uc *ResendVerificationEmailUseCase) Execute(ctx context.Context, input ResendVerificationEmailInput) (*ResendVerificationEmailOutput, error) {
	var verificationTokenStr string
	var username string
	var userEmail string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Find user by email
		user, err := uc.userService.GetUserByEmail(txCtx, input.Email)
		if err != nil {
			// Only hide ErrUserNotFound for security - propagate other errors
			if errors.Is(err, usererrors.ErrUserNotFound) {
				return usererrors.ErrUserNotFound
			}
			return err
		}

		// Check if user is already active
		if user.IsActive() {
			return usererrors.ErrEmailNotVerified // User is already verified
		}

		// Check rate limiting
		canResend, waitDuration, err := uc.tokenService.CanResendVerificationEmail(txCtx, user.UserID)
		if err != nil {
			return err
		}

		if !canResend {
			return fmt.Errorf("%w: please wait %v before requesting another email", usererrors.ErrTooManyRequests, waitDuration.Round(1))
		}

		// Create new verification token
		verificationToken, err := uc.tokenService.CreateEmailVerificationToken(txCtx, user.UserID)
		if err != nil {
			return err
		}

		// Store for email sending (outside transaction)
		verificationTokenStr = verificationToken.Token
		username = user.Username
		userEmail = user.Email

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Send verification email (outside transaction)
	if err := uc.emailService.SendVerificationEmail(userEmail, username, verificationTokenStr); err != nil {
		log.Printf("[EMAIL_ERROR] Failed to resend verification email | email=%s | error=%v",
			userEmail, err)
		// TODO: Consider adding metric/alert for email sending failures
		return nil, usererrors.ErrEmailSendFailed
	}

	log.Printf("[EMAIL_SUCCESS] Verification email resent successfully | email=%s", userEmail)

	return &ResendVerificationEmailOutput{
		Message: "Verification email has been resent",
	}, nil
}
