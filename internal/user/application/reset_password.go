package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
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
}

func NewResetPasswordUseCase(
	tokenService service.TokenService,
	authService service.AuthService,
	userService service.UserService,
	txManager db.TxManager,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		tokenService: tokenService,
		authService:  authService,
		userService:  userService,
		txManager:    txManager,
	}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) (*ResetPasswordOutput, error) {
	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Validate new password
		if len(input.NewPassword) < 8 {
			return service.ErrWeakPassword
		}

		// Validate and use token
		validatedToken, err := uc.tokenService.ValidateAndUseToken(
			txCtx,
			input.Token,
			token.TokenTypePasswordReset,
		)
		if err != nil {
			return err
		}

		// Hash new password
		hashedPassword, err := uc.authService.HashPassword(input.NewPassword)
		if err != nil {
			return err
		}

		// Update user password
		if err := uc.userService.UpdatePassword(txCtx, validatedToken.UserID, hashedPassword); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ResetPasswordOutput{
		Message: "Password has been reset successfully",
	}, nil
}
