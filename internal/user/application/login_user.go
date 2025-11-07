package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type LoginUserInput struct {
	Email    string
	Password string
}

type LoginUserOutput struct {
	UserID   uint
	Token    string
	Email    string
	Nickname string
}

type LoginUserUseCase struct {
	authService service.AuthService
	logger      logger.Logger
}

func NewLoginUserUseCase(authService service.AuthService, log logger.Logger) *LoginUserUseCase {
	return &LoginUserUseCase{
		authService: authService,
		logger:      log,
	}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error) {
	uc.logger.Info(ctx, "user login started",
		zap.String("email", input.Email),
	)

	// Authenticate user through AuthenticationService
	user, token, err := uc.authService.AuthenticateUser(ctx, input.Email, input.Password)
	if err != nil {
		uc.logger.Error(ctx, "authentication failed",
			zap.Error(err),
			zap.String("email", input.Email),
		)
		return nil, err
	}

	// Prepare output
	output := &LoginUserOutput{
		UserID:   user.UserID,
		Token:    token,
		Email:    user.Email,
		Nickname: user.Nickname,
	}

	uc.logger.Info(ctx, "user login completed",
		zap.Uint("user_id", user.UserID),
		zap.String("email", user.Email),
	)

	return output, nil
}
