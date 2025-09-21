package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type RegisterUserInput struct {
	Username string
	Password string
	Email    string
	Name     string
}

type RegisterUserOutput struct {
	UserID uint
	Token  string
}

type RegisterUserUseCase struct {
	authService service.AuthService
	txManager   db.TxManager
}

func NewRegisterUserUseCase(authService service.AuthService, txManager db.TxManager) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		authService: authService,
		txManager:   txManager,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	var output *RegisterUserOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		var name *string
		if input.Name != "" {
			name = &input.Name
		}

		// Register user through AuthenticationService
		user, token, err := uc.authService.RegisterUser(txCtx, input.Username, input.Password, input.Email, name)
		if err != nil {
			return err
		}

		output = &RegisterUserOutput{
			UserID: user.UserID,
			Token:  token,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return output, nil
}
