package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type ResetPasswordInput struct {
	Token       string
	NewPassword string
}

type ResetPasswordOutput struct {
	Message string
}

type ResetPasswordUseCase struct {
	tokenService service.TokenService
	authService  service.AuthService
	userService  service.UserService
	txManager    db.TxManager
	logger       logger.Logger
}

func NewResetPasswordUseCase(
	tokenService service.TokenService,
	authService service.AuthService,
	userService service.UserService,
	txManager db.TxManager,
	log logger.Logger,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		tokenService: tokenService,
		authService:  authService,
		userService:  userService,
		txManager:    txManager,
		logger:       log,
	}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) (*ResetPasswordOutput, error) {
	uc.logger.Info(ctx, "reset password started",
		zap.String("token_prefix", input.Token[:min(10, len(input.Token))]),
	)

	var userID uint

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Validate new password
		if len(input.NewPassword) < 8 {
			uc.logger.Warn(ctx, "new password too weak",
				zap.Int("password_length", len(input.NewPassword)),
			)
			return service.ErrWeakPassword
		}

		// Validate and use token
		validatedToken, err := uc.tokenService.ValidateAndUseToken(
			txCtx,
			input.Token,
			token.TokenTypePasswordReset,
		)
		if err != nil {
			uc.logger.Error(ctx, "failed to validate reset token",
				zap.Error(err),
				zap.String("token_prefix", input.Token[:min(10, len(input.Token))]),
			)
			return err
		}

		userID = validatedToken.UserID

		// Hash new password
		hashedPassword, err := uc.authService.HashPassword(input.NewPassword)
		if err != nil {
			uc.logger.Error(ctx, "failed to hash new password",
				zap.Error(err),
				zap.Uint("user_id", userID),
			)
			return err
		}

		// Update user password
		if err := uc.userService.UpdatePassword(txCtx, validatedToken.UserID, hashedPassword); err != nil {
			uc.logger.Error(ctx, "failed to update password",
				zap.Error(err),
				zap.Uint("user_id", userID),
			)
			return err
		}

		uc.logger.Info(ctx, "password reset successfully",
			zap.Uint("user_id", userID),
		)

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "reset password completed",
		zap.Uint("user_id", userID),
	)

	return &ResetPasswordOutput{
		Message: "Password has been reset successfully",
	}, nil
}
