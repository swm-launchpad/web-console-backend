package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type VerifyEmailInput struct {
	Token string
}

type VerifyEmailOutput struct {
	UserID uint
	Email  string
}

type VerifyEmailUseCase struct {
	tokenService service.TokenService
	userService  service.UserService
	txManager    db.TxManager
}

func NewVerifyEmailUseCase(
	tokenService service.TokenService,
	userService service.UserService,
	txManager db.TxManager,
) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{
		tokenService: tokenService,
		userService:  userService,
		txManager:    txManager,
	}
}

func (uc *VerifyEmailUseCase) Execute(ctx context.Context, input VerifyEmailInput) (*VerifyEmailOutput, error) {
	var output *VerifyEmailOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Validate and use token
		verificationToken, err := uc.tokenService.ValidateAndUseToken(
			txCtx,
			input.Token,
			token.TokenTypeEmailVerification,
		)
		if err != nil {
			return err
		}

		// Activate user
		if err := uc.userService.ActivateUser(txCtx, verificationToken.UserID); err != nil {
			return err
		}

		// Get user details for response
		user, err := uc.userService.GetUserByID(txCtx, verificationToken.UserID)
		if err != nil {
			return err
		}

		output = &VerifyEmailOutput{
			UserID: user.UserID,
			Email:  user.Email,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return output, nil
}
