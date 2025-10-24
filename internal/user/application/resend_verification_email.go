package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
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
	logger       logger.Logger
}

func NewResendVerificationEmailUseCase(
	userService service.UserService,
	tokenService service.TokenService,
	emailService email.Service,
	txManager db.TxManager,
	log logger.Logger,
) *ResendVerificationEmailUseCase {
	return &ResendVerificationEmailUseCase{
		userService:  userService,
		tokenService: tokenService,
		emailService: emailService,
		txManager:    txManager,
		logger:       log,
	}
}

func (uc *ResendVerificationEmailUseCase) Execute(ctx context.Context, input ResendVerificationEmailInput) (*ResendVerificationEmailOutput, error) {
	uc.logger.Info(ctx, "resend verification email started",
		zap.String("email", input.Email),
	)

	var verificationTokenStr string
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
				return usererrors.ErrUserNotFound
			}
			return err
		}

		// Check if user is already active
		if user.IsActive() {
			uc.logger.Warn(ctx, "user is already active",
				zap.Uint("user_id", user.UserID),
				zap.String("email", input.Email),
			)
			return usererrors.ErrEmailNotVerified // User is already verified
		}

		// Check rate limiting
		canResend, waitDuration, err := uc.tokenService.CanResendVerificationEmail(txCtx, user.UserID)
		if err != nil {
			uc.logger.Error(ctx, "failed to check rate limiting",
				zap.Error(err),
				zap.Uint("user_id", user.UserID),
			)
			return err
		}

		if !canResend {
			uc.logger.Warn(ctx, "rate limit exceeded for resend verification email",
				zap.Uint("user_id", user.UserID),
				zap.Duration("wait_duration", waitDuration),
			)
			return fmt.Errorf("%w: please wait %v before requesting another email", usererrors.ErrTooManyRequests, waitDuration.Round(1))
		}

		// Create new verification token
		verificationToken, err := uc.tokenService.CreateEmailVerificationToken(txCtx, user.UserID)
		if err != nil {
			uc.logger.Error(ctx, "failed to create verification token",
				zap.Error(err),
				zap.Uint("user_id", user.UserID),
			)
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
	if err := uc.emailService.SendVerificationEmail(ctx, userEmail, username, verificationTokenStr); err != nil {
		uc.logger.Error(ctx, "failed to resend verification email",
			zap.Error(err),
			zap.String("email", userEmail),
		)
		return nil, usererrors.ErrEmailSendFailed
	}

	uc.logger.Info(ctx, "verification email resent successfully",
		zap.String("email", userEmail),
	)

	return &ResendVerificationEmailOutput{
		Message: "Verification email has been resent",
	}, nil
}
